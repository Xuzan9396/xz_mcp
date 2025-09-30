#!/bin/bash

# XZ MCP 编译和安装脚本
# 用法: ./build.sh

set -e  # 遇到错误立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  XZ MCP 编译脚本${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# 获取脚本所在目录
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo -e "${YELLOW}[1/5]${NC} 清理旧文件..."
rm -f xz_mcp

echo -e "${YELLOW}[2/5]${NC} 下载依赖..."
go mod tidy

echo -e "${YELLOW}[3/5]${NC} 编译项目..."
go build -ldflags "-s -w" -o xz_mcp main.go

echo -e "${YELLOW}[4/5]${NC} 设置执行权限..."
chmod +x xz_mcp

echo -e "${YELLOW}[5/5]${NC} 复制到系统路径..."
cp -f xz_mcp /Users/admin/go/bin/

echo ""
echo -e "${GREEN}✅ 编译完成！${NC}"
echo ""
echo "可执行文件位置:"
echo "  - 本地: $SCRIPT_DIR/xz_mcp"
echo "  - 系统: /Users/admin/go/bin/xz_mcp"
echo ""

# 显示版本信息
echo "版本信息:"
./xz_mcp --version

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  使用方法${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "1. 直接运行:"
echo "   ./xz_mcp"
echo ""
echo "2. 从任意位置运行:"
echo "   xz_mcp"
echo ""
echo "3. Claude Desktop 配置:"
echo "   claude mcp add-json xz_mcp -s user '{\"type\":\"stdio\",\"command\":\"/Users/admin/go/bin/xz_mcp\",\"args\":[],\"env\":{}}'"
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  配置建议${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "如需在其他工具中使用，可添加以下配置："
echo ""
echo "1. Codex 配置文件 (~/.codex/config.toml):"
echo ""
echo "   [mcp_servers.xz_mcp]"
echo "   command = \"/Users/admin/go/bin/xz_mcp\""
echo ""
echo "2. 添加到 PATH（可选）:"
echo "   export PATH=\"/Users/admin/go/bin:\$PATH\""
echo ""
echo "3. 验证安装:"
echo "   xz_mcp --version"
echo ""
echo "4. 查看工具列表（通过 MCP Inspector）:"
echo "   npm install -g @modelcontextprotocol/inspector"
echo "   mcp-inspector /Users/admin/go/bin/xz_mcp"
echo ""