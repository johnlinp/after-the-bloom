package analytics

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type recordingRecorder struct {
	events []Event
	err    error
}

func (r *recordingRecorder) Record(_ context.Context, event Event) error {
	r.events = append(r.events, event)
	return r.err
}

type fixedCountryResolver struct{ code string }

func (r fixedCountryResolver) CountryCode(*http.Request) *string { return &r.code }

func TestHandlerRecordsServerOwnedFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := &recordingRecorder{}
	handler := NewHandler(recorder, fixedCountryResolver{code: "US"}, log.New(io.Discard, "", 0))
	fixedNow := time.Date(2026, 10, 2, 12, 0, 0, 0, time.FixedZone("test", -7*60*60))
	handler.now = func() time.Time { return fixedNow }

	router := gin.New()
	router.POST("/api/v1/analytics/events", handler.Record)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/analytics/events",
		strings.NewReader(`{"event_type":"page_landing","metadata":{"path":"/"}}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "8.8.8.8")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if len(recorder.events) != 1 {
		t.Fatalf("recorded %d events, want 1", len(recorder.events))
	}
	event := recorder.events[0]
	if event.ID == "" || event.EventType != EventPageLanding {
		t.Fatalf("unexpected event: %+v", event)
	}
	if event.ClientCountryCode == nil || *event.ClientCountryCode != "US" {
		t.Fatalf("country = %v, want US", event.ClientCountryCode)
	}
	if !event.CreatedAt.Equal(fixedNow.UTC()) || event.CreatedAt.Location() != time.UTC {
		t.Fatalf("created_at = %v, want %v", event.CreatedAt, fixedNow.UTC())
	}
}

func TestHandlerRejectsInvalidAndOversizedEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(&recordingRecorder{}, nil, log.New(io.Discard, "", 0))
	router := gin.New()
	router.POST("/api/v1/analytics/events", handler.Record)

	tests := []struct {
		name   string
		body   string
		status int
	}{
		{name: "invalid", body: `{"event_type":"unknown","metadata":{}}`, status: http.StatusBadRequest},
		{
			name:   "oversized",
			body:   `{"event_type":"page_landing","metadata":{"path":"` + strings.Repeat("x", maxRequestBytes) + `"}}`,
			status: http.StatusRequestEntityTooLarge,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/analytics/events", strings.NewReader(test.body))
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != test.status {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, test.status, rec.Body.String())
			}
		})
	}
}

func TestHandlerReturnsUnavailableWhenRecorderFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := &recordingRecorder{err: errors.New("database unavailable")}
	handler := NewHandler(recorder, nil, log.New(io.Discard, "", 0))
	router := gin.New()
	router.POST("/api/v1/analytics/events", handler.Record)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/analytics/events",
		strings.NewReader(`{"event_type":"page_landing","metadata":{"path":"/"}}`),
	)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}
