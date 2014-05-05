package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"regexp"

	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
	_ "github.com/go-sql-driver/mysql"
)

const (
	PictureBufferSize = 1024 * 1024 * 4  // 4MB
	Md5BufferSize     = 1024 * 4         // 4KB
)

func picture(c web.C, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	conn, err := GetDbConnection()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	stmt, _ := conn.Prepare("SELECT body FROM pictures WHERE hash = ?")

	var body []byte

	row := stmt.QueryRow(c.URLParams["hash"])
	row.Scan(&body)

	w.Write(body)
}

func picturePage(c web.C, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<img src="/%s.png">`, c.URLParams["hash"])
}

func upload(c web.C, w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(PictureBufferSize)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	m := r.MultipartForm

	files := m.File["imagedata"]

	for i, _ := range files {
		file, err := files[i].Open()
		defer file.Close()

		if err != nil {
			RespondInternalServerError(w, err, "Failed to load uploaded picture")
			return
		}

		hash, err := UploadPictureFile(file)

		if err != nil {
			RespondInternalServerError(w, err, "Failed to upload picture")
			return
		} else {
			fmt.Fprintf(w, "http://localhost:8000/%s", hash)
			return
		}
	}

	RespondBadRequest(w, err, "imagedata is required")
}

func UploadPictureFile(file multipart.File) (string, error) {
	var pictureBuffer []byte

	pictureBuffer, err := ioutil.ReadAll(file)

	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to load uploaded file: %s", err.Error()))
	}

	conn, err := GetDbConnection()

	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to connect to database: %s", err.Error()))
	}

	stmt, err := conn.Prepare("INSERT INTO pictures (`hash`, `user_id`, `body`, `created_at`, `updated_at`) VALUES (?, ?, ?, NOW(), NOW())")

	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed on prepared statement: %s", err.Error()))
	}

	hash, err := GetMd5Hash(pictureBuffer)

	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to calculate MD5: %s", err.Error()))
	}

	result, err := stmt.Exec(hash, "1234", pictureBuffer)

	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to execute prepared statement: %s", err.Error()))
	}

	id, _ := result.LastInsertId()

	fmt.Printf("id = %d\n", id)

	return hash, nil
}

func GetMd5Hash(bytes []byte) (string, error) {
	hasher := md5.New()
	hasher.Write(bytes)
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func GetDbConnection() (*sql.DB, error) {
	return sql.Open("mysql", "root:@/golang_test")
}

func RespondHttpError(w http.ResponseWriter, err error, message string, status int) {
	http.Error(w, err.Error(), status)
	errorMessage := fmt.Sprintf("%s: %s", message, err.Error())
	fmt.Printf("%s\n", errorMessage)
}

func RespondBadRequest(w http.ResponseWriter, err error, message string) {
	RespondHttpError(w, err, message, http.StatusBadRequest)
}

func RespondInternalServerError(w http.ResponseWriter, err error, message string) {
	RespondHttpError(w, err, message, http.StatusInternalServerError)
}

func main() {
	goji.Get(regexp.MustCompile(`^/(?P<hash>[a-z0-9]+)\.png$`), picture)
	goji.Get(regexp.MustCompile(`^/(?P<hash>[a-z0-9]+)$`), picturePage)
	goji.Post("/upload.cgi", upload)
	goji.Serve()
}
