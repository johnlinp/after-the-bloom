package app

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type District struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	ZipCode string   `json:"zip_code"`
	Aliases []string `json:"aliases,omitempty"`
}

type Spot struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	ShortCode             string `json:"short_code"`
	ZipCode               string `json:"zip_code"`
	GoogleMapURL          string `json:"google_map_url"`
	WikipediaURL          string `json:"wikipedia_url"`
	CurrentBusinessStatus string `json:"current_business_status"`
	PermanentlyClosedOn   string `json:"permanently_closed_on"`
}

type dataset struct {
	Spots []Spot `json:"spots"`
}

type zipcodeDataset map[string]map[string]string

type zipcodeIndex struct {
	districts []District
	byZipCode map[string]string
	byName    map[string]District
}

type SpotResponse struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	ShortCode             string `json:"short_code"`
	ZipCode               string `json:"zip_code"`
	GoogleMapURL          string `json:"google_map_url"`
	WikipediaURL          string `json:"wikipedia_url"`
	CurrentBusinessStatus string `json:"current_business_status"`
	PermanentlyClosedOn   string `json:"permanently_closed_on"`
	DistrictID            string `json:"district_id,omitempty"`
	DistrictName          string `json:"district_name,omitempty"`
}

type Store struct {
	districts          []District
	spots              []Spot
	spotByID           map[string]Spot
	spotByShortCode    map[string]Spot
	districtByID       map[string]District
	districtIDBySpotID map[string]string
	spotsByDistrictID  map[string][]Spot
	sortedAllSpotView  []SpotResponse
}

func LoadStore(datasetPath, zipcodePath string) (*Store, error) {
	file, err := os.Open(datasetPath)
	if err != nil {
		return nil, fmt.Errorf("open dataset %q: %w", datasetPath, err)
	}
	defer file.Close()

	var raw dataset
	if err := json.NewDecoder(file).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode dataset %q: %w", datasetPath, err)
	}

	zipcodes, err := loadZipcodes(zipcodePath)
	if err != nil {
		return nil, err
	}

	store := &Store{
		spots:              append([]Spot(nil), raw.Spots...),
		spotByID:           make(map[string]Spot, len(raw.Spots)),
		spotByShortCode:    make(map[string]Spot, len(raw.Spots)),
		districts:          append([]District(nil), zipcodes.districts...),
		districtByID:       make(map[string]District, len(zipcodes.districts)),
		districtIDBySpotID: make(map[string]string, len(raw.Spots)),
		spotsByDistrictID:  make(map[string][]Spot),
	}

	for _, district := range store.districts {
		store.districtByID[district.ID] = district
	}

	for _, spot := range raw.Spots {
		store.spotByID[spot.ID] = spot
		if spot.ShortCode != "" {
			store.spotByShortCode[spot.ShortCode] = spot
		}

		districtName, ok := zipcodes.byZipCode[normalizeZipCode(spot.ZipCode)]
		if !ok {
			continue
		}

		district, exists := zipcodes.byName[districtName]
		if !exists {
			continue
		}

		districtID := district.ID
		store.districtIDBySpotID[spot.ID] = districtID
		store.spotsByDistrictID[districtID] = append(store.spotsByDistrictID[districtID], spot)
	}

	sortDistricts(store.districts)

	for districtID, spots := range store.spotsByDistrictID {
		sortSpots(spots)
		store.spotsByDistrictID[districtID] = spots
	}

	all := make([]SpotResponse, 0, len(store.spots))
	for _, spot := range store.sortedSpots() {
		all = append(all, store.toSpotResponse(spot))
	}
	store.sortedAllSpotView = all

	return store, nil
}

func loadZipcodes(path string) (*zipcodeIndex, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open zipcode dataset %q: %w", path, err)
	}
	defer file.Close()

	var raw zipcodeDataset
	if err := json.NewDecoder(file).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode zipcode dataset %q: %w", path, err)
	}

	index := &zipcodeIndex{
		districts: []District{},
		byZipCode: make(map[string]string),
		byName:    make(map[string]District),
	}

	for city, districts := range raw {
		for district, zip := range districts {
			fullName := strings.TrimSpace(city) + strings.TrimSpace(district)
			normalizedZip := strings.TrimSpace(zip)
			districtRecord := District{
				ID:      districtID(normalizedZip),
				Name:    fullName,
				ZipCode: normalizedZip,
				Aliases: districtAliases(fullName),
			}
			index.districts = append(index.districts, districtRecord)
			index.byName[fullName] = districtRecord
			if _, exists := index.byZipCode[normalizedZip]; !exists {
				index.byZipCode[normalizedZip] = fullName
			}
		}
	}

	sortDistricts(index.districts)

	return index, nil
}

func normalizeZipCode(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) >= 3 {
		return trimmed[:3]
	}
	return trimmed
}

func districtAliases(name string) []string {
	if len([]rune(name)) <= 3 {
		return nil
	}
	alias := string([]rune(name)[3:])
	if alias == name {
		return nil
	}
	return []string{alias}
}

func districtID(zipCode string) string {
	return "tw-" + zipCode
}

func sortDistricts(districts []District) {
	sort.Slice(districts, func(i, j int) bool {
		leftZip, leftErr := strconv.Atoi(strings.TrimSpace(districts[i].ZipCode))
		rightZip, rightErr := strconv.Atoi(strings.TrimSpace(districts[j].ZipCode))

		if leftErr == nil && rightErr == nil && leftZip != rightZip {
			return leftZip < rightZip
		}
		if districts[i].ZipCode != districts[j].ZipCode {
			return districts[i].ZipCode < districts[j].ZipCode
		}
		return districts[i].Name < districts[j].Name
	})
}

func (s *Store) Districts() []District {
	return append([]District(nil), s.districts...)
}

func (s *Store) SpotsPage(page, limit int) ([]SpotResponse, int) {
	return paginate(s.sortedAllSpotView, page, limit)
}

func (s *Store) SpotsPageByDistrict(districtID string, page, limit int) ([]SpotResponse, int, bool) {
	if _, ok := s.districtByID[districtID]; !ok {
		return nil, 0, false
	}

	spots := s.spotsByDistrictID[districtID]
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

func (s *Store) SpotByShortCode(shortCode string) (SpotResponse, bool) {
	spot, ok := s.spotByShortCode[strings.TrimSpace(shortCode)]
	if !ok {
		return SpotResponse{}, false
	}
	return s.toSpotResponse(spot), true
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
		ShortCode:             spot.ShortCode,
		ZipCode:               spot.ZipCode,
		GoogleMapURL:          spot.GoogleMapURL,
		WikipediaURL:          spot.WikipediaURL,
		CurrentBusinessStatus: spot.CurrentBusinessStatus,
		PermanentlyClosedOn:   spot.PermanentlyClosedOn,
	}

	if districtID, ok := s.districtIDBySpotID[spot.ID]; ok {
		resp.DistrictID = districtID
		if district, exists := s.districtByID[districtID]; exists {
			resp.DistrictName = district.Name
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
