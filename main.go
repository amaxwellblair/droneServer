package main

import "net/http"

func main() {
	h := NewHandler()
	http.ListenAndServe(":9000", h)
}
