// THANKS: https://github.com/fiatjaf/go-lnurl/blob/d50a8e916232580895822178fe36e0f5cf400554/base.go
// only using the LNURL types here
package lnurl

import "net/url"

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
