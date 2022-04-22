# Traefik Get Real IP address

<!-- cspell:words traefik middlewares proxyHeadername proxyHeadervalue Kubernetes -->

When traefik is deployed behind multiple load balancers, use this plugin to detect the different load balancers and get the real IP from different header fields

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
