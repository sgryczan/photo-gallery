package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/sgryczan/photoGallery/uploader/models"
	"github.com/sgryczan/photoGallery/uploader/utils"
)

var DestinationBucket string
var GalleryUpdateURL string
var S3AccessKey string
var S3SecretKeyID string
var AllowedSenders []string

// SMSHandler accepts inbound MMS messages and copies media to S3
func SMSHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /sms SMS sms
	//
	// Upload a new photo via MMS
	// ---
	// consumes:
	// - application/json
	// produces:
	// - text/plain
	// parameters:
	// - name: payload
	//   in: body
	//   description: MMS Message from Twilio
	//   required: true
	//   example: ToCountry=US&MediaContentType0=image%2Fjpeg&ToState=CO&SmsMessageSid=MM0ab821fa95eeecea1eadd2f9d2414997&NumMedia=1&ToCity=&FromZip=80204&SmsSid=MM0ab821fa95eeecea1eadd2f9d2414997&FromState=CO&SmsStatus=received&FromCity=DENVER&Body=Test&FromCountry=US&To=%2B18881112233&ToZip=&NumSegments=1&MessageSid=MM0ab821fa95eeecea1eadd2f9d2414997&AccountSid=AC51065da8ef8171360073ba0023137ba3&From=%2B17208884444&MediaUrl0=https%3A%2F%2Fapi.twilio.com%2F2010-04-01%2FAccounts%2FAC51065da8ef8171360073ba0023137ba3%2FMessages%2FMM0ab821fa95eeecea1eadd2f9d2414997%2FMedia%2FMEcf6f27737f63106f76c895195ade7a29&ApiVersion=2010-04-01
	//   #schema:
	//   # "$ref": "#/definitions/InboundMMSQuery"
	// responses:
	//   '200':
	//     description: Success
	//     type: string
	//   '400':
	//     description: Bad Request
	//     type: string
	//   '401':
	//     description: Unauthorized
	//     type: string
	//   '500':
	//     description: Server Error
	//     type: string

	var resp string
	// Dump the request body (for debugging only)
	r.Body = utils.DumpRequestBody(r.Body)
	buf, bodyErr := ioutil.ReadAll(r.Body)
	if bodyErr != nil {
		log.Print("bodyErr ", bodyErr.Error())
		http.Error(w, bodyErr.Error(), http.StatusInternalServerError)
		return
	}

	// Twilio passes the POST body as a URLEncoded Query string
	// Here we will parse the query string into a map,
	// then convert the map to JSON,
	// then finally convert the JSON into a struct
	params, err := url.ParseQuery(string(buf))
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Println("Query Params: ")
	for key, value := range params {
		log.Printf("  %v = %v\n", key, value)
	}

	cleanParams, _ := utils.ExtractDict(params)

	// Convert the dict to a json string
	log.Printf("Encoded JSON:")
	js, _ := json.Marshal(cleanParams)
	log.Println(string(js))

	// Convert the JSON string to a struct
	inboundMMS := &models.InboundMMS{}
	err = json.Unmarshal(js, &inboundMMS)
	if err != nil {
		log.Print(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error decoding body: %s", err.Error())
		return
	}
	log.Printf("InboundMMS: %+v", inboundMMS)

	// Save a copy of this request for debugging.
	//utils.DumpRequest(r)

	// Is the sender authorized?
	// Must return 200 for Twilio to relay the message back to the sender
	if allowed := utils.IsWhiteListed(inboundMMS.From, &AllowedSenders); allowed != true {
		resp = `<?xml version="1.0" encoding="UTF-8"?>
		<Response>
		<Message>Sorry, this number is not allowed!</Message>
		</Response>`

		log.Printf("Sender %s is not allowed\n", inboundMMS.From)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s", resp)
		return
	}

	// Does the request contain media?
	// Must return 200 for Twilio to relay the message back to the sender
	if inboundMMS.NumMedia == 0 {
		resp = `<?xml version="1.0" encoding="UTF-8"?>
		<Response>
		<Message>Your message didn't contain any media!</Message>
		</Response>`

		log.Println("Message didn't contain any media.")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s", resp)
		return
	}

	for i := 0; i < inboundMMS.NumMedia; i++ {

		log.Printf("Processing image %d/%d", i+1, inboundMMS.NumMedia)
		// Copy image from source S3 bucket to destination S3 Bucket
		key := fmt.Sprintf("MediaUrl%d", i)
		mediaLocation, _ := utils.GetFileLocation(inboundMMS.MediaURLs[key])
		log.Printf("Media URL: %s", mediaLocation.Hostname)
		log.Printf("Media Bucket: %s", mediaLocation.Bucket)
		log.Printf("Media Key: %s", mediaLocation.Key)

		log.Printf("Media FullKey: %s", mediaLocation.FullKey)

		file, err := utils.GetFileBytes(mediaLocation.Hostname)
		if err != nil {
			log.Print(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Error Reading file from S3: %s", err.Error())
			return
		}

		// We need to set the Content-Type to make sure clients decode it as an image
		contentType := http.DetectContentType(file)

		var caption string
		if inboundMMS.NumMedia > 1 {
			caption = fmt.Sprintf("%s (%d/%d)", inboundMMS.Body, i+1, inboundMMS.NumMedia)
		} else {
			caption = inboundMMS.Body
		}

		uploadInput := &s3manager.UploadInput{
			Body:        utils.BytesToReader(file),
			Bucket:      aws.String(DestinationBucket),
			Key:         aws.String(fmt.Sprintf("photos/%s", mediaLocation.Key)),
			ACL:         aws.String("public-read"),
			ContentType: &contentType,
			Metadata: map[string]*string{
				"caption": &caption,
			},
		}

		err = utils.S3UploadFile(uploadInput)
		if err != nil {
			log.Print(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Error copying to S3: %s", err.Error())
			return
		}
	}

	if inboundMMS.NumMedia == 1 {
		resp = `<?xml version="1.0" encoding="UTF-8"?>
			<Response>
			<Message>Photo uploaded successfully!</Message>
			</Response>`
	} else {
		resp = fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
			<Response>
			<Message>%d photos uploaded successfully!</Message>
			</Response>`, inboundMMS.NumMedia)
	}

	go utils.InvokeUpdate(GalleryUpdateURL)
	w.WriteHeader(http.StatusOK)
	//json, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Fprintf(w, "%s", resp)
}
