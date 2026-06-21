package httpapi

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tranmh/pgnize/internal/config"
)

// When RATE_LIMIT_DISABLED is set, rateLimit must allow every request and never
// touch the Store. A nil Store here proves the bypass short-circuits before the
// DB call: any attempt to consult the limiter would panic.
func TestRateLimitDisabledBypassesStore(t *testing.T) {
	s := &Server{Cfg: config.Config{RateLimitDisabled: true}} // Store intentionally nil
	for i := 0; i < 100; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/convert", nil)
		if !s.rateLimit(rr, req, "convert:1.2.3.4", 10, time.Hour) {
			t.Fatalf("expected bypass when RateLimitDisabled, got 429 at request %d", i+1)
		}
		if rr.Code != 200 {
			t.Fatalf("bypass should not write a response, got status %d", rr.Code)
		}
	}
}
