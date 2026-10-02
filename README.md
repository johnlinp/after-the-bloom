# 空折枝 / After the Bloom

A Gin-based single-page web application for browsing permanently closed local businesses, backed by the `data/atb-20260601.json` dataset loaded into memory at startup.

## Requirements

- Go 1.26+

## Run locally

```bash
go run ./cmd/after-the-bloom
```

The server binds to `PORT` when present and falls back to `8080`.

## MongoDB

MongoDB is the application's persistent database.

Configure the database connection with:

```bash
export MONGODB_URI='mongodb+srv://...'
export MONGODB_DATABASE='after_the_bloom'
```

Keep the connection URI in environment configuration rather than source
control. The application creates the collections and indexes required by its
enabled features.

### Test MongoDB locally

Start MongoDB 8 in Docker:

```bash
docker run --name after-the-bloom-mongo \
  -p 27017:27017 \
  -e MONGO_INITDB_ROOT_USERNAME=atb \
  -e MONGO_INITDB_ROOT_PASSWORD=local-development-only \
  -d mongo:8.0
```

On later runs, use `docker start after-the-bloom-mongo`. The local connection
configuration is:

```bash
export MONGODB_URI='mongodb://atb:local-development-only@127.0.0.1:27017/?authSource=admin'
export MONGODB_DATABASE='after_the_bloom_dev'
```

Open an interactive shell without installing `mongosh` locally:

```bash
docker exec -it after-the-bloom-mongo mongosh \
  --username atb \
  --password local-development-only \
  --authenticationDatabase admin \
  after_the_bloom_dev
```

## Analytics

Analytics events help us understand how visitors use the site and where the experience can be improved.

Analytics is disabled unless enabled in the environment:

```bash
export ANALYTICS_ENABLED=true
```

The Heroku Go buildpack runs `bin/go-post-compile`, which downloads the pinned
GeoIP database to `data/geoip/dbip-country-lite.mmdb` in the deployed slug.
Update `DBIP_COUNTRY_LITE_VERSION` in `bin/download-geoip` when adopting a newer
monthly DB-IP release.

## API

- `POST /api/v1/analytics/events`
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
