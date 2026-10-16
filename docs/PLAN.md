# hosts-rules 重写实施计划

> **状态：已完成。** 本文件保留为历史实施记录（原计划针对 `hosts-new`，已归档；
> 现项目为 `hosts-rules`）。文中 `hosts-new` 字样为当时的命名。

## 决策摘要

| 决策 | 结论 | ADR |
|---|---|---|
| 语言 | Go（单二进制） | 0001 |
| 分类判据 | 消费端语法路由 | 0003 |
| 黑名单 | 删除 | 0002 |
| 输出桶 | 3 桶保留，消费端重新路由 | 0003 |
| accelerate 语义 | 纯加速，广告域名差集排除 | CONTEXT |
| 域名校验 | 严格（小写/拒通配/拒IP/拒单标签），AGH 吃不了的落 uBO | CONTEXT |
| base/full | `--full` flag，同一二进制 | — |
| diy.txt | 外部文件，运行时读取 | — |
| 旧脚本 | Go 验证通过后立即删除 | — |
| 前端 | Hugo + GitHub Pages | — |
| 统计 | stats.json(最新) + history.jsonl(趋势) | — |
| 分发 | Release(txt+二进制) + Pages(前端) | — |
| CI | 两 workflow：data.yml(cron) + pages.yml(push) | — |

## 目标文件结构

```
hosts-new/
├── go.mod                    # module github.com/zy-xyz/hosts-rules
├── main.go                   # 入口、--full flag、编排、门禁
├── fetch.go                  # goroutine 并发抓取
├── classify.go               # 消费端语法路由
├── write.go                  # 去重 + 尾换行
├── stats.go                  # stats.json + history.jsonl
├── links.txt                 # 41 base 源
├── links_full.txt            # 74 full 源
├── diy.txt                   # 164 手写规则
├── CONTEXT.md                # 域术语表
├── docs/
│   ├── adr/                  # 3 条 ADR
│   ├── data/
│   │   ├── stats.json        # 最新统计（管线生成）
│   │   └── history.jsonl    # 趋势数据（管线追加）
│   └── (Hugo 站点：config.toml, content/, layouts/, static/)
├── .github/workflows/
│   ├── data.yml              # cron: Go build+run+release+stats
│   └── pages.yml             # push docs/: Hugo build+deploy
└── README.md
```

## 阶段 0：清理 + 初始化（可验证：git status 干净）

- [ ] `git rm hosts.py hosts-full.py requirements.txt`
- [ ] `git rm --cached accelerate*.txt easylist*.txt adblock*.txt`（从 main 移除大文件，保留 .gitignore）
- [ ] 删除旧 workflow：`git rm .github/workflows/daily-update*.yml`
- [ ] 删除误提交图片：`git rm *.jpg *.png`
- [ ] `git rm cache/ __pycache__/`（如有）
- [ ] `go mod init github.com/zy-xyz/hosts-rules`
- [ ] 创建 `links.txt`（从旧 hosts.py links 数组提取 41 URL）和 `links_full.txt`（74 URL）
- [ ] 保留 `diy.txt`
- [ ] `.gitignore`：`cache/`、`*.txt`（输出产物不进 main，走 Release）

**验证**：`git status` 干净；`go build` 空项目成功；`ls links*.txt diy.txt` 存在。

## 阶段 1：fetch.go — 并发抓取（可验证：cache/ 落盘 42+ 文件）

```go
// fetch.go
type FetchResult struct {
    URL      string
    Data     []byte
    Duration time.Duration
    Err      error
}

func fetch(urls []string, workers int) (results []FetchResult, failed []string, duration time.Duration)
// - goroutine + buffered channel，workers=32
// - http.Client{Timeout: 30s}，UA: "Mozilla/5.0"
// - 每个 URL 写 cache/<hash>.txt
// - 返回失败列表
```

- [ ] 实现 `fetch()`
- [ ] 测试：`go run . --fetch-only` → cache/ 落盘 42+ 文件

**验证**：`ls cache/ | wc -l` ≥ 40；失败数 < 5。

## 阶段 2：classify.go — 消费端语法路由（可验证：三桶行数合理）

```go
// classify.go
type Bucket int
const (
    BucketAccelerate Bucket = iota
    BucketEasylist           // AGH
    BucketAdblock            // uBO
)

var aghModifiers = map[string]bool{
    "client": true, "denyallow": true, "dnstype": true,
    "dnsrewrite": true, "important": true, "badfilter": true,
    "ctag": true, "respgeo": true,
}

func classify(lines []string, diy []string) (accelerate, easylist, adblock []string)
// 流程：
// 1. 遍历 lines
// 2. routeLine(line) -> Bucket + normalized line
// 3. 追加到对应桶
// 4. diy 规则按同样逻辑路由
// 5. 差集排除：acc_domains -= easy_domains
// 6. 逐桶 dict.fromkeys 去重

func routeLine(line string) (Bucket, string)
// - strip + 跳过注释
// - 归一化 sink IP (0.0.0.0/::/::1/127.0.0.x -> 127.0.0.1)
// - hosts 行 -> 127.0.0.1 -> easylist ||domain^; 外部 IP -> accelerate
// - ||domain^ / @@||domain^ -> 检查修饰符 -> easylist 或 adblock
// - /regex/ -> easylist
// - cosmetic (##, #$#) -> adblock
// - 通配符/IP域名/单标签 -> adblock
```

- [ ] 实现 `classify()` + `routeLine()`
- [ ] 加载 diy.txt 作为额外输入行
- [ ] 差集排除逻辑
- [ ] 测试：`go run . --classify-only` → 三桶行数在合理范围

**验证**：
- `accelerate.txt` 无 `::1` / `127.0.0.2` 行
- `easylist.txt` 无 `$third-party` 行
- `adblock.txt` 无纯 `||domain^$important` 行（应在 easylist）
- `sort -u easylist.txt | wc -l` == `wc -l < easylist.txt`（去重生效）
- diy.txt 的 `@@||a0.app.xiaomi.com^` 在 easylist.txt
- accelerate 域名 ∩ easylist 域名 == 0（差集排除）

## 阶段 3：write.go — 去重 + 尾换行（可验证：wc -l == sort -u | wc -l）

```go
// write.go
func writeOutput(path string, lines []string) error
// - dict.fromkeys 去重（保持顺序）
// - f.write("\n".join(lines) + "\n")  // 尾换行
// - 空列表跳过
```

- [ ] 实现 `writeOutput()`
- [ ] main.go 编排：fetch → classify → write(3 files) → gate

**验证**：`tail -c 1 accelerate.txt | od -An -c` 输出 `\n`；去重生效。

## 阶段 4：stats.go + 门禁（可验证：stats.json 生成 + 失败时 exit 2）

```go
// stats.go
type Stats struct {
    Timestamp     string            `json:"timestamp"`
    Duration      float64           `json:"duration_s"`
    Full          bool              `json:"full"`
    TotalSources  int               `json:"total_sources"`
    FailedSources []string          `json:"failed_sources"`
    Buckets       map[string]Bucket `json:"buckets"`
    SourceContrib []SourceContrib   `json:"source_contrib"`
}

func generateStats(...) Stats
func writeStatsJSON(path string, s Stats) error
func appendHistory(path string, s Stats) error  // JSONL append
```

门禁（main.go）：
```go
if len(failed) > 0 {
    log.Fatalf("fetch failed: %d sources: %v", len(failed), failed)
}
```

- [ ] 实现 stats 生成 + history 追加
- [ ] 门禁：failed > 0 → exit 2

**验证**：人为让一个 URL 失败 → `echo $?` == 2；`stats.json` 存在且可 `jq .`；`history.jsonl` 有 1 行。

## 阶段 5：CI workflow — data.yml（可验证：Actions 跑通 + Release 创建）

```yaml
# .github/workflows/data.yml
name: data
on:
  schedule:
    - cron: '0 16 * * *'   # base
    - cron: '0 20 * * *'   # full (需在 step 里判断)
  workflow_dispatch:

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: go build -o hosts-new .
      - run: ./hosts-new                    # base
      - run: ./hosts-new --full             # full
        # 门禁：失败 exit 2 → job 红
      - uses: softprops/action-gh-release@v2
        with:
          tag_name: ${{ github.run_id }}
          files: |
            accelerate.txt easylist.txt adblock.txt
            accelerate_full.txt easylist_full.txt adblock_full.txt
            hosts-new
      - run: |
          git config user.name "bot"
          git config user.email "bot@zy"
          git add docs/data/stats.json docs/data/history.jsonl
          git commit -m "data: update stats" && git push
```

- [ ] 写 data.yml
- [ ] push → Actions 跑通 → Release 创建 → stats.json 提交

**验证**：Actions 页面绿色；Release 页面有 7 个文件；`docs/data/stats.json` 在 main。

## 阶段 6：Hugo 前端 + pages.yml（可验证：Pages URL 打开有内容）

```
docs/
├── config.toml          # Hugo 配置
├── content/
│   └── _index.md        # 首页（读 data/ 画图表）
├── layouts/
│   └── _default/
│       └── baseof.html  # HTML + JS + Chart.js
├── data/
│   ├── stats.json       # 管线生成
│   └── history.jsonl   # 管线追加
└── static/
    └── (favicon 等)
```

```yaml
# .github/workflows/pages.yml
name: pages
on:
  push:
    paths: ['docs/**']
  workflow_dispatch:

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: peaceiris/actions-hugo@v3
      - run: cd docs && hugo --minify
      - uses: actions/upload-pages-artifact@v3
        with: { path: docs/public }
  deploy:
    needs: build
    runs-on: ubuntu-latest
    permissions: { pages: write, id-token: write }
    steps:
      - uses: actions/deploy-pages@v4
```

- [ ] 初始化 Hugo：`cd docs && hugo new site .`
- [ ] 写 `content/_index.md` + `layouts/_default/baseof.html`（内嵌 JS 读 stats.json + history.jsonl，Chart.js CDN 画饼图/趋势图）
- [ ] 写 pages.yml
- [ ] Settings → Pages → Source: GitHub Actions

**验证**：push `docs/` → Pages 部署 → 打开 URL 看到统计页 + 图表。

## 阶段 7：旧文件清理 + README + git push（可验证：仓库干净）

- [ ] 删除旧 accelerate*.txt 等（已在 .gitignore，不再进 main）
- [ ] 写 README.md（项目说明 + 订阅链接 + 架构说明）
- [ ] `git add -A && git commit -m "feat: Go rewrite + Hugo frontend + consumer-syntax routing"`
- [ ] `git push origin main`

**验证**：`git log --oneline -1`；仓库 < 5MB；Pages 有内容；Release 有产物。
