---
description: 如何发布 Goupter CLI 工具
---

# Goupter 发布指南

本文档介绍如何将 Goupter 发布为可安装的 CLI 工具。

## 📦 发布方式概览

### 1. 通过 `go install` 安装（最简单）

用户可以直接从 GitHub 安装：

```bash
go install github.com/goupter/goupter/cmd/goupter@latest
```

或指定版本：

```bash
go install github.com/goupter/goupter/cmd/goupter@v1.0.0
```

**优点**：

- 无需额外配置
- 自动编译适配用户系统
- Go 开发者熟悉

**缺点**：

- 需要用户安装 Go 环境
- 编译时间较长

---

### 2. 使用 GoReleaser 自动化发布（推荐）

GoReleaser 可以自动构建多平台二进制文件、创建 GitHub Release、生成 checksums 等。

#### 2.1 安装 GoReleaser

```bash
# macOS
brew install goreleaser

# Linux
brew install goreleaser
# 或
go install github.com/goreleaser/goreleaser@latest

# Windows
scoop install goreleaser
```

#### 2.2 测试发布配置

在发布前，先测试配置是否正确：

```bash
# 测试构建（不发布）
goreleaser build --snapshot --clean

# 完整测试（包括归档）
goreleaser release --snapshot --clean
```

#### 2.3 创建 GitHub Release

**方式 A: 使用 Git Tag 触发**

```bash
# 创建并推送 tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# 运行 GoReleaser
export GITHUB_TOKEN="your_github_token"
goreleaser release --clean
```

**方式 B: 使用 GitHub Actions 自动化**

创建 `.github/workflows/release.yml`（见下文）

#### 2.4 配置文件说明

项目已包含 `.goreleaser.yaml` 配置文件，主要功能：

- ✅ 构建多平台二进制文件（Linux/macOS/Windows，amd64/arm64）
- ✅ 生成归档文件（tar.gz/zip）
- ✅ 计算 checksums
- ✅ 创建 GitHub Release
- ✅ 生成更新日志
- ✅ 支持 Homebrew Tap（可选）
- ✅ 支持 Scoop Bucket（可选）
- ✅ 构建 Docker 镜像（可选）

---

### 3. 发布到包管理器

#### 3.1 Homebrew (macOS/Linux)

**步骤 1**: 创建 Homebrew Tap 仓库

```bash
# 在 GitHub 创建仓库: goupter/homebrew-tap
```

**步骤 2**: 配置 GitHub Token

```bash
# 创建 Personal Access Token (需要 repo 权限)
# 设置环境变量
export HOMEBREW_TAP_GITHUB_TOKEN="your_token"
```

**步骤 3**: 发布时自动更新 Tap

GoReleaser 会自动更新 Homebrew formula。

**用户安装**：

```bash
brew tap goupter/tap
brew install goupter
```

#### 3.2 Scoop (Windows)

**步骤 1**: 创建 Scoop Bucket 仓库

```bash
# 在 GitHub 创建仓库: goupter/scoop-bucket
```

**步骤 2**: 配置 GitHub Token

```bash
export SCOOP_BUCKET_GITHUB_TOKEN="your_token"
```

**用户安装**：

```powershell
scoop bucket add goupter https://github.com/goupter/scoop-bucket
scoop install goupter
```

---

### 4. 通过安装脚本

项目包含 `install.sh` 脚本，支持一键安装：

```bash
curl -sSL https://raw.githubusercontent.com/goupter/goupter/main/install.sh | bash
```

脚本功能：

- 自动检测操作系统和架构
- 下载对应的二进制文件
- 安装到合适的目录
- 验证安装

---

## 🚀 完整发布流程

### 方式 A: 手动发布

```bash
# 1. 确保代码已提交
git add .
git commit -m "chore: prepare for release v1.0.0"
git push

# 2. 创建 tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# 3. 设置 GitHub Token
export GITHUB_TOKEN="your_github_token"

# 4. 运行 GoReleaser
goreleaser release --clean

# 5. 检查 GitHub Release
# 访问 https://github.com/goupter/goupter/releases
```

### 方式 B: GitHub Actions 自动化（推荐）

创建 `.github/workflows/release.yml`：

```yaml
name: Release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.25"

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v5
        with:
          distribution: goreleaser
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
          SCOOP_BUCKET_GITHUB_TOKEN: ${{ secrets.SCOOP_BUCKET_GITHUB_TOKEN }}
```

**使用流程**：

```bash
# 1. 提交代码
git add .
git commit -m "feat: add new feature"
git push

# 2. 创建并推送 tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# 3. GitHub Actions 自动构建和发布
# 查看进度: https://github.com/goupter/goupter/actions
```

---

## 📝 版本管理建议

### 语义化版本（Semantic Versioning）

格式：`MAJOR.MINOR.PATCH`

- **MAJOR**: 不兼容的 API 变更
- **MINOR**: 向后兼容的功能新增
- **PATCH**: 向后兼容的问题修复

示例：

- `v1.0.0` - 首个稳定版本
- `v1.1.0` - 新增功能
- `v1.1.1` - 修复 bug
- `v2.0.0` - 重大变更

### 预发布版本

- `v1.0.0-alpha.1` - Alpha 版本
- `v1.0.0-beta.1` - Beta 版本
- `v1.0.0-rc.1` - Release Candidate

---

## 🔍 发布检查清单

发布前确保：

- [ ] 所有测试通过 (`make test`)
- [ ] 代码已格式化 (`make fmt`)
- [ ] Lint 检查通过 (`make lint`)
- [ ] 更新 `CHANGELOG.md`
- [ ] 更新版本号（如果手动管理）
- [ ] 更新文档（README 等）
- [ ] 提交所有更改
- [ ] 创建 Git tag
- [ ] 推送 tag 到 GitHub

---

## 🛠️ 故障排查

### 问题 1: GoReleaser 找不到 Git tag

**解决**：

```bash
git fetch --tags
git tag -l
```

### 问题 2: GitHub Token 权限不足

**解决**：

确保 token 有以下权限：

- `repo` (完整仓库访问)
- `write:packages` (如果发布 Docker 镜像)

### 问题 3: 构建失败

**解决**：

```bash
# 本地测试构建
goreleaser build --snapshot --clean

# 查看详细日志
goreleaser release --clean --debug
```

### 问题 4: Homebrew/Scoop 更新失败

**解决**：

1. 检查 Tap/Bucket 仓库是否存在
2. 检查 Token 权限
3. 手动更新 formula/manifest

---

## 📚 相关资源

- [GoReleaser 文档](https://goreleaser.com/)
- [GitHub Releases 文档](https://docs.github.com/en/repositories/releasing-projects-on-github)
- [Homebrew Tap 创建指南](https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap)
- [Scoop Bucket 创建指南](https://github.com/ScoopInstaller/Scoop/wiki/Buckets)

---

## 🎯 快速开始

如果你只想快速发布，最简单的方式：

```bash
# 1. 安装 GoReleaser
brew install goreleaser

# 2. 创建 tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# 3. 发布
export GITHUB_TOKEN="your_token"
goreleaser release --clean
```

完成！用户现在可以通过以下方式安装：

```bash
# 方式 1: go install
go install github.com/goupter/goupter/cmd/goupter@v1.0.0

# 方式 2: 下载二进制文件
# 访问 https://github.com/goupter/goupter/releases

# 方式 3: 安装脚本
curl -sSL https://raw.githubusercontent.com/goupter/goupter/main/install.sh | bash
```
