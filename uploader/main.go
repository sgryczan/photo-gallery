package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/sgryczan/photoGallery/uploader/handlers"
)

var (
	listenPort = flag.Int("port", 8080, "Port to listen on")
)

// AWS_ACCESS_KEY_ID and
// AWS_SECRET_ACCESS_KEY
// AWS_REGION
func main() {

	handlers.DestinationBucket = os.Getenv("S3_BUCKET")
	handlers.GalleryUpdateURL = os.Getenv("UPDATE_API_URL")
	awsRegion := os.Getenv("AWS_REGION")

	if awsRegion == "" {
		log.Printf("AWS_REGION not set. Defaulting to us-east-1")
		os.Setenv("AWS_REGION", "us-east-1")
	}
	if handlers.DestinationBucket == "" {
		log.Fatalf("S3_BUCKET environment variable not set!")
	}
	if handlers.GalleryUpdateURL == "" {
		log.Fatalf("UPDATE_API_URL environment variable not set!")
	}
	if key := os.Getenv("AWS_ACCESS_KEY_ID"); key == "" {
		log.Fatalf("AWS_ACCESS_KEY_ID not set!")
	}
	if key := os.Getenv("AWS_SECRET_ACCESS_KEY"); key == "" {
		log.Fatalf("AWS_SECRET_ACCESS_KEY not set!")
	}

	// Grab Destination Bucket from Environment
	// Grab AWS Credentials from Environment
	r := mux.NewRouter()

	fmt.Printf("Started photo-uploader v%s\n", handlers.Version)
	fmt.Printf("AWS Region: %s\n", awsRegion)

	r.HandleFunc("/", handlers.HomeHandler)
	r.HandleFunc("/about", handlers.AboutHandler)
	r.HandleFunc("/sms", handlers.SMSHandler)

	sh := http.StripPrefix("/api",
		http.FileServer(http.Dir("./swaggerui/")))
	r.PathPrefix("/api/").Handler(sh)

	srv := &http.Server{
		Handler:      r,
		Addr:         ":" + strconv.Itoa(*listenPort),
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}
