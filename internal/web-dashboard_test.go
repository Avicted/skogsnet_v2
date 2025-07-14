package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServeAPI_Endpoint(t *testing.T) {
	db, err := openDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	serveAPI(db, mux)
	req := httptest.NewRequest("GET", "/api/measurements?range=1h", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200 OK, got %d", w.Code)
	}
	if len(w.Body.Bytes()) == 0 {
		t.Error("Expected non-empty response body")
	}
}

func TestServeAPI_NoRange(t *testing.T) {
	db, err := openDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	serveAPI(db, mux)
	req := httptest.NewRequest("GET", "/api/measurements", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200 OK, got %d", w.Code)
	}
	if len(w.Body.Bytes()) == 0 {
		t.Error("Expected non-empty response body")
	}
}

func TestServeAPI_DBError(t *testing.T) {
	db, err := openDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	// Simulate a DB error by closing the connection
	db.Close()

	mux := http.NewServeMux()
	serveAPI(db, mux)
	req := httptest.NewRequest("GET", "/api/measurements?range=1h", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Errorf("Expected 500 Internal Server Error, got %d", w.Code)
	}
	if w.Body.String() != "DB error" {
		t.Errorf("Expected 'DB error', got '%s'", w.Body.String())
	}
}
