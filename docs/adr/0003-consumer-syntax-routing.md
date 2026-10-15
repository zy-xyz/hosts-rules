# ADR-0003: 按消费端语法路由，非字符串模式匹配

**状态**: accepted
**日期**: 2026-10-15

## 背景
原管线用 `line.endswith('^')` 做路由判据。后果：
- `diy.txt` 164 条 AGH 规则中 157 条错放进 adblock.txt（uBO）
- 全部 54 条 `@@` 放行规则，AGH 订阅者一条都收不到
- `$important`/`$dnsrewrite` 等 DNS 级规则被丢进 uBO 专用文件

## 决策
分类判据改为**消费端能否消费这条规则**，基于 AGH 修饰符白名单（8 项）和消费者格式矩阵。

## 路由规则
1. hosts 行（`IP hostname`）：loopback/reserved → 丢弃；`127.0.0.1 domain` → easylist；外部 IP + domain → accelerate
2. `||domain^` / `@@||domain^`（无修饰符或修饰符 ⊆ AGH 白名单）→ easylist
3. `||domain^$<浏览器专有修饰符>`（含 path、`$third-party`、`$generichide` 等）→ adblock
4. `/regex/` → easylist（AGH 支持）
5. cosmetic 规则（`##`、`#$#`）→ adblock
6. 通配符/裸 IP/单标签域名 → adblock（uBO 能处理）
7. accelerate 输出前做差集排除：`acc_domains - easy_domains`

## 理由
`endswith('^')` 是字符串特征，不是语义判据。`||domain^$important` 和 `||domain/path$important` 一个进 AGH 一个进 uBO，由修饰符和路径决定，不由尾部字符决定。

## 后果
- diy.txt 157 条错放规则回归正确桶
- AGH 订阅者获得放行例外
- 需要维护 AGH 修饰符白名单（8 项，文档地址 adguard-dns.io/kb/general/dns-filtering-syntax）
