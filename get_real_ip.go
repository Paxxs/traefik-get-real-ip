package traefik_get_real_ip

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
)

const (
	xRealIP       = "X-Real-Ip"
	xForwardedFor = "X-Forwarded-For"
)

// Proxy 配置文件中的数组结构
type Proxy struct {
	ProxyHeadername  string `yaml:"proxyHeadername"`
	ProxyHeadervalue string `yaml:"proxyHeadervalue"`
	RealIP           string `yaml:"realIP"`
	OverwriteXFF     bool   `yaml:"overwriteXFF"` // override X-Forwarded-For
}

// Config the plugin configuration.
type Config struct {
	Proxy     []Proxy `yaml:"proxy"`
	EnableLog bool    `yaml:"enableLog"` // Enable logging output
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		EnableLog: false, // Logging disabled by default
	}
}

// GetRealIP Define plugin
type GetRealIP struct {
	next      http.Handler
	name      string
	proxy     []Proxy
	enableLog bool
}

// New creates and returns a new realip plugin instance.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	// Always log instance creation regardless of log settings
	fmt.Printf("[get-realip] Instance created with %d proxy configurations\n", len(config.Proxy))

	return &GetRealIP{
		next:      next,
		name:      name,
		proxy:     config.Proxy,
		enableLog: config.EnableLog,
	}, nil
}

// 真正干事情了
func (g *GetRealIP) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	var realIPStr string
	for _, proxy := range g.proxy {
		if proxy.ProxyHeadername == "*" || req.Header.Get(proxy.ProxyHeadername) == proxy.ProxyHeadervalue {
			g.log("Processing proxy configuration: %s (%s)", proxy.ProxyHeadervalue, proxy.ProxyHeadername)

			// CDN来源确定
			nIP := req.Header.Get(proxy.RealIP)
			if proxy.RealIP == "RemoteAddr" {
				nIP, _, _ = net.SplitHostPort(req.RemoteAddr)
			}
			forwardedIPs := strings.Split(nIP, ",") // 从头部获取到IP并分割（主要担心xff有多个IP）

			g.log("Processing IP addresses: %v (%d found)", forwardedIPs, len(forwardedIPs))
			// 如果有多个，得到第一个 IP
			for i := 0; i <= len(forwardedIPs)-1; i++ {
				trimmedIP := strings.TrimSpace(forwardedIPs[i])
				finalIP := g.getIP(trimmedIP)
				g.log("Validating IP: %s (index: %d, parsed: %s)", trimmedIP, i, finalIP)
				if finalIP != nil {
					realIPStr = finalIP.String()
					break
				}
			}
		}
		// 获取到后直接设定 realIP
		if realIPStr != "" {
			if proxy.OverwriteXFF {
				g.log("Overwriting X-Forwarded-For header with: %s", realIPStr)
				req.Header.Set(xForwardedFor, realIPStr)
			}
			req.Header.Set(xRealIP, realIPStr)
			break
		}
	}
	g.next.ServeHTTP(rw, req)
}

// getIP is used to obtain valid IP addresses. The parameter s is the input IP text,
// which should be in the format of x.x.x.x or x.x.x.x:1234.
func (g *GetRealIP) getIP(s string) net.IP {
	pureIP, _, err := net.SplitHostPort(s) // 如果有端口号则分离得到ip
	if err != nil {
		pureIP = s
	}
	ip := net.ParseIP(pureIP) // 解析是否为合法 ip
	return ip
}

// log is a method of GetRealIP that outputs logs only if logging is enabled.
// Usage is similar to fmt.Sprintf, but it automatically includes a prefix and newline.
func (g *GetRealIP) log(format string, a ...interface{}) {
	if g.enableLog {
		os.Stdout.WriteString("[get-realip] " + fmt.Sprintf(format, a...) + "\n")
	}
}
