package server

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// limiter is a token bucket per client IP for the endpoints anyone on the network can
// reach without signing in. Tokens are long and random, so guessing is hopeless anyway;
// the limit keeps a misbehaving client from flooding the log and the database.
type limiter struct {
	mu      sync.Mutex
	perSec  float64
	burst   float64
	buckets map[string]*bucket
	clock   func() time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

func newLimiter(perMinute, burst int, clock func() time.Time) *limiter {
	return &limiter{perSec: float64(perMinute) / 60, burst: float64(burst), buckets: map[string]*bucket{}, clock: clock}
}

// allow takes a token for key, or reports how long until the next one.
func (l *limiter) allow(key string) (bool, time.Duration) {
	now := l.clock()
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.buckets[key]
	if !ok {
		if len(l.buckets) >= 4096 {
			l.prune(now)
		}
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	b.tokens = math.Min(l.burst, b.tokens+now.Sub(b.last).Seconds()*l.perSec)
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	return false, time.Duration((1 - b.tokens) / l.perSec * float64(time.Second))
}

// prune forgets clients whose bucket has refilled: they are indistinguishable from new ones.
func (l *limiter) prune(now time.Time) {
	for key, b := range l.buckets {
		if b.tokens+now.Sub(b.last).Seconds()*l.perSec >= l.burst {
			delete(l.buckets, key)
		}
	}
}

func (l *limiter) wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ok, wait := l.allow(clientIP(r)); !ok {
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
			writeError(w, http.StatusTooManyRequests, "Too many attempts. Wait a moment and try again.")
			return
		}
		next(w, r)
	}
}

// clientIP is the TCP peer. X-Forwarded-For is deliberately ignored: any client can set it.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
