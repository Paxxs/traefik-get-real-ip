package traefik_get_real_ip_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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

func TestLogging(t *testing.T) {
	// Create a pipe to capture stdout
	r, w, _ := os.Pipe()
	originalStdout := os.Stdout
	os.Stdout = w

	// Create plugin config with logging enabled
	cfg := plugin.CreateConfig()
	cfg.EnableLog = true
	cfg.Proxy = []plugin.Proxy{
		{
			ProxyHeadername:  "X-From-Cdn",
			ProxyHeadervalue: "1",
			RealIP:           "X-Forwarded-For",
		},
	}

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {})

	handler, err := plugin.New(ctx, next, cfg, "traefik-get-real-ip")
	if err != nil {
		t.Fatal(err)
	}

	// Create a test request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.RemoteAddr = "192.168.1.1:1234"
	req.Header.Set("X-From-Cdn", "1")
	req.Header.Set("X-Forwarded-For", "10.0.0.1")

	// Serve the request
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	// Restore stdout and read the captured output
	w.Close()
	os.Stdout = originalStdout
	var buf strings.Builder
	io.Copy(&buf, r)
	output := buf.String()

	// Check if logs contain expected messages
	expectedLogs := []string{
		"[get-realip] Instance created",
		"[get-realip] Processing proxy configuration",
		"[get-realip] Processing IP addresses",
		"[get-realip] Validating IP",
	}

	for _, expected := range expectedLogs {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected log output to contain '%s', but got:\n%s", expected, output)
		}
	}

	// Verify that X-Real-Ip was set correctly
	if req.Header.Get("X-Real-Ip") != "10.0.0.1" {
		t.Errorf("Expected X-Real-Ip to be set to '10.0.0.1', but got '%s'", req.Header.Get("X-Real-Ip"))
	}
}

func TestDeny403OnFail(t *testing.T) {
	// Create plugin config with deny403OnFail enabled
	cfg := plugin.CreateConfig()
	cfg.Deny403OnFail = true
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
	}

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// This should not be called when 403 is returned
		rw.WriteHeader(http.StatusOK)
	})

	handler, err := plugin.New(ctx, next, cfg, "traefik-get-real-ip")
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		desc           string
		xFromProxy     string
		expectedStatus int
	}{
		{
			desc:           "Matching proxy - should pass through",
			xFromProxy:     "1",
			expectedStatus: http.StatusOK,
		},
		{
			desc:           "Non-matching proxy - should return 403",
			xFromProxy:     "3", // Not in config
			expectedStatus: http.StatusForbidden,
		},
		{
			desc:           "Empty proxy header - should return 403",
			xFromProxy:     "",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			recorder := httptest.NewRecorder()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
			if err != nil {
				t.Fatal(err)
			}

			req.RemoteAddr = "192.168.1.1:1234"
			if test.xFromProxy != "" {
				req.Header.Set("X-From-Cdn", test.xFromProxy)
			}
			req.Header.Set("X-Forwarded-For", "10.0.0.1")

			handler.ServeHTTP(recorder, req)

			// Check response status code
			if recorder.Code != test.expectedStatus {
				t.Errorf("Expected status %d, got %d", test.expectedStatus, recorder.Code)
			}

			// For successful requests, verify the X-Real-Ip header was set
			if test.expectedStatus == http.StatusOK {
				if req.Header.Get("X-Real-Ip") == "" {
					t.Error("X-Real-Ip header not set for valid request")
				}
			}
		})
	}
}

// Test that deny403OnFail=false allows all requests
func TestDeny403Disabled(t *testing.T) {
	// Create plugin config with deny403OnFail disabled
	cfg := plugin.CreateConfig()
	cfg.Deny403OnFail = false // Default behavior
	cfg.Proxy = []plugin.Proxy{
		{
			ProxyHeadername:  "X-From-Cdn",
			ProxyHeadervalue: "1",
			RealIP:           "X-Forwarded-For",
		},
	}

	var nextHandlerCalled bool
	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		nextHandlerCalled = true
		rw.WriteHeader(http.StatusOK)
	})

	handler, err := plugin.New(ctx, next, cfg, "traefik-get-real-ip")
	if err != nil {
		t.Fatal(err)
	}

	// Create request with non-matching proxy header
	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.RemoteAddr = "192.168.1.1:1234"
	req.Header.Set("X-From-Cdn", "999") // Not in config
	req.Header.Set("X-Forwarded-For", "10.0.0.1")

	// Execute request
	handler.ServeHTTP(recorder, req)

	// Verify next handler was called and status is OK
	if !nextHandlerCalled {
		t.Error("Next handler not called when deny403OnFail is disabled")
	}

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

// Test the EraseProxyHeaders functionality
func TestEraseProxyHeaders(t *testing.T) {
	// Create plugin config with EraseProxyHeaders enabled
	cfg := plugin.CreateConfig()
	cfg.EraseProxyHeaders = true
	cfg.EnableLog = true // Enable logging for verification
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
			ProxyHeadername:  "*",
			ProxyHeadervalue: "4",
			RealIP:           "RemoteAddr",
		},
	}

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// Handler doesn't do anything
	})

	handler, err := plugin.New(ctx, next, cfg, "traefik-get-real-ip")
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		desc            string
		xFromProxy      string
		realIPHeader    string
		realIP          string
		remoteAddr      string
		expectedRealIP  string
		headersShouldBe map[string]bool // Header name -> should exist after processing
	}{
		{
			desc:           "Proxy 1 - should erase X-From-Cdn but keep X-Forwarded-For",
			xFromProxy:     "1",
			realIPHeader:   "X-Forwarded-For",
			realIP:         "10.0.0.1",
			remoteAddr:     "192.168.1.1:1234",
			expectedRealIP: "10.0.0.1",
			headersShouldBe: map[string]bool{
				"X-From-Cdn":      false, // Should be erased
				"X-Forwarded-For": true,  // Should be kept (standard header)
				"X-Real-Ip":       true,  // Should be set by the plugin
			},
		},
		{
			desc:           "Proxy 2 - should erase X-From-Cdn and Client-Ip",
			xFromProxy:     "2",
			realIPHeader:   "Client-Ip",
			realIP:         "10.0.1.1",
			remoteAddr:     "192.168.1.2:1234",
			expectedRealIP: "10.0.1.1",
			headersShouldBe: map[string]bool{
				"X-From-Cdn": false, // Should be erased
				"Client-Ip":  false, // Should be erased (non-standard header)
				"X-Real-Ip":  true,  // Should be set by the plugin
			},
		},
		{
			desc:           "Proxy 3 - should erase X-From-Cdn and Cf-Connecting-Ip",
			xFromProxy:     "3",
			realIPHeader:   "Cf-Connecting-Ip",
			realIP:         "10.0.2.1",
			remoteAddr:     "192.168.1.3:1234",
			expectedRealIP: "10.0.2.1",
			headersShouldBe: map[string]bool{
				"X-From-Cdn":       false, // Should be erased
				"Cf-Connecting-Ip": false, // Should be erased (non-standard header)
				"X-Real-Ip":        true,  // Should be set by the plugin
			},
		},
		{
			desc:           "Proxy 4 - wildcard proxy should keep RemoteAddr",
			xFromProxy:     "4",
			realIPHeader:   "RemoteAddr", // Not actually a header, but a special case
			realIP:         "",           // Not used for RemoteAddr
			remoteAddr:     "192.168.1.4:1234",
			expectedRealIP: "192.168.1.4",
			headersShouldBe: map[string]bool{
				"X-From-Cdn": true, // Should not be erased for wildcard
				"X-Real-Ip":  true, // Should be set by the plugin
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			recorder := httptest.NewRecorder()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
			if err != nil {
				t.Fatal(err)
			}

			req.RemoteAddr = test.remoteAddr
			req.Header.Set("X-From-Cdn", test.xFromProxy)

			// Set any IP headers specified in the test
			if test.realIPHeader != "RemoteAddr" && test.realIP != "" {
				req.Header.Set(test.realIPHeader, test.realIP)
			}

			// Always set X-Forwarded-For for convenience
			if test.realIPHeader != "X-Forwarded-For" {
				req.Header.Set("X-Forwarded-For", "default-xff-value")
			}

			// Process the request
			handler.ServeHTTP(recorder, req)

			// Verify X-Real-Ip is set correctly
			if req.Header.Get("X-Real-Ip") != test.expectedRealIP {
				t.Errorf("X-Real-Ip not set correctly: expected '%s', got '%s'",
					test.expectedRealIP, req.Header.Get("X-Real-Ip"))
			}

			// Verify headers that should be present or absent
			for header, shouldExist := range test.headersShouldBe {
				headerExists := req.Header.Get(header) != ""
				if headerExists != shouldExist {
					if shouldExist {
						t.Errorf("Header '%s' should exist but was erased", header)
					} else {
						t.Errorf("Header '%s' should be erased but still exists with value: %s",
							header, req.Header.Get(header))
					}
				}
			}
		})
	}
}
