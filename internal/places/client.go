package places

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	textSearchURL = "https://places.googleapis.com/v1/places:searchText"
	basePlaceURL  = "https://places.googleapis.com/v1/places/"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient() (*Client, error) {
	apiKey := strings.TrimSpace(os.Getenv("PLACES_API_KEY"))
	if apiKey == "" {
		return nil, fmt.Errorf("PLACES_API_KEY is required")
	}

	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}, nil
}

type rectangle struct {
	Low  latLng `json:"low"`
	High latLng `json:"high"`
}

type latLng struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type textSearchRequest struct {
	TextQuery           string `json:"textQuery"`
	PageToken           string `json:"pageToken,omitempty"`
	LocationRestriction struct {
		Rectangle rectangle `json:"rectangle"`
	} `json:"locationRestriction"`
}

type TextSearchResponse struct {
	Places        []PlaceSummary `json:"places"`
	NextPageToken string         `json:"nextPageToken"`
}

type PlaceSummary struct {
	ID              string           `json:"id"`
	BusinessStatus  string           `json:"businessStatus"`
	Location        *latLng          `json:"location"`
	OpeningDate     *Date            `json:"openingDate"`
	GoogleMapsLinks *GoogleMapsLinks `json:"googleMapsLinks"`
	PostalAddress   *PostalAddress   `json:"postalAddress"`
	Rating          *float64         `json:"rating"`
	UserRatingCount int              `json:"userRatingCount"`
	DisplayName     struct {
		Text string `json:"text"`
	} `json:"displayName"`
}

type Date struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

type GoogleMapsLinks struct {
	PlaceURI        string `json:"placeUri"`
	DirectionsURI   string `json:"directionsUri"`
	WriteAReviewURI string `json:"writeAReviewUri"`
	ReviewsURI      string `json:"reviewsUri"`
	PhotosURI       string `json:"photosUri"`
}

type PostalAddress struct {
	Revision           int      `json:"revision"`
	RegionCode         string   `json:"regionCode"`
	LanguageCode       string   `json:"languageCode"`
	PostalCode         string   `json:"postalCode"`
	SortingCode        string   `json:"sortingCode"`
	AdministrativeArea string   `json:"administrativeArea"`
	Locality           string   `json:"locality"`
	Sublocality        string   `json:"sublocality"`
	AddressLines       []string `json:"addressLines"`
	Recipients         []string `json:"recipients"`
	Organization       string   `json:"organization"`
}

type PlaceDetails struct {
	BusinessStatus   string `json:"businessStatus"`
	FormattedAddress string `json:"formattedAddress"`
	DisplayName      struct {
		Text string `json:"text"`
	} `json:"displayName"`
}

func (c *Client) SearchText(ctx context.Context, query string, lowLat, lowLng, highLat, highLng float64, pageToken string) (*TextSearchResponse, error) {
	reqBody := textSearchRequest{
		TextQuery: query,
		PageToken: pageToken,
	}
	reqBody.LocationRestriction.Rectangle = rectangle{
		Low: latLng{
			Latitude:  lowLat,
			Longitude: lowLng,
		},
		High: latLng{
			Latitude:  highLat,
			Longitude: highLng,
		},
	}

	var payload bytes.Buffer
	if err := json.NewEncoder(&payload).Encode(reqBody); err != nil {
		return nil, fmt.Errorf("encode text search request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, textSearchURL, &payload)
	if err != nil {
		return nil, fmt.Errorf("build text search request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", c.apiKey)
	req.Header.Set("X-Goog-FieldMask", "places.id,places.displayName.text,places.businessStatus,places.location,places.openingDate,places.googleMapsLinks,places.postalAddress,places.rating,places.userRatingCount,nextPageToken")

	var respData TextSearchResponse
	if err := c.doJSON(req, &respData); err != nil {
		return nil, err
	}

	return &respData, nil
}

func (c *Client) GetPlaceDetails(ctx context.Context, placeID string) (*PlaceDetails, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, basePlaceURL+url.PathEscape(placeID), nil)
	if err != nil {
		return nil, fmt.Errorf("build place details request for %s: %w", placeID, err)
	}

	req.Header.Set("X-Goog-Api-Key", c.apiKey)
	req.Header.Set("X-Goog-FieldMask", "businessStatus,displayName.text,formattedAddress")

	var details PlaceDetails
	if err := c.doJSON(req, &details); err != nil {
		return nil, fmt.Errorf("fetch place details for %s: %w", placeID, err)
	}

	return &details, nil
}

func (c *Client) doJSON(req *http.Request, dest any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request %s %s failed: %w", req.Method, req.URL.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("request %s %s returned %s: %s", req.Method, req.URL.String(), resp.Status, strings.TrimSpace(string(body)))
	}

	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("decode response from %s %s: %w", req.Method, req.URL.String(), err)
	}

	return nil
}
