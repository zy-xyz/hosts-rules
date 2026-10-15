# ADR-0001: 管线用 Go 重写

**状态**: accepted
**日期**: 2026-10-15

## 背景
hosts-new 原管线用 Python（`hosts.py` 224 行 / `hosts-full.py` 234 行）。审计发现 13 项缺陷（路由判据错、无去重、无门禁、ProcessPool 倒挂 34% 等）。

实测数据：
- 分类阶段 Python 0.6–2.5s，ProcessPool(4) 比单进程慢 34%（pickle + GIL）
- Go/Rust 重写的性能上限 = 省 1.33s/天（0.05% 墙钟）
- 抓取占 99.89% 墙钟，但用户确认网络/Actions 不是约束

## 决策
用 Go 重写管线，单 package 多文件。Python 仅做离线辅助工具（数据分析、Hugo content 生成）。

## 理由
性能不是换语言的理由（分类已够快）。真正收益在工程层：
1. **单二进制**：CI 不用 pip install，`gh release` 直接挂二进制
2. **goroutine**：并发抓取 32+ 无 GIL
3. **类型安全**：`enum Bucket` + `func classify(line string) Bucket` 让 `endswith('^')` 这类 bug 更难犯
4. Rust 不值得 2–3 倍开发时间

## 替代方案
- Python 重构：最快写，但丢掉单二进制 + 类型安全
- Rust：最快跑分，但学习曲线 + 开发时间不值
- Go+Python 双语言：Python 辅助分类，但分类逻辑本质是 30 行 regex，不值得引入 Python runtime 依赖

## 后果
- CI 需 `actions/setup-go@v5`，不需要 Python runtime
- 分类逻辑手写 regex（无现成 Go adblock 库），但逻辑简单
