# Traefik 获取真实 IP 地址

<!-- cspell:words traefik middlewares proxyHeadername proxyHeadervalue Kubernetes -->

当 Traefik 部署在多个负载均衡器后面时，可以使用此插件检测不同的负载均衡器并从不同的头部字段提取真实 IP，然后将该值输出到 `x-real-ip` 头部。

此插件可以通过检查负载均衡器接收到的头部信息值是否匹配来防止 IP 欺骗，然后再提取 IP 地址。

例如，在下面展示的 `CloudFlare` 负载均衡器配置中，我们将其配置为只接受头部 `x-from-cdn` 且值等于 `cf-foo`，并从 `Cf-Connecting-Ip` 头部提取 IP 地址。由于用户永远不知道 `x-from-cdn` 头部的存在或其所需的值 `cf-foo`，因此它保持安全 🛡️。为了增加复杂性并避免被猜测，您可以使用随机字符串 :)

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

## CDN 配置

例如 Cloudflare:

规则 > 转换规则 > HTTP 请求头部修改 > 添加
- 设置静态头部: `X-From-Cdn`
  - 值: `cf-foo`

![image](https://user-images.githubusercontent.com/10364775/164590908-43edab8a-cdc8-4d4c-abd6-542b6c798f3b.png)

![image](https://user-images.githubusercontent.com/10364775/164591134-4dd2fc97-cd0e-4deb-8fe3-bcd4555ebbde.png)

## Traefik 配置
### 静态配置

插件信息:
- 模块名称: `github.com/Paxxs/traefik-get-real-ip`
- 版本: `v1.0.2`

Traefik 配置:
- yml
- toml
- docker-labels

```yml
pilot:
  token: [已编辑]

experimental:
  plugins:
    real-ip:
      moduleName: github.com/Paxxs/traefik-get-real-ip
      version: [请填写最新版本!]
```

### 动态配置

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
          enableLog: false # 默认: false，启用以查看详细日志
          Proxy:
            - proxyHeadername: X-From-Cdn
              proxyHeadervalue: mf-fun
              realIP: X-Forwarded-For
            - proxyHeadername: X-From-Cdn
              proxyHeadervalue: mf-bar
              realIP: Client-Ip
              OverwriteXFF: true # 默认: false，v1.0.2 或更高版本
            - proxyHeadername: X-From-Cdn
              proxyHeadervalue: cf-foo
              realIP: Cf-Connecting-Ip
              OverwriteXFF: true # 默认: false，v1.0.2 或更高版本
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

## 配置选项

| 选项            | 类型   | 必需   | 默认值  | 说明                                                     |
|----------------|--------|--------|---------|----------------------------------------------------------|
| enableLog      | bool   | 否     | false   | 启用详细日志记录，用于调试目的                              |
| Proxy          | array  | 是     | -       | 代理配置数组                                               |

### Proxy 配置

| 选项             | 类型   | 必需   | 默认值  | 说明                                                     |
|-----------------|--------|--------|---------|----------------------------------------------------------|
| proxyHeadername | string | 是     | -       | 用于CDN识别的头部名称。使用"*"可匹配所有来源                |
| proxyHeadervalue| string | 否     | -       | 头部的预期值。当proxyHeadername为"*"时不需要                |
| realIP          | string | 是     | -       | 用于提取真实IP的头部。特殊值"RemoteAddr"将使用连接的远程地址  |
| OverwriteXFF    | bool   | 否     | false   | 设为true时，还会用提取的真实IP覆盖X-Forwarded-For头部(v1.0.2+)|

### 处理逻辑

1. 插件按顺序检查每个代理配置。
2. 对于每个配置，它检查请求是否具有指定的`proxyHeadername`和值`proxyHeadervalue`（如果`proxyHeadername`为"*"，则接受任何请求）。
3. 找到匹配项后，从`realIP`指定的头部提取IP。
4. 提取的IP被设置为`X-Real-Ip`头部。
5. 如果`OverwriteXFF`为true，`X-Forwarded-For`头部也会被提取的IP覆盖。
6. 然后将请求传递给下一个处理程序。