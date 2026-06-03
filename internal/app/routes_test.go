package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAPIEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store, err := LoadStore(filepath.Join("..", "..", "data", "atb-20260601.json"))
	if err != nil {
		t.Fatalf("LoadStore() error = %v", err)
	}

	router := gin.New()
	RegisterRoutes(router, store, filepath.Join("..", "..", "photos"), filepath.Join("..", "..", "web", "index.html"))

	t.Run("spots pagination", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/spots?page=1&limit=10", nil)
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var payload struct {
			Page  int            `json:"page"`
			Limit int            `json:"limit"`
			Total int            `json:"total"`
			Items []SpotResponse `json:"items"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode spots response: %v", err)
		}

		if payload.Page != 1 || payload.Limit != 10 {
			t.Fatalf("unexpected pagination: %+v", payload)
		}
		if len(payload.Items) != 10 {
			t.Fatalf("len(items) = %d, want 10", len(payload.Items))
		}
		if payload.Total == 0 {
			t.Fatalf("total = %d, want > 0", payload.Total)
		}
	})

	t.Run("hubs", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/hubs", nil)
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var payload []Hub
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode hubs response: %v", err)
		}
		if len(payload) == 0 {
			t.Fatal("expected at least one hub")
		}
	})

	t.Run("hub spots pagination", func(t *testing.T) {
		hubID := store.Hubs()[0].ID
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/hubs/"+hubID+"/spots?page=1&limit=10", nil)
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var payload struct {
			Items []SpotResponse `json:"items"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode hub spots response: %v", err)
		}
		if len(payload.Items) == 0 {
			t.Fatal("expected hub to have spots")
		}
	})

	t.Run("thumbnail", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/spots/705300c0-e469-443e-9d5b-c5bf599f8e0b/thumbnail", nil)
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})
}
