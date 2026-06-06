package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/johnlinp/after-the-bloom/internal/places"
)

const defaultTileSizeDegrees = 0.002

type boundingBox struct {
	lowLat  float64
	lowLng  float64
	highLat float64
	highLng float64
}

func main() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags)

	if err := run(); err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		textQuery  = flag.String("text-query", "", "Search term for Places text search")
		lowLat     = flag.Float64("low-lat", 0, "Southwest latitude")
		lowLng     = flag.Float64("low-lng", 0, "Southwest longitude")
		highLat    = flag.Float64("high-lat", 0, "Northeast latitude")
		highLng    = flag.Float64("high-lng", 0, "Northeast longitude")
		tileSize   = flag.Float64("tile-size-degree", defaultTileSizeDegrees, "Tile size in degrees for bounding box subdivision")
		maxRecords = flag.Int("max-records", 100, "Maximum number of records to collect")
		outputPath = flag.String("output", "", "Output CSV file path")
	)
	flag.Parse()

	if *textQuery == "" {
		return fmt.Errorf("-text-query is required")
	}
	if *maxRecords <= 0 {
		return fmt.Errorf("-max-records must be greater than zero")
	}
	if *tileSize <= 0 {
		return fmt.Errorf("-tile-size-degree must be greater than zero")
	}
	if *lowLat > *highLat {
		return fmt.Errorf("-low-lat must be less than or equal to -high-lat")
	}
	if *lowLng > *highLng {
		return fmt.Errorf("-low-lng must be less than or equal to -high-lng")
	}
	if *outputPath == "" {
		*outputPath = defaultRetrieverOutputPath(*textQuery)
	}

	client, err := places.NewClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	boxes := splitBoundingBox(*lowLat, *lowLng, *highLat, *highLng, *tileSize)
	collected := make([]places.RetrievedPlace, 0, *maxRecords)
	seenIDs := make(map[string]struct{}, *maxRecords)

	for boxIndex, box := range boxes {
		if len(collected) >= *maxRecords {
			break
		}

		log.Printf(
			"searching tile %d/%d: low=(%.6f, %.6f) high=(%.6f, %.6f)",
			boxIndex+1,
			len(boxes),
			box.lowLat,
			box.lowLng,
			box.highLat,
			box.highLng,
		)

		nextPageToken := ""
		page := 1
		for len(collected) < *maxRecords {
			log.Printf("retrieving tile %d page %d for query %q", boxIndex+1, page, *textQuery)

			resp, err := client.SearchText(ctx, *textQuery, box.lowLat, box.lowLng, box.highLat, box.highLng, nextPageToken)
			if err != nil {
				return err
			}

			for _, place := range resp.Places {
				if _, exists := seenIDs[place.ID]; exists {
					continue
				}
				seenIDs[place.ID] = struct{}{}
				var locationLat string
				var locationLng string
				var rating string
				if place.Location != nil {
					locationLat = formatCoordinate(place.Location.Latitude)
					locationLng = formatCoordinate(place.Location.Longitude)
				}
				if place.Rating != nil {
					rating = formatRating(*place.Rating)
				}
				collected = append(collected, places.RetrievedPlace{
					ID:                  place.ID,
					Name:                place.DisplayName.Text,
					BusinessStatus:      place.BusinessStatus,
					LocationLatitude:    locationLat,
					LocationLongitude:   locationLng,
					GoogleMapsLinksJSON: places.MustMarshalJSON(place.GoogleMapsLinks),
					PostalAddressJSON:   places.MustMarshalJSON(place.PostalAddress),
					PhotosJSON:          places.MustMarshalJSON(place.Photos),
					Rating:              rating,
					UserRatingCount:     place.UserRatingCount,
				})
				if len(collected) >= *maxRecords {
					break
				}
			}

			log.Printf("collected %d/%d unique places", len(collected), *maxRecords)

			if len(collected) >= *maxRecords || resp.NextPageToken == "" {
				break
			}

			nextPageToken = resp.NextPageToken
			page++
			log.Printf("waiting 2s before requesting next page")
			time.Sleep(2 * time.Second)
		}
	}

	if err := places.WriteRetrievedPlacesCSV(*outputPath, collected); err != nil {
		return err
	}

	log.Printf("wrote %d places to %s", len(collected), *outputPath)
	return nil
}

func splitBoundingBox(lowLat, lowLng, highLat, highLng, step float64) []boundingBox {
	if step <= 0 {
		return []boundingBox{{
			lowLat:  lowLat,
			lowLng:  lowLng,
			highLat: highLat,
			highLng: highLng,
		}}
	}

	latTiles := maxInt(1, int(math.Ceil((highLat-lowLat)/step)))
	lngTiles := maxInt(1, int(math.Ceil((highLng-lowLng)/step)))

	boxes := make([]boundingBox, 0, latTiles*lngTiles)
	for lat := lowLat; lat < highLat; lat += step {
		nextLat := math.Min(lat+step, highLat)
		for lng := lowLng; lng < highLng; lng += step {
			nextLng := math.Min(lng+step, highLng)
			boxes = append(boxes, boundingBox{
				lowLat:  lat,
				lowLng:  lng,
				highLat: nextLat,
				highLng: nextLng,
			})
		}
	}

	if len(boxes) == 0 {
		boxes = append(boxes, boundingBox{
			lowLat:  lowLat,
			lowLng:  lowLng,
			highLat: highLat,
			highLng: highLng,
		})
	}

	return boxes
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func defaultRetrieverOutputPath(query string) string {
	return fmt.Sprintf("places-%s.csv", slugify(query))
}

func slugify(value string) string {
	var b strings.Builder
	lastHyphen := false

	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastHyphen = false
		case !lastHyphen:
			b.WriteByte('-')
			lastHyphen = true
		}
	}

	result := strings.Trim(b.String(), "-")
	if result == "" {
		return "places"
	}

	return result
}

func formatCoordinate(value float64) string {
	return fmt.Sprintf("%.8f", value)
}

func formatRating(value float64) string {
	return fmt.Sprintf("%.1f", value)
}
