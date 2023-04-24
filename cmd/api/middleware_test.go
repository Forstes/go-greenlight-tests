package main

import (
	"expvar"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"greenlight.bcc/internal/data"
)

func Test_enableCORS(t *testing.T) {
	app := &application{
		config: config{
			cors: struct{ trustedOrigins []string }{
				trustedOrigins: []string{"http://localhost:3000"},
			},
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := app.enableCORS(handler)

	req := httptest.NewRequest(http.MethodOptions, "/path", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")

	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	if rr.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("Access-Control-Allow-Origin header not set correctly")
	}

	if rr.Header().Get("Access-Control-Allow-Methods") != "OPTIONS, PUT, PATCH, DELETE" {
		t.Errorf("Access-Control-Allow-Methods header not set correctly")
	}

	if rr.Header().Get("Access-Control-Allow-Headers") != "Authorization, Content-Type" {
		t.Errorf("Access-Control-Allow-Headers header not set correctly")
	}

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func Test_metrics(t *testing.T) {

	app := newTestApplication(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/foo", nil)

	handler := app.metrics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Millisecond * 1)
		fmt.Fprint(w, "Cheese and bread")
	}))

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	totalRequestsReceived := expvar.Get("total_requests_received").(*expvar.Int).Value()
	if totalRequestsReceived != 1 {
		t.Errorf("total_requests_received should be 1, but got %d", totalRequestsReceived)
	}

	totalResponsesSent := expvar.Get("total_responses_sent").(*expvar.Int).Value()
	if totalResponsesSent != 1 {
		t.Errorf("total_responses_sent should be 1, but got %d", totalResponsesSent)
	}

	totalProcessingTimeMicroseconds := expvar.Get("total_processing_time_μs").(*expvar.Int).Value()
	if totalProcessingTimeMicroseconds <= 0 {
		t.Errorf("total_processing_time_μs should be greater than 0, but got %d", totalProcessingTimeMicroseconds)
	}

	totalResponsesSentByStatus := expvar.Get("total_responses_sent_by_status").(*expvar.Map)
	if totalResponsesSentByStatus == nil {
		t.Errorf("total_responses_sent_by_status should not be nil")
	} else {
		// Check entry for the response status code
		num := totalResponsesSentByStatus.Get(strconv.Itoa(rr.Code)).(*expvar.Int).Value()
		if num != 1 {
			t.Errorf("total_processing_time_μs should be greater than 0, but got %d", num)
		}
	}
}

func Test_requireAuthenticatedUser(t *testing.T) {
	app := newTestApplication(t)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	req = app.contextSetUser(req, &data.User{ID: 1})

	var nextCalled bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	handler := app.requireAuthenticatedUser(next)
	handler.ServeHTTP(rr, req)

	if !nextCalled {
		t.Error("next handler should have been called")
	}
}

func Test_requireActivatedUser(t *testing.T) {
	app := newTestApplication(t)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	req = app.contextSetUser(req, &data.User{ID: 1, Activated: true})

	var nextCalled bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	handler := app.requireAuthenticatedUser(next)
	handler.ServeHTTP(rr, req)

	if !nextCalled {
		t.Error("next handler should have been called")
	}
}

func Test_requirePermission(t *testing.T) {
	app := newTestApplication(t)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	req = app.contextSetUser(req, &data.User{ID: 1, Activated: true})

	app.models.Permissions = &data.MockPermissionModel{
		Permissions: data.Permissions{"1"},
		Err:         nil,
	}

	var nextCalled bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	handler := app.requirePermission("1", next)
	handler.ServeHTTP(rr, req)

	if !nextCalled {
		t.Error("next handler should have been called")
	}
}

func Test_authenticate(t *testing.T) {

	app := newTestApplication(t)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	tests := []struct {
		name               string
		user               *data.User
		authorizationToken string
		statusCode         int
	}{
		{
			name:               "no token",
			user:               data.AnonymousUser,
			authorizationToken: "",
			statusCode:         http.StatusOK,
		},
		{
			name:               "bad token",
			user:               nil,
			authorizationToken: "invalid_token",
			statusCode:         http.StatusUnauthorized,
		},
		{
			name:               "valid token",
			user:               &data.User{},
			authorizationToken: "Bearer kjkjkjkjkjkjkjkjkjkjkjkjki",
			statusCode:         http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			handler := app.authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctxUser := app.contextGetUser(r)

				if tt.user != nil && ctxUser.ID != tt.user.ID {
					t.Errorf("Expected user to be %v, but got %v", tt.user, ctxUser)
				}
				w.WriteHeader(http.StatusOK)
			}))

			rr := httptest.NewRecorder()
			req.Header.Set("Authorization", tt.authorizationToken)

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.statusCode {
				t.Errorf("Expected status code %d, but got %d", tt.statusCode, rr.Code)
			}
		})
	}
}

func Test_rateLimit(t *testing.T) {
	app := newTestApplication(t)

	app.config.limiter.enabled = true
	app.config.limiter.rps = 1
	app.config.limiter.burst = 2

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := app.rateLimit(handler)

	// Call the middleware 3 times in a row with a delay between calls
	for i := 0; i < 3; i++ {
		res := httptest.NewRecorder()
		mw.ServeHTTP(res, req)

		// First 2 responses are OK
		if status := res.Result().StatusCode; status != http.StatusOK {
			t.Errorf("Expected status code %v, but got %v", http.StatusOK, status)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// It should exceed the rate limit
	res := httptest.NewRecorder()
	mw.ServeHTTP(res, req)

	if status := res.Result().StatusCode; status != http.StatusTooManyRequests {
		t.Errorf("Expected status code %v, but got %v", http.StatusTooManyRequests, status)
	}
}

func Test_recoverPanic(t *testing.T) {
	app := newTestApplication(t)
	rr := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/path", nil)
	if err != nil {
		t.Fatal(err)
	}

	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("MAMMA MIA!")
	})

	mw := app.recoverPanic(mockHandler)
	mw.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusInternalServerError)
	}

	expected := `{"error":"the server encountered a problem and could not process your request"}`
	actual := strings.TrimSpace(rr.Body.String())
	if actual != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			actual, expected)
	}
}
