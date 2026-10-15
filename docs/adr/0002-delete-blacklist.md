# ADR-0002: 删除黑名单（dead list）过滤

**状态**: accepted
**日期**: 2026-10-15

## 背景
原管线从 `notracking/hosts-blocklists` 抓 dead 域名列表（173551 条），在分类时做精确匹配过滤。

实测：
- 精确命中 487/151078 候选域名（0.32%）
- 后缀语义可多删 ~96 个，但 dead 列表停更于 2023-06-27（过期 2 年+）
- `clean_lines()` 死代码 + `line[2:-1]` off-by-two bug + 子串误杀陷阱（`ellads.com` 杀 `loads.com`）

## 决策
删除黑名单过滤。不抓 dead 列表，不做 dead 域名过滤。

## 理由
583 个潜在删除（0.39%）对 151078 域名的价值 < 维护负担（过期数据、误杀风险、额外 fetch、两个独立 bug 路径）。YAGNI：如果将来需要，加 `--dead` 参数用新列表即可。

## 后果
- 少 2 个 fetch 任务
- 消除 off-by-two bug 和子串误杀风险
- 输出可能多 ~583 个死域名（可接受，订阅者的 DNS 解析器会自然 NXDOMAIN）
