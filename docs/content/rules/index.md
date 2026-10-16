---
title: "规则订阅"
---

三份产物互斥，按消费端语法拆分。每条规则只投递给真正能消费它的客户端。

## 订阅地址

| 文件 | 消费端 | 说明 |
|---|---|---|
| `easylist.txt` | **AdGuard Home** | DNS 级规则：`\|\|domain^`、`@@\|\|domain^`、`/regex/` + AGH 原生修饰符 |
| `adblock.txt` | **uBlock Origin** | 全语法：浏览器专有修饰符、cosmetic 规则、通配符域名 |
| `accelerate.txt` | /etc/hosts、加速 DNS | `IP hostname` 映射（GitHub / Steam 加速） |

`_full` 后缀为完整源集（81 源，base 为 40 源）。

### 源链（GitHub 直连）

```
https://github.com/zy-xyz/hosts-rules/releases/latest/download/easylist.txt
https://github.com/zy-xyz/hosts-rules/releases/latest/download/adblock.txt
https://github.com/zy-xyz/hosts-rules/releases/latest/download/accelerate.txt
```

### 加速链（`ghfast.top` 镜像，国内直连不稳时用）

```
https://ghfast.top/https://github.com/zy-xyz/hosts-rules/releases/latest/download/easylist.txt
https://ghfast.top/https://github.com/zy-xyz/hosts-rules/releases/latest/download/adblock.txt
https://ghfast.top/https://github.com/zy-xyz/hosts-rules/releases/latest/download/accelerate.txt
```

## 路由判据

分类依据是**消费端能否消费这条规则**，而非规则的字符串特征：

- `IP hostname`（sink IP）→ easylist；外部 IP → accelerate
- `||domain^` / `@@||domain^` + 修饰符 ⊆ AGH 白名单（8 项）→ easylist
- 含 path、通配符、或 AGH 不认的修饰符 → adblock
- `/regex/` → easylist

AGH 修饰符白名单：`client` `denyallow` `dnstype` `dnsrewrite` `important` `badfilter` `ctag` `respgeo`
