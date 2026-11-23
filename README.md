<p align="center">
  <img src="/images/banner.png" width="400" />
</p>

<p align="center">
  <a target="_blank" href="https://github.com/MaydayV/PandaWiki">乘风版仓库</a> &nbsp;|&nbsp;
  <a target="_blank" href="https://github.com/chaitin/PandaWiki">原版仓库（致谢）</a>
</p>

## 项目介绍

PandaWiki 乘风版（Fly Version）是基于 [PandaWiki 开源项目](https://github.com/chaitin/PandaWiki) 二次开发的 AI 知识库系统，用于搭建产品文档、技术文档、FAQ 等内容站点，并提供 AI 创作、问答与搜索能力。

当前主线分支 **`main`** 已完成部署栈精简：默认 **5 容器**、向量检索使用 **Postgres pgvector**、附件存储默认 **阿里云 OSS**（S3 兼容 API）。

## 版本

| 项 | 说明 |
| --- | --- |
| 版本号 | `FV2.6.19.2111` |
| 默认分支 | `main` |
| 运行时 | Caddy + Postgres + Redis + API + App |
| RAG | `RAG_PROVIDER=pg`（pgvector） |
| 对象存储 | OSS / S3 兼容端点 |

详细变更见 [`docs/CHANGELOG_FV.md`](docs/CHANGELOG_FV.md) 与 [`docs/VERSION_NOTES.md`](docs/VERSION_NOTES.md)。

## 功能

- AI 辅助创作、问答、搜索（Postgres `pgvector` + 全文检索）
- 富文本 / Markdown 编辑，支持多种导出格式
- GitHub 私有仓库 MD 文章同步：以 Git 为唯一内容真源，自动按目录与 front-matter 识别分类、图片转存 OSS、增量自动发布并触发 RAG 更新
- 知识库更新群聊推送（支持飞书、钉钉 Webhook 通知模板）
- 网页挂件、钉钉 / 飞书 / 企业微信等机器人集成
- 安全与凭据加密：支持 `ENCRYPTION_KEY` 加密落库 API Token / 访问口令 / GitHub PAT，以及 `TRUST_PROXY` 可信代理限频防护

## 部署

### 一键安装（推荐）

Linux 服务器，已安装 Docker Compose v2。准备好阿里云 OSS（或其它 S3 兼容存储）。

```bash
curl -fsSL https://github.com/MaydayV/PandaWiki/releases/latest/download/install.sh | bash
```

脚本会下载发布包（含 `backup.sh` 备份脚本）、生成数据库/JWT/管理员密码及 `ENCRYPTION_KEY`，并询问 OSS 凭证，然后拉起 5 个容器。部署完成后请妥善备份安装目录下的 `.env`（尤其是 `ENCRYPTION_KEY`），可使用目录内的 `./backup.sh` 定时备份数据库。

指定版本与目录：

```bash
curl -fsSL https://github.com/MaydayV/PandaWiki/releases/latest/download/install.sh | \
  INSTALL_DIR=/opt/pandawiki PANDAWIKI_VERSION=latest bash
```

非交互安装可预先导出 `S3_ACCESS_KEY`、`S3_SECRET_KEY`、`S3_BUCKET`、`S3_PUBLIC_BASE_URL`。

启动后：

- **管理后台**：`https://<服务器IP>:2443`
- **Wiki 前台**：`http://<服务器IP>:3010`（经 Caddy）
- 管理员密码见安装目录 `.env` 中的 `ADMIN_PASSWORD`

完整说明：[docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)

### 从源码使用 Compose

```bash
git clone https://github.com/MaydayV/PandaWiki.git
cd PandaWiki
cd docs/deploy
./generate-env.sh
./quickstart.sh
```

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
| [docs/CHANGELOG_FV.md](docs/CHANGELOG_FV.md) | 乘风版各版本更新日志 |
| [docs/VERSION_NOTES.md](docs/VERSION_NOTES.md) | 详细改造与迭代记录 |
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
