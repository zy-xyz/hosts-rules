# hosts-new 域术语表

本文件只定义域概念，不含实现细节。

## 核心概念

### 三桶 (Three Buckets)
管线将上游规则按**消费端**分为三个互斥输出文件：
- **accelerate.txt** — hosts 格式的加速/CDN 映射（`IP hostname`），供 /etc/hosts 或加速 DNS 使用
- **easylist.txt** — AdGuard Home (AGH) DNS 级规则（`||domain^`、`@@||domain^`、`/regex/` + AGH 原生修饰符）
- **adblock.txt** — uBlock Origin (uBO) 全语法规则（含浏览器专有修饰符、cosmetic 规则）

### 消费端语法路由 (Consumer-Syntax Routing)
分类判据是**消费端能否消费这条规则**，而非规则的字符串特征。例：`||domain^$important` 属于 easylist（AGH 支持 `$important`），`||domain^$third-party` 属于 adblock（AGH 不支持 `$third-party`，整条丢弃）。

### accelerate 语义
accelerate = 纯加速/CDN 映射。广告域名通过**差集排除**（`accelerate 域名集合 - easylist 域名集合`）确保不与屏蔽规则自相矛盾。

### cosmetic 例外
`@@||domain^$generichide` 等带浏览器专有修饰符的例外规则**留在 adblock.txt**（uBO），不进 easylist。AGH 不做 cosmetic filtering，不需要它们。

### AGH 修饰符白名单
AdGuard Home DNS 语法仅支持 8 种修饰符：`client`、`denyallow`、`dnstype`、`dnsrewrite`、`important`、`badfilter`、`ctag`、`respgeo`。规则含白名单外修饰符时**整条忽略**。

### 域名校验
域名 token 必须通过校验才能进输出：小写归一；拒通配符 `*`/`?`；拒裸 IP；拒单标签（无点）。AGH 吃不了的 token 落到 adblock.txt（uBO 能处理）。punycode `xn--` 保留原样。

### stats.json / history.jsonl
管线每次运行生成 `stats.json`（最新统计）并追加一条记录到 `history.jsonl`（趋势数据）。两者均为小文件（~10KB/次），提交到 `docs/data/`，供 Hugo 前端读取。
