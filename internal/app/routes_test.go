package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAPIEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store, err := LoadStore(
		filepath.Join("..", "..", "data", "atb-20260601.json"),
		filepath.Join("..", "..", "data", "tw-zipcode.json"),
	)
	if err != nil {
		t.Fatalf("LoadStore() error = %v", err)
	}

	router := gin.New()
	RegisterRoutes(router, store, filepath.Join("..", "..", "photos"), filepath.Join("..", "..", "web", "index.html"))

	knownShortCode := ""
	knownShortCodeWithPhoto := ""
	knownSpotIDWithPhoto := ""
	for _, spot := range store.spots {
		if spot.ShortCode != "" {
			if knownShortCode == "" {
				knownShortCode = spot.ShortCode
			}
			if matches, _ := filepath.Glob(filepath.Join("..", "..", "photos", spot.ID+".*")); len(matches) > 0 {
				knownShortCodeWithPhoto = spot.ShortCode
				knownSpotIDWithPhoto = spot.ID
				break
			}
		}
	}
	if knownShortCode == "" {
		t.Fatal("expected at least one spot to have a short code")
	}
	if knownShortCodeWithPhoto == "" {
		t.Fatal("expected at least one spot with both a short code and a thumbnail")
	}

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
		if payload.Items[0].ShortCode == "" {
			t.Fatal("expected spot response to include short code")
		}
	})

	t.Run("district page route", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/distrct/tw-106", nil)
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if rec.Body.Len() == 0 {
			t.Fatal("expected district page route to serve index html")
		}
	})

	t.Run("spot page route", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/spot/"+knownShortCodeWithPhoto, nil)
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if rec.Body.Len() == 0 {
			t.Fatal("expected spot page route to serve index html")
		}
		body := rec.Body.String()
		spot, ok := store.SpotByShortCode(knownShortCodeWithPhoto)
		if !ok {
			t.Fatalf("expected short code %q to resolve to a spot", knownShortCodeWithPhoto)
		}
		if !strings.Contains(body, `<meta property="og:url" content="https://afterthebloom.com/spot/`+knownShortCodeWithPhoto+`" id="meta-og-url">`) {
			t.Fatal("expected spot page route to serve spot-specific og:url metadata")
		}
		if !strings.Contains(body, `<meta property="og:title" content="空折枝 - `+spot.Name+`" id="meta-og-title">`) {
			t.Fatal("expected spot page route to serve spot-specific og:title metadata")
		}
		if !strings.Contains(body, `<meta property="og:image" content="https://afterthebloom.com/api/v1/spots/`+knownSpotIDWithPhoto+`/thumbnail" id="meta-og-image">`) {
			t.Fatal("expected spot page route to serve spot-specific og:image metadata")
		}
	})

	t.Run("districts", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/districts", nil)
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var payload []District
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode districts response: %v", err)
		}
		if len(payload) == 0 {
			t.Fatal("expected at least one district")
		}
		found := false
		foundDaanID := false
		foundChiayiID := false
		emptyDistrictID := ""
		for _, district := range payload {
			if district.Name == "桃園市龍潭區" {
				found = true
				emptyDistrictID = district.ID
			}
			if district.Name == "台北市大安區" && district.ID == "tw-106" {
				foundDaanID = true
			}
			if district.Name == "嘉義市東區/西區" && district.ID == "tw-600" {
				foundChiayiID = true
			}
		}
		if !found {
			t.Fatal("expected 桃園市龍潭區 to be present in districts response")
		}
		if !foundDaanID {
			t.Fatal("expected 台北市大安區 to use district id tw-106")
		}
		if !foundChiayiID {
			t.Fatal("expected 嘉義市東區/西區 to use district id tw-600")
		}
		if emptyDistrictID == "" {
			t.Fatal("expected 桃園市龍潭區 to have a district id")
		}
		for i := 1; i < len(payload); i++ {
			prevZip, err := strconv.Atoi(payload[i-1].ZipCode)
			if err != nil {
				t.Fatalf("invalid zip code %q: %v", payload[i-1].ZipCode, err)
			}
			currZip, err := strconv.Atoi(payload[i].ZipCode)
			if err != nil {
				t.Fatalf("invalid zip code %q: %v", payload[i].ZipCode, err)
			}
			if prevZip > currZip {
				t.Fatalf("districts not sorted by zip code: %s (%s) before %s (%s)", payload[i-1].Name, payload[i-1].ZipCode, payload[i].Name, payload[i].ZipCode)
			}
		}
	})

	t.Run("district spots pagination", func(t *testing.T) {
		districtID := ""
		for _, district := range store.Districts() {
			if items, total, ok := store.SpotsPageByDistrict(district.ID, 1, 1); ok && total > 0 && len(items) > 0 {
				districtID = district.ID
				break
			}
		}
		if districtID == "" {
			t.Fatal("expected at least one district to have spots")
		}
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/districts/"+districtID+"/spots?page=1&limit=10", nil)
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var payload struct {
			Items []SpotResponse `json:"items"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode district spots response: %v", err)
		}
		if len(payload.Items) == 0 {
			t.Fatal("expected district to have spots")
		}
	})

	t.Run("empty district spots pagination", func(t *testing.T) {
		emptyDistrictID := ""
		for _, district := range store.Districts() {
			if district.Name == "桃園市龍潭區" {
				emptyDistrictID = district.ID
				break
			}
		}
		if emptyDistrictID == "" {
			t.Fatal("expected 桃園市龍潭區 to have a district id")
		}
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/districts/"+emptyDistrictID+"/spots?page=1&limit=10", nil)
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var payload struct {
			Total int            `json:"total"`
			Items []SpotResponse `json:"items"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode empty district spots response: %v", err)
		}
		if payload.Total != 0 {
			t.Fatalf("total = %d, want 0", payload.Total)
		}
		if len(payload.Items) != 0 {
			t.Fatalf("len(items) = %d, want 0", len(payload.Items))
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

	t.Run("spot by short code", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/spots/short-code/"+knownShortCode, nil)
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var payload SpotResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode short-code spot response: %v", err)
		}
		if payload.ShortCode != knownShortCode {
			t.Fatalf("short code = %q, want %q", payload.ShortCode, knownShortCode)
		}
	})
}
