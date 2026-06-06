package places

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	legacyPlaceDetailsURL = "https://maps.googleapis.com/maps/api/place/details/json"
	maxExpandRedirects    = 10
)

type legacyPlaceDetailsResponse struct {
	Result struct {
		PlaceID string `json:"place_id"`
	} `json:"result"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message"`
}

func ExpandGoogleMapsURL(ctx context.Context, rawURL string) (string, error) {
	trimmedURL := strings.TrimSpace(rawURL)
	if trimmedURL == "" {
		return "", fmt.Errorf("Google Maps URL is required")
	}

	parsedURL, err := url.Parse(trimmedURL)
	if err != nil {
		return "", fmt.Errorf("parse Google Maps URL: %w", err)
	}

	if !needsGoogleMapsExpansion(parsedURL.Host) {
		return trimmedURL, nil
	}

	httpClient := &http.Client{
		Timeout: 20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxExpandRedirects {
				return fmt.Errorf("stopped after %d redirects", maxExpandRedirects)
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, trimmedURL, nil)
	if err != nil {
		return "", fmt.Errorf("build short Google Maps URL expansion request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("expand short Google Maps URL: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	if resp.Request == nil || resp.Request.URL == nil {
		return "", fmt.Errorf("expand short Google Maps URL: missing final URL after redirects")
	}

	return resp.Request.URL.String(), nil
}

func ExtractCIDFromGoogleMapsURL(rawURL string) (string, error) {
	decodedURL, err := url.QueryUnescape(strings.TrimSpace(rawURL))
	if err != nil {
		return "", fmt.Errorf("decode Google Maps URL: %w", err)
	}

	marker := "!1s0x"
	start := strings.Index(decodedURL, marker)
	if start == -1 {
		return "", fmt.Errorf("extract CID from Google Maps URL: missing %q marker", marker)
	}

	value := decodedURL[start+len(marker):]
	separator := strings.Index(value, ":0x")
	if separator == -1 {
		return "", fmt.Errorf("extract CID from Google Maps URL: missing :0x separator")
	}

	cidHex := value[separator+len(":0x"):]
	end := strings.IndexAny(cidHex, "!&?")
	if end >= 0 {
		cidHex = cidHex[:end]
	}

	cidHex = strings.TrimSpace(cidHex)
	if cidHex == "" {
		return "", fmt.Errorf("extract CID from Google Maps URL: empty CID")
	}

	if _, err := strconv.ParseUint(cidHex, 16, 64); err != nil {
		return "", fmt.Errorf("parse CID hex %q: %w", cidHex, err)
	}

	return "0x" + strings.ToLower(cidHex), nil
}

func needsGoogleMapsExpansion(host string) bool {
	normalizedHost := strings.ToLower(strings.TrimSpace(host))
	return normalizedHost == "maps.app.goo.gl"
}

func CIDHexToDecimalString(cidHex string) (string, error) {
	trimmed := strings.TrimSpace(strings.ToLower(cidHex))
	trimmed = strings.TrimPrefix(trimmed, "0x")
	if trimmed == "" {
		return "", fmt.Errorf("CID hex is required")
	}

	value, err := strconv.ParseUint(trimmed, 16, 64)
	if err != nil {
		return "", fmt.Errorf("parse CID hex %q: %w", cidHex, err)
	}

	return strconv.FormatUint(value, 10), nil
}

func (c *Client) GetPlaceIDFromCID(ctx context.Context, cidDecimal string) (string, error) {
	return c.getPlaceIDFromCID(ctx, legacyPlaceDetailsURL, cidDecimal)
}

func (c *Client) getPlaceIDFromCID(ctx context.Context, endpoint, cidDecimal string) (string, error) {
	if strings.TrimSpace(cidDecimal) == "" {
		return "", fmt.Errorf("CID decimal is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("build place details request for CID %s: %w", cidDecimal, err)
	}

	query := req.URL.Query()
	query.Set("cid", cidDecimal)
	query.Set("fields", "place_id")
	query.Set("key", c.apiKey)
	req.URL.RawQuery = query.Encode()

	var respData legacyPlaceDetailsResponse
	if err := c.doJSON(req, &respData); err != nil {
		return "", fmt.Errorf("fetch place ID for CID %s: %w", cidDecimal, err)
	}

	if respData.Status != "OK" {
		if strings.TrimSpace(respData.ErrorMessage) != "" {
			return "", fmt.Errorf("fetch place ID for CID %s: API status %s: %s", cidDecimal, respData.Status, respData.ErrorMessage)
		}
		return "", fmt.Errorf("fetch place ID for CID %s: API status %s", cidDecimal, respData.Status)
	}

	if strings.TrimSpace(respData.Result.PlaceID) == "" {
		return "", fmt.Errorf("fetch place ID for CID %s: empty place_id in response", cidDecimal)
	}

	return respData.Result.PlaceID, nil
}
