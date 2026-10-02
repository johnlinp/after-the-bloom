package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johnlinp/after-the-bloom/internal/analytics"
	"github.com/johnlinp/after-the-bloom/internal/app"
)

const (
	dataPath                 = "data/atb-20260601.json"
	zipcodePath              = "data/tw-zipcode.json"
	photosPath               = "photos"
	indexPath                = "web/index.html"
	defaultAnalyticsDatabase = "after_the_bloom"
	geoIPPath                = "data/geoip/dbip-country-lite.mmdb"
)

func main() {
	log.SetFlags(log.LstdFlags)

	store, err := app.LoadStore(dataPath, zipcodePath)
	if err != nil {
		log.Fatalf("load store: %v", err)
	}

	recorder, closeRecorder := configureAnalyticsRecorder()
	defer closeRecorder()
	countryResolver, closeCountryResolver := configureCountryResolver()
	defer closeCountryResolver()
	analyticsHandler := analytics.NewHandler(recorder, countryResolver, log.Default())

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	if err := router.SetTrustedProxies(nil); err != nil {
		log.Fatalf("configure trusted proxies: %v", err)
	}
	app.RegisterRoutes(router, store, analyticsHandler, photosPath, indexPath)

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	server := &http.Server{Addr: ":" + port, Handler: router}
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	shutdownSignal, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("run server: %v", err)
		}
	case <-shutdownSignal.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shut down server: %v", err)
		}
	}
}

func configureAnalyticsRecorder() (analytics.Recorder, func()) {
	if !environmentEnabled("ANALYTICS_ENABLED") {
		log.Print("analytics storage disabled")
		return analytics.NoopRecorder{}, func() {}
	}

	uri := strings.TrimSpace(os.Getenv("MONGODB_URI"))
	if uri == "" {
		log.Fatal("MONGODB_URI is required when ANALYTICS_ENABLED is true")
	}
	database := strings.TrimSpace(os.Getenv("MONGODB_DATABASE"))
	if database == "" {
		database = defaultAnalyticsDatabase
	}

	ctx, cancel := context.WithTimeout(context.Background(), analytics.ConnectTimeout())
	defer cancel()
	recorder, err := analytics.NewMongoRecorder(ctx, uri, database)
	if err != nil {
		log.Fatalf("connect analytics database: %v", err)
	}
	return recorder, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := recorder.Close(ctx); err != nil {
			log.Printf("close analytics database: %v", err)
		}
	}
}

func configureCountryResolver() (analytics.CountryResolver, func()) {
	if !environmentEnabled("ANALYTICS_ENABLED") {
		return nil, func() {}
	}
	resolver, err := analytics.OpenMMDBCountryResolver(geoIPPath)
	if err != nil {
		log.Printf("country lookup disabled: %v", err)
		return nil, func() {}
	}
	return resolver, func() {
		if err := resolver.Close(); err != nil {
			log.Printf("close country database: %v", err)
		}
	}
}

func environmentEnabled(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
