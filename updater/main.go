package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gorilla/mux"
)

// SET the following vars
// S3_BUCKET
// AWS_ACCESS_KEY_ID
// AWS_SECRET_ACCESS_KEY
// AWS_REGION

// PhotoBucket is the S3 bucket from which files will be read
var PhotoBucket string

// SiteBucket hosts the static files for the website
var SiteBucket string

var listenPort = flag.Int("port", 8080, "Port to listen on")

func main() {

	PhotoBucket = os.Getenv("PHOTO_BUCKET")
	SiteBucket = os.Getenv("SITE_BUCKET")
	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		log.Printf("AWS_REGION not set. Defaulting to us-east-1")
		os.Setenv("AWS_REGION", "us-east-1")
	}
	if PhotoBucket == "" {
		log.Fatalf("PHOTO_BUCKET environment variable not set!")
	}
	if SiteBucket == "" {
		log.Fatalf("SITE_BUCKET environment variable not set!")
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

	fmt.Printf("AWS Region: %s\n", awsRegion)

	r.HandleFunc("/update", UpdateHandler)

	srv := &http.Server{
		Handler:      r,
		Addr:         ":" + strconv.Itoa(*listenPort),
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

// S3ListObjects lists the photos in the destination bucket
func S3ListObjects(bucket string) (*s3.ListObjectsV2Output, error) {
	svc := s3.New(session.New())
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String("photos/"),
	}

	result, err := svc.ListObjectsV2(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket:
				fmt.Println(s3.ErrCodeNoSuchBucket, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return result, nil
	}

	//fmt.Printf("Result: %+v", result)
	return result, nil
}

// S3GetMetadata returns the metadata of an object
func S3GetMetadata(key string) (*s3.HeadObjectOutput, error) {
	svc := s3.New(session.New())
	input := &s3.HeadObjectInput{
		Bucket: aws.String(PhotoBucket),
		Key:    aws.String(key),
	}

	result, err := svc.HeadObject(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
	}
	//fmt.Println(result)
	return result, nil
}

// ParseObjects returns the parsed figure lines
func ParseObjects(o *s3.ListObjectsV2Output) ([]string, error) {
	result := []string{}
	contents := o.Contents
	for _, o := range contents {
		if *o.Key == "photos/" {
			continue
		}
		obj, err := S3GetMetadata(*o.Key)
		if err != nil {
			fmt.Println(err)
		}
		//fmt.Println(obj.Metadata)
		var caption string
		if c := obj.Metadata["Caption"]; c != nil {
			caption = *c
		}
		line := fmt.Sprintf(`{{< figure link="https://files.czan.io/%s" caption="%s" >}}`, *o.Key, caption)
		result = append(result, line)
	}
	return result, nil
}

// GenerateManifest creates the Hugo manifest
func GenerateManifest(objects []string) {
	filename := "content/_index.md"
	manifest := []string{
		`####`,
		"",
		`{{< gallery >}}`,
	}
	manifest = append(manifest, objects...)
	manifest = append(manifest, `{{< /gallery >}}`)
	for _, line := range manifest {
		fmt.Printf("%s\n", line)
	}
	if _, err := os.Stat(filename); err == nil {
		_ = os.Remove(filename)
	}
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	datawriter := bufio.NewWriter(file)

	for _, data := range manifest {
		_, _ = datawriter.WriteString(data + "\n")
	}

	datawriter.Flush()
	file.Close()
}

// HugoMinify runs hugo --minify
func HugoMinify() {
	cmd := exec.Command("hugo", "--minify")
	log.Println("Running Hugo..")
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
	}
}

// UploadIndex uploads the index file to S3
func UploadIndex() error {
	filename := "public/index.html"
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file %q, %v", filename, err)
	}

	contentType := "text/html"
	uploadInput := &s3manager.UploadInput{
		Body:        f,
		Bucket:      aws.String(SiteBucket),
		Key:         aws.String("index.html"),
		ACL:         aws.String("public-read"),
		ContentType: &contentType,
	}
	err = S3UploadFile(uploadInput)
	if err != nil {
		fmt.Println(err)
	}
	return nil
}

// S3UploadFile writes a file to a destination bucket
func S3UploadFile(input *s3manager.UploadInput) error {
	sess := session.Must(session.NewSession())
	uploader := s3manager.NewUploader(sess)
	result, err := uploader.Upload(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return nil
	}
	log.Println(result)
	return nil
}

// UpdateHandler accepts inbound MMS messages and copies media to S3
func UpdateHandler(w http.ResponseWriter, r *http.Request) {

	// List all files in the Bucket
	objects, err := S3ListObjects(PhotoBucket)
	if err != nil {
		fmt.Println(err)
	}

	// For each file, grab the name and metadata, and add it to a slice of string
	lines, err := ParseObjects(objects)

	GenerateManifest(lines)
	HugoMinify()
	UploadIndex()

	w.WriteHeader(http.StatusOK)
	//json, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Fprintf(w, "%s", "OK")
}
