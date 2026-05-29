# After the Bloom

Two command-line tools for working with the Google Places API (New):

- `place-retriever`: Searches for places inside a bounding box and writes `id,name` CSV output.
- `place-validator`: Reads a CSV of place IDs, checks business status, and writes `id,name,status` CSV output.

## Requirements

- Go 1.26+
- `PLACES_API_KEY` environment variable set

## Build

```bash
go build ./cmd/place-retriever
go build ./cmd/place-validator
```

## Usage

Retrieve places:

```bash
go run ./cmd/place-retriever \
  -text-query="restaurant" \
  -low-lat=25.010784236124856 \
  -low-lng=121.52779034443573 \
  -high-lat=25.024676516055315 \
  -high-lng=121.54806758579765 \
  -max-records=100 \
  -output=places.csv
```

Validate places:

```bash
go run ./cmd/place-validator \
  -input=places.csv \
  -output=statuses.csv
```

Operational logs are written to stderr so CSV output stays clean.
