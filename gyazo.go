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
	"time"

	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
	_ "github.com/go-sql-driver/mysql"
)

const (
	PictureBufferSize = 1024 * 1024 * 4 // 4MB

	PictureNotFoundError = 0
	InternalMysqlError   = 1
)

type Picture struct {
	Id        int
	Hash      string
	UserId    string
	Body      []byte
	CreatedAt time.Time
	UpdatedAt time.Time
}

type PictureFetchingError struct {
	s    string
	code int
}

func (err *PictureFetchingError) Error() string {
	return err.s
}

func (err *PictureFetchingError) IsPictureNotFound() bool {
	return err.code == PictureNotFoundError
}

func (err *PictureFetchingError) IsInternalMysqlError() bool {
	return err.code == InternalMysqlError
}

func NewPictureFetchingError(s string, code int, err error) *PictureFetchingError {
	return &PictureFetchingError{fmt.Sprintf("%s: %s", s, err.Error()), code}
}

func PngPicture(c web.C, w http.ResponseWriter, r *http.Request) {
	picture, err := FetchPictureByHash(c.URLParams["hash"])

	if err != nil {
		if err.IsPictureNotFound() {
			RespondNotFound(w, err, "No picture is found")
			return
		} else {
			RespondInternalServerError(w, err, "Failed to fetch Picture")
			return
		}
	}

	w.Header().Set("Content-Type", "image/png")
	w.Write(picture.Body)
}

func PicturePage(c web.C, w http.ResponseWriter, r *http.Request) {
	picture, err := FetchPictureByHash(c.URLParams["hash"])

	if err != nil {
		if err.IsPictureNotFound() {
			RespondNotFound(w, err, "No picture is found")
			return
		} else {
			RespondInternalServerError(w, err, "Failed to fetch Picture")
			return
		}
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<img src="/%s.png">`, picture.Hash)
}

func FetchPictureByHash(hashToFind string) (*Picture, *PictureFetchingError) {
	conn, err := GetDbConnection()
	defer conn.Close()

	if err != nil {
		return nil, NewPictureFetchingError("Failed to connect to database", InternalMysqlError, err)
	}

	stmt, err := conn.Prepare("SELECT id, hash, user_id, body, created_at, updated_at FROM pictures WHERE hash = ?")

	if err != nil {
		return nil, NewPictureFetchingError("Failed on prepared statement", InternalMysqlError, err)
	}

	row := stmt.QueryRow(hashToFind)

	var Id        int
	var Hash      string
	var UserId    string
	var Body      []byte
	var CreatedAt time.Time
	var UpdatedAt time.Time

	err = row.Scan(&Id, &Hash, &UserId, &Body, &CreatedAt, &UpdatedAt)

	if err != nil {
		return nil, NewPictureFetchingError("No picture is found", PictureNotFoundError, err)
	}

	return &Picture{Id, Hash, UserId, Body, CreatedAt, UpdatedAt}, nil
}

func Upload(c web.C, w http.ResponseWriter, r *http.Request) {
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

	_, err = stmt.Exec(hash, "1234", pictureBuffer)

	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to execute prepared statement: %s", err.Error()))
	}

	return hash, nil
}

func GetMd5Hash(bytes []byte) (string, error) {
	hasher := md5.New()
	hasher.Write(bytes)
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func GetDbConnection() (*sql.DB, error) {
	return sql.Open("mysql", "root:@/golang_test?parseTime=true")
}

func RespondHttpError(w http.ResponseWriter, err error, message string, status int) {
	http.Error(w, err.Error(), status)
	errorMessage := fmt.Sprintf("%s: %s", message, err.Error())
	fmt.Printf("%s\n", errorMessage)
}

func RespondBadRequest(w http.ResponseWriter, err error, message string) {
	RespondHttpError(w, err, message, http.StatusBadRequest)
}

func RespondNotFound(w http.ResponseWriter, err error, message string) {
	RespondHttpError(w, err, message, http.StatusNotFound)
}

func RespondInternalServerError(w http.ResponseWriter, err error, message string) {
	RespondHttpError(w, err, message, http.StatusInternalServerError)
}

func main() {
	goji.Get(regexp.MustCompile(`^/(?P<hash>[a-z0-9]+)\.png$`), PngPicture)
	goji.Get(regexp.MustCompile(`^/(?P<hash>[a-z0-9]+)$`), PicturePage)
	goji.Post("/upload.cgi", Upload)
	goji.Serve()
}
