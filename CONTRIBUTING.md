# 贡献指南

感谢你有兴趣为 ShieldFlow CDN 贡献代码！本文档指导你如何参与开发。

## 🚀 快速开始

### 环境要求

| 依赖 | 最低版本 | 用途 |
|------|---------|------|
| Go | 1.22+ | 后端编译 |
| Node.js | 18+ | 前端编译 |
| PostgreSQL | 14+ | 配置数据库 |
| ClickHouse | 23+ | 日志数据库 |
| Redis | 6+ | 缓存 |
| Make | 任意 | 构建工具 |
| protoc | 3.0+ | gRPC 代码生成（可选） |

### 克隆 & 编译

```bash
git clone https://github.com/717315051/shieldflow.git
cd shieldflow

# 编译后端
go build ./...

# 编译前端
cd web && npm install && npm run build

# 生成所有二进制
make build
```

### 开发模式

```bash
# 后端热重载（需安装 air）
go install github.com/air-verse/air@latest
air

# 前端热重载
cd web && npm run dev
```

## 📁 项目结构

详见 [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) 中的目录结构详解。

## 📝 代码规范

### Go 代码

- 遵循 [Effective Go](https://go.dev/doc/effective_go) 和 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- 使用 `gofmt` / `goimports` 格式化
- 包名小写单词，如 `handlers`、`models`
- 导出函数必须有文档注释
- 错误必须处理，不得使用 `_` 忽略
- 命名规范：
  - 结构体：`PascalCase`（如 `UserPackage`）
  - 常量：`PascalCase` 或 `ALL_CAPS`
  - 私有变量：`camelCase`

### Vue 代码

- 使用 `<script setup>` 语法
- 组件名 `PascalCase`（如 `DomainDetail.vue`）
- Props 使用 `defineProps` 并声明类型
- API 调用统一通过 `@/api` 模块
- 样式使用 `scoped`

### 通用

- 提交信息格式：`<type>: <description>`
  - `feat:` 新功能
  - `fix:` 修复
  - `docs:` 文档
  - `refactor:` 重构
  - `chore:` 杂项
- 分支命名：`feature/xxx`、`fix/xxx`、`docs/xxx`

## 🔄 贡献流程

1. **Fork** 仓库
2. 创建分支：`git checkout -b feature/your-feature`
3. 编写代码，确保通过编译：
   ```bash
   go build ./...
   cd web && npm run build
   ```
4. 提交：`git commit -m "feat: 添加 XXX 功能"`
5. 推送：`git push origin feature/your-feature`
6. 创建 **Pull Request**，描述变更内容

## ✅ 提交前检查清单

- [ ] `go build ./...` 通过
- [ ] `go vet ./...` 无警告
- [ ] `cd web && npm run build` 通过
- [ ] 提交信息符合规范
- [ ] 新功能有对应的配置项或文档
- [ ] 不引入新的第三方依赖（除非必要）
- [ ] 敏感信息（密码、密钥）已替换为占位符

## 🐛 报告 Bug

请在 [Issues](https://github.com/717315051/shieldflow/issues) 中提交，包含：

- 问题描述
- 复现步骤
- 期望行为 vs 实际行为
- 环境信息（OS、Go版本、浏览器等）
- 日志/截图（如有）

## 💡 功能建议

欢迎在 [Issues](https://github.com/717315051/shieldflow/issues) 中提交功能建议，标注 `enhancement` 标签。

## 📄 开源协议

本项目采用 [MIT License](LICENSE)。
