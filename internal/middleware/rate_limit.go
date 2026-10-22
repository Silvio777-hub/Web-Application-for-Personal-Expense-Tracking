package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// RateLimiter implements a simple token bucket rate limiter.
type RateLimiter struct {
	ips            sync.Map // Map[string]*visitor
	trustedProxies map[string]bool
}

type visitor struct {
	lastSeen time.Time
	tokens   float64
	rate     float64
	burst    float64
	sync.Mutex
}

func NewRateLimiter(trustedProxyStr string) *RateLimiter {
	proxies := make(map[string]bool)
	if trustedProxyStr != "" {
		for _, p := range strings.Split(trustedProxyStr, ",") {
			proxies[strings.TrimSpace(p)] = true
		}
	}

	rl := &RateLimiter{
		trustedProxies: proxies,
	}
	// Cleanup old entries every hour
	go func() {
		for {
			time.Sleep(1 * time.Hour)
			rl.ips.Range(func(key, value interface{}) bool {
				v := value.(*visitor)
				v.Lock()
				if time.Since(v.lastSeen) > 24*time.Hour {
					rl.ips.Delete(key)
				}
				v.Unlock()
				return true
			})
		}
	}()
	return rl
}

// Limit is a middleware that limits requests per IP for a given key (e.g. endpoint name).
func (rl *RateLimiter) Limit(keyPrefix string, rate float64, burst float64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			remoteIP := r.RemoteAddr
			if lastColon := strings.LastIndex(remoteIP, ":"); lastColon != -1 {
				remoteIP = remoteIP[:lastColon]
			}

			ip := remoteIP

			// If from trusted proxy, we can trust X-Forwarded-For
			if rl.trustedProxies[remoteIP] || remoteIP == "127.0.0.1" || remoteIP == "::1" || remoteIP == "" {
				xff := r.Header.Get("X-Forwarded-For")
				if xff != "" {
					if comma := strings.Index(xff, ","); comma != -1 {
						ip = strings.TrimSpace(xff[:comma])
					} else {
						ip = strings.TrimSpace(xff)
					}
				}
			}

			key := keyPrefix + ":" + ip

			v, _ := rl.ips.LoadOrStore(key, &visitor{
				lastSeen: time.Now(),
				tokens:   burst,
				rate:     rate,
				burst:    burst,
			})

			vis := v.(*visitor)
			vis.Lock()
			defer vis.Unlock()

			// Refill tokens based on time passed
			now := time.Now()
			dt := now.Sub(vis.lastSeen).Seconds()
			vis.tokens += dt * vis.rate
			if vis.tokens > vis.burst {
				vis.tokens = vis.burst
			}
			vis.lastSeen = now

			if vis.tokens < 1 {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			vis.tokens--
			next.ServeHTTP(w, r)
		})
	}
}
