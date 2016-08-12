package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	UploadBufferSize = 1024 * 1024 * 4 // 4MB
)

var client = createClient()

func handlePing(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "pong\n")
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	var err error

	err = r.ParseMultipartForm(UploadBufferSize)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	m := r.MultipartForm
	files := m.File["imagedata"]

	for i, _ := range files {
		file, err := files[i].Open()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()

		hash, err := upload(file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "http://localhost:3000/%s", filename)
		return
	}

	http.Error(w, "imagedata is not specified", http.StatusBadRequest)
}

func upload(file multipart.File) (string, error) {
	hasher := md5.New()
	io.Copy(hasher, file)
	hash := hex.EncodeToString(hasher.Sum(nil))
	filename := fmt.Sprintf("%s.png", hash)

	bucketName := os.Getenv("S3_BUCKET_NAME")
	param := &s3.PutObjectInput{
		Bucket: &bucketName,
		Key:    &filename,
		Body:   file,
	}
	_, err := client.PutObject(param)
	if err != nil {
		return "", err
	}

	return hash, nil
}

func createClient() *s3.S3 {
	return s3.New(session.New(), createConfig())
}

func createConfig() *aws.Config {
	config := aws.NewConfig().WithCredentials(credentials.NewEnvCredentials())
	return config
}
