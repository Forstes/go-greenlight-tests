package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"greenlight.bcc/internal/assert"
)

func TestShowMovie(t *testing.T) {
	app := newTestApplication(t)

	ts := newTestServer(t, app.routesTest())
	defer ts.Close()

	tests := []struct {
		name     string
		urlPath  string
		wantCode int
		wantBody string
	}{
		{
			name:     "Valid ID",
			urlPath:  "/v1/movies/1",
			wantCode: http.StatusOK,
		},
		{
			name:     "Non-existent ID",
			urlPath:  "/v1/movies/2",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "Negative ID",
			urlPath:  "/v1/movies/-1",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "Decimal ID",
			urlPath:  "/v1/movies/1.23",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "String ID",
			urlPath:  "/v1/movies/foo",
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			code, _, body := ts.get(t, tt.urlPath)

			assert.Equal(t, code, tt.wantCode)

			if tt.wantBody != "" {
				assert.StringContains(t, body, tt.wantBody)
			}

		})
	}

}

func TestCreateMovie(t *testing.T) {
	app := newTestApplication(t)
	ts := newTestServer(t, app.routesTest())
	defer ts.Close()

	const (
		validTitle   = "Test Title"
		validYear    = 2021
		validRuntime = "105 mins"
	)

	validGenres := []string{"comedy", "drama"}

	tests := []struct {
		name     string
		Title    string
		Year     int32
		Runtime  string
		Genres   []string
		wantCode int
	}{
		{
			name:     "Valid submission",
			Title:    validTitle,
			Year:     validYear,
			Runtime:  validRuntime,
			Genres:   validGenres,
			wantCode: http.StatusCreated,
		},
		{
			name:     "Empty Title",
			Title:    "",
			Year:     validYear,
			Runtime:  validRuntime,
			Genres:   validGenres,
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name:     "year < 1888",
			Title:    validTitle,
			Year:     1500,
			Runtime:  validRuntime,
			Genres:   validGenres,
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name:     "test for wrong input",
			Title:    validTitle,
			Year:     validYear,
			Runtime:  validRuntime,
			Genres:   validGenres,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputData := struct {
				Title   string   `json:"title"`
				Year    int32    `json:"year"`
				Runtime string   `json:"runtime"`
				Genres  []string `json:"genres"`
			}{
				Title:   tt.Title,
				Year:    tt.Year,
				Runtime: tt.Runtime,
				Genres:  tt.Genres,
			}

			b, err := json.Marshal(&inputData)
			if err != nil {
				t.Fatal("wrong input data")
			}
			if tt.name == "test for wrong input" {
				b = append(b, 'a')
			}

			code, _, _ := ts.postForm(t, "/v1/movies", b)
			assert.Equal(t, code, tt.wantCode)
		})
	}
}

func TestDeleteMovie(t *testing.T) {
	app := newTestApplication(t)
	ts := newTestServer(t, app.routesTest())
	defer ts.Close()

	tests := []struct {
		name     string
		urlPath  string
		wantCode int
		wantBody string
	}{
		{
			name:     "deleting existing movie",
			urlPath:  "/v1/movies/1",
			wantCode: http.StatusOK,
		},
		{
			name:     "Non-existent ID",
			urlPath:  "/v1/movies/2",
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			code, _, body := ts.deleteReq(t, tt.urlPath)
			assert.Equal(t, code, tt.wantCode)

			if tt.wantBody != "" {
				assert.StringContains(t, body, tt.wantBody)
			}
		})
	}
}

func TestUpdateMovie(t *testing.T) {
	app := newTestApplication(t)

	ts := newTestServer(t, app.routesTest())
	defer ts.Close()

	tests := []struct {
		name     string
		urlPath  string
		body     io.Reader
		wantCode int
	}{
		{
			name:     "Valid request",
			urlPath:  "/v1/movies/1",
			body:     strings.NewReader(`{"title":"New Movie Title","year":2022,"runtime":"100 mins","genres":["Comedy","Drama"]}`),
			wantCode: http.StatusOK,
		},
		{
			name:     "Unreadable ID",
			urlPath:  "/v1/movies/cupcake1",
			body:     strings.NewReader(`invalid json`),
			wantCode: http.StatusNotFound,
		},
		{
			name:     "Invalid request body",
			urlPath:  "/v1/movies/1",
			body:     strings.NewReader(`invalid json`),
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "Failed validation",
			urlPath:  "/v1/movies/1",
			body:     strings.NewReader(`{"title":"","year":1337,"runtime":"-5 mins","genres":["Comedy","Comedy"]}`),
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name:     "Non-existent movie ID",
			urlPath:  "/v1/movies/2",
			body:     strings.NewReader(`{"title":"New Movie Title","year":2022,"runtime":"100 mins","genres":["Comedy","Drama"]}`),
			wantCode: http.StatusNotFound,
		},
		{
			name:     "Edit conflict",
			urlPath:  "/v1/movies/3",
			body:     strings.NewReader(`{"title":"New Movie Title","year":2022,"runtime":"100 mins","genres":["Comedy","Drama"]}`),
			wantCode: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, _ := ts.updateReq(t, ts.URL+tt.urlPath, tt.body, http.MethodPatch)

			assert.Equal(t, code, tt.wantCode)
		})
	}
}

func TestListMovies(t *testing.T) {
	app := newTestApplication(t)

	ts := newTestServer(t, app.routesTest())
	defer ts.Close()

	tests := []struct {
		name     string
		urlPath  string
		wantCode int
	}{
		{
			name:     "Valid",
			urlPath:  "/v1/movies",
			wantCode: http.StatusOK,
		},
		{
			name:     "Bad params",
			urlPath:  "/v1/movies?page_size=haha",
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name:     "Invalid filters",
			urlPath:  "/v1/movies?page_size=5000&sort=cake",
			wantCode: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			code, _, _ := ts.get(t, tt.urlPath)
			assert.Equal(t, code, tt.wantCode)
		})
	}
}
