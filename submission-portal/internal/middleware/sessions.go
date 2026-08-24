package middleware

import (
	"net"
	"sync"
	"sync/atomic"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
)

// MaxSessionsMiddleware caps concurrent sessions globally and per source IP.
func MaxSessionsMiddleware(maxGlobal int32, maxPerIP int) wish.Middleware {
	var (
		globalCount int32
		perIP       = make(map[string]int)
		mu          sync.Mutex
	)

	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			ip, _, err := net.SplitHostPort(sess.RemoteAddr().String())
			if err != nil {
				ip = sess.RemoteAddr().String()
			}

			if atomic.LoadInt32(&globalCount) >= maxGlobal {
				wish.Fatalln(sess, "Server at maximum capacity.")
				return
			}

			mu.Lock()
			if perIP[ip] >= maxPerIP {
				mu.Unlock()
				wish.Fatalln(sess, "Too many sessions from your IP.")
				return
			}
			perIP[ip]++
			mu.Unlock()
			atomic.AddInt32(&globalCount, 1)

			defer func() {
				atomic.AddInt32(&globalCount, -1)
				mu.Lock()
				perIP[ip]--
				if perIP[ip] <= 0 {
					delete(perIP, ip)
				}
				mu.Unlock()
			}()

			next(sess)
		}
	}
}
