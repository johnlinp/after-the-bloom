package analytics

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	maxRequestBytes = 16 * 1024
	recordTimeout   = 3 * time.Second
	requestsPerMin  = 60
	maxRateWindows  = 10_000
)

type Handler struct {
	recorder  Recorder
	countries CountryResolver
	logger    *log.Logger
	limiter   *rateLimiter
	now       func() time.Time
}

func NewHandler(recorder Recorder, countries CountryResolver, logger *log.Logger) *Handler {
	if recorder == nil {
		recorder = NoopRecorder{}
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Handler{
		recorder: recorder, countries: countries, logger: logger,
		limiter: newRateLimiter(requestsPerMin, time.Minute), now: time.Now,
	}
}

func (h *Handler) Record(c *gin.Context) {
	clientIP, hasClientIP := FirstPublicIP(c.GetHeader("X-Forwarded-For"))
	if hasClientIP && !h.limiter.Allow(clientIP.String(), h.now()) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBytes)
	eventType, metadata, err := DecodeEventRequest(c.Request.Body)
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request too large"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid analytics event"})
		return
	}

	id, err := newUUID()
	if err != nil {
		h.logger.Printf("generate analytics event ID: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "analytics unavailable"})
		return
	}

	var countryCode *string
	if h.countries != nil {
		countryCode = h.countries.CountryCode(c.Request)
	}
	event := Event{
		ID: id, EventType: eventType, ClientCountryCode: countryCode,
		Metadata: metadata, CreatedAt: h.now().UTC(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), recordTimeout)
	defer cancel()
	if err := h.recorder.Record(ctx, event); err != nil {
		h.logger.Printf("record analytics event %q: %v", eventType, err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "analytics unavailable"})
		return
	}
	c.Status(http.StatusNoContent)
}

type rateWindow struct {
	startedAt time.Time
	count     int
}

type rateLimiter struct {
	mu       sync.Mutex
	limit    int
	duration time.Duration
	windows  map[string]rateWindow
}

func newRateLimiter(limit int, duration time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, duration: duration, windows: make(map[string]rateWindow)}
}

func (l *rateLimiter) Allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	window, exists := l.windows[key]
	if !exists && len(l.windows) >= maxRateWindows {
		for candidate, existingWindow := range l.windows {
			if now.Sub(existingWindow.startedAt) >= l.duration {
				delete(l.windows, candidate)
			}
		}
		if len(l.windows) >= maxRateWindows {
			return false
		}
	}
	if !exists || now.Sub(window.startedAt) >= l.duration {
		l.windows[key] = rateWindow{startedAt: now, count: 1}
		return true
	}
	if window.count >= l.limit {
		return false
	}
	window.count++
	l.windows[key] = window
	return true
}
