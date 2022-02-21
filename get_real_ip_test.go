package traefik_get_real_ip_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	plugin "github.com/paxxs/traefik-get-real-ip"
)

func TestNew(t *testing.T) {
	cfg := plugin.CreateConfig()
	cfg.Proxy = []plugin.Proxy{
		{
			ProxyHeadername:  "X-From-Cdn",
			ProxyHeadervalue: "1",
			RealIP:           "X-Forwarded-For",
		},
		{
			ProxyHeadername:  "X-From-Cdn",
			ProxyHeadervalue: "2",
			RealIP:           "Client-Ip",
		},
		{
			ProxyHeadername:  "X-From-Cdn",
			ProxyHeadervalue: "3",
			RealIP:           "Cf-Connecting-Ip",
		},
		{
			ProxyHeadername: "*",
			RealIP:          "RemoteAddr",
		},
	}
	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {})

	handler, err := plugin.New(ctx, next, cfg, "traefik-get-real-ip")
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		xff          string // X-Forwarded-For
		xFromProxy   string // cdn标识
		realIPHeader string // CDN传递IP字段
		realIP       string // CDN传递IP字段值
		desc         string
		expected     string
		remoteAddr   string
	}{
		{
			xff:          "奇怪的,东西🤣,10.0.0.1, 2.2.2.2,3.3.3.3",
			xFromProxy:   "1",
			realIPHeader: "Client-Ip",
			realIP:       "10.0.0.2",
			desc:         "Proxy 1 通过 xff 传递IP",
			expected:     "10.0.0.1",
		},
		{
			xff:          "10.0.0.2",
			xFromProxy:   "2",
			realIPHeader: "Client-Ip",
			realIP:       "10.0.0.1",
			desc:         "Proxy 2 通过 Client-Ip 传递IP",
			expected:     "10.0.0.1",
		},
		{
			xff:          "10.0.0.2",
			xFromProxy:   "3",
			realIPHeader: "Cf-Connecting-Ip",
			realIP:       "10.0.0.1",
			desc:         "Proxy 3 通过 Cf-Connecting-Ip 传递IP",
			expected:     "10.0.0.1",
		},
		{
			xff:          "10.0.0.2",
			xFromProxy:   "4",
			realIPHeader: "Cf-Connecting-Ip",
			realIP:       "10.0.0.1",
			desc:         "Proxy 4 不存在",
			remoteAddr:   "1.1.1.1",
			expected:     "1.1.1.1",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			reorder := httptest.NewRecorder()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.RemoteAddr = test.remoteAddr
			req.Header.Set(test.realIPHeader, test.realIP)
			req.Header.Set("X-From-Cdn", test.xFromProxy)
			req.Header.Set("X-Forwarded-For", test.xff)

			handler.ServeHTTP(reorder, req)

			assertHeader(t, req, "X-Real-Ip", test.expected)
		})
	}
}

func assertHeader(t *testing.T, req *http.Request, key, expected string) {
	t.Helper()
	if req.Header.Get(key) != expected {
		t.Errorf("invalid header value: %s", req.Header.Get(key))
	}
}
