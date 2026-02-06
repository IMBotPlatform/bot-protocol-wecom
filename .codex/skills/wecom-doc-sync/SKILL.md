---
name: wecom-doc-sync
description: 同步企业微信智能机器人文档。用于从 docs/appendix/wecom-official/wecom_ai_bot 的 Markdown YAML frontmatter 读取 doc_id/source_url/doc_name，调用 docFetch/fetchCnt 获取 data.content_md 并覆盖正文内容。
---

# WeCom AI Bot 文档同步

## Frontmatter 约定
在每个 Markdown 文件顶部提供以下字段（必须）：

```yaml
---
doc_name: 接收消息
source_url: https://developer.work.weixin.qq.com/document/path/100719
doc_id: 57141
---
```

- 必须显式提供 `doc_id`（本 skill 不做自动推断）。

## 快速使用

```bash
python3 .codex/skills/wecom-doc-sync/scripts/update_wecom_docs.py
```

```bash
# 同步完成后输出改动范围（git diff --numstat）
python3 .codex/skills/wecom-doc-sync/scripts/update_wecom_docs.py --show-diff
```

```bash
# 关闭重要变化输出
python3 .codex/skills/wecom-doc-sync/scripts/update_wecom_docs.py --no-show-changes
```

## 常用参数

- `--dir <path>`：目标目录，默认 `docs/appendix/wecom-official/wecom_ai_bot`（固定默认，推荐不改）
- `--file <path>`：只更新单个文件
- `--dry-run`：仅拉取与日志，不写回
- `--show-changes`：输出重要变化（diff -u -w -B，忽略空白/换行，默认开启）
- `--no-show-changes`：关闭重要变化输出
- `--show-diff`：同步完成后输出改动范围（git diff --numstat）
- `--timeout <sec>`：curl 超时秒数
- `--cookie <cookie>`：可选 cookie；或设置环境变量 `WEWORK_COOKIE`

## 行为约定

- 只读取 YAML frontmatter 的 `doc_id/source_url/doc_name`
- 使用 curl 请求 `docFetch/fetchCnt`，读取 `data.content_md`
- 同步顺序：拉取 `content_md` → 输出重要变化（忽略空白/换行） → 按规则格式化 → 写回
- 保留 frontmatter，正文被 `content_md`（格式化后）完全替换

## Markdown 格式修复规则

- 标题必须有空格：`##标题` → `## 标题`
- 去除标题多余井号：`# # 标题` / `## # 标题` → `# 标题` / `## 标题`
- 标题与正文之间保留空行
- 标题后面需要空一行
- 列表与段落之间保留空行
- 去除行尾多余空格
- 代码块前后保留空行
- frontmatter 与正文之间保留空行
- 表格开始前需要空一行
- 表格与正文之间保留空行

## 错误处理

- 缺少 frontmatter 或关键字段：跳过并记录
- curl/JSON 失败或 `content_md` 为空：跳过并记录

## 最小依赖

- `python3`
- `curl`
