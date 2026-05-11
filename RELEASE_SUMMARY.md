# Goupter 发布配置总结

本文档总结了为 Goupter 项目添加的发布相关配置和文件。

## 📁 新增文件清单

### 1. 核心配置文件

#### `.goreleaser.yaml`

- **用途**: GoReleaser 主配置文件
- **功能**:
  - 多平台构建（Linux/macOS/Windows，amd64/arm64/arm）
  - 自动生成归档文件和 checksums
  - 创建 GitHub Release
  - 生成更新日志
  - 支持 Homebrew Tap 和 Scoop Bucket
  - 构建 Docker 镜像（可选）
- **文档**: https://goreleaser.com/

#### `.github/workflows/release.yml`

- **用途**: GitHub Actions 自动发布工作流
- **触发条件**: 推送 `v*` 格式的 tag
- **功能**:
  - 自动运行测试
  - 自动构建多平台二进制文件
  - 自动创建 GitHub Release
  - 自动更新 Homebrew/Scoop（如果配置）

### 2. 安装脚本

#### `install.sh`

- **用途**: 通用安装脚本
- **支持系统**: Linux/macOS/Windows
- **功能**:
  - 自动检测操作系统和架构
  - 下载对应的二进制文件
  - 安装到合适的目录
  - 验证安装
- **使用方式**:
  ```bash
  curl -sSL https://raw.githubusercontent.com/goupter/goupter/main/install.sh | bash
  ```

#### `Dockerfile.goreleaser`

- **用途**: GoReleaser 专用 Dockerfile
- **功能**: 构建轻量级容器镜像
- **基础镜像**: Alpine Linux

### 3. 文档文件

#### `doc/RELEASE.md`

- **用途**: 完整的发布指南
- **内容**:
  - 多种发布方式详解
  - GoReleaser 配置说明
  - 包管理器发布流程
  - 版本管理建议
  - 故障排查指南

#### `RELEASE_QUICKSTART.md`

- **用途**: 快速发布指南
- **内容**:
  - 简化的发布流程
  - 常用命令参考
  - 快速问题解答

#### `CHANGELOG.md`

- **用途**: 版本变更记录
- **格式**: Keep a Changelog 标准
- **维护**: 每次发布前更新

### 4. 更新的文件

#### `Makefile`

- **新增命令**:
  - `make help` - 显示所有可用命令
  - `make release-test` - 测试 GoReleaser 配置
  - `make release-snapshot` - 创建快照版本
  - `make release` - 发布新版本
  - `make tag` - 创建并推送 git tag
  - `make install-goreleaser` - 安装 GoReleaser
- **改进**: 所有命令都有中文描述

#### `README.md`

- **新增章节**: "安装 CLI 工具"
- **内容**: 4 种安装方式的说明
- **更新**: 添加发布指南链接

## 🚀 使用指南

### 快速开始

1. **查看所有可用命令**:

   ```bash
   make help
   ```

2. **测试发布配置**:

   ```bash
   make release-test
   ```

3. **发布新版本**:

   ```bash
   # 方式 1: 使用 make 命令
   make tag      # 创建 tag（会提示输入版本号）
   make release  # 发布

   # 方式 2: 手动步骤
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   export GITHUB_TOKEN="your_token"
   goreleaser release --clean
   ```

### 安装 GoReleaser

```bash
# 使用 make 命令
make install-goreleaser

# 或手动安装
brew install goreleaser
```

### 配置 GitHub Token

```bash
# 创建 Personal Access Token
# 访问: https://github.com/settings/tokens/new
# 权限: repo (完整仓库访问)

# 设置环境变量
export GITHUB_TOKEN="your_github_token_here"

# 永久保存（可选）
echo 'export GITHUB_TOKEN="your_github_token_here"' >> ~/.zshrc
source ~/.zshrc
```

## 📦 发布流程

### 方式 A: 手动发布

```bash
# 1. 更新 CHANGELOG.md
vim CHANGELOG.md

# 2. 提交更改
git add .
git commit -m "chore: prepare for release v1.0.0"
git push

# 3. 创建 tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# 4. 发布
export GITHUB_TOKEN="your_token"
goreleaser release --clean
```

### 方式 B: 使用 Makefile

```bash
# 1. 更新 CHANGELOG.md
vim CHANGELOG.md

# 2. 提交更改
git add .
git commit -m "chore: prepare for release v1.0.0"
git push

# 3. 创建并推送 tag
make tag
# 输入: v1.0.0

# 4. 发布
make release
```

### 方式 C: GitHub Actions（自动化）

```bash
# 1. 更新 CHANGELOG.md
vim CHANGELOG.md

# 2. 提交更改
git add .
git commit -m "chore: prepare for release v1.0.0"
git push

# 3. 创建并推送 tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# GitHub Actions 会自动构建和发布
# 查看进度: https://github.com/goupter/goupter/actions
```

## 🎯 用户安装方式

发布后，用户可以通过以下方式安装：

### 1. 安装脚本（推荐）

```bash
curl -sSL https://raw.githubusercontent.com/goupter/goupter/main/install.sh | bash
```

### 2. go install

```bash
# 最新版本
go install github.com/goupter/goupter/cmd/goupter@latest

# 指定版本
go install github.com/goupter/goupter/cmd/goupter@v1.0.0
```

### 3. 下载二进制文件

访问 [Releases](https://github.com/goupter/goupter/releases) 页面下载。

### 4. Homebrew（需要配置 Tap）

```bash
brew tap goupter/tap
brew install goupter
```

### 5. Scoop（需要配置 Bucket）

```powershell
scoop bucket add goupter https://github.com/goupter/scoop-bucket
scoop install goupter
```

### 6. Docker

```bash
docker pull ghcr.io/goupter/goupter:latest
docker run --rm ghcr.io/goupter/goupter:latest --version
```

## 📋 发布前检查清单

- [ ] 所有测试通过: `make test`
- [ ] 代码已格式化: `make fmt`
- [ ] Lint 检查通过: `make lint`
- [ ] 更新 `CHANGELOG.md`
- [ ] 更新版本号（如果需要）
- [ ] 更新文档（如果有新功能）
- [ ] 所有更改已提交并推送
- [ ] 测试 GoReleaser 配置: `make release-test`
- [ ] 设置 `GITHUB_TOKEN` 环境变量

## 🔧 常用命令

```bash
# 查看帮助
make help

# 构建项目
make build

# 运行测试
make test

# 测试发布配置
make release-test

# 创建快照版本（不发布到 GitHub）
make release-snapshot

# 创建 tag
make tag

# 发布（需要先创建 tag）
make release

# 安装 GoReleaser
make install-goreleaser

# 查看当前版本
./bin/goupter --version

# 查看所有 tags
git tag -l

# 删除本地 tag
git tag -d v1.0.0

# 删除远程 tag
git push --delete origin v1.0.0
```

## 📚 相关文档

- **快速发布指南**: `RELEASE_QUICKSTART.md`
- **完整发布指南**: `doc/RELEASE.md`
- **项目文档**: `README.md`

## 🔗 外部资源

- [GoReleaser 官方文档](https://goreleaser.com/)
- [GitHub Actions 文档](https://docs.github.com/en/actions)
- [语义化版本规范](https://semver.org/lang/zh-CN/)
- [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)

## ⚠️ 注意事项

1. **首次发布前**:
   - 确保已设置 `GITHUB_TOKEN`
   - 测试 GoReleaser 配置: `make release-test`
   - 检查 `.goreleaser.yaml` 中的仓库信息

2. **版本号规则**:
   - 使用语义化版本: `vMAJOR.MINOR.PATCH`
   - Tag 必须以 `v` 开头: `v1.0.0`
   - 预发布版本: `v1.0.0-alpha.1`, `v1.0.0-beta.1`

3. **GitHub Actions**:
   - 确保仓库有 Actions 权限
   - 检查 Secrets 配置（如果使用 Homebrew/Scoop）

4. **可选功能**:
   - Homebrew Tap: 需要创建 `goupter/homebrew-tap` 仓库
   - Scoop Bucket: 需要创建 `goupter/scoop-bucket` 仓库
   - Docker 镜像: 需要配置 GitHub Container Registry

## 🎉 完成！

现在你的项目已经配置好了完整的发布流程！

下一步：

1. 阅读 `RELEASE_QUICKSTART.md` 了解快速发布流程
2. 运行 `make release-test` 测试配置
3. 准备好后，创建第一个 release: `make tag` → `make release`

祝发布顺利！🚀
