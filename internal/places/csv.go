package places

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type RetrievedPlace struct {
	ID                  string
	Name                string
	BusinessStatus      string
	LocationLatitude    string
	LocationLongitude   string
	OpeningDate         string
	GoogleMapsLinksJSON string
	PostalAddressJSON   string
	Rating              string
	UserRatingCount     int
}

type ValidatedPlace struct {
	ID     string
	Name   string
	Status string
}

func WriteRetrievedPlacesCSV(path string, places []RetrievedPlace) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create output CSV %q: %w", path, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{
		"id",
		"name",
		"business_status",
		"location_latitude",
		"location_longitude",
		"opening_date",
		"google_maps_links",
		"postal_address",
		"rating",
		"user_rating_count",
	}); err != nil {
		return fmt.Errorf("write header to %q: %w", path, err)
	}

	for _, place := range places {
		if err := writer.Write([]string{
			place.ID,
			place.Name,
			place.BusinessStatus,
			place.LocationLatitude,
			place.LocationLongitude,
			place.OpeningDate,
			place.GoogleMapsLinksJSON,
			place.PostalAddressJSON,
			place.Rating,
			strconv.Itoa(place.UserRatingCount),
		}); err != nil {
			return fmt.Errorf("write row to %q: %w", path, err)
		}
	}

	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush CSV %q: %w", path, err)
	}

	return nil
}

func MustMarshalJSON(value any) string {
	if value == nil {
		return ""
	}

	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}

	return string(data)
}

func ReadPlaceIDsCSV(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open input CSV %q: %w", path, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read CSV %q: %w", path, err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("input CSV %q is empty", path)
	}

	var ids []string
	for idx, row := range rows[1:] {
		if len(row) == 0 {
			return nil, fmt.Errorf("row %d in %q has no columns", idx+2, path)
		}
		id := strings.TrimSpace(row[0])
		if id == "" {
			continue
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("input CSV %q contains no place IDs", path)
	}

	return ids, nil
}

func WriteValidatedPlacesCSV(path string, places []ValidatedPlace) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create output CSV %q: %w", path, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{"id", "name", "status"}); err != nil {
		return fmt.Errorf("write header to %q: %w", path, err)
	}

	for _, place := range places {
		if err := writer.Write([]string{place.ID, place.Name, place.Status}); err != nil {
			return fmt.Errorf("write row to %q: %w", path, err)
		}
	}

	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush CSV %q: %w", path, err)
	}

	return nil
}
