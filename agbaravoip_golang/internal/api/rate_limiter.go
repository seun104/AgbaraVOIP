package api

import (
	"sync"
	"time"
	"net/http"

	"golang.org/x/time/rate"
	"github.com/gin-gonic/gin"
	// "github.com/sirupsen/logrus" // Not directly used in this version of PerClientRateLimiter
)

// ClientLimiter stores a rate limiter for each client IP address.
type ClientLimiter struct {
	IP        string
	Limiter   *rate.Limiter
	LastSeen  time.Time
}

var (
	clients = make(map[string]*ClientLimiter)
	mu      sync.Mutex
)

// InitRateLimiterCleanup starts a goroutine to periodically clean up old client limiters.
// This should be called once at application startup (e.g., in main.go).
// cleanupInterval: How often to run the cleanup process.
// clientExpiry: How long a client can be inactive before being removed.
func InitRateLimiterCleanup(cleanupInterval time.Duration, clientExpiry time.Duration) {
	go func() {
		for {
			time.Sleep(cleanupInterval)
			mu.Lock()
			for ip, client := range clients {
				if time.Since(client.LastSeen) > clientExpiry {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()
}

// PerClientRateLimiter is a middleware that limits requests per client IP.
// rps is requests per second, burst is the maximum burst size.
func PerClientRateLimiter(rps rate.Limit, burst int) gin.HandlerFunc {
	// Ensure InitRateLimiterCleanup is called once from main.go or a similar central place.
	// Example: InitRateLimiterCleanup(10*time.Minute, 30*time.Minute)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		mu.Lock()
		client, exists := clients[ip]
		if !exists {
			client = &ClientLimiter{
				IP:      ip,
				Limiter: rate.NewLimiter(rps, burst),
			}
			clients[ip] = client
			// logrus.Debugf("Rate limiter created for IP: %s with RPS: %v, Burst: %d", ip, rps, burst) // Example logging
		}
		client.LastSeen = time.Now()
		mu.Unlock()

		if !client.Limiter.Allow() {
			// logrus.Warnf("Rate limit exceeded for IP: %s", ip) // Example logging
			c.AbortWithStatusJSON(http.StatusTooManyRequests, GenericErrorResponse{
				Error: "Too Many Requests",
				Details: "You have exceeded the request limit. Please try again later.",
			})
			return
		}
		c.Next()
	}
}
