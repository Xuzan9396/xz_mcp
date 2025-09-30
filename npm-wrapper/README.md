# @xuzan/xz-mcp

> Unified MCP server for MySQL, PostgreSQL, Redis, and SQLite databases

一个统一的 MCP 服务器，支持 MySQL、PostgreSQL、Redis 和 SQLite 数据库操作。

## 📦 安装

### 使用 npx (推荐)

无需安装，直接使用：

```bash
npx -y @xuzan/xz-mcp
```

### 全局安装

```bash
npm install -g @xuzan/xz-mcp
```

### 本地安装

```bash
npm install @xuzan/xz-mcp
```

## 🚀 使用方法

### 在 Claude Desktop 中使用

添加到 Claude Desktop 配置：

```bash
claude mcp add-json xz_mcp -s user '{"type":"stdio","command":"npx","args":["-y","@xuzan/xz-mcp"],"env":{}}'
claude mcp add-json test_mcp -s user '{"type":"stdio","command":"npx","args":["-y","@xuzan/xz-mcp"],"env":{}}'
```

或者手动编辑配置文件：

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "xz_mcp": {
      "command": "npx",
      "args": ["-y", "@xuzan/xz-mcp"]
    }
  }
}
```

### 在 Codex 中使用

编辑 `~/.codex/config.toml`：

```toml
[mcp_servers.xz_mcp]
command = "npx"
args = ["-y", "@xuzan/xz-mcp"]
```

## ✨ 功能特性

### 支持的数据库

- **MySQL** (15 个工具)
  - 连接、查询、执行、表管理、存储过程等

- **PostgreSQL** (3 个工具)
  - 连接、查询、执行

- **Redis** (13 个工具)
  - 字符串、哈希、列表、集合、有序集合
  - Lua 脚本、数据库管理等

- **SQLite** (1 个工具)
  - SQL 查询执行

### 所有工具列表

```
MySQL 工具:
- mysql_connect          - 连接到 MySQL 数据库
- mysql_query            - 执行 SELECT 查询
- mysql_exec             - 执行 INSERT/UPDATE/DELETE
- mysql_exec_get_id      - 执行 INSERT 并返回 ID
- mysql_create_table     - 创建表
- mysql_drop_table       - 删除表
- mysql_describe_table   - 查看表结构
- mysql_show_tables      - 列出所有表
- mysql_show_create_table- 查看建表语句
- mysql_create_procedure - 创建存储过程
- mysql_drop_procedure   - 删除存储过程
- mysql_call_procedure   - 调用存储过程
- mysql_show_procedures  - 列出所有存储过程

PostgreSQL 工具:
- pgsql_connect          - 连接到 PostgreSQL
- pgsql_query            - 执行 SELECT 查询
- pgsql_exec             - 执行 INSERT/UPDATE/DELETE

Redis 工具:
- redis_connect          - 连接到 Redis
- redis_disconnect       - 断开连接
- redis_ping             - 测试连接
- redis_string           - 字符串操作
- redis_hash             - 哈希操作
- redis_list             - 列表操作
- redis_set              - 集合操作
- redis_zset             - 有序集合操作
- redis_keys             - 获取键列表
- redis_del              - 删除键
- redis_expire           - 设置过期时间
- redis_lua              - 执行 Lua 脚本
- redis_db               - 数据库管理

SQLite 工具:
- sqlite_query           - 执行 SQL 查询
```

## 📋 系统要求

- **Node.js**: >= 14.0.0
- **操作系统**: macOS, Linux, Windows
- **架构**: x64, arm64

## 🔧 工作原理

1. **安装时**: `postinstall` 脚本自动从 GitHub Releases 下载对应平台的二进制文件
2. **运行时**: Node.js 包装器启动 Go 编译的 MCP 服务器
3. **通信**: 通过标准输入输出 (stdio) 与 Claude 通信

## 🛠️ 开发

查看源代码和贡献：

```bash
git clone https://github.com/Xuzan9396/xz_mcp.git
cd xz_mcp
go build
```

## 📖 更多文档

- [完整使用指南](https://github.com/Xuzan9396/xz_mcp/blob/main/USAGE.md)
- [部署指南](https://github.com/Xuzan9396/xz_mcp/blob/main/DEPLOYMENT.md)

## 📄 许可证

MIT License

## 🔗 链接

- [GitHub Repository](https://github.com/Xuzan9396/xz_mcp)
- [Issues](https://github.com/Xuzan9396/xz_mcp/issues)
- [Releases](https://github.com/Xuzan9396/xz_mcp/releases)

## 💡 提示

如果安装失败，可以使用传统安装方式：

```bash
curl -fsSL https://raw.githubusercontent.com/Xuzan9396/xz_mcp/main/install.sh | bash
```