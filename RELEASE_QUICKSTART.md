# 快速发布指南

这是一个简化的发布流程，适合快速上手。完整文档请参考 `doc/RELEASE.md`。

## 🚀 首次发布准备

### 1. 安装 GoReleaser

```bash
# macOS
brew install goreleaser

# 或使用 go install
go install github.com/goreleaser/goreleaser@latest
```

### 2. 设置 GitHub Token

```bash
# 创建 Personal Access Token
# 访问: https://github.com/settings/tokens/new
# 权限: repo (完整仓库访问)

# 设置环境变量
export GITHUB_TOKEN="your_github_token_here"

# 可选：添加到 ~/.zshrc 或 ~/.bashrc
echo 'export GITHUB_TOKEN="your_github_token_here"' >> ~/.zshrc
```

### 3. 测试构建

```bash
# 测试 GoReleaser 配置
make release-test

# 或直接运行
goreleaser build --snapshot --clean
```

## 📦 发布新版本

### 方式 A: 使用 Makefile（推荐）

```bash
# 1. 确保所有更改已提交
git add .
git commit -m "chore: prepare for release"
git push

# 2. 创建并推送 tag（会提示输入版本号）
make tag
# 输入: v1.0.0

# 3. 发布
make release
```

### 方式 B: 手动步骤

```bash
# 1. 提交代码
git add .
git commit -m "chore: prepare for release v1.0.0"
git push

# 2. 创建 tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# 3. 运行 GoReleaser
goreleaser release --clean
```

### 方式 C: GitHub Actions（自动化）

只需推送 tag，GitHub Actions 会自动构建和发布：

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# 查看构建进度
# https://github.com/goupter/goupter/actions
```

## ✅ 验证发布

发布成功后，检查：

1. **GitHub Release**: https://github.com/goupter/goupter/releases
   - 查看是否创建了新的 Release
   - 确认二进制文件已上传
   - 检查 checksums.txt

2. **测试安装**:

   ```bash
   # 通过 go install
   go install github.com/goupter/goupter/cmd/goupter@v1.0.0

   # 验证版本
   goupter --version
   ```

3. **测试安装脚本**:
   ```bash
   curl -sSL https://raw.githubusercontent.com/goupter/goupter/main/install.sh | bash
   ```

## 🔄 版本号规则

遵循 [语义化版本](https://semver.org/lang/zh-CN/)：

- `v1.0.0` - 首个稳定版本
- `v1.1.0` - 新增功能（向后兼容）
- `v1.0.1` - Bug 修复（向后兼容）
- `v2.0.0` - 重大变更（不兼容）

预发布版本：

- `v1.0.0-alpha.1` - Alpha 测试版
- `v1.0.0-beta.1` - Beta 测试版
- `v1.0.0-rc.1` - Release Candidate

## 📝 发布前检查清单

- [ ] 所有测试通过: `make test`
- [ ] 代码已格式化: `make fmt`
- [ ] Lint 检查通过: `make lint`
- [ ] 更新 `CHANGELOG.md`
- [ ] 更新文档（如有需要）
- [ ] 所有更改已提交并推送

## 🛠️ 常见问题

### Q: GoReleaser 报错 "git tag not found"

**A**: 确保你已经创建并推送了 tag：

```bash
git tag -l  # 查看本地 tags
git push origin v1.0.0  # 推送 tag
```

### Q: GitHub Token 权限不足

**A**: 确保 token 有 `repo` 权限，重新生成 token：
https://github.com/settings/tokens/new

### Q: 如何撤销发布？

**A**: 在 GitHub Release 页面删除 Release 和 tag：

```bash
# 删除远程 tag
git push --delete origin v1.0.0

# 删除本地 tag
git tag -d v1.0.0
```

### Q: 如何发布预览版？

**A**: 使用 snapshot 模式：

```bash
make release-snapshot
```

## 📚 更多信息

- 完整发布指南: `doc/RELEASE.md`
- GoReleaser 文档: https://goreleaser.com/
- 项目文档: `README.md`

## 🎯 快速命令参考

```bash
# 安装 GoReleaser
make install-goreleaser

# 测试构建
make release-test

# 创建 tag
make tag

# 发布
make release

# 查看所有 make 命令
make help
```
