package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"mime/multipart"
	"net/http"
	"regexp"

	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
	_ "github.com/go-sql-driver/mysql"
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
	err := r.ParseMultipartForm(1024 * 100)

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
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
		}

		buf := make([]byte, 1024 * 100)

		n, _ := file.Read(buf)

		fmt.Printf("%d\n", n)

		conn, err := GetDbConnection()

		if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				fmt.Printf("Error on DB connection\n")
				return
		}

		stmt, err := conn.Prepare("INSERT INTO pictures (`hash`, `user_id`, `body`, `created_at`, `updated_at`) VALUES (?, ?, ?, NOW(), NOW())")

		if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				fmt.Printf("Error on prepared statement\n")
				return
		}

		hash := GetMd5Hash(file)
		result, err := stmt.Exec(hash, "1234", buf)

		if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				fmt.Printf("Error on query execution\n")
				fmt.Printf(err.Error() + "\n")
				return
		}

		id, err := result.LastInsertId()

		fmt.Printf("ID = %d\n", id)

		fmt.Fprintf(w, "http://localhost:8000/%s", hash)
	}
}

const (
	md5_buf_length = 4096;
)

func GetMd5Hash(file multipart.File) string {
	file.Seek(0, 0)
	hasher := md5.New()
	buf := make([]byte, md5_buf_length)

	for {
		n, _ := file.Read(buf)

		if n == 0 {
			break
		}

		hasher.Write(buf[0:n])
	}

	file.Seek(0, 0)

	return hex.EncodeToString(hasher.Sum(nil))
}

func GetDbConnection() (*sql.DB, error) {
	return sql.Open("mysql", "root:@/golang_test")
}

func main() {
	goji.Get(regexp.MustCompile(`^/(?P<hash>[a-z0-9]+)\.png$`), picture)
	goji.Get(regexp.MustCompile(`^/(?P<hash>[a-z0-9]+)$`), picturePage)
	goji.Post("/upload.cgi", upload)
	goji.Serve()
}
