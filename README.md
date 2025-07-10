# Traefik Get Real IP address

[中文文档](README.cn.md)

<!-- cspell:words traefik middlewares proxyHeadername proxyHeadervalue Kubernetes -->

When traefik is deployed behind multiple load balancers, this plugin can be used to detect different load balancers and extract the real IP from different header fields, then output the value to the `x-real-ip` header.

This plugin can prevent IP spoofing by checking if the values form the received header information of the load balancer match before extracting the IP address.

For example, in the configuration of `CloudFlare` load balancer shown below, we configure it to only accept the header `x-from-cdn` with a value equal to `cf-foo`, and extract the IP address from the `Cf-Connecting-Ip` header. Since users never know about the existence of the `x-from-cdn` header or its required value `cf-foo`, it remains secure 🛡️. To increase complexity and avoid being guessed, you can use a random string :)

```
 CloudFlare
┌─────────┐
│         ├────────────────────────────────►  ┌───────┬────────┐
└─────────┘ x-from-cdn:cf-foo                 │       │        │
            Cf-Connecting-Ip: realip          │       │        │
 CDN2                                         │       │        │
┌─────────┐                                   │       │ paxxs's│
│         ├────────────────────────────────►  │traefik│        │ x-real-ip:realip
└─────────┘ x-from-cdn:mf-bar                 │       │Get-rea ├─────────────►
            Client-iP: realip                 │       │ l-ip   │
 CDN3                                         │       │Plugin  │
┌─────────┐                                   │       │        │
│         ├───────────────────────────────►   │       │        │
└─────────┘ x-from-cdn:mf-fun                 └───────┴────────┘
            x-forwarded-for: realip,x.x.x.x
                           (truthedIP)          ▲  ▲
                                                │  │
 ┌────────┐                                     │  │
 └────────┘ ────────────────────────────────────┘  │
   "*"                                             │
 ┌────────┐                RemoteAddr/etc..        │
 └────────┘ ───────────────────────────────────────┘
```

## CDN Configuration

E.g. Cloudflare:

Rules > Transform Rules > HTTP Request Header Modification > Add
- Set static Header: `X-From-Cdn`
  - Value: `cf-foo`

![image](https://user-images.githubusercontent.com/10364775/164590908-43edab8a-cdc8-4d4c-abd6-542b6c798f3b.png)

![image](https://user-images.githubusercontent.com/10364775/164591134-4dd2fc97-cd0e-4deb-8fe3-bcd4555ebbde.png)

## Traefik Configuration
### Static

Plugin Info:
- moduleName: `github.com/Paxxs/traefik-get-real-ip`
- version: `v1.0.2`

Traefik Configuration:
- yml
- toml
- docker-labels

```yml
pilot:
  token: [REDACTED]

experimental:
  plugins:
    real-ip:
      moduleName: github.com/Paxxs/traefik-get-real-ip
      version: [Please fill the latest version !]
```

### Dynamic

- yml
- toml
- docker labels
- Kubernetes

```yml
http:
  middlewares:
    real-ip-foo:
      plugin:
        real-ip:
          enableLog: false # default: false, enable to see detailed logs
          deny403OnFail: true # default: false, when true returns 403 if no matching CDN header found
          eraseProxyHeaders: true # default: false, erase CDN-specific headers after processing
          Proxy:
            - proxyHeadername: X-From-Cdn
              proxyHeadervalue: mf-fun
              realIP: X-Forwarded-For
            - proxyHeadername: X-From-Cdn
              proxyHeadervalue: mf-bar
              realIP: Client-Ip
              OverwriteXFF: true # default: false, v1.0.2 or above
            - proxyHeadername: X-From-Cdn
              proxyHeadervalue: cf-foo
              realIP: Cf-Connecting-Ip
              OverwriteXFF: true # default: false, v1.0.2 or above
            - proxyHeadername: "*"
              realIP: RemoteAddr

  routers:
    my-router:
      rule: Host(`localhost`)
      middlewares:
        - real-ip-foo
      service: my-service

  services:
    my-service:
      loadBalancer:
        servers:
          - url: 'http://127.0.0.1'
```

## Configuration Options

| Option            | Type   | Required | Default | Description                                                 |
|-------------------|--------|----------|---------|-------------------------------------------------------------|
| enableLog         | bool   | No       | false   | Enable detailed logging for debugging purposes              |
| deny403OnFail     | bool   | No       | false   | When true, returns a 403 Forbidden response if no matching CDN header is found |
| eraseProxyHeaders | bool   | No       | false   | When true, erases CDN-specific headers after processing to prevent leaking CDN identification |
| Proxy             | array  | Yes      | -       | Array of proxy configurations                               |

### Proxy Configuration

| Option           | Type   | Required | Default | Description                                                 |
|------------------|--------|----------|---------|-------------------------------------------------------------|
| proxyHeadername  | string | Yes      | -       | The header name to check for CDN identification. Use "*" to match all sources |
| proxyHeadervalue | string | No       | -       | The expected value for the header. Not required when proxyHeadername is "*" |
| realIP           | string | Yes      | -       | The header to extract the real IP from. Special value "RemoteAddr" will use the connection's remote address |
| OverwriteXFF     | bool   | No       | false   | When set to true, also overwrites the X-Forwarded-For header with the extracted real IP (v1.0.2+) |

### Processing Logic

1. The plugin checks each proxy configuration in order.
2. For each configuration, it checks if the request has the specified `proxyHeadername` with value `proxyHeadervalue` (or accepts any if `proxyHeadername` is "*").
3. When a match is found, it extracts the IP from the header specified in `realIP`.
4. The extracted IP is set as the `X-Real-Ip` header.
5. If `OverwriteXFF` is true, the `X-Forwarded-For` header is also overwritten with the extracted IP.
6. If `eraseProxyHeaders` is true, the plugin removes the CDN-specific headers (the matched `proxyHeadername` and `realIP` headers) to prevent leaking CDN identification to downstream services. Standard headers like X-Forwarded-For are preserved.
7. If no matching proxy configuration is found and `deny403OnFail` is set to true, the plugin returns a 403 Forbidden response, preventing further request processing.
8. Otherwise, the request is then passed to the next handler.
