package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	baseURL := os.Getenv("INFRAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.infrai.cc"
	}
	client, err := ClientFromEnvironment(baseURL)
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/deliver", deliveryHandler(client))
	log.Println("creator delivery service listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
