package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"greenlight.bcc/internal/data"
)

func Test_healthcheck(t *testing.T) {
	app := newTestApplication(t)
	app.config.env = "test"

	req, err := http.NewRequest("GET", "/v1/healthcheck", nil)
	if err != nil {
		t.Fatal(err)
	}
	req = app.contextSetUser(req, &data.User{Name: "Ramsay"})

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.healthcheckHandler)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			rr.Code, http.StatusOK)
	}

	expected := `{"status":"available","system_info":{"environment":"test","user_name":"Ramsay","version":"1.0.0"}}`
	if strings.TrimSpace(rr.Body.String()) != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}
