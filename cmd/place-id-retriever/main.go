package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/johnlinp/after-the-bloom/internal/places"
)

func main() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags)

	if err := run(); err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	mapsURL := flag.String("maps-url", "", "Full Google Maps place URL")
	flag.Parse()

	if *mapsURL == "" {
		return fmt.Errorf("-maps-url is required")
	}

	expandedURL, err := places.ExpandGoogleMapsURL(context.Background(), *mapsURL)
	if err != nil {
		return err
	}

	cidHex, err := places.ExtractCIDFromGoogleMapsURL(expandedURL)
	if err != nil {
		return err
	}

	cidDecimal, err := places.CIDHexToDecimalString(cidHex)
	if err != nil {
		return err
	}

	client, err := places.NewClient()
	if err != nil {
		return err
	}

	placeID, err := client.GetPlaceIDFromCID(context.Background(), cidDecimal)
	if err != nil {
		return err
	}

	fmt.Printf("place_id=%s\n", placeID)
	return nil
}
