# Traefik Get Real IP address

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