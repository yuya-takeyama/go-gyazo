package main

import (
	"github.com/yuya-takeyama/go-gyazo/picture"
	"github.com/yuya-takeyama/go-gyazo/picture_fetching_error"

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
)

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

func FetchPictureByHash(hashToFind string) (*picture.Picture, *picture_fetching_error.PictureFetchingError) {
	conn, err := GetDbConnection()
	defer conn.Close()

	if err != nil {
		return nil, picture_fetching_error.New("Failed to connect to database", picture_fetching_error.InternalMysqlError, err)
	}

	stmt, err := conn.Prepare("SELECT id, hash, user_id, body, created_at, updated_at FROM pictures WHERE hash = ?")

	if err != nil {
		return nil, picture_fetching_error.New("Failed on prepared statement", picture_fetching_error.InternalMysqlError, err)
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
		return nil, picture_fetching_error.New("No picture is found", picture_fetching_error.PictureNotFoundError, err)
	}

	return &picture.Picture{Id, Hash, UserId, Body, CreatedAt, UpdatedAt}, nil
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
