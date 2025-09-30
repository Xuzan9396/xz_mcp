# XZ MCP 部署和分发指南

## 🎯 自动更新解决方案

针对你提出的问题："用户如何方便地获取和更新 xz_mcp，而不需要每次手动配置路径？"

我们提供了 **3种分发方案**：

---

## 方案1：GitHub Releases + 自动安装脚本 ⭐（已实现）

### 用户安装（一行命令）

```bash
curl -fsSL https://raw.githubusercontent.com/Xuzan9396/xz_mcp/main/install.sh | bash
```

### 工作原理
1. 安装脚本自动检测系统架构（macOS/Linux, Intel/ARM）
2. 从 GitHub Releases 下载对应平台的最新版本
3. 安装到 `~/go/bin/xz_mcp`
4. 显示配置建议

### 用户配置（只需一次）

**Codex** (`~/.codex/config.toml`):
```toml
[mcp_servers.xz_mcp]
command = "/Users/admin/go/bin/xz_mcp"
```

**Claude Desktop**:
```bash
claude mcp add-json xz_mcp -s user '{"type":"stdio","command":"/Users/admin/go/bin/xz_mcp","args":[],"env":{}}'
```

### 更新（同样一行命令）
```bash
curl -fsSL https://raw.githubusercontent.com/Xuzan9396/xz_mcp/main/install.sh | bash
```

### 发布新版本流程

1. **创建 Git 标签**：
```bash
cd /Users/admin/go/empty/go/mcp_server/xz_mcp
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

2. **GitHub Actions 自动执行**：
   - ✅ 编译 5 个平台版本（macOS Intel/ARM, Linux amd64/arm64, Windows amd64）
   - ✅ 创建 GitHub Release
   - ✅ 上传所有二进制文件

3. **用户自动获取最新版**：
   - 重新运行安装脚本即可

---

## 方案2：Smithery 平台托管（推荐给公开项目）

Smithery 是专门的 MCP 服务托管平台，类似 npm 但专为 MCP 设计。

### 发布到 Smithery

```bash
# 1. 安装 Smithery CLI
npm install -g @smithery/cli

# 2. 登录 Smithery
smithery login

# 3. 初始化配置
smithery init

# 4. 发布
smithery publish
```

### 用户使用（无需任何手动配置）

```bash
# Claude Desktop 一行配置
claude mcp add-json xz_mcp -s user '{"type":"stdio","command":"npx","args":["-y","@smithery/cli@latest","run","@xuzan/xz_mcp"],"env":{}}'
```

### 优点
- ✅ 用户无需手动下载安装
- ✅ 每次使用自动获取最新版本
- ✅ Smithery 自动处理多平台编译
- ✅ 内置版本管理和回滚

---

## 方案3：Homebrew（适合 macOS）

如果项目流行，可以发布到 Homebrew。

### 创建 Homebrew Formula

```ruby
# xz_mcp.rb
class XzMcp < Formula
  desc "XZ MCP - 统一数据库 MCP 服务器"
  homepage "https://github.com/Xuzan9396/xz_mcp"
  url "https://github.com/Xuzan9396/xz_mcp/archive/v1.0.0.tar.gz"
  sha256 "..."
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w")
  end

  test do
    system "#{bin}/xz_mcp", "--version"
  end
end
```

### 用户安装
```bash
brew tap xuzan9396/tap
brew install xz_mcp

# 更新
brew upgrade xz_mcp
```

---

## 📊 方案对比

| 方案 | 安装复杂度 | 更新便捷性 | 适用场景 |
|------|-----------|-----------|---------|
| **GitHub + install.sh** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | 当前实现，推荐 |
| **Smithery** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 公开项目最佳 |
| **Homebrew** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | macOS 用户 |

---

## 🚀 当前实现状态

### ✅ 已完成
1. **GitHub Actions 工作流** (`.github/workflows/release.yml`)
   - 自动编译 5 个平台版本
   - 创建 GitHub Release
   - 上传二进制文件

2. **自动安装脚本** (`install.sh`)
   - 检测系统架构
   - 下载最新版本
   - 自动安装和配置

3. **本地编译脚本** (`build.sh`)
   - 一键编译
   - 复制到系统路径
   - 显示配置建议

### 📝 使用流程

#### 对于最终用户
```bash
# 1. 安装（只需一次）
curl -fsSL https://raw.githubusercontent.com/Xuzan9396/xz_mcp/main/install.sh | bash

# 2. 配置 Codex（只需一次）
# 编辑 ~/.codex/config.toml，添加：
[mcp_servers.xz_mcp]
command = "/Users/admin/go/bin/xz_mcp"

# 3. 使用
# 在 Codex 或 Claude Desktop 中直接使用 35 个数据库工具

# 4. 更新到最新版
curl -fsSL https://raw.githubusercontent.com/Xuzan9396/xz_mcp/main/install.sh | bash
```

#### 对于开发者
```bash
# 1. 发布新版本
git tag -a v1.0.1 -m "Release v1.0.1"
git push origin v1.0.1

# 2. GitHub Actions 自动执行
# - 编译所有平台版本
# - 创建 Release
# - 上传文件

# 3. 用户自动获取
# 用户重新运行安装脚本即可获取最新版本
```

---

## 💡 推荐方案

**当前阶段**：使用 **方案1（GitHub Releases）**
- ✅ 已完全实现
- ✅ 用户安装简单（一行命令）
- ✅ 更新方便（重新运行安装脚本）
- ✅ 支持多平台

**未来优化**：发布到 **Smithery**
- ⭐ 完全自动更新（类似 npx）
- ⭐ 用户零配置路径
- ⭐ Smithery 处理所有平台编译

---

## 🎯 下一步建议

1. **测试 GitHub Actions**：
   ```bash
   git tag -a v1.0.0 -m "First release"
   git push origin v1.0.0
   ```

2. **验证安装脚本**：
   ```bash
   curl -fsSL https://raw.githubusercontent.com/Xuzan9396/xz_mcp/main/install.sh | bash
   ```

3. **发布到 Smithery**（可选）：
   - 注册 Smithery 账号
   - 运行 `smithery publish`
   - 获得更好的用户体验

---

**总结**：通过 GitHub Releases + install.sh，用户体验已经和 npx 非常接近，只需一行命令安装和更新！