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
	Proxy []Proxy `yaml:"proxy"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

// GetRealIP Define plugin
type GetRealIP struct {
	next  http.Handler
	name  string
	proxy []Proxy
}

// New creates and returns a new realip plugin instance.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	log("☃️  Config loaded.(%d) %v", len(config.Proxy), config)

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
		if req.Header.Get(proxy.ProxyHeadername) == "*" || (req.Header.Get(proxy.ProxyHeadername) == proxy.ProxyHeadervalue) {
			log("🐸  Current Proxy：%s", proxy.ProxyHeadervalue)

			// CDN来源确定
			nIP := req.Header.Get(proxy.RealIP)
			if proxy.RealIP == "RemoteAddr" {
				nIP, _, _ = net.SplitHostPort(req.RemoteAddr)
			}
			forwardedIPs := strings.Split(nIP, ",")
			// 从头部获取到IP并分割（主要担心xff有多个IP）
			// 只有单个IP也只会返回单个IP slice
			log("👀  IPs:'%v'-%d", forwardedIPs, len(forwardedIPs))
			// 如果有多个，得到第一个 IP
			for i := 0; i <= len(forwardedIPs)-1; i++ {
				trimmedIP := strings.TrimSpace(forwardedIPs[i])
				excluded := g.excludedIP(trimmedIP)
				log("exluded:%t， currentIP:%s, index:%d", excluded, trimmedIP, i)
				if !excluded {
					realIP = trimmedIP
					break
				}
			}
		}
		// 获取到后直接设定 realIP
		if realIP != "" {
			if proxy.OverwriteXFF {
				log("🐸  Modify XFF to:%s", realIP)
				req.Header.Set(xForwardedFor, realIP)
			}
			req.Header.Set(xRealIP, realIP)
			break
		}
	}
	g.next.ServeHTTP(rw, req)
}

// excludedIP 判断给定的字符串是否是一个被排除的 IP 地址。
// 参数 s 是待检查的 IP 地址字符串。
// 返回值是一个布尔值，若给定的字符串不是一个合法的 IP 地址，则返回 true；否则返回 false。
func (g *GetRealIP) excludedIP(s string) bool {
	ip := net.ParseIP(s)
	return ip == nil
}

// log 是用于输出日志，使用方法类似 Sprintf，但末尾已经包含换行
//
// log is used for logging output, with a usage similar to Sprintf,
// but it already includes a newline character at the end.
func log(format string, a ...interface{}) {
	os.Stdout.WriteString("[get-realip] " + fmt.Sprintf(format, a...) + "\n")
}

// err是用于输出错误日志，使用方法类似 Sprintf，但末尾已经包含换行
//
// err is used for output err logs, and it usage is simillar to Sprintf,
// but with a newline character already included at the end.
// func err(format string, a ...interface{}) {
// 	os.Stderr.WriteString(fmt.Sprintf(format, a...) + "\n")
// }
