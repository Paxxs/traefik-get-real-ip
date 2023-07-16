package traefik_get_real_ip_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	plugin "github.com/Paxxs/traefik-get-real-ip"
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
			ProxyHeadername:  "X-From-Cdn",
			ProxyHeadervalue: "4",
			RealIP:           "X-Forwarded-For",
		},
		{
			ProxyHeadername:  "X-From-Cdn",
			ProxyHeadervalue: "5",
			RealIP:           "Client-Ip",
		},
		{
			ProxyHeadername:  "*",
			ProxyHeadervalue: "6",
			RealIP:           "RemoteAddr",
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
			remoteAddr:   "172.18.0.1:1000",
			expected:     "10.0.0.1",
		},
		{
			xff:          "10.0.1.2",
			xFromProxy:   "2",
			realIPHeader: "Client-Ip",
			realIP:       "10.0.1.1",
			desc:         "Proxy 2 通过 Client-Ip 传递IP",
			remoteAddr:   "172.18.0.2:2000",
			expected:     "10.0.1.1",
		},
		{
			xff:          "10.0.0.2",
			xFromProxy:   "3",
			realIPHeader: "Cf-Connecting-Ip",
			realIP:       "10.0.2.1",
			desc:         "Proxy 3 通过 Cf-Connecting-Ip 传递IP",
			remoteAddr:   "172.18.0.3:3000",
			expected:     "10.0.2.1",
		},
		{
			xff:          "奇怪的,东西🤣,10.0.3.1:2345, 2.2.2.2,3.3.3.3",
			xFromProxy:   "4",
			realIPHeader: "Client-Ip",
			realIP:       "10.0.3.2",
			desc:         "Proxy 4 通过 xff 传递IP带端口号",
			remoteAddr:   "172.18.0.4:4000",
			expected:     "10.0.3.1",
		},
		{
			xff:        "10.0.5.1",
			xFromProxy: "5",
			realIP:     "RemoteAddr",
			desc:       "Proxy 5 取远程地址",
			remoteAddr: "172.18.0.5:55122",
			expected:   "172.18.0.5",
		},
		{
			xff:          "sss",
			xFromProxy:   "6",
			realIPHeader: "Client-Ip",
			realIP:       "6",
			desc:         "Proxy 6 未正确传递",
			remoteAddr:   "172.18.0.6:6000",
			expected:     "172.18.0.6",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			reorder := httptest.NewRecorder()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
			if err != nil {
				t.Fatal(err)
			}

			fmt.Println("\n😊 测试:", test.desc)
			fmt.Println(test)

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
		t.Errorf("invalid header value: got %s, want %s", req.Header.Get(key), expected)
	}
}
