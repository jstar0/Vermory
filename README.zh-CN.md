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

仓库同时已经包含 workspace、conversation、Global Defaults、durable bridge、显式可信来源修订、OpenClaw external-turn lifecycle、authenticated multi-tenant HTTP profile 和原生 PostgreSQL 恢复的生产形态运行切片。来源修订切片可以用新来源替代一个被明确指定的当前事实，同时保留无关事实和历史，并保证旧事实在投影重建后仍不会进入当前上下文；该链路已由真实 Grok MCP 任务消费和回写。认证 profile 使用服务端发行且只保存 digest 的 token、角色路由、非 owner PostgreSQL runtime identity、tenant-aware foreign keys，以及覆盖当前 continuity graph 的 RLS。恢复证据覆盖迁移重放、原生 dump/restore、投影重建、runtime role 重建和有界数据库中断恢复。每份证据只对实际执行过的客户端、模型、故障条件和确定性硬门负责，任何单一切片都不被当成“整个平台已经完成”的证明。

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
