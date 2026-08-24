// Package middleware implements Wish SSH server security middleware.
package middleware

import (
	"net"
	"sync"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimitMiddleware enforces a per-IP token bucket on new connections.
func RateLimitMiddleware(r rate.Limit, burst int) wish.Middleware {
	var (
		visitors = make(map[string]*visitor)
		mu       sync.Mutex
	)

	// Evict stale entries every minute.
	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, v := range visitors {
				if time.Since(v.lastSeen) > 3*time.Minute {
					delete(visitors, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			ip, _, err := net.SplitHostPort(sess.RemoteAddr().String())
			if err != nil {
				ip = sess.RemoteAddr().String()
			}

			mu.Lock()
			v, exists := visitors[ip]
			if !exists {
				v = &visitor{limiter: rate.NewLimiter(r, burst)}
				visitors[ip] = v
			}
			v.lastSeen = time.Now()
			mu.Unlock()

			if !v.limiter.Allow() {
				wish.Fatalln(sess, "Rate limit exceeded. Try again later.")
				return
			}
			next(sess)
		}
	}
}
