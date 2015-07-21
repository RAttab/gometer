// Copyright (c) 2014 Datacratic. All rights reserved.

package meter

import (
	"github.com/RAttab/goklog/klog"
	"github.com/RAttab/gorest/rest"

	"net/http"
	"net/url"
	"sync"
	"time"
)

// HTTPHandler is used forward the recorded meter values to a remote HTTP
// endpoint.
type HTTPHandler struct {

	// URL is the remote HTTP endpoint where meter values should be sent
	// to. This field must be set.
	URL string

	// Method is the HTTP method used when forwarding over HTTP. This field must
	// be set.
	Method string

	// HTTPClient can be used to optionally customize the HTTPClient used to
	// make the HTTP requests.
	HTTPClient *http.Client

	initialize sync.Once
	client     *rest.Client
}

// Init initializes the object. Note that calling this is optional in which case
// the object lazily initializes itself as needed.
func (handler *HTTPHandler) Init() {
	handler.initialize.Do(handler.init)
}

func (handler *HTTPHandler) init() {
	if handler.URL == "" {
		klog.KFatal("meter.http.init.error", "no URL configured")
	}

	if handler.Method == "" {
		klog.KFatal("meter.http.init.error", "no HTTP method configured")
	}

	if _, err := url.Parse(handler.URL); err != nil {
		klog.KFatalf("meter.http.init.error", "invalid URL '%s': %s", handler.URL, err)
	}

	if handler.HTTPClient == nil {
		handler.HTTPClient = http.DefaultClient
	}

	handler.client = &rest.Client{
		Client: handler.HTTPClient,
		Root:   handler.URL,
	}
}

// HandleMeters sends the given values to the configured remote HTTP endpoint.
func (handler *HTTPHandler) HandleMeters(values map[string]float64) {
	handler.Init()

	var body struct {
		Timestamp int64              `json:"timestamp"`
		Values    map[string]float64 `json:"values"`
	}

	body.Timestamp = time.Now().Unix()
	body.Values = values

	resp := handler.client.NewRequest(handler.Method).SetBody(body).Send()

	if err := resp.GetBody(nil); err != nil {
		klog.KPrintf("meter.http.send.error", "unable to send metrics: %s", err)
	}
}
