# 空折枝 / After the Bloom

A Gin-based single-page web application for browsing permanently closed local businesses, backed by the `data/atb-20260601.json` dataset loaded into memory at startup.

## Requirements

- Go 1.26+

## Run locally

```bash
go run ./cmd/after-the-bloom
```

The server binds to `PORT` when present and falls back to `8080`.

## API

- `GET /api/v1/spots?page=1&limit=10`
- `GET /api/v1/hubs`
- `GET /api/v1/hubs/:hubId/spots?page=1&limit=10`
- `GET /api/v1/spots/:spotId/thumbnail`

## Existing tools

The earlier Google Places helpers are still available:

- `go run ./cmd/place-discoverer`
- `go run ./cmd/place-validator`

Retrieve places:

```bash
go run ./cmd/place-discoverer \
  -text-query="restaurant" \
  -low-lat=25.010784236124856 \
  -low-lng=121.52779034443573 \
  -high-lat=25.024676516055315 \
  -high-lng=121.54806758579765 \
  -tile-size-degree=0.002 \
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

## Heroku

The repo includes a `Procfile`:

```bash
web: bin/after-the-bloom
```

That keeps the app compatible with Heroku's `PORT`-based runtime model.
