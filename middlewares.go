package main

import (
	"fmt"
	"net/http"
)

func handle(path string, handler http.HandlerFunc) {
	http.HandleFunc(path, logger(handler))
}

type myResponseWriter struct {
	status int
	http.ResponseWriter
}

func (w *myResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func logger(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wrappedWriter := &myResponseWriter{http.StatusOK, w}
		handler(wrappedWriter, r)
		fmt.Printf("%s %s %d\n", r.Method, r.URL, wrappedWriter.status)
	}
}

type methodHandlerMap map[string]http.HandlerFunc

func routeByMethods(handlers methodHandlerMap) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if handler, ok := handlers[r.Method]; ok {
			handler(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(w, "405: Method Not Allowed: %s is not allowed for %s\n", r.Method, r.URL)
		}
	}
}
