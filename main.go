package main

import (
	"fmt"
	"net/http"
	"os"
)

var port string

func main() {
	/*
		client := createClient()
		bucketName := "yuyat-s3-sandbox"
		params := &s3.ListObjectsInput{
			Bucket: &bucketName,
		}
		err := client.ListObjectsPages(params, func(p *s3.ListObjectsOutput, lastPage bool) bool {
			pp.Println(p)
			pp.Println(lastPage)

			return true
		})
		if err != nil {
			panic(err)
		}
	*/

	envPort := os.Getenv("PORT")
	if envPort != "" {
		port = envPort
	} else {
		port = "3000"
	}

	addr := fmt.Sprintf("0.0.0.0:%s", port)

	handle("/ping", handlePing)
	handle("/upload.cgi", routeByMethods(methodHandlerMap{"POST": handleUpload}))
	fmt.Printf("Listening on %s...\n", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		panic(err)
	}
}
