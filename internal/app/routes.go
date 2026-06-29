package app

import (
	"html"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	defaultPage            = 1
	defaultLimit           = 10
	maxLimit               = 100
	siteName               = "空折枝"
	siteURL                = "https://afterthebloom.com/"
	defaultPageTitle       = "空折枝 - 哪裡有熄燈的店家？"
	defaultPageDescription = "空折枝整理熄燈店家的資訊，讓你看看哪些熟悉的地方已經悄悄告別。"
)

type paginatedResponse struct {
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
	Total int            `json:"total"`
	Items []SpotResponse `json:"items"`
}

type pageMetadata struct {
	Title       string
	Description string
	URL         string
	ImageURL    string
	SiteName    string
}

func RegisterRoutes(router *gin.Engine, store *Store, photosDir, indexPath string) {
	router.GET("/", func(c *gin.Context) {
		serveIndexPage(c, indexPath, defaultPageMetadata())
	})
	router.GET("/distrct/:districtId", func(c *gin.Context) {
		serveIndexPage(c, indexPath, defaultPageMetadata())
	})
	router.GET("/spot/:shortCode", func(c *gin.Context) {
		spot, ok := store.SpotByShortCode(c.Param("shortCode"))
		if !ok {
			serveIndexPage(c, indexPath, defaultPageMetadata())
			return
		}
		serveIndexPage(c, indexPath, spotPageMetadata(spot))
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

func defaultPageMetadata() pageMetadata {
	return pageMetadata{
		Title:       defaultPageTitle,
		Description: defaultPageDescription,
		URL:         siteURL,
		ImageURL:    "",
		SiteName:    siteName,
	}
}

func spotPageMetadata(spot SpotResponse) pageMetadata {
	districtName := strings.TrimSpace(spot.DistrictName)
	if districtName == "" {
		districtName = "未知鄉鎮市區"
	}

	description := spot.Name + "位於" + districtName + "。"
	if closedOn := strings.TrimSpace(spot.PermanentlyClosedOn); closedOn != "" {
		description = spot.Name + "位於" + districtName + "，已於" + closedOn + "熄燈。"
	}
	description += "空折枝持續整理這些悄悄告別的地方。"

	return pageMetadata{
		Title:       siteName + " - " + spot.Name,
		Description: description,
		URL:         siteURL + "spot/" + spot.ShortCode,
		ImageURL:    siteURL + "api/v1/spots/" + spot.ID + "/thumbnail",
		SiteName:    siteName,
	}
}

func serveIndexPage(c *gin.Context, indexPath string, meta pageMetadata) {
	indexHTML, err := os.ReadFile(indexPath)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	rendered := string(indexHTML)
	rendered = replaceTitleTag(rendered, meta.Title)
	rendered = replaceMetaContentByID(rendered, "meta-og-url", meta.URL)
	rendered = replaceMetaContentByID(rendered, "meta-og-title", meta.Title)
	rendered = replaceMetaContentByID(rendered, "meta-og-description", meta.Description)
	rendered = replaceMetaContentByID(rendered, "meta-og-image", meta.ImageURL)
	rendered = replaceMetaContentByID(rendered, "meta-og-site-name", meta.SiteName)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(rendered))
}

func replaceTitleTag(doc, value string) string {
	start := strings.Index(doc, "<title>")
	if start == -1 {
		return doc
	}
	end := strings.Index(doc[start:], "</title>")
	if end == -1 {
		return doc
	}
	end += start
	return doc[:start+len("<title>")] + html.EscapeString(value) + doc[end:]
}

func replaceMetaContentByID(doc, id, value string) string {
	marker := `id="` + id + `"`
	markerIndex := strings.Index(doc, marker)
	if markerIndex == -1 {
		return doc
	}

	tagStart := strings.LastIndex(doc[:markerIndex], "<meta")
	if tagStart == -1 {
		return doc
	}

	tagEnd := strings.Index(doc[markerIndex:], ">")
	if tagEnd == -1 {
		return doc
	}
	tagEnd += markerIndex

	tag := doc[tagStart : tagEnd+1]
	contentMarker := `content="`
	contentStart := strings.Index(tag, contentMarker)
	if contentStart == -1 {
		return doc
	}
	contentStart += len(contentMarker)

	contentEnd := strings.Index(tag[contentStart:], `"`)
	if contentEnd == -1 {
		return doc
	}
	contentEnd += contentStart

	replacedTag := tag[:contentStart] + html.EscapeString(value) + tag[contentEnd:]
	return doc[:tagStart] + replacedTag + doc[tagEnd+1:]
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
