# Fresh Deploy SQL

乘风版（stack-slim）**仅支持全新部署**：数据库结构由单一文件提供，不再维护增量 `*.up.sql`。

## 文件

- `backend/store/pg/migration/full_fresh_deploy.sql` — 含业务表、`vector` 扩展、`rag_chunks`、`rag_jobs` 等

## 步骤

1. 创建空 PostgreSQL 库（compose 使用 `pgvector/pgvector:pg16`）。
2. 导入合并 SQL：

```bash
psql -U panda-wiki -d panda-wiki -f backend/store/pg/migration/full_fresh_deploy.sql
```

或使用 `docs/deploy/quickstart.sh`（检测到空库时自动导入）。

3. 启动其余服务。

## 说明

- API 启动时若 `migration/` 下无 `*.up.sql`，**跳过** golang-migrate 自动迁移（见 `store/pg/pg.go`）。
- **不支持**从旧版 Raglite 栈原地升级；请新库 + 全量重新学习，或保留旧 compose 与数据卷。
- `schema_migrations` 在 fresh SQL 末尾记为 version `43`。
