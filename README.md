<p align="center">
  <img src="/images/banner.png" width="400" />
</p>

<p align="center">
  <a target="_blank" href="https://github.com/MaydayV/PandaWiki">乘风版仓库</a> &nbsp;|&nbsp;
  <a target="_blank" href="https://github.com/chaitin/PandaWiki">原版仓库（致谢）</a>
</p>

## 项目介绍

PandaWiki 乘风版（Fly Version）是基于 [PandaWiki 开源项目](https://github.com/chaitin/PandaWiki) 二次开发的 AI 知识库系统，用于搭建产品文档、技术文档、FAQ 等内容站点，并提供 AI 创作、问答与搜索能力。

当前主线分支 **`feat/stack-slim`** 已完成部署栈精简：默认 **6 容器**、向量检索使用 **Postgres pgvector**、附件存储默认 **阿里云 OSS**（S3 兼容 API）。

## 版本

| 项 | 说明 |
| --- | --- |
| 版本号 | `FV2.6.16.2111` |
| 默认分支 | `feat/stack-slim` |
| 运行时 | Caddy + Postgres + Redis + NATS + API + App |
| RAG | `RAG_PROVIDER=pg`（pgvector） |
| 对象存储 | OSS / S3 兼容端点 |

详细变更见 [`docs/VERSION_NOTES.md`](docs/VERSION_NOTES.md)。

## 功能

- AI 辅助创作、问答、搜索
- 富文本 / Markdown 编辑，支持多种导出格式
- 网页挂件、钉钉 / 飞书 / 企业微信等机器人集成
- URL、Sitemap、RSS、离线文件等多种内容导入

## 部署

### Docker Compose（推荐）

适用于 Linux 服务器生产或测试环境。

```bash
git clone https://github.com/MaydayV/PandaWiki.git
cd PandaWiki
git checkout feat/stack-slim
cd docs/deploy
./generate-env.sh
./quickstart.sh
```

`.env` 中至少配置：`POSTGRES_PASSWORD`、`REDIS_PASSWORD`、`S3_*`、`NATS_PASSWORD`、`JWT_SECRET`、`ADMIN_PASSWORD`。

启动后：

- **管理后台**：HTTPS 2443（或由 Caddy 映射的管理端口）
- **Wiki 前台**：按知识库访问设置中的域名 / 端口访问
- 默认管理员密码见 `.env` 中的 `ADMIN_PASSWORD`

完整说明：[docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)

### Debian 原生（无 Docker）

适用于已有 Postgres / Nginx（含宝塔）的环境：

```bash
cd docs/deploy
cp .env.example .env   # 填写变量
bash native/deploy.sh
```

原生部署需设置 `CADDY_API=disabled`，由 Nginx 承担入口，API 不再连接 Caddy Unix Socket。脚本路径：[docs/deploy/native/deploy.sh](docs/deploy/native/deploy.sh)

### 相关文档

| 文档 | 内容 |
| --- | --- |
| [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) | Docker 三种部署方式 |
| [docs/STACK_SLIM.md](docs/STACK_SLIM.md) | 栈精简架构与 RAG 方案 |
| [docs/FEATURE_COMPARISON.md](docs/FEATURE_COMPARISON.md) | 开源版与乘风版功能对比 |

## 首次使用

1. 登录管理后台，按向导配置 chat / embedding / rerank 三类 AI 模型。
2. 创建知识库（Wiki 站点），填写访问域名与端口。
3. 在后台录入文档并等待向量学习任务完成。
4. 打开对应 Wiki 前台验证浏览与 AI 问答。

> 未配置模型时，AI 相关功能不可用。切换 RAG 方案或全新部署后，需对文档执行全量重新学习。

## 贡献

- 乘风版问题与功能建议：[MaydayV/PandaWiki Issues](https://github.com/MaydayV/PandaWiki/issues)
- 原版问题：请先在原版环境复现后，再到 [chaitin/PandaWiki](https://github.com/chaitin/PandaWiki) 提交

## 许可证

本项目采用 [AGPL-3.0](LICENSE) 许可证。
