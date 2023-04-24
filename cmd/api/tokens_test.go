package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"greenlight.bcc/internal/assert"
)

func TestCreateAuthenticationToken(t *testing.T) {
	app := newTestApplication(t)
	ts := newTestServer(t, app.routesTest())
	defer ts.Close()

	const (
		validEmail    = "test@example.com"
		validPassword = "12345678"
	)

	tests := []struct {
		name     string
		Email    string
		Password string
		wantCode int
	}{
		{
			name:     "Valid submission",
			Email:    validEmail,
			Password: validPassword,
			wantCode: http.StatusCreated,
		},
		{
			name:     "Invalid Email",
			Email:    "@aaaaaz1",
			Password: validPassword,
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name:     "Invalid Password",
			Email:    validEmail,
			Password: "12345",
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name:     "test for wrong input",
			Email:    validEmail,
			Password: validPassword,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputData := struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}{
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

			code, _, _ := ts.postForm(t, "/v1/tokens/authentication", b)
			assert.Equal(t, code, tt.wantCode)
		})
	}
}
