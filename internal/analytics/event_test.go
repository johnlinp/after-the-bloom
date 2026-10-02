package analytics

import (
	"strings"
	"testing"
)

func TestDecodeEventRequest(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		eventType string
		wantError bool
	}{
		{
			name:      "page landing",
			body:      `{"event_type":"page_landing","metadata":{"path":"/"}}`,
			eventType: EventPageLanding,
		},
		{
			name:      "load more",
			body:      `{"event_type":"load_more_click","metadata":{"path":"/district/tw-106","next_page":2,"district_id":"tw-106"}}`,
			eventType: EventLoadMoreClick,
		},
		{
			name:      "district search",
			body:      `{"event_type":"district_search","metadata":{"query":"大安","result_count":1}}`,
			eventType: EventDistrictSearch,
		},
		{
			name:      "district click",
			body:      `{"event_type":"district_click","metadata":{"district_id":"tw-106","source":"search_result"}}`,
			eventType: EventDistrictClick,
		},
		{
			name:      "spot click",
			body:      `{"event_type":"spot_click","metadata":{"spot_id":"spot-id","short_code":"abc123","source":"thumbnail"}}`,
			eventType: EventSpotClick,
		},
		{
			name:      "unknown event",
			body:      `{"event_type":"unknown","metadata":{}}`,
			wantError: true,
		},
		{
			name:      "unknown top-level field",
			body:      `{"event_type":"page_landing","metadata":{"path":"/"},"created_at":"spoofed"}`,
			wantError: true,
		},
		{
			name:      "unknown metadata field",
			body:      `{"event_type":"page_landing","metadata":{"path":"/","ip":"spoofed"}}`,
			wantError: true,
		},
		{
			name:      "invalid source",
			body:      `{"event_type":"spot_click","metadata":{"spot_id":"spot-id","short_code":"abc123","source":"card"}}`,
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			eventType, metadata, err := DecodeEventRequest(strings.NewReader(test.body))
			if test.wantError {
				if err == nil {
					t.Fatal("DecodeEventRequest() error = nil, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("DecodeEventRequest() error = %v", err)
			}
			if eventType != test.eventType {
				t.Fatalf("event type = %q, want %q", eventType, test.eventType)
			}
			if metadata == nil {
				t.Fatal("metadata = nil")
			}
		})
	}
}

func TestNewUUID(t *testing.T) {
	first, err := newUUID()
	if err != nil {
		t.Fatalf("newUUID() error = %v", err)
	}
	second, err := newUUID()
	if err != nil {
		t.Fatalf("newUUID() error = %v", err)
	}
	if len(first) != 36 || first == second {
		t.Fatalf("unexpected UUIDs: %q and %q", first, second)
	}
}
