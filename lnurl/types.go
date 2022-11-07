// THANKS: https://github.com/fiatjaf/go-lnurl/blob/d50a8e916232580895822178fe36e0f5cf400554/base.go
// only using the LNURL types here
package lnurl

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
)

type LNURLResponse struct {
	Status string `json:"status,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type LNURLPayResponse1 struct {
	LNURLResponse
	Callback        string   `json:"callback"`
	CallbackURL     *url.URL `json:"-"`
	Tag             string   `json:"tag"`
	MaxSendable     int64    `json:"maxSendable"`
	MinSendable     int64    `json:"minSendable"`
	EncodedMetadata string   `json:"metadata"`
	Metadata        Metadata `json:"-"`
	CommentAllowed  int64    `json:"commentAllowed"`
}

type LNURLPayResponse2 struct {
	LNURLResponse
	SuccessAction *SuccessAction `json:"successAction"`
	Routes        [][]RouteInfo  `json:"routes"`
	PR            string         `json:"pr"`
	Disposable    bool           `json:"disposable,omitempty"`
}

type RouteInfo struct {
	NodeId        string `json:"nodeId"`
	ChannelUpdate string `json:"channelUpdate"`
}

type SuccessAction struct {
	Tag         string `json:"tag"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
	Message     string `json:"message,omitempty"`
	Ciphertext  string `json:"ciphertext,omitempty"`
	IV          string `json:"iv,omitempty"`
}

type LNURLErrorResponse struct {
	Status string   `json:"status,omitempty"`
	Reason string   `json:"reason,omitempty"`
	URL    *url.URL `json:"-"`
}

type Metadata [][]string

func (metadata Metadata) Identifier(identifier string) Metadata {
	return append(metadata, []string{"text/identifier", identifier})
}

func (metadata Metadata) Description(description string) Metadata {
	return append(metadata, []string{"text/plain", description})
}

func (metadata Metadata) Thumbnail(imageData []byte) Metadata {
	if len(imageData) == 0 {
		return metadata
	}

	mimeType := http.DetectContentType(imageData)
	imageDataBase64 := base64.StdEncoding.EncodeToString(imageData)

	return append(metadata, []string{mimeType + ";base64", imageDataBase64})
}

func (metadata Metadata) String() string {
	bytes, _ := json.Marshal(metadata)

	return string(bytes)
}
