package models

// InboundMMS represents an inbound MMS from Twilio
// swagger:model
type InboundMMS struct {
	ToCountry         string
	MediaContentType0 string
	ToState           string
	SMSMessageSid     string
	NumMedia          string
	ToCity            string
	FromZip           string
	Body              string
	FromCountry       string
	To                string
	ToZip             string
	NumSegments       string
	MessageSid        string
	AccountSid        string
	From              string
	MediaURL0         string
	APIVersion        string
}

// InboundMMSQuery blah
// swagger:model
type InboundMMSQuery string
