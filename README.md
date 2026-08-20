# WuxiangAIHub - 南宁五象新区南A人工智能产业服务平台

WuxiangAIHub 面向五象新区企业、跨境线路运营人员、服务机构、园区审核员和 OPC 社区运营者，管理人工智能项目从申报、算力配额、跨境数字线路、服务采购到场景验收的完整生命周期。平台对应“北上广研发 + 广西集成 + 东盟应用”的协作路径，并把企业服务、算力资源、远程教育、远程医疗、无人配送和 OPC 创业载体放在同一套可追溯流程中。

- 语言：Go 1.26
- 服务端口：49660
- 存储：分片 JSONL 事件与 SQLite 关系索引，支持迁移、事务、重建和重启恢复

## 业务边界

- **项目申报与分办**：企业提交跨境 AI 项目，按行业、东盟国家、风险等级和服务窗口分配牵头机构。
- **算力与线路**：申请智算集群配额，关联跨境通信线路和时延目标，审批通过后生成可审计的资源安排。
- **企业服务**：服务机构发布远程教育、远程医疗、跨境结算、数据加工等服务产品，企业可在有效期内提交申请并跟踪履约。
- **场景验收**：社区、学校、医院和物流车队提交 AI 场景验收材料，复核通过后才能上线展示或进入东博会展区。
- **OPC 社区**：创业者申请入驻南A东盟谷等特色社区，平台维护席位、导师服务、阶段里程碑和退出记录。

## 目录

```
cmd/wuxiangai/       HTTP 服务入口、优雅关闭和健康检查
cmd/hubctl/          运维命令（初始化、导入、导出、重建索引、诊断）
internal/domain/     申报、规则、分配、升级、审计和失败记录
internal/hub/        算力、线路、场景、企业服务与 OPC 领域校验
internal/service/    跨实体事务和状态流编排
internal/repo/       JSONL 事件与 SQLite 索引的复合持久化
internal/index/      迁移、约束、分页过滤和组合查询
internal/httpapi/    JSON API、鉴权、错误映射和请求 ID
internal/scheduler/  超期升级、重试退避和永久失败任务
internal/worker/     规则版本变更后的重新评估
```

## 启动

```bash
go build ./...
go run ./cmd/hubctl init -data-dir ./data
WUXIANG_AI_HUB_AUTH_BOOTSTRAP_USERS='[{"id":"u-admin","username":"admin","password":"<strong-password>","role":"admin"}]' \
  go run ./cmd/wuxiangai config.example.yaml
```

存活检查为 `GET /healthz`，就绪检查为 `GET /readyz`。登录后可调用 `/api/items`、`/api/rules`、`/api/batches`、`/api/audit` 和 `/api/failures` 等接口；所有请求都经过会话撤销、过期和角色鉴权。

## 质量门禁

```bash
go test ./... -count=1
go test -race ./... -count=1
go vet ./...
go build ./...
go run .agents/skills/go-base-project-create/scripts/measure_project.go -root . -enforce
```

Dockerfile 使用仓库中的 `cmd/wuxiangai` 和 `cmd/hubctl` 构建入口，构建时从 `go.mod` 选择 Go 版本，容器默认启动 HTTP 服务。
