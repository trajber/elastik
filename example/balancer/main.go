package main

import (
	"elastik"
	"log"
	"net/http"
)

func main() {
	balancer := elastik.NewBalancer(9988, 8899)
	go func() {
		balancer.ListenIncomingHeartbeats()
	}()

	log.Println("Load balancer listen on port 8080")

	log.Fatal(http.ListenAndServe(":8080", balancer))
}
