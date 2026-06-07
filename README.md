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
- `GET /api/v1/districts`
- `GET /api/v1/districts/:districtId/spots?page=1&limit=10`
- `GET /api/v1/spots/:spotId/thumbnail`

## Existing tools

The earlier Google Places helpers are still available:

- `go run ./cmd/place-discoverer`
- `go run ./cmd/place-id-retriever`
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

Retrieve a Google Place ID from a full or short Google Maps URL:

```bash
go run ./cmd/place-id-retriever \
  -maps-url='https://www.google.com/maps/place/AYin+Seafood+Porridge/@25.0246242,121.5432538,17z/data=!3m2!4b1!5s0x3442abd9dc7473b5:0x597d336dd8d813e9!4m6!3m5!1s0x3442aa2f4ecd9c5f:0xfa43f9af0fd3cee8!8m2!3d25.0246242!4d121.5432538!16s%2Fg%2F1v76xbkc?entry=ttu&g_ep=EgoyMDI2MDYwMS4wIKXMDSoASAFQAw%3D%3D'
```

```bash
go run ./cmd/place-id-retriever \
  -maps-url='https://maps.app.goo.gl/YwQgzE4vP4GWr2A59'
```

Operational logs are written to stderr so CSV output stays clean.

## Heroku

The repo includes a `Procfile`:

```bash
web: bin/after-the-bloom
```

That keeps the app compatible with Heroku's `PORT`-based runtime model.
