package main

import (
	"elastik"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
)

func main() {
	instance := elastik.NewInstance(8899, 9988)
	u, _ := url.Parse("http://myhost:8081")
	instance.AddURL(u)

	go func() {
		instance.ListenIncomingHeartbeats()
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	log.Println("HTTP server listen on port 8081")

	log.Fatal(http.ListenAndServe(":8081", nil))
}
