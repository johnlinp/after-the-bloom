package main

import (
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/johnlinp/after-the-bloom/internal/app"
)

const (
	dataPath   = "data/atb-20260601.json"
	photosPath = "photos"
	indexPath  = "web/index.html"
)

func main() {
	log.SetFlags(log.LstdFlags)

	store, err := app.LoadStore(dataPath)
	if err != nil {
		log.Fatalf("load store: %v", err)
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	if err := router.SetTrustedProxies(nil); err != nil {
		log.Fatalf("configure trusted proxies: %v", err)
	}
	app.RegisterRoutes(router, store, photosPath, indexPath)

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
