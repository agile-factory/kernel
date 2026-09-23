package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"
)

func size(l *TokenBucketLimiter) int {
	n := 0
	l.m.Range(func(any, any) bool { n++; return true })
	return n
}

func TestNew_RejectsNonPositive(t *testing.T) {
	for _, tc := range []struct{ rate, burst int }{{0, 1}, {1, 0}, {-1, 5}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("New(%d, %d) should panic", tc.rate, tc.burst)
				}
			}()
			New(tc.rate, tc.burst)
		}()
	}
}

func TestAllow_BurstThenDeny(t *testing.T) {
	l := NewRate(1.0/60, 3)
	for i := 0; i < 3; i++ {
		if !l.Allow("k") {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	if l.Allow("k") {
		t.Fatal("4th request should be denied")
	}
	if !l.Allow("other") {
		t.Fatal("keys are independent")
	}
}

func TestSweep_DropsIdleBuckets(t *testing.T) {
	l := New(100, 1) // refills in 10ms
	for _, k := range []string{"a", "b", "c"} {
		l.Allow(k)
	}
	// Pretend the last sweep was long ago and the buckets are idle.
	l.lastSweep.Store(time.Now().Add(-2 * sweepInterval).UnixNano())
	time.Sleep(20 * time.Millisecond)
	l.Allow("d")
	if n := size(l); n != 1 {
		t.Fatalf("idle buckets should be swept, %d keys left", n)
	}
}

func req(remote string, xff ...string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = remote
	for _, v := range xff {
		r.Header.Add("X-Forwarded-For", v)
	}
	return r
}

func TestIPKeyFn_IgnoresForwardedFor(t *testing.T) {
	if got := IPKeyFn(req("203.0.113.7:4242", "1.2.3.4")); got != "203.0.113.7" {
		t.Fatalf("got %q", got)
	}
	if got := IPKeyFn(req("[2001:db8::1]:443")); got != "2001:db8::1" {
		t.Fatalf("IPv6: got %q", got)
	}
}

func TestTrustedProxyIPKeyFn(t *testing.T) {
	fn := TrustedProxyIPKeyFn([]netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")})
	cases := []struct {
		name string
		r    *http.Request
		want string
	}{
		{"untrusted peer, XFF ignored", req("203.0.113.7:1", "1.2.3.4"), "203.0.113.7"},
		{"trusted peer", req("10.0.0.2:1", "198.51.100.9"), "198.51.100.9"},
		{"spoofed left entry ignored", req("10.0.0.2:1", "1.2.3.4, 198.51.100.9"), "198.51.100.9"},
		{"proxy chain", req("10.0.0.2:1", "198.51.100.9, 10.0.0.5"), "198.51.100.9"},
		{"multiple headers", req("10.0.0.2:1", "1.2.3.4", "198.51.100.9"), "198.51.100.9"},
		{"no XFF", req("10.0.0.2:1"), "10.0.0.2"},
		{"garbage", req("10.0.0.2:1", "not-an-ip"), "10.0.0.2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := fn(tc.r); got != tc.want {
				t.Fatalf("want %q, got %q", tc.want, got)
			}
		})
	}
}

func TestTenantKeyFn(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/t-1/settings", nil)
	if got := TenantKeyFn(r); got != "t-1" {
		t.Fatalf("got %q", got)
	}
	a := TenantKeyFn(httptest.NewRequest(http.MethodGet, "/random/1", nil))
	b := TenantKeyFn(httptest.NewRequest(http.MethodGet, "/random/2", nil))
	if a != b {
		t.Fatal("non-tenant paths must share one key")
	}
}

func TestMiddleware_429(t *testing.T) {
	l := New(1, 1)
	h := Middleware(l, IPKeyFn)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	codes := []int{}
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req("203.0.113.7:1"))
		codes = append(codes, rec.Code)
	}
	if codes[0] != http.StatusOK || codes[1] != http.StatusTooManyRequests {
		t.Fatalf("codes = %v", codes)
	}
}
