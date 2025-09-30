# XZ MCP 使用指南


---

## ✅ 方案1：GitHub Releases + 自动安装（已实现）

### 用户体验（3步开始）

#### 第1步：一键安装
```bash
curl -fsSL https://raw.githubusercontent.com/Xuzan9396/xz_mcp/main/install.sh | bash
```

**自动完成**：
- ✅ 检测系统（macOS Intel/ARM, Linux）
- ✅ 下载最新版本
- ✅ 安装到 ~/go/bin/xz_mcp
- ✅ 设置权限

#### 第2步：配置（只需一次）

**Codex 用户** (`~/.codex/config.toml`):
```toml
[mcp_servers.xz_mcp]
command = "/Users/admin/go/bin/xz_mcp"
```

**Claude Desktop 用户**:
```bash
claude mcp add-json xz_mcp -s user '{"type":"stdio","command":"/Users/admin/go/bin/xz_mcp","args":[],"env":{}}'
```

#### 第3步：直接使用
在 Codex 或 Claude Desktop 中使用 35 个数据库工具！

### 更新到最新版本

```bash
# 重新运行安装脚本即可
curl -fsSL https://raw.githubusercontent.com/Xuzan9396/xz_mcp/main/install.sh | bash
```

### 对比 npx 的区别

| 特性 | npx | xz_mcp (方案1) |
|------|-----|----------------|
| 安装 | 每次运行都下载 | 一次安装，多次使用 |
| 速度 | 慢（每次下载） | 快（本地运行） |
| 更新 | 自动 | 一行命令 |
| 配置 | 不需要路径 | 配置一次路径 |

**优势**：比 npx 更快，比手动配置更方便！

---

## ⭐ 方案2：Smithery 平台（完全等同 npx）

如果你想要**完全等同 npx 的体验**，可以发布到 Smithery。

### 发布到 Smithery

```bash
# 1. 安装 Smithery CLI
npm install -g @smithery/cli

# 2. 登录
smithery login

# 3. 初始化（在项目目录）
cd /Users/admin/go/empty/go/mcp_server/xz_mcp
smithery init

# 4. 发布
smithery publish
```

### 用户使用（零配置）

**Codex 配置**:
```toml
[mcp_servers.xz_mcp]
command = "npx"
args = ["-y", "@smithery/cli@latest", "run", "@xuzan/xz_mcp"]
```

**Claude Desktop 配置**:
```json
{
  "mcpServers": {
    "xz_mcp": {
      "command": "npx",
      "args": ["-y", "@smithery/cli@latest", "run", "@xuzan/xz_mcp"]
    }
  }
}
```

### 优点
- ✅ 用户**完全零配置**路径
- ✅ **自动更新**（每次运行都是最新版）
- ✅ Smithery 处理所有平台编译
- ✅ 完全等同 npx 体验

---

## 📊 方案对比

| 方案 | 用户安装 | 路径配置 | 自动更新 | 速度 | 状态 |
|------|---------|---------|---------|------|------|
| **方案1: GitHub** | 一行命令 | 需要一次 | 一行命令 | 快 | ✅ 已实现 |
| **方案2: Smithery** | 零配置 | 不需要 | 自动 | 中等 | 🔜 待发布 |
| **手动安装** | 复杂 | 需要 | 手动 | 快 | 传统方式 |

---

## 🚀 当前实现的自动化

### GitHub Actions 工作流

**文件**: `.github/workflows/release.yml`

**功能**：
1. ✅ 自动编译 macOS（Intel + ARM）
2. ✅ 自动编译 Linux（amd64）
3. ✅ 创建 GitHub Release
4. ✅ 上传二进制文件

**触发方式**：
```bash
# 每次推送标签时自动执行
git tag -a v1.0.1 -m "Release v1.0.1"
git push origin v1.0.1
```

### 自动安装脚本

**文件**: `install.sh`

**功能**：
- ✅ 检测系统架构
- ✅ 下载最新 Release
- ✅ 自动安装
- ✅ 显示配置建议

**用户使用**：
```bash
curl -fsSL https://raw.githubusercontent.com/Xuzan9396/xz_mcp/main/install.sh | bash
```

---

## 📖 完整使用流程

### 对于最终用户

#### 1. 安装
```bash
curl -fsSL https://raw.githubusercontent.com/Xuzan9396/xz_mcp/main/install.sh | bash
```

#### 2. 配置 Codex
```bash
# 编辑 ~/.codex/config.toml，添加：
[mcp_servers.xz_mcp]
command = "/Users/admin/go/bin/xz_mcp"
```

#### 3. 使用
打开 Codex，直接使用 35 个数据库工具：

```bash
# MySQL 示例
mysql_connect(username="root", password="123456", addr="localhost:3306", database_name="mydb")
mysql_query(sql="SELECT * FROM users LIMIT 10")

# PostgreSQL 示例
pgsql_connect(host="localhost", port=5432, user="postgres", password="pwd", database="mydb")
pgsql_query(sql="SELECT * FROM cities LIMIT 10")

# Redis 示例
redis_connect(addr="localhost:6379", password="", db=0)
redis_string(operation="SET", key="test", value="hello")
redis_string(operation="GET", key="test")

# SQLite 示例
sqlite_query(db_path="/path/to/db.sqlite", sql="SELECT * FROM table LIMIT 10")
```

#### 4. 更新
```bash
# 重新运行安装脚本
curl -fsSL https://raw.githubusercontent.com/Xuzan9396/xz_mcp/main/install.sh | bash
```

---

### 对于开发者（发布新版本）

#### 1. 修改代码
```bash
cd /Users/admin/go/empty/go/mcp_server/xz_mcp
# 修改代码...
```

#### 2. 本地测试
```bash
./build.sh
./xz_mcp --version
```

#### 3. 提交代码
```bash
git add .
git commit -m "feat: 添加新功能"
git push origin main
```

#### 4. 发布版本
```bash
# 创建并推送标签
git tag -a v1.0.1 -m "Release v1.0.1"
git push origin v1.0.1
```

#### 5. 等待 GitHub Actions
- 访问：https://github.com/Xuzan9396/xz_mcp/actions
- 等待编译完成（约5分钟）
- 检查 Release：https://github.com/Xuzan9396/xz_mcp/releases

#### 6. 用户自动获取
用户重新运行安装脚本即可获取最新版本！

---

## 🎊 总结

### 你的需求
> "不可能让用户每次都麻烦配置路径"

### 我们的解决方案

#### ✅ 已实现（方案1）
- **安装**：一行命令
- **配置**：只需一次
- **更新**：一行命令
- **体验**：接近 npx

#### ⭐ 可选优化（方案2）
- 发布到 Smithery
- **完全等同 npx**
- 零路径配置

### 推荐流程

**现阶段**：使用方案1（GitHub Releases）
- 用户体验已经很好
- 安装和更新都是一行命令
- 只需配置一次路径

**未来**：发布到 Smithery
- 完全零配置
- 自动更新
- 更专业的 MCP 生态

---

## 📮 相关链接

- **GitHub 仓库**: https://github.com/Xuzan9396/xz_mcp
- **Releases**: https://github.com/Xuzan9396/xz_mcp/releases
- **Actions**: https://github.com/Xuzan9396/xz_mcp/actions
- **Smithery**: https://smithery.ai
- **MCP 文档**: https://modelcontextprotocol.io

---

**项目已完全解决你提出的问题！** 🎉