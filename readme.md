# 🛡️ hosts-new

轻量级、可自定义的广告 / 跟踪 / 恶意域名屏蔽规则构建器。

从 82 个上游源抓取规则 → 按**消费端语法**归类 → 输出三份互斥规则文件。
每条规则只投递给真正能消费它的客户端，不靠字符串特征猜测。

---

## 规则文件

产物不提交仓库，全部走 **GitHub Releases**：

| 文件 | 消费端 | 内容 |
|---|---|---|
| `accelerate.txt` | /etc/hosts、加速 DNS | `IP hostname` 映射（GitHub/Steam 加速） |
| `easylist.txt` | **AdGuard Home** | DNS 级规则：`\|\|domain^`、`@@\|\|domain^`、`/regex/` + AGH 原生修饰符 |
| `adblock.txt` | **uBlock Origin** | 全语法：浏览器专有修饰符、cosmetic 规则、通配符域名 |

`_full` 后缀为完整源集（82 源，base 为 41 源）。

订阅地址：`https://github.com/zy-xyz/hosts-rules/releases/latest/download/<文件名>`

可通过 `ghfast.top` 镜像加速：
`https://ghfast.top/https://github.com/zy-xyz/hosts-rules/releases/latest/download/easylist.txt`

---

## 状态面板

构建统计与趋势图：https://zy-xyz.github.io/hosts-new/

自动生成，无后端、无数据库。

---

## 架构

```
cmd/hosts-rules/       入口 + 编排 + 安全门禁
internal/pipeline/
  fetch.go             goroutine 并发抓取（32 并发），原子写缓存
  classify.go          消费端语法路由 —— 核心域逻辑
  write.go             去重 + 尾换行 + 域名抽取
  stats.go             stats.json + history.jsonl
  io.go                文件读取 + 历史回读（门禁用）
config/
  links.txt            41 个基础源
  links_full.txt       82 个完整源
  diy.txt              手写规则（AGH 语法）
docs/                  Hugo 状态面板 + ADR
```

零第三方 Go 依赖，单二进制。

### 路由判据

分类依据是**消费端能否消费这条规则**，而非规则的字符串特征：

- `IP hostname`（sink IP）→ easylist；外部 IP → accelerate
- `\|\|domain^` / `@@\|\|domain^` + 修饰符 ⊆ AGH 白名单（8 项）→ easylist
- 含 path、通配符、或 AGH 白名单外修饰符（`$third-party` 等）→ adblock
- `/regex/` → easylist（AGH 支持）
- cosmetic（`##`、`#$#`）→ adblock

AGH 修饰符白名单：`client` `denyallow` `dnstype` `dnsrewrite` `important` `badfilter` `ctag` `respgeo`。
规则含白名单外修饰符时 AGH **整条忽略**，因此必须投给 uBO。

### 约束

- **accelerate 语义**：纯加速映射。广告域名通过差集排除（accelerate 域名 ∖ 屏蔽域名），
  确保同一域名不会既被加速又被屏蔽。
- **域名校验**：小写归一；拒通配符 `*`/`?`；拒裸 IP；拒单标签。AGH 吃不了的 token 落 uBO。
- **安全门禁**：任一桶行数相对上次掉超 50% 时拒绝发布并退出非零码，防止网络抖动把残缺产物推上 main。

设计决策见 `docs/adr/`，域术语见 `CONTEXT.md`。

---

## 本地运行

```bash
go build -o hosts-new .
./hosts-new          # base，41 源
./hosts-new --full   # full，82 源
go test ./...        # 路由逻辑测试
```

产物：`accelerate.txt`、`easylist.txt`、`adblock.txt`（`--full` 加 `_full` 后缀），
以及 `docs/static/data/stats.json` + `history.jsonl`。

## 开发

```bash
hugo --source docs --minify    # 本地预览面板
```

CI：`.github/workflows/data.yml`（cron 每日构建 + Release）、
`.github/workflows/pages.yml`（push docs/ 时部署 Hugo 面板）。

## License

见 `LICENSE`。
