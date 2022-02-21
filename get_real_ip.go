package traefik_get_real_ip

import (
	"context"
	"log"
	"net"
	"net/http"
	"strings"
)

const (
	xRealIP       = "X-Real-Ip"
	xForwardedFor = "X-Forwarded-For"
)

type Proxy struct {
	ProxyHeadername  string `yaml:"proxyHeadername"`
	ProxyHeadervalue string `yaml:"proxyHeadervalue"`
	RealIP           string `yaml:"realIP"`
}

// Config the plugin configuration.
type Config struct {
	Proxy []Proxy `yaml:"proxy"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

// Define plugin
type GetRealIP struct {
	next  http.Handler
	name  string
	proxy []Proxy
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	log.Printf("☃️ All Config：'%v',Proxy Settings len: '%d'", config, len(config.Proxy))

	return &GetRealIP{
		next:  next,
		name:  name,
		proxy: config.Proxy,
	}, nil
}

// 真正干事情了
func (g *GetRealIP) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// fmt.Println("☃️当前配置：", g.proxy, "remoteaddr", req.RemoteAddr)
	var realIP string
	for _, proxy := range g.proxy {
		log.Printf("🐸 Current Proxy：%s", proxy.ProxyHeadervalue)
		if req.Header.Get(proxy.ProxyHeadername) == "*" || (req.Header.Get(proxy.ProxyHeadername) == proxy.ProxyHeadervalue) {
			// CDN来源确定
			nIP := req.Header.Get(proxy.RealIP)
			if proxy.RealIP == "RemoteAddr" {
				nIP = req.RemoteAddr
			}
			forwardedIPs := strings.Split(nIP, ",")
			// 从头部获取到IP并分割（主要担心xff有多个IP）
			// 只有单个IP也只会返回单个IP slice
			log.Printf("👀 IPs: '%d' detail:'%v'", len(forwardedIPs), forwardedIPs)
			// 如果有多个，得到第一个 IP
			for i := 0; i <= len(forwardedIPs)-1; i++ {
				trimmedIP := strings.TrimSpace(forwardedIPs[i])
				excluded := g.excludedIP(trimmedIP)
				log.Printf("exluded:%t， currentIP:%s, index:%d", excluded, trimmedIP, i)
				if !excluded {
					realIP = trimmedIP
					break
				}
			}
		}
		// 获取到后直接设定 realIP
		if realIP != "" {
			// req.Header.Set(xForwardedFor, realIP)
			req.Header.Set(xRealIP, realIP)
			break
		}
	}
	g.next.ServeHTTP(rw, req)
}

// 排除非IP
func (g *GetRealIP) excludedIP(s string) bool {
	ip := net.ParseIP(s)
	return ip == nil
}
