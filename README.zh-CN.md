# Vermory

**面向 AI 的可治理记忆与上下文连续性平台**

[![CI](https://github.com/jstar0/Vermory/actions/workflows/ci.yml/badge.svg)](https://github.com/jstar0/Vermory/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

[English](README.md)

Vermory 是一个面向 AI 客户端的、以真实场景和可验证证据驱动的记忆与上下文连续性平台。它服务于 AI Coding 工具、Web Chat、个人助手及其他需要持续处理真实事务的 AI 系统。

Vermory 不只是保存几段 memo，也不只是给 PostgreSQL 套一层向量检索。它要解决的是：

- 当前交互到底属于哪个持续空间；
- 哪些观察值得形成长期记忆；
- 多个来源冲突时应该相信谁；
- 哪些事实仍然有效、已经过期、仅在局部有效或已经删除；
- 当前模型和任务真正需要看到哪些上下文；
- 记忆的形成、修正、召回、桥接和删除怎样保持可解释、可审计。

## 三种连续性模式

| 模式 | 主要锚点 | 用户侧行为 |
|---|---|---|
| Workspace-backed continuity | 仓库根目录、工作区路径、manifest、显式绑定 | 同一工作区可跨 Codex、Claude Code、Grok 等客户端接续；不同工作区默认隔离。 |
| Conversation-backed continuity | thread、渠道、联系人、命名事务或话题 | 日常事务可以跨会话继续，但无关话题不能被擅自合并。 |
| Global Defaults | 用户显式确认的稳定偏好和长期设置 | 只保留薄而稳定的默认层，临时任务要求不能污染全局偏好。 |

`promote`、`link`、`export`、`adopt`、`rebind` 等跨模式操作属于显式治理动作，而不是底层自动混合。

## 核心约束

Vermory 当前的产品宪法要求：

- 标注为硬门的测试中，跨租户、跨 continuity 禁止事实泄漏必须为零；
- 强锚点识别不确定时必须拒绝猜测或请求确认；
- 过期和被替代的事实不能继续作为当前事实使用；
- 已删除目标不能通过精确、改写、语义、缓存、历史或可选后端再次泄漏；
- 当前仓库和事实源可以纠正旧记忆；
- 模型推断不能悄悄覆盖用户明确意图；
- PostgreSQL 保存权威状态，检索投影必须可销毁、可重建；
- mem0、MemOS、Supermemory 等只能作为可选投影适配器，不能成为第二套事实源。

详见 [产品宪法](docs/superpowers/specs/2026-07-11-vermory-product-constitution.md)。

## 当前进度

Experiment 0 已完成，当前仓库已经具备：

- 严格的 reality case manifest 与 JSONL 事件合同；
- 来源授权、匿名化、fixture 哈希和路径越界校验；
- 确定性的 `fixture-lock.json` 与冻结后变更检测；
- `public` 和 `withheld_local` 证据等级，并拒绝把本地可读目录伪装成 sealed；
- 外部 sealed evaluator 的 Ed25519 attestation 验签能力；
- 9 个覆盖 workspace、conversation、Global Defaults、删除、source injection、durable bridge、OpenClaw 日常事务连续性、authenticated multi-tenant RLS 与 PostgreSQL 运维恢复的公开冻结案例；
- JSON 和 Markdown 实验报告。

仓库同时已经包含 workspace、conversation、Global Defaults、durable bridge、显式可信来源修订、按稳定事实 key 治理的 source candidate、可信来源无 key 时的 provider 闭集目标匹配、OpenClaw external-turn lifecycle、authenticated multi-tenant HTTP profile、原生 PostgreSQL 恢复、可选 active-only pgvector runtime，以及可并行构建和测量切换的版本化语义投影。来源变化可以先形成候选而不改变 AI 当前上下文；拒绝候选不会改动当前事实，接受候选则原子替代仍然有效的同 key 目标。可信来源只有精确内容和 revision、没有内部 key 时，provider 只能从当前 scope 的闭合集合中选一个现有 key 或 abstain；Vermory 会验证并审计结果，仍然要求操作者明确接受。真实 Grok MCP 任务已经在投影重建后只消费被接受的新事实，并把结果回写为 proposed。语义检索 profile 共享 PostgreSQL 权威事实，但拥有独立 cursor、vector、audit、reset/rebuild 与 promotion decision；当前实测 v2 仍保留为 candidate，lexical 和 v1 默认均未被擅自切换。认证 profile 使用服务端发行且只保存 digest 的 token、角色路由、非 owner PostgreSQL runtime identity、tenant-aware foreign keys，以及覆盖当前 continuity graph 的 RLS。恢复证据覆盖迁移重放、原生 dump/restore、投影重建、runtime role 重建和有界数据库中断恢复。Pull Request CI 会在干净 Ubuntu runner 上启动 PostgreSQL 18，并自动执行数据库 Go 测试、关键 runtime race、release build 和 OpenClaw 安装/检查/打包链路。每份证据只对实际执行过的客户端、模型、故障条件和确定性硬门负责，任何单一切片都不被当成“整个平台已经完成”的证明。

完整状态见 [Experiment 0 读数](docs/experiment-0-readout.md)。

## 快速开始

要求：

- Go `1.25.7` 或兼容的新版本；
- 使用 `jq` 查看生成的 JSON 证据；
- 只有运行原生后端和旧垂直切片时才需要 PostgreSQL。

```bash
go test ./...
go test -race ./internal/reality
go vet ./...
```

```bash
go run ./cmd/vermory --help
```

验证冻结案例：

```bash
go run ./cmd/vermory reality-validate \
  --case-root reality/cases \
  --artifact-root ./artifacts \
  --run-id experiment-0-public-v1
```

生成 Experiment 0 报告：

```bash
go run ./cmd/vermory experiment-0 \
  --case-root reality/cases \
  --artifact-root ./artifacts \
  --run-id experiment-0-v1
```

## 本地工作区治理

普通 AI 客户端只通过 MCP 获取已确认工作区的有效上下文，并把任务结果写回为待确认观察。工作区确认、来源事实记录、指定事实纠正和指定事实遗忘由本机操作者显式执行，不作为模型工具开放。完整命令与边界见[本地工作区治理指南](docs/integrations/local-operator-workspace-slice.md)；[Codex MCP 真实客户端实证](docs/evidence/2026-07-14-codex-mcp-real-client.md)记录了 Codex 自行调用 `prepare_context`、生成并验证文件、调用 `commit_observation`，以及 PostgreSQL 将结果保持为 `proposed` 的完整链路。

可信来源发生变化时，`memory revise-source` 会替代一个被明确指定的当前事实；它不会用语义相似度猜测目标，也不会覆盖同工作区中的无关事实。[显式来源修订运行实证](docs/evidence/2026-07-14-source-revision-runtime.md)记录了软件发布命令更新、投影重建、旧事实精确与改写探针、真实 Grok MCP 消费回写，以及本轮 Codex 因模型或账户配额在 MCP 前失败的边界。

当可信 ingestor 能提供稳定事实 key 时，`memory propose-source` 会先生成可审查候选，不改变当前检索；操作者再使用 `memory accept-candidate` 或 `memory reject-candidate` 明确裁决，普通 MCP 客户端看不到 proposed/rejected 内容。[来源冲突候选运行实证](docs/evidence/2026-07-14-source-conflict-candidate-runtime.md)记录了拒绝、接受、跨租户阻断、投影重建、旧事实原文与改写探针，以及真实 Grok MCP 消费回写的完整链路。

当可信来源只有精确事实与 revision、没有 Vermory 内部 key 时，`memory match-source` 会让已配置 provider 只从当前 workspace 的闭集 key 中选择一个目标或 abstain。provider 不能创造 authority、跨 scope 或直接激活 memory；合法匹配只会形成原有的可审查 source candidate。[无 key 来源目标匹配运行实证](docs/evidence/2026-07-14-unkeyed-source-target-matching-runtime.md)记录了真实 Grok matched/abstained、proposal 隔离、显式接受、RLS 审计、投影重建、stale probes 与真实 MCP coder 回写。该能力是闭集匹配，不是任意文档抽取。

## 生产检索运行线

W09 把 active-only PostgreSQL/pgvector 检索接到了真实 workspace MCP、Web
Chat 和 authenticated API。固定租户的受限 worker 消费 durable projection
events；`shadow` 和 `vector` 必须显式启用，默认仍是 lexical。projection
lag 或 embedding provider 故障时，运行时按原 ID 与顺序退回 lexical，不能
影响 PostgreSQL authority。

真实回放覆盖了 Grok MCP 语义事实与技术标识消费、proposed 回写、链接会话
Web Chat、shadow 字节等价、cursor lag、HTTP 503、vector 清空重建、RLS、
原生 dump/restore 和恢复库重建。该结果只表示 `production_path_integrated`，
不表示已经切换默认检索，也不表示完成规模、embedding migration 或最终发布
验收。详见[生产检索运行实证](docs/evidence/2026-07-14-production-retrieval-runtime.md)。

## 发布产物

每个 Pull Request 都会生成保留 7 天的可下载 snapshot，包括带 SHA-256 校验的 `linux/amd64`、`linux/arm64`、`darwin/amd64`、`darwin/arm64` 归档，以及独立的 `@vermory/openclaw` 包。每个 Go 归档固定包含 `vermory`、`LICENSE`、`README.md` 和 `README.zh-CN.md`。

```bash
vermory version
```

发布二进制会输出注入的版本、完整 revision、构建时间和 Go runtime 版本。手动 Release workflow 只生成不发布的 snapshot；只有 `v*` tag 可以创建 draft GitHub Release。当前 Draft PR 不创建 tag，也不创建 GitHub Release。精确 checksum、两次构建可复现性、Actions 下载产物、本机执行和明确不承诺项见[发布打包实证](docs/evidence/2026-07-14-release-packaging.md)。

## OpenClaw 接入

`@vermory/openclaw` 使用 OpenClaw 的 canonical `sessionKey` 和 `runId`：在 `before_prompt_build` 注入当前有效的语义上下文，在 `agent_end` 记录最终 turn lifecycle。它不替代 OpenClaw 的 transcript、memory slot、渠道或模型路由。

```bash
PATH="/opt/homebrew/opt/node@24/bin:$PATH" \
  pnpm -C integrations/openclaw install --frozen-lockfile
PATH="/opt/homebrew/opt/node@24/bin:$PATH" \
  pnpm -C integrations/openclaw check
```

loopback 部署、OpenClaw trust 配置、runtime inspection、确认/纠正/删除、显式 link、故障语义、隔离状态重放和卸载步骤见 [OpenClaw 运行接入指南](docs/integrations/openclaw-runtime.md)。

authenticated 部署、token 生命周期、runtime role 授权、TLS 规则、RLS 验证、备份、恢复、投影重建与撤销边界见[身份授权与 PostgreSQL RLS 指南](docs/integrations/identity-authorization-rls.md)。[身份授权实证](docs/evidence/2026-07-14-identity-authorization-rls.md)包含确定性租户隔离硬门和真实 OpenClaw/Grok 认证回放；[PostgreSQL 运维恢复实证](docs/evidence/2026-07-14-postgresql-operations-recovery.md)记录原生 dump/restore、投影丢失与重建、数据库中断恢复。

## 开发原则

后续能力按照以下顺序推进：

```text
真实失败或长期轨迹
-> 冻结当前事实、禁止行为和验收标准
-> 建立 public、withheld 和 sealed 证据
-> 跑简单基线
-> 提出可证伪实现假设
-> 接入真实客户端执行
-> 验证删除、故障、迁移和规模
```

不要因为某个表、状态机、服务或中间件看起来“架构完整”就直接把它写死。永久设计必须先对应真实案例和可证伪假设。

## 许可证

项目采用 [Apache License 2.0](LICENSE)。
