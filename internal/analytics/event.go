package analytics

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	EventPageLanding    = "page_landing"
	EventLoadMoreClick  = "load_more_click"
	EventDistrictSearch = "district_search"
	EventDistrictClick  = "district_click"
	EventSpotClick      = "spot_click"
)

var (
	errInvalidEventType = errors.New("invalid event type")
	errInvalidMetadata  = errors.New("invalid metadata")
)

type Event struct {
	ID                string         `bson:"_id" json:"id"`
	EventType         string         `bson:"event_type" json:"event_type"`
	ClientCountryCode *string        `bson:"client_country_code" json:"client_country_code"`
	Metadata          map[string]any `bson:"metadata" json:"metadata"`
	CreatedAt         time.Time      `bson:"created_at" json:"created_at"`
}

type eventRequest struct {
	EventType string          `json:"event_type"`
	Metadata  json.RawMessage `json:"metadata"`
}

func DecodeEventRequest(r io.Reader) (string, map[string]any, error) {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()

	var request eventRequest
	if err := decoder.Decode(&request); err != nil {
		return "", nil, fmt.Errorf("decode event: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return "", nil, err
	}

	eventType := strings.TrimSpace(request.EventType)
	if len(request.Metadata) == 0 || string(request.Metadata) == "null" {
		request.Metadata = json.RawMessage(`{}`)
	}

	metadata, err := validateMetadata(eventType, request.Metadata)
	if err != nil {
		return "", nil, err
	}
	return eventType, metadata, nil
}

func validateMetadata(eventType string, raw json.RawMessage) (map[string]any, error) {
	switch eventType {
	case EventPageLanding:
		var value struct {
			Path string `json:"path"`
		}
		if err := decodeMetadata(raw, &value); err != nil || !validRequiredString(value.Path, 2048) {
			return nil, errInvalidMetadata
		}
		return map[string]any{"path": value.Path}, nil

	case EventLoadMoreClick:
		var value struct {
			Path       string `json:"path"`
			NextPage   int    `json:"next_page"`
			DistrictID string `json:"district_id,omitempty"`
		}
		if err := decodeMetadata(raw, &value); err != nil ||
			!validRequiredString(value.Path, 2048) || value.NextPage < 2 || value.NextPage > 1_000_000 ||
			!validOptionalString(value.DistrictID, 64) {
			return nil, errInvalidMetadata
		}
		metadata := map[string]any{"path": value.Path, "next_page": value.NextPage}
		if value.DistrictID != "" {
			metadata["district_id"] = value.DistrictID
		}
		return metadata, nil

	case EventDistrictSearch:
		var value struct {
			Query       string `json:"query"`
			ResultCount int    `json:"result_count"`
		}
		if err := decodeMetadata(raw, &value); err != nil ||
			!validRequiredString(value.Query, 100) || value.ResultCount < 0 || value.ResultCount > 8 {
			return nil, errInvalidMetadata
		}
		return map[string]any{"query": value.Query, "result_count": value.ResultCount}, nil

	case EventDistrictClick:
		var value struct {
			DistrictID string `json:"district_id"`
			Source     string `json:"source"`
		}
		if err := decodeMetadata(raw, &value); err != nil ||
			!validRequiredString(value.DistrictID, 64) ||
			!oneOf(value.Source, "search_result", "spot_card", "spot_detail") {
			return nil, errInvalidMetadata
		}
		return map[string]any{"district_id": value.DistrictID, "source": value.Source}, nil

	case EventSpotClick:
		var value struct {
			SpotID    string `json:"spot_id"`
			ShortCode string `json:"short_code"`
			Source    string `json:"source"`
		}
		if err := decodeMetadata(raw, &value); err != nil ||
			!validRequiredString(value.SpotID, 64) || !validRequiredString(value.ShortCode, 64) ||
			!oneOf(value.Source, "name", "thumbnail") {
			return nil, errInvalidMetadata
		}
		return map[string]any{
			"spot_id": value.SpotID, "short_code": value.ShortCode, "source": value.Source,
		}, nil

	default:
		return nil, errInvalidEventType
	}
}

func decodeMetadata(raw json.RawMessage, destination any) error {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	return ensureJSONEOF(decoder)
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request must contain one JSON value")
		}
		return err
	}
	return nil
}

func validRequiredString(value string, maxLength int) bool {
	return value == strings.TrimSpace(value) && value != "" && utf8.RuneCountInString(value) <= maxLength
}

func validOptionalString(value string, maxLength int) bool {
	return value == "" || validRequiredString(value, maxLength)
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func newUUID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80

	encoded := make([]byte, 36)
	hex.Encode(encoded[0:8], value[0:4])
	encoded[8] = '-'
	hex.Encode(encoded[9:13], value[4:6])
	encoded[13] = '-'
	hex.Encode(encoded[14:18], value[6:8])
	encoded[18] = '-'
	hex.Encode(encoded[19:23], value[8:10])
	encoded[23] = '-'
	hex.Encode(encoded[24:36], value[10:16])
	return string(encoded), nil
}
