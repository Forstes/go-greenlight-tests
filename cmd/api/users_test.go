package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"greenlight.bcc/internal/assert"
)

func TestRegisterUser(t *testing.T) {
	app := newTestApplication(t)
	ts := newTestServer(t, app.routesTest())
	defer ts.Close()

	const (
		validName     = "User"
		validEmail    = "test@example.com"
		validPassword = "12345678"
	)

	tests := []struct {
		name     string
		Name     string
		Email    string
		Password string
		wantCode int
	}{
		{
			name:     "Valid submission",
			Name:     validName,
			Email:    validEmail,
			Password: validPassword,
			wantCode: http.StatusCreated,
		},
		{
			name:     "Invalid Name",
			Name:     "",
			Email:    validEmail,
			Password: validPassword,
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name:     "Invalid Email",
			Name:     validName,
			Email:    "@aaaaaz1",
			Password: validPassword,
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name:     "Invalid Password",
			Name:     validName,
			Email:    validEmail,
			Password: "12345",
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name:     "Duplicate Email",
			Name:     validName,
			Email:    "duplicate@gmail.com",
			Password: validPassword,
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name:     "test for wrong input",
			Name:     validName,
			Email:    validEmail,
			Password: validPassword,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputData := struct {
				Name     string `json:"name"`
				Email    string `json:"email"`
				Password string `json:"password"`
			}{
				Name:     tt.Name,
				Email:    tt.Email,
				Password: tt.Password,
			}

			b, err := json.Marshal(&inputData)
			if err != nil {
				t.Fatal("wrong input data")
			}
			if tt.name == "test for wrong input" {
				b = append(b, 'a')
			}

			code, _, _ := ts.postForm(t, "/v1/users", b)
			assert.Equal(t, code, tt.wantCode)
		})
	}
}

func TestActivateUser(t *testing.T) {
	app := newTestApplication(t)

	ts := newTestServer(t, app.routesTest())
	defer ts.Close()

	tests := []struct {
		name     string
		body     io.Reader
		wantCode int
	}{
		{
			name:     "Valid token",
			body:     strings.NewReader(`{"token":"samplevalidtoken123456789a"}`),
			wantCode: http.StatusOK,
		},
		{
			name:     "Invalid json",
			body:     strings.NewReader(`invalid json`),
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "Invalid token",
			body:     strings.NewReader(`{"token":"bad_guy"}`),
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name:     "Expired token",
			body:     strings.NewReader(`{"token":"badtoken123456789abcdefghk"}`),
			wantCode: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, _ := ts.updateReq(t, ts.URL+"/v1/users/activated", tt.body, http.MethodPut)
			assert.Equal(t, code, tt.wantCode)
		})
	}
}
