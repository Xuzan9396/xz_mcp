# @xuzan/xz-mcp

XZ MCP - 统一数据库 MCP 服务器（MySQL, PostgreSQL, Redis, SQLite）

## 快速开始

### 使用 npx（推荐）

```bash
npx -y @xuzan/xz-mcp
```

### 安装到项目

```bash
npm install @xuzan/xz-mcp
```

## 配置到 Claude Desktop

编辑配置文件：
- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

添加配置：

```json
{
  "mcpServers": {
    "xz-mcp": {
      "command": "npx",
      "args": ["-y", "@xuzan/xz-mcp"]
    }
  }
}
```

重启 Claude Desktop 即可。

## 集成的数据库

- ✅ **MySQL** - 14个工具（连接、查询、表管理、存储过程）
- ✅ **PostgreSQL** - 3个工具（连接、查询、DML操作）
- ✅ **Redis** - 3个工具（连接、命令执行、Lua脚本）
- ✅ **SQLite** - 1个工具（统一查询接口）
- ✅ **MongoDB** - 4个工具（动态连接、查询、聚合、原生命令全权限）

## 使用方法

在 Claude Desktop 中：

```
连接到我的 MySQL 数据库
查询 Redis 中所有以 user: 开头的键
执行 PostgreSQL 查询
```

## 支持平台

- macOS (Intel / Apple Silicon)
- Linux (x64)
- Windows (x64 / ARM64)

## 更多信息

- 📖 [完整文档](https://github.com/Xuzan9396/xz_mcp)
- 🐛 [问题反馈](https://github.com/Xuzan9396/xz_mcp/issues)
- 📝 [更新日志](https://github.com/Xuzan9396/xz_mcp/releases)

## License

MIT
