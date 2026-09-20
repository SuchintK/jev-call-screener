package twilio

import (
	"bytes"
	"encoding/xml"
)

const contentTypeXML = "application/xml; charset=utf-8"

type twimlResponse struct {
	XMLName xml.Name  `xml:"Response"`
	Gather  *gather   `xml:"Gather,omitempty"`
	Say     *string   `xml:"Say,omitempty"`
	Dial    *string   `xml:"Dial,omitempty"`
	Hangup  *struct{} `xml:"Hangup,omitempty"`
}

type gather struct {
	Input               string `xml:"input,attr"`
	Action              string `xml:"action,attr"`
	Method              string `xml:"method,attr"`
	ActionOnEmptyResult bool   `xml:"actionOnEmptyResult,attr"`
	Say                 string `xml:"Say"`
}

func gatherXML(message string) []byte {
	return marshalTwiML(twimlResponse{Gather: &gather{
		Input: "speech", Action: "/webhooks/twilio/gather", Method: "POST", ActionOnEmptyResult: true, Say: message,
	}})
}

func rejectXML(message string) []byte {
	return marshalTwiML(twimlResponse{Say: &message, Hangup: &struct{}{}})
}

func forwardXML(number string) []byte {
	return marshalTwiML(twimlResponse{Dial: &number})
}

func marshalTwiML(response twimlResponse) []byte {
	var buffer bytes.Buffer
	buffer.WriteString(xml.Header)
	encoder := xml.NewEncoder(&buffer)
	_ = encoder.Encode(response)
	return buffer.Bytes()
}
