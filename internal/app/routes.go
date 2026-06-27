package app

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	defaultPage  = 1
	defaultLimit = 10
	maxLimit     = 100
)

type paginatedResponse struct {
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
	Total int            `json:"total"`
	Items []SpotResponse `json:"items"`
}

func RegisterRoutes(router *gin.Engine, store *Store, photosDir, indexPath string) {
	router.GET("/", func(c *gin.Context) {
		c.File(indexPath)
	})
	router.GET("/distrct/:districtId", func(c *gin.Context) {
		c.File(indexPath)
	})
	router.GET("/spot/:shortCode", func(c *gin.Context) {
		c.File(indexPath)
	})

	api := router.Group("/api/v1")
	{
		api.GET("/spots", func(c *gin.Context) {
			page, limit := parsePagination(c)
			items, total := store.SpotsPage(page, limit)
			c.JSON(http.StatusOK, paginatedResponse{
				Page:  page,
				Limit: limit,
				Total: total,
				Items: items,
			})
		})

		api.GET("/districts", func(c *gin.Context) {
			c.JSON(http.StatusOK, store.Districts())
		})

		api.GET("/districts/:districtId/spots", func(c *gin.Context) {
			page, limit := parsePagination(c)
			items, total, ok := store.SpotsPageByDistrict(c.Param("districtId"), page, limit)
			if !ok {
				c.JSON(http.StatusNotFound, gin.H{"error": "district not found"})
				return
			}
			c.JSON(http.StatusOK, paginatedResponse{
				Page:  page,
				Limit: limit,
				Total: total,
				Items: items,
			})
		})

		api.GET("/spots/short-code/:shortCode", func(c *gin.Context) {
			spot, ok := store.SpotByShortCode(c.Param("shortCode"))
			if !ok {
				c.JSON(http.StatusNotFound, gin.H{"error": "spot not found"})
				return
			}
			c.JSON(http.StatusOK, spot)
		})

		api.GET("/spots/:spotId/thumbnail", func(c *gin.Context) {
			spotID := strings.TrimSpace(c.Param("spotId"))
			if spotID == "" || !store.HasSpot(spotID) {
				c.JSON(http.StatusNotFound, gin.H{"error": "spot not found"})
				return
			}

			matches, _ := filepath.Glob(filepath.Join(photosDir, spotID+".*"))
			if len(matches) == 0 {
				c.Status(http.StatusNotFound)
				return
			}

			c.File(matches[0])
		})
	}
}

func parsePagination(c *gin.Context) (int, int) {
	page := parsePositiveInt(c.Query("page"), defaultPage)
	limit := parsePositiveInt(c.Query("limit"), defaultLimit)
	if limit > maxLimit {
		limit = maxLimit
	}
	return page, limit
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
