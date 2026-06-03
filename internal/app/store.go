package app

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Hub struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Aliases []string `json:"aliases"`
}

type Spot struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	GoogleMapURL          string `json:"google_map_url"`
	CurrentBusinessStatus string `json:"current_business_status"`
	PermanentlyClosedOn   string `json:"permanently_closed_on"`
}

type HubSpot struct {
	ID     string `json:"id"`
	HubID  string `json:"hub_id"`
	SpotID string `json:"spot_id"`
}

type dataset struct {
	Hubs     []Hub     `json:"hubs"`
	Spots    []Spot    `json:"spots"`
	HubXSpot []HubSpot `json:"hub_x_spot"`
}

type SpotResponse struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	GoogleMapURL          string `json:"google_map_url"`
	CurrentBusinessStatus string `json:"current_business_status"`
	PermanentlyClosedOn   string `json:"permanently_closed_on"`
	HubID                 string `json:"hub_id,omitempty"`
	HubName               string `json:"hub_name,omitempty"`
}

type Store struct {
	hubs              []Hub
	spots             []Spot
	spotByID          map[string]Spot
	hubByID           map[string]Hub
	hubIDBySpotID     map[string]string
	spotsByHubID      map[string][]Spot
	sortedAllSpotView []SpotResponse
}

func LoadStore(path string) (*Store, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open dataset %q: %w", path, err)
	}
	defer file.Close()

	var raw dataset
	if err := json.NewDecoder(file).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode dataset %q: %w", path, err)
	}

	store := &Store{
		hubs:          append([]Hub(nil), raw.Hubs...),
		spots:         append([]Spot(nil), raw.Spots...),
		spotByID:      make(map[string]Spot, len(raw.Spots)),
		hubByID:       make(map[string]Hub, len(raw.Hubs)),
		hubIDBySpotID: make(map[string]string, len(raw.HubXSpot)),
		spotsByHubID:  make(map[string][]Spot, len(raw.Hubs)),
	}

	for _, hub := range raw.Hubs {
		store.hubByID[hub.ID] = hub
	}

	for _, spot := range raw.Spots {
		store.spotByID[spot.ID] = spot
	}

	for _, link := range raw.HubXSpot {
		spot, ok := store.spotByID[link.SpotID]
		if !ok {
			continue
		}
		if _, exists := store.hubByID[link.HubID]; !exists {
			continue
		}
		if _, seen := store.hubIDBySpotID[link.SpotID]; !seen {
			store.hubIDBySpotID[link.SpotID] = link.HubID
		}
		store.spotsByHubID[link.HubID] = append(store.spotsByHubID[link.HubID], spot)
	}

	sort.Slice(store.hubs, func(i, j int) bool {
		return store.hubs[i].Name < store.hubs[j].Name
	})

	for hubID, spots := range store.spotsByHubID {
		sortSpots(spots)
		store.spotsByHubID[hubID] = spots
	}

	all := make([]SpotResponse, 0, len(store.spots))
	for _, spot := range store.sortedSpots() {
		all = append(all, store.toSpotResponse(spot))
	}
	store.sortedAllSpotView = all

	return store, nil
}

func (s *Store) Hubs() []Hub {
	return append([]Hub(nil), s.hubs...)
}

func (s *Store) SpotsPage(page, limit int) ([]SpotResponse, int) {
	return paginate(s.sortedAllSpotView, page, limit)
}

func (s *Store) SpotsPageByHub(hubID string, page, limit int) ([]SpotResponse, int, bool) {
	if _, ok := s.hubByID[hubID]; !ok {
		return nil, 0, false
	}

	spots := s.spotsByHubID[hubID]
	views := make([]SpotResponse, 0, len(spots))
	for _, spot := range spots {
		views = append(views, s.toSpotResponse(spot))
	}

	items, total := paginate(views, page, limit)
	return items, total, true
}

func (s *Store) HasSpot(spotID string) bool {
	_, ok := s.spotByID[spotID]
	return ok
}

func (s *Store) sortedSpots() []Spot {
	spots := append([]Spot(nil), s.spots...)
	sortSpots(spots)
	return spots
}

func (s *Store) toSpotResponse(spot Spot) SpotResponse {
	resp := SpotResponse{
		ID:                    spot.ID,
		Name:                  spot.Name,
		GoogleMapURL:          spot.GoogleMapURL,
		CurrentBusinessStatus: spot.CurrentBusinessStatus,
		PermanentlyClosedOn:   spot.PermanentlyClosedOn,
	}

	if hubID, ok := s.hubIDBySpotID[spot.ID]; ok {
		resp.HubID = hubID
		if hub, exists := s.hubByID[hubID]; exists {
			resp.HubName = hub.Name
		}
	}

	return resp
}

func paginate[T any](items []T, page, limit int) ([]T, int) {
	total := len(items)
	start := (page - 1) * limit
	if start >= total {
		return []T{}, total
	}

	end := start + limit
	if end > total {
		end = total
	}

	return append([]T(nil), items[start:end]...), total
}

func sortSpots(spots []Spot) {
	sort.SliceStable(spots, func(i, j int) bool {
		left := closureSortKey(spots[i].PermanentlyClosedOn)
		right := closureSortKey(spots[j].PermanentlyClosedOn)
		if left.year != right.year {
			return left.year > right.year
		}
		if left.month != right.month {
			return left.month > right.month
		}
		return spots[i].Name < spots[j].Name
	})
}

type sortKey struct {
	year  int
	month int
}

func closureSortKey(value string) sortKey {
	parts := strings.FieldsFunc(strings.TrimSpace(value), func(r rune) bool {
		return r == '/' || r == '-'
	})

	key := sortKey{}
	if len(parts) > 0 {
		key.year, _ = strconv.Atoi(parts[0])
	}
	if len(parts) > 1 {
		key.month, _ = strconv.Atoi(parts[1])
	}
	return key
}
