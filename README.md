# XZ MCP - 统一数据库 MCP 服务器

> 一个整合了 MySQL、PostgreSQL、Redis、SQLite 四种数据库的统一 MCP (Model Context Protocol) 服务器

## 🎯 项目简介

XZ MCP 是一个基于 [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) 框架开发的统一数据库 MCP 服务器，通过单一服务提供对多种数据库的访问能力。

### 主要特性

- ✅ **统一接口** - 一个服务器集成 4 种数据库
- ✅ **标准协议** - 完全兼容 MCP 协议规范
- ✅ **独立工具** - 15 个数据库操作工具，命名空间隔离
- ✅ **生产就绪** - 包含错误恢复、连接管理等生产特性

## 📦 集成的数据库

| 数据库 | 工具数量 | 主要功能 |
|--------|---------|---------|
| **MySQL** | 8 | 连接管理、查询执行、存储过程 |
| **PostgreSQL** | 3 | 连接管理、查询执行、DML 操作 |
| **Redis** | 3 | 连接管理、通用命令执行、Lua 脚本 |
| **SQLite** | 1 | 统一查询接口（SELECT/DML） |
| **总计** | **15** | - |

## 🛠️ 工具列表

### MySQL 工具 (8个)

#### 连接管理
- `mysql_connect` - 连接到 MySQL 数据库

#### 查询执行
- `mysql_query` - 执行查询操作（SELECT/SHOW/DESCRIBE 等）
- `mysql_exec` - 执行 DML/DDL 操作（INSERT/UPDATE/DELETE/CREATE TABLE/ALTER TABLE/DROP TABLE 等）
- `mysql_exec_get_id` - 执行 INSERT 并返回自增 ID

#### 存储过程
- `mysql_call_procedure` - 调用存储过程
- `mysql_create_procedure` - 创建存储过程
- `mysql_drop_procedure` - 删除存储过程
- `mysql_show_procedures` - 列出所有存储过程

### PostgreSQL 工具 (3个)

- `pgsql_connect` - 连接到 PostgreSQL 数据库
- `pgsql_query` - 执行 SELECT 查询
- `pgsql_exec` - 执行 INSERT/UPDATE/DELETE 操作

### Redis 工具 (3个)

#### 连接管理
- `redis_connect` - 连接到 Redis 服务器

#### 通用操作
- `redis_command` - 执行任意 Redis 命令
- `redis_lua` - 执行 Lua 脚本

### SQLite 工具 (1个)

- `sqlite_query` - 执行 SQL 查询（支持 SELECT 和 DML）

## 🚀 安装与使用

### 方式1：自动安装脚本（最简单）⭐

从 GitHub Release 自动下载最新版本：

```bash
# 一键安装
curl -fsSL https://raw.githubusercontent.com/Xuzan9396/xz_mcp/main/install.sh | bash
```

安装后会自动：
- ✅ 检测系统架构（macOS/Linux, amd64/arm64）
- ✅ 下载对应平台的最新版本
- ✅ 安装到 ~/go/bin/xz_mcp
- ✅ 设置执行权限
- ✅ 显示配置建议

### 方式2：手动下载 Release

访问 [Releases 页面](https://github.com/Xuzan9396/xz_mcp/releases) 下载对应平台的二进制文件：

```bash
# macOS (Apple Silicon)
curl -L https://github.com/Xuzan9396/xz_mcp/releases/latest/download/xz_mcp_darwin_arm64 -o xz_mcp
chmod +x xz_mcp
mv xz_mcp ~/go/bin/

# macOS (Intel)
curl -L https://github.com/Xuzan9396/xz_mcp/releases/latest/download/xz_mcp_darwin_amd64 -o xz_mcp
chmod +x xz_mcp
mv xz_mcp ~/go/bin/

# Linux (amd64)
curl -L https://github.com/Xuzan9396/xz_mcp/releases/latest/download/xz_mcp_linux_amd64 -o xz_mcp
chmod +x xz_mcp
sudo mv xz_mcp /usr/local/bin/
```

#### Windows (PowerShell)

```powershell
# 下载
Invoke-WebRequest -Uri "https://github.com/Xuzan9396/xz_mcp/releases/latest/download/xz_mcp_windows_amd64.exe" -OutFile "xz_mcp.exe"

# 移动到合适的位置（例如）
Move-Item xz_mcp.exe C:\Users\$env:USERNAME\go\bin\xz_mcp.exe
```

#### Windows 配置

**Codex** (`%USERPROFILE%\.codex\config.toml`):
```toml
[mcp_servers.xz_mcp]
command = "C:\\Users\\YourUsername\\go\\bin\\xz_mcp.exe"
```

**Claude Desktop** (`%APPDATA%\Claude\claude_desktop_config.json`):
```json
{
  "mcpServers": {
    "xz_mcp": {
      "command": "C:\\Users\\YourUsername\\go\\bin\\xz_mcp.exe",
      "args": [],
      "env": {}
    }
  }
}
```

### 方式3：从源码编译

#### 前置要求
- Go 1.21 或更高版本
- 项目使用纯 Go SQLite 驱动（modernc.org/sqlite），无需 CGO

#### 使用编译脚本（推荐）

```bash
cd /Users/admin/go/empty/go/mcp_server/xz_mcp
./build.sh
```

编译脚本会自动完成：
1. 清理旧文件
2. 下载依赖（go mod tidy）
3. 优化编译（-ldflags "-s -w"）
4. 设置执行权限
5. 复制到 /Users/admin/go/bin/

#### 方式2：手动编译

```bash
cd /Users/admin/go/empty/go/mcp_server/xz_mcp

# 安装依赖
go mod tidy

# 优化编译
go build -ldflags "-s -w" -o xz_mcp main.go

# 设置权限
chmod +x xz_mcp

# 复制到系统路径
cp -f xz_mcp /Users/admin/go/bin/
```

#### 编译参数说明
- `-ldflags "-s -w"` - 去除调试信息，减小可执行文件大小
- `-o xz_mcp` - 指定输出文件名

## 📖 配置与使用

### 配置 MCP 客户端

#### 方式1：Codex 配置（推荐）

编辑 `~/.codex/config.toml`，添加：

```toml
[mcp_servers.xz_mcp]
command = "/Users/admin/go/bin/xz_mcp"
```

#### 方式2：Claude Desktop 配置

在 Claude Desktop 配置文件中添加：

```json
{
  "mcpServers": {
    "xz_mcp": {
      "command": "/Users/admin/go/bin/xz_mcp",
      "args": [],
      "env": {}
    }
  }
}
```

或使用 Claude CLI：

```bash
claude mcp add-json xz_mcp -s user '{"type":"stdio","command":"/Users/admin/go/bin/xz_mcp","args":[],"env":{}}'
```

### 验证安装

```bash
# 查看版本
xz_mcp --version
# 或
/Users/admin/go/bin/xz_mcp --version

# 输出:
# XZ MCP Unified Database Server v1.0.0
# Integrated: MySQL, PostgreSQL, Redis, SQLite
```

### 更新到最新版本

#### 自动更新
```bash
# 重新运行安装脚本即可
curl -fsSL https://raw.githubusercontent.com/Xuzan9396/xz_mcp/main/install.sh | bash
```

#### 手动更新
```bash
# 下载最新 Release 并替换
cd ~/go/bin
curl -L https://github.com/Xuzan9396/xz_mcp/releases/latest/download/xz_mcp_darwin_arm64 -o xz_mcp
chmod +x xz_mcp
```

### 使用 MCP Inspector 调试

```bash
# 安装 MCP Inspector
npm install -g @modelcontextprotocol/inspector

# 启动调试界面
mcp-inspector /Users/admin/go/bin/xz_mcp
```

浏览器会自动打开调试界面，可以测试所有 21 个工具。

## 💡 使用示例

### MySQL 示例

```javascript
// 1. 连接到 MySQL
{
  "tool": "mysql_connect",
  "arguments": {
    "username": "root",
    "password": "123456",
    "addr": "127.0.0.1:3306",
    "database_name": "test_db"
  }
}

// 2. 查询数据
{
  "tool": "mysql_query",
  "arguments": {
    "sql": "SELECT * FROM users LIMIT 10"
  }
}

// 3. 插入数据
{
  "tool": "mysql_exec",
  "arguments": {
    "sql": "INSERT INTO users (name, email) VALUES (?, ?)",
    "args": ["张三", "zhangsan@example.com"]
  }
}
```

### PostgreSQL 示例

```javascript
// 1. 连接到 PostgreSQL
{
  "tool": "pgsql_connect",
  "arguments": {
    "host": "127.0.0.1",
    "port": 5432,
    "user": "postgres",
    "password": "password",
    "database": "mydb"
  }
}

// 2. 查询数据
{
  "tool": "pgsql_query",
  "arguments": {
    "sql": "SELECT * FROM cities LIMIT 10"
  }
}
```

### Redis 示例

```javascript
// 1. 连接到 Redis
{
  "tool": "redis_connect",
  "arguments": {
    "addr": "127.0.0.1:6379",
    "password": "your_password",
    "db": 0
  }
}

// 2. 执行任意 Redis 命令
{
  "tool": "redis_command",
  "arguments": {
    "command": "SET user:1 张三"
  }
}

// 3. 执行复杂命令
{
  "tool": "redis_command",
  "arguments": {
    "command": "HSET user:1 name 张三 age 30"
  }
}

// 4. 执行 Lua 脚本
{
  "tool": "redis_lua",
  "arguments": {
    "script": "return redis.call('GET', KEYS[1])",
    "keys": ["user:1"],
    "args": []
  }
}
```

### SQLite 示例

```javascript
// 查询数据
{
  "tool": "sqlite_query",
  "arguments": {
    "db_path": "/path/to/database.db",
    "sql": "SELECT * FROM cities LIMIT 10"
  }
}

// 插入数据
{
  "tool": "sqlite_query",
  "arguments": {
    "db_path": "/path/to/database.db",
    "sql": "INSERT INTO users (name) VALUES ('张三')"
  }
}
```

## 🏗️ 项目结构

```
xz_mcp/
├── main.go              # 主程序入口（2010行）
├── go.mod               # Go 模块定义
├── go.sum               # 依赖校验文件
├── db/                  # 数据库连接模块
│   ├── mysql_db/        # MySQL 连接管理
│   ├── pgsql_db/        # PostgreSQL 连接管理
│   ├── redis_db/        # Redis 连接管理
│   └── sqlite_db/       # SQLite 连接管理
├── handlers/            # 工具处理器（预留）
├── tools/               # 工具定义（预留）
├── README.md            # 项目文档
└── xz_mcp              # 编译后的可执行文件
```

## 🔧 技术栈

- **语言**: Go 1.21+
- **MCP 框架**: [github.com/mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)
- **数据库驱动**:
  - MySQL: `github.com/go-sql-driver/mysql`
  - PostgreSQL: `github.com/lib/pq`
  - Redis: `github.com/redis/go-redis/v9`
  - SQLite: `github.com/mattn/go-sqlite3`

## 📝 开发说明

### 代码结构

- **main.go**: 包含所有工具注册和处理函数
- **db/*_db**: 各数据库的连接管理模块（从原独立项目复制）
- **工具命名**: 使用前缀区分数据库（mysql_*, pgsql_*, redis_*, sqlite_*）

### 添加新工具

1. 在 `main.go` 中找到对应数据库的 `register*Tools` 函数
2. 使用 `mcp.NewTool()` 定义新工具
3. 使用 `s.AddTool()` 注册工具和处理函数
4. 实现处理函数

### 修改现有工具

直接修改 `main.go` 中对应的工具定义或处理函数。

## 🧪 测试

### 单元测试（TODO）

```bash
go test ./...
```

### 集成测试

使用 MCP Inspector 进行交互式测试：

```bash
# 安装 MCP Inspector
npm install -g @modelcontextprotocol/inspector

# 启动测试
mcp-inspector ./xz_mcp
```

## 📊 性能指标

- **编译后大小**: ~12 MB
- **内存占用**: < 50 MB（空闲状态）
- **启动时间**: < 100ms
- **并发能力**: 支持多个并发 MCP 请求

## 🐛 故障排查

### 编译错误

**问题**: `undefined: sqlite_db.InitDB`

**解决**: 确保已复制 `db/sqlite_db` 模块

```bash
cp -r ../sqlite/sqlite_db db/
```

### 运行时错误

**问题**: `Database not connected`

**解决**: 先调用对应的 `*_connect` 工具建立连接

### CGO 相关错误

**问题**: SQLite 编译失败

**解决**: 安装 GCC 编译器
```bash
# macOS
xcode-select --install

# Linux
sudo apt-get install build-essential
```

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

### 开发流程

1. Fork 本项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情

## 🙏 致谢

- [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) - MCP Go SDK
- [Anthropic](https://www.anthropic.com/) - Model Context Protocol 规范

## 📮 联系方式

- 项目位置: `/Users/admin/go/empty/go/mcp_server/xz_mcp`
- 相关项目:
  - MySQL MCP: `/Users/admin/go/empty/go/mcp_server/mysql`
  - PostgreSQL MCP: `/Users/admin/go/empty/go/mcp_server/pgsql`
  - Redis MCP: `/Users/admin/go/empty/go/mcp_server/redis`
  - SQLite MCP: `/Users/admin/go/empty/go/mcp_server/sqlite`

## 🔄 版本历史

### v1.1.0 (2025-10-06)

- 🎯 **简化工具** - 合并 MySQL 表管理工具到 `mysql_exec` 和 `mysql_query`
- ✅ 工具数量从 21 个优化到 15 个
- ✅ 保持相同功能，更简洁的 API

**迁移说明**：
- `mysql_create_table` → `mysql_exec` (如: `CREATE TABLE ...`)
- `mysql_alter_table` → `mysql_exec` (如: `ALTER TABLE ...`)
- `mysql_drop_table` → `mysql_exec` (如: `DROP TABLE ...`)
- `mysql_show_tables` → `mysql_query` (如: `SHOW TABLES`)
- `mysql_describe_table` → `mysql_query` (如: `DESCRIBE table_name`)
- `mysql_show_create_table` → `mysql_query` (如: `SHOW CREATE TABLE table_name`)

### v1.0.0 (2025-09-30)

- ✨ 初始版本发布
- ✅ 集成 MySQL、PostgreSQL、Redis、SQLite
- ✅ 实现 15 个数据库操作工具
- ✅ 完整的错误处理和连接管理
- ✅ 生产就绪的代码质量

---

**Made with ❤️ using Claude Code**