package server

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/lucasrafaldini/wavetrack/internal/store"
)

const (
	sessionCookie   = "wavetrack_session"
	sessionLifetime = 30 * 24 * time.Hour
	adminTokenKey   = "admin_token_sha256"
)

// NewToken returns a random token with a readable prefix, e.g. "wt_k3j…".
func NewToken(prefix string) string {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand never fails on supported platforms
	}
	return prefix + strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b))
}

func Hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// EnsureAdminToken creates the admin token on first run. It returns the plain token only
// when it was just created, because only its hash is stored.
func EnsureAdminToken(st *store.Store) (string, error) {
	if _, err := st.Meta(adminTokenKey); err == nil {
		return "", nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return "", err
	}
	return ResetAdminToken(st)
}

// ResetAdminToken issues a new admin token and signs every browser out.
func ResetAdminToken(st *store.Store) (string, error) {
	token := NewToken("wt_")
	if err := st.SetMeta(adminTokenKey, Hash(token)); err != nil {
		return "", err
	}
	return token, st.DeleteAllSessions()
}

func (s *Server) checkAdminToken(token string) bool {
	want, err := s.store.Meta(adminTokenKey)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(Hash(strings.TrimSpace(token))), []byte(want)) == 1
}

func (s *Server) signedIn(r *http.Request) bool {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return false
	}
	ok, err := s.store.SessionValid(Hash(cookie.Value), s.clock())
	return err == nil && ok
}

// startSession signs the browser in and returns the session token it was given.
func (s *Server) startSession(w http.ResponseWriter, r *http.Request) (string, error) {
	token := NewToken("")
	now := s.clock()
	if err := s.store.CreateSession(Hash(token), now, now.Add(sessionLifetime)); err != nil {
		return "", err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(sessionLifetime.Seconds()),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})
	return token, nil
}

func (s *Server) endSession(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		_ = s.store.DeleteSession(Hash(cookie.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
}

// requireSession guards the admin API.
func (s *Server) requireSession(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.signedIn(r) {
			writeError(w, http.StatusUnauthorized, "Sign in to continue.")
			return
		}
		next(w, r)
	}
}

// sameOrigin rejects state-changing requests from other sites. Browsers always send
// Origin on cross-site POST, PATCH and DELETE; the SameSite cookie covers the rest.
func sameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if origin := r.Header.Get("Origin"); origin != "" {
				host := strings.TrimPrefix(strings.TrimPrefix(origin, "https://"), "http://")
				if host != r.Host {
					writeError(w, http.StatusForbidden, "Cross-site request blocked.")
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self'; script-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		if r.TLS != nil {
			h.Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}
