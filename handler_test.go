package main_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/amaxwellblair/droneServer"
)

func TestHandler_Connect_ErrMethodNotAllowed(t *testing.T) {
	h := main.NewHandler()
	req, err := http.NewRequest("GET", buildURL("/connect").String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if b := w.Body.String(); b != "method not allowed\n" {
		t.Fatalf("unexpected error: %s", b)
	}
}

func TestHandler_Actions_ErrMethodNotAllowed(t *testing.T) {
	h := main.NewHandler()
	req, err := http.NewRequest("PATCH", buildURL("/actions").String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if b := w.Body.String(); b != "method not allowed\n" {
		t.Fatalf("unexpected error: %s", b)
	}
}

func TestHandler_ErrPath(t *testing.T) {
	h := main.NewHandler()
	req, err := http.NewRequest("GET", buildURL("/checkplus").String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if b := w.Body.String(); b != "not found\n" {
		t.Fatalf("unexpected error: %s", b)
	}
}

func TestHandler_ServeHTTP_ValidateRoutePostActions(t *testing.T) {
	h := main.NewHandler()
	req, err := http.NewRequest("POST", buildURL("/actions").String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if b := w.Body.String(); b == "method not allowed" || b == "not found" {
		t.Fatalf("unexpected method: %s", b)
	}
}

func TestHandler_ServeHTTP_ValidateRouteGetActions(t *testing.T) {
	h := main.NewHandler()
	req, err := http.NewRequest("GET", buildURL("/actions").String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if b := w.Body.String(); b == "method not allowed" || b == "not found" {
		t.Fatalf("unexpected method: %s", b)
	}
}

func TestHandler_postConnectHandler(t *testing.T) {
	s, h := NewServerHandler()
	defer s.Close()

	buf, err := json.Marshal(&main.DroneRequest{
		DroneID: "1",
	})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("POST", s.URL+"/connect", bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}

	c := make(chan *http.Response, 1)
	go LongPollClient(req, c, t)

	// Waits for the queue to be filled - not the most eloquent solution!
	for len(h.Queue) == 0 {
		time.Sleep(100)
	}
	d := h.Queue[0]
	if err != nil {
		t.Fatal(err)
	}
	d.Wait <- new(struct{})

	buf, err = json.Marshal(&main.ActionsRequest{
		ItemID:  "1",
		Actions: []string{"deploy"},
	})
	if err != nil {
		t.Fatal(err)
	}
	req, err = http.NewRequest("POST", s.URL+"/actions", bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}
	_, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	resp := <-c

	var ar main.ActionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		t.Fatal(err)
	}

	if ar.ItemID != 1 {
		t.Fatalf("Unexpected item ID: %d", ar.ItemID)
	} else if ar.Actions[0] != "deploy" {
		t.Fatalf("Unexpected actions: %s", ar.Actions[0])
	} else if resp.StatusCode != 200 {
		t.Fatalf("Unexpected status code: %d", resp.StatusCode)
	}
}

func TestHandler_postActionsHandler_NoDroneAvailable(t *testing.T) {
	s, _ := NewServerHandler()
	defer s.Close()

	// Build request
	buf, err := json.Marshal(&main.ActionsRequest{
		ItemID:  "1",
		Actions: []string{"deploy"},
	})
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", s.URL+"/actions", bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var body []byte
	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 500 {
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	} else if b := string(body); b != "no available drones\n" {
		t.Fatalf("unexpected error: %s", b)
	}
}

func TestHandler_PopDrone(t *testing.T) {
	h := main.NewHandler()

	h.Queue = append(h.Queue, main.NewDrone(1))
	h.Queue = append(h.Queue, main.NewDrone(2))
	h.Queue = append(h.Queue, main.NewDrone(3))
	h.Queue = append(h.Queue, main.NewDrone(4))
	d, err := h.PopDrone(1)
	if err != nil {
		t.Fatal(err)
	} else if d.DroneID != 1 {
		t.Fatalf("unexpected id: %d", d.DroneID)
	} else if l := len(h.Queue); l != 3 {
		t.Fatalf("unexpected length: %d", l)
	}
}

func TestHandler_PopDrone_NoDronesToPop(t *testing.T) {
	h := main.NewHandler()

	d, err := h.PopDrone(1)
	if err.Error() != "no drone for this ID" {
		t.Fatal(err)
	} else if d != nil {
		t.Fatalf("unexpected drone: %#v", d)
	}
}

func buildURL(path string) *url.URL {
	// Build the URL
	u := url.URL{}
	u.Scheme = "http"
	u.Host = "localhost:9000"
	u.Path = path
	return &u
}

func NewServerHandler() (*httptest.Server, *main.Handler) {
	h := main.NewHandler()
	return httptest.NewServer(h), h
}

func LongPollClient(req *http.Request, respChannel chan *http.Response, t *testing.T) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	respChannel <- resp
}
