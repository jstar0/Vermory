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
- 首批 4 个 workspace、conversation、Global Defaults、删除与 source injection 案例；
- JSON 和 Markdown 实验报告。

当前的 `pass=true` 只代表首批证据有效且已经冻结，不代表完整生产记忆内核已经通过这些案例。Experiment 1 将开始让生产形态的记忆切片和真实 AI 客户端消费这些轨迹。

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
