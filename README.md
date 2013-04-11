elastik
=======
elastik is a library in Go that allows the creation of an architecture of HTTP servers that grows and shrinks at run time. It uses [Maestro] (http://github.com/trajber/maestro) as http load balancer.

## On load balancer

	package main

	import (
		"elastik"
		"log"
		"net/http"
	)

	func main() {
		// these are the UDP ports for send and receive heartbeat messages
		balancer := elastik.NewBalancer(9988, 8899)
		go func() {
			balancer.ListenIncomingHeartbeats()
		}()

		log.Println("Load balancer listen on port 8080")

		log.Fatal(http.ListenAndServe(":8080", balancer))
	}

## On instances

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
		// these are the UDP ports for send and receive heartbeat messages
		instance := elastik.NewInstance(8899, 9988)
		u, _ := url.Parse("http://myhost:8081")
		// I must send the URL where I'm serving HTTP
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
