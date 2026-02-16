<p align="center">
  <img src="/images/banner.png" width="400" />
</p>

<p align="center">
  <a target="_blank" href="https://ly.safepoint.cloud/Br48PoX">📖 官方网站</a> &nbsp; | &nbsp;
  <a target="_blank" href="https://github.com/MaydayV/PandaWiki">🌬️ 乘风版仓库</a> &nbsp; | &nbsp;
  <a target="_blank" href="https://github.com/chaitin/PandaWiki">🙏 原版仓库（致谢）</a>
</p>

## 👋 项目介绍

PandaWiki 是一款 AI 大模型驱动的**开源知识库搭建系统**，帮助你快速构建智能化的 **产品文档、技术文档、FAQ、博客系统**，借助大模型的力量为你提供 **AI 创作、AI 问答、AI 搜索** 等能力。

## 🌬️ 乘风版故事

乘风版（Fly Version）诞生于一个很朴素的目标：把开源能力真正落到长期、稳定、可演进的业务现场。  
我们在 PandaWiki 的开源基础上持续深挖，把“能跑起来”升级为“能长期跑、能放心改、能快速交付”。

“乘风”这两个字，不是另起炉灶的姿态，而是对开源精神的延续。  
风，来自社区共享的技术势能；乘，是在尊重原作、理解原作之上，把工程细节打磨得更实、更稳、更贴近真实场景。

因此，乘风版坚持两件事：
- 坚持致谢：感谢 PandaWiki 原作者与社区，让我们站在巨人的肩膀上继续前进。
- 坚持负责：乘风版的问题、节奏与交付由乘风版仓库独立承担，不让原作者为二开问题“背锅”。

<p align="center">
  <img src="/images/setup.png" width="800" />
</p>

## ⚡️ 界面展示

| PandaWiki 控制台                                 | Wiki 网站前台                                    |
| ------------------------------------------------ | ------------------------------------------------ |
| <img src="/images/screenshot-1.png" width=370 /> | <img src="/images/screenshot-2.png" width=370 /> |
| <img src="/images/screenshot-3.png" width=370 /> | <img src="/images/screenshot-4.png" width=370 /> |

## 🔥 功能与特色

- AI 驱动智能化：AI 辅助创作、AI 辅助问答、AI 辅助搜索。
- 强大的富文本编辑能力：兼容 Markdown 和 HTML，支持导出为 word、pdf、markdown 等多种格式。
- 轻松与第三方应用进行集成：支持做成网页挂件挂在其他网站上，支持做成钉钉、飞书、企业微信等聊天机器人。
- 通过第三方来源导入内容：根据网页 URL 导入、通过网站 Sitemap 导入、通过 RSS 订阅、通过离线文件导入等。

## 🛫 乘风版版本信息

- 产品型号：Fly Version（乘风版）
- 当前版本：`FV2.6.1.2111`
- 版本规则：`FV{大版本}.{功能序号}.{提交序号}.{原版版本号去点}`
- 说明：乘风版基于 PandaWiki 开源项目进行深度二次开发，保留对原作者与开源社区的致谢；功能边界与发布节奏以本仓库为准。

## 🎯 乘风版愿景

- 让知识库系统从“演示可用”走向“生产可用”：关注稳定性、可观测性、可维护性。
- 让二开迭代从“改得动”走向“改得稳”：尽量减少隐式耦合，提升功能扩展的一致性。
- 让交付体验从“功能堆叠”走向“体系化演进”：每次改动都服务于长期可持续的工程能力。

我们欢迎共建，也欢迎质疑与讨论。只要方向是让系统更可靠、更清晰、更可持续，乘风版就会继续向前。

## 📦 部署文档

- 乘风版部署指南（手动安装环境 + 方案 B 预构建镜像）：[`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md)
- 开源版与乘风版功能对比：[`docs/FEATURE_COMPARISON.md`](docs/FEATURE_COMPARISON.md)
- 版本更新记录笔记：[`docs/VERSION_NOTES.md`](docs/VERSION_NOTES.md)

## 🚀 上手指南

> 以下内容为原版安装步骤，保留作历史对照，请勿用于乘风版部署。

~~### 安装 PandaWiki（原版）~~

~~你需要一台支持 Docker 20.x 以上版本的 Linux 系统来安装 PandaWiki。~~

~~使用 root 权限登录你的服务器，然后执行以下命令。~~

~~`bash -c "$(curl -fsSLk https://release.baizhi.cloud/panda-wiki/manager.sh)"`~~

~~根据命令提示的选项进行安装，命令执行过程将会持续几分钟，请耐心等待。~~

~~> 关于安装与部署的更多细节请参考 [安装 PandaWiki](https://pandawiki.docs.baizhi.cloud/node/01971602-bb4e-7c90-99df-6d3c38cfd6d5)。~~

~~### 登录 PandaWiki（原版）~~

~~在上一步中，安装命令执行结束后，你的终端会输出以下内容。~~

~~`SUCCESS 控制台信息...（略）`~~

~~使用浏览器打开上述内容中的 “访问地址”，你将看到 PandaWiki 的控制台登录入口，使用上述内容中的 “用户名” 和 “密码” 登录即可。~~

### 配置 AI 模型

> PandaWiki 是由 AI 大模型驱动的 Wiki 系统，在未配置大模型的情况下 AI 创作、AI 问答、AI 搜索 等功能无法正常使用。
> 
首次登录时会提示需要先配置 AI 模型，可自行选择一键配置或手动配置。

<div align="center">
  <img src="/images/model-config-1.png" width="800" />
  <p><em>一键自动配置 AI 模型</em></p>

  <img src="/images/model-config-2.png" width="800" />
  <p><em>手动自定义配置 AI 模型</em></p>
</div>



> 推荐使用 [百智云模型广场](https://baizhi.cloud/) 快速接入 AI 模型，注册即可获赠 5 元的模型使用额度。
> 关于大模型的更多配置细节请参考 [接入 AI 模型](https://pandawiki.docs.baizhi.cloud/node/01971616-811c-70e1-82d9-706a202b8498)。

### 创建知识库

“知识库” 是一组文档的集合，PandaWiki 将会根据知识库中的文档，为不同的知识库分别创建 “Wiki 网站”。
<img src="/images/createkb.png" width="800" />

### 💪 开始使用

如果你顺利完成了以上步骤，那么恭喜你，属于你的 PandaWiki 搭建成功，你可以：

- 访问 **控制台** 来管理你的知识库并上传文档等待学习成功
- 访问 **Wiki 网站** 使用知识库并测试AI问答效果
<img src="/images/AI-QA.png" width="700" />

### 💬 遇到问题

如在使用产品过程中遇到问题，可通过以下方式获取帮助：
- 📘查阅官方文档：[常见问题](https://pandawiki.docs.baizhi.cloud/node/019b4952-4ed3-7514-ba57-c93a8ca13608)，更多内容请参考文档目录。
- 🤖不想翻文档？试试 [AI 问答](https://pandawiki.docs.baizhi.cloud/node/0197160c-782c-74ad-a4b7-857dae148f84)，快速获取答案。

## 🙋‍♂️ 贡献

- 乘风版问题与改进建议：请提交到 [MaydayV/PandaWiki](https://github.com/MaydayV/PandaWiki)。
- 原版问题反馈：请先在原版环境复现后，再到 [chaitin/PandaWiki](https://github.com/chaitin/PandaWiki) 提交。

## 📝 许可证

本项目采用 GNU Affero General Public License v3.0 (AGPL-3.0) 许可证。这意味着：

- 你可以自由使用、修改和分发本软件
- 你必须以相同的许可证开源你的修改
- 如果你通过网络提供服务，也必须开源你的代码
- 商业使用需要遵守相同的开源要求


## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=chaitin/PandaWiki&type=Date)](https://www.star-history.com/#chaitin/PandaWiki&Date)
