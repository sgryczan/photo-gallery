package utils

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// ExtractDict accepts a parsed query, and returns
// an inboundMMS object with the field values extracted
func ExtractDict(m map[string][]string) (map[string]interface{}, error) {
	dict := map[string]interface{}{}
	mediaURLs := map[string]string{}

	// Convert dictionary to usable struct
	for key, value := range m {
		if key == "NumSegments" || key == "NumMedia" {
			dict[key], _ = strconv.Atoi(value[0])
		}
		if strings.HasPrefix(key, "MediaUrl") {
			log.Printf("Extracting %s", key)
			mediaURLs[key] = value[0]
			continue
		}
		dict[key] = value[0]
	}
	dict["MediaURLs"] = mediaURLs
	return dict, nil
}

// DumpRequestBody dumps the body of an incoming request.
// Pass r.Body to this function
func DumpRequestBody(r io.ReadCloser) io.ReadCloser {
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return nil
	}

	rdr1 := ioutil.NopCloser(bytes.NewBuffer(buf))
	rdr2 := ioutil.NopCloser(bytes.NewBuffer(buf))
	log.Printf("BODY: %q", rdr1)
	return rdr2
}

// DumpRequest dumps the raw headers of a request
func DumpRequest(r *http.Request) error {
	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Println(err)
		return err
	}
	log.Println(string(requestDump))
	return nil
}

func followRedirect(req *http.Request, via []*http.Request) error {
	log.Printf("Redirected to: %s://%s%s", req.URL.Scheme, req.URL.Host, req.URL.Path)
	if strings.HasPrefix(req.URL.Host, "s3-external-") {
		return errors.New("error")
	}

	return nil
}

// FileLocation represents the location of a file in S3
type FileLocation struct {
	Hostname string
	Bucket   string
	Key      string
	FullKey  string
}

// GetFileLocation determines the S3 location where the media is hosted
func GetFileLocation(url string) (*FileLocation, error) {
	client := http.Client{
		CheckRedirect: followRedirect,
	}
	resp, err := client.Get(url)
	if err != nil {
		// If we got a 301/302/307, use the Location method of the response to find the URL we
		// were redirected to
		if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusTemporaryRedirect {
			url, _ := resp.Location()
			locs := &FileLocation{
				Hostname: fmt.Sprintf("%s://%s%s", url.Scheme, url.Host, url.Path),
				Bucket:   strings.Split(url.Path, "/")[1],
				// TODO this may break eventually, find a cleaner way of obtaining the file key
				Key:     strings.SplitAfterN(url.Path, "/", 4)[3],
				FullKey: url.Path,
			}
			return locs, nil
		}
		log.Print(err)
		return nil, err
	}

	return nil, errors.New("Unable to find the file location")
}

// GetFileBytes downloads the contents of a media file into a buffer
func GetFileBytes(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// BytesToReader converts a byte slice to an io.Reader
func BytesToReader(b []byte) io.Reader {
	return bytes.NewReader(b)
}

// S3CopyFile copies a file between 2 buckets
func S3CopyFile(input *s3.CopyObjectInput) error {
	svc := s3.New(session.New())
	result, err := svc.CopyObject(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeObjectNotInActiveTierError:
				log.Println(s3.ErrCodeObjectNotInActiveTierError, aerr.Error())
			default:
				log.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Println(err.Error())
		}
		return nil
	}
	log.Println(result)
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

// IsWhiteListed determines if the sending number is allowed to post
func IsWhiteListed(number string, allowed *[]string) bool {
	for _, n := range *allowed {
		if n == number {
			return true
		}
	}
	return false
}

// InvokeUpdate invokes the update API
func InvokeUpdate(url string) error {
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Printf(string(buf))
	return nil
}
