# resolve

CoreDNS 插件：为 HTTPS (DoH) 服务器提供 `/resolve` JSON 解析接口，支持 EDNS0 Client Subnet。

## 编译安装

### 前置条件

- Go 1.21+
- CoreDNS 源码

```bash
# 克隆 CoreDNS 源码
git clone https://github.com/coredns/coredns.git /opt/coredns
cd /opt/coredns
```

### 1. 克隆插件

```bash
git clone https://github.com/qist/resolve.git plugin/resolve
```

### 2. 应用上游补丁

```bash
cd /opt/coredns
git apply plugin/resolve/server_https.patch
```

补丁修改以下文件：
- `core/dnsserver/server_https.go` — 新增 `/resolve` 处理、EDNS0 Client Subnet 注入
- `core/dnsserver/config.go` — 新增 `ResolveEDNS0` 配置字段

### 3. 注册插件到 plugin.cfg

```bash
# 在 https3 行后添加 edns0 和 resolve 指令
#sed -i '/^https3:https3$/a edns0:resolve\nresolve:resolve' plugin.cfg
补丁里面添加了 不需要 手动添加
```

验证：

```bash
grep -n "edns0\|resolve" plugin.cfg
# 应输出：
# 32:edns0:resolve
# 33:resolve:resolve
```

### 4. 生成代码并编译

```bash
# 生成 zplugin.go 和 zdirectives.go
go generate

# 编译
go build -o coredns .

# 验证二进制包含 resolve 插件
strings ./coredns | grep "resolve" | head -3
```

### 5. 安装

```bash
# 替换原有二进制
cp coredns /usr/local/bin/coredns
# 或
cp coredns /opt/coredns/coredns
```

## 配置

### Corefile

```bash
# 自动添加到 Corefile 的 https server block
sed -i '/tls .*/a\    edns0 on\n    resolve' /path/to/Corefile
```

完整配置：

```
https://.:443 {
    tls /path/to/fullchain.crt /path/to/private.key
    edns0 on
    resolve
    forward . 8.8.8.8 1.1.1.1 223.5.5.5 {
        max_concurrent 1000
    }
    log
    errors
}
```

### 指令说明

| 指令 | 说明 |
|------|------|
| `resolve` | 启用 `/resolve` JSON 接口 |
| `edns0 on` | 自动注入 EDNS0 Client Subnet（默认） |
| `edns0 off` | 关闭自动注入，仅 `cip` 参数生效 |

## 接口

### GET /resolve

```
GET /resolve?name=<domain>&type=<rrtype>&cip=<client_ip>
```

| 参数 | 必填 | 说明 | 示例 |
|------|------|------|------|
| `name` | 是 | 查询域名 | `example.com` |
| `type` | 否 | 记录类型，默认 `A` | `AAAA`、`MX`、`CNAME`、`NS`、`TXT`、`SRV` |
| `cip` | 否 | 客户端 IP，用于 EDNS0 Client Subnet | `8.8.8.8` |

### 响应格式

成功（200）：

```json
{
  "Name": "www.qq.com.",
  "Type": "A",
  "Status": 0,
  "TTL": 30,
  "Answer": [
    {"name": "www.qq.com.", "type": "CNAME", "ttl": 30, "data": "www.qq.com.eo.dnse2.com."},
    {"name": "www.qq.com.eo.dnse2.com.", "type": "A", "ttl": 30, "data": "43.159.109.55"}
  ]
}
```

错误（400/405/500）：

```json
{"Error": "missing required 'name' parameter"}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `Name` | string | 查询域名（FQDN） |
| `Type` | string | 查询记录类型 |
| `Status` | int | DNS 响应码（0=成功，3=NXDOMAIN） |
| `TTL` | uint32 | 最小 TTL，无记录时为 0 |
| `Answer` | array | 解析记录列表 |
| `Answer[].name` | string | 记录名称 |
| `Answer[].type` | string | 记录类型 |
| `Answer[].ttl` | uint32 | 记录 TTL |
| `Answer[].data` | string | 记录数据 |

## EDNS0 Client Subnet

### 行为

默认开启（不写 `edns0` 指令等同于 `edns0 on`）。

| edns0 | cip | /resolve | /dns-query |
|-------|-----|----------|-----------|
| on（默认） | 无 | 自动提取客户端 IP | 自动提取客户端 IP |
| on（默认） | 有 | 以 cip 为准 | 自动提取客户端 IP |
| off | 无 | 不注入 | 不注入 |
| off | 有 | 以 cip 为准 | 不注入 |

### 客户端 IP 优先级

1. `cip` 查询参数（仅 `/resolve`）
2. `X-Forwarded-For` 头（取第一个 IP）
3. `X-Real-IP` 头
4. 连接 `RemoteAddr`

### 子网掩码

| 协议 | 掩码 |
|------|------|
| IPv4 | /24 |
| IPv6 | /48 |

## 反向代理

Nginx 传递真实客户端 IP：

```nginx
location /resolve {
    proxy_pass https://coredns:443;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}
```

## 测试

```bash
# 基本查询
curl -sk "https://localhost:443/resolve?name=example.com"

# 指定记录类型
curl -sk "https://localhost:443/resolve?name=example.com&type=AAAA"

# 指定客户端 IP（EDNS0 Client Subnet）
curl -sk "https://localhost:443/resolve?name=example.com&cip=8.8.8.8"

# 格式化输出
curl -sk "https://localhost:443/resolve?name=example.com" | python3 -m json.tool
curl -sk "https://localhost:443/resolve?name=example.com" | jq .
```

## 卸载

```bash
# 1. 从 plugin.cfg 移除
sed -i '/^edns0:resolve$/d; /^resolve:resolve$/d' plugin.cfg

# 2. 从 Corefile 移除
sed -i '/edns0/d; /resolve/d' Corefile

# 3. 回退上游补丁
git apply -R plugin/resolve/server_https.patch

# 4. 删除插件目录
rm -rf plugin/resolve

# 5. 重新编译
go generate && go build
```

## 文件说明

```
plugin/resolve/
├── README.md              # 本文档
├── setup.go               # 注册 resolve + edns0 指令，包装 validator
├── resolve.go             # 透传 handler（实现 plugin.Handler 接口）
└── server_https.patch     # 上游补丁
```

### server_https.patch 改动

| 文件 | 行数 | 改动 |
|------|------|------|
| `core/dnsserver/config.go` | +3 | 新增 `ResolveEDNS0 bool` 字段 |
| `core/dnsserver/server_https.go` | +252 | serveResolve()、addECSFromHTTP()、clientIPFromRequest()、resolveEDNS0() |
| `plugin.cfg` | +2 | `edns0:resolve`、`resolve:resolve` |
