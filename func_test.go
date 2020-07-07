package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMyHandler(t *testing.T) {
	handler := &MyHandler{sem: make(chan struct{}, MaxClients)}
	server := httptest.NewServer(handler)
	defer server.Close()

	checkHandlerValid(t, server)
	checkErrorValidation(t, server)
	ensureURLValidation(t, server)
}

func ensureURLValidation(t *testing.T, server *httptest.Server) {
	body := []byte(`["invlid_url"]`)
	res, err := http.Post(server.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != http.StatusBadRequest {
		t.Fatal("expected bad-request response for invalid url")
	}
}

func checkHandlerValid(t *testing.T, server *httptest.Server) {
	urls := genUrls(20)
	body, err := json.Marshal(urls)
	if err != nil {
		t.Fatal(err)
	}

	res, err := http.Post(server.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("Received non-200 resonse: %d\n", res.StatusCode)
	}
}

func checkErrorValidation(t *testing.T, server *httptest.Server) {
	urls := genUrls(21)
	body, err := json.Marshal(urls)
	if err != nil {
		t.Fatal(err)
	}

	res, err := http.Post(server.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 400 {
		t.Fatalf("Validation length is't work: %d\n", res.StatusCode)
	} else {
		t.Log("Validation is work")
	}
}

func genUrls(count int) []string {
	url := "https://jsonplaceholder.typicode.com/posts/"
	urls := make([]string, 0, count)
	for i := 0; i < count; i++ {
		urls = append(urls, fmt.Sprint(url, i+1))
	}
	return urls
}
