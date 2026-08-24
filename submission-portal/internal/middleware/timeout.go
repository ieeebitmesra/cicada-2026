package middleware

import (
	"context"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
)

// TimeoutMiddleware kills sessions that exceed maxDuration.
func TimeoutMiddleware(maxDuration, _ time.Duration) wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			ctx, cancel := context.WithTimeout(sess.Context(), maxDuration)
			defer cancel()

			done := make(chan struct{})
			go func() {
				select {
				case <-ctx.Done():
					wish.Fatalln(sess, "\nSession timed out. Reconnect to continue.")
					sess.Close()
				case <-done:
					return
				}
			}()

			defer close(done)
			next(sess)
		}
	}
}
