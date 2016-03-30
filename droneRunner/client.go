package main

import (
	"bytes"
	"net/http"
	"net/url"
)

func clientPostConnect(host string) (*http.Response, error) {
	// Build URL
	u := new(url.URL)
	u.Scheme = "http"
	u.Host = host + ":9000"
	u.Path = "/connect"

	// Build request
	buf := []byte(`{"droneID":"1"}`)
	req, err := http.NewRequest("POST", u.String(), bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}

	// Send the build request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
