package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler_StatusCode_200(t *testing.T) {
	// Create a fake request to pass to the method.
	r := httptest.NewRequest("GET", "/health", nil)

	// Create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response
	rr := httptest.NewRecorder()

	// Test the handler with the request and record the result
	handler := http.HandlerFunc(makeHealthHandler())
	handler.ServeHTTP(rr, r)

	if body := rr.Body.String(); body != "Worker is running" {
		t.Errorf("Wrong body: got %v, want %v", body, "Worker is running")
	}
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Wrong status code: got %v, want %v", status, http.StatusOK)
	}
}
