#!/bin/bash

# XZ MCP 自动安装脚本
# 用法: curl -fsSL https://raw.githubusercontent.com/yourname/xz_mcp/main/install.sh | bash

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# 配置
REPO="Xuzan9396/xz_mcp"
INSTALL_DIR="${HOME}/go/bin"
BINARY_NAME="xz_mcp"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  XZ MCP 自动安装脚本${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# 检测操作系统和架构
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# 检查是否是 Windows (Git Bash/WSL)
if [[ "$OS" == *"mingw"* ]] || [[ "$OS" == *"msys"* ]] || [[ "$OS" == *"cygwin"* ]]; then
    echo -e "${YELLOW}检测到 Windows 系统${NC}"
    echo ""
    echo "请使用 PowerShell 运行以下命令："
    echo ""
    echo "  Invoke-WebRequest -Uri \"https://github.com/Xuzan9396/xz_mcp/releases/latest/download/xz_mcp_windows_amd64.exe\" -OutFile \"xz_mcp.exe\""
    echo ""
    echo "然后移动到合适的位置，例如："
    echo "  Move-Item xz_mcp.exe C:\Users\\\$env:USERNAME\go\bin\xz_mcp.exe"
    echo ""
    echo "配置 Codex (~/.codex/config.toml):"
    echo "  [mcp_servers.xz_mcp]"
    echo "  command = \"C:\\\\Users\\\\YourUsername\\\\go\\\\bin\\\\xz_mcp.exe\""
    echo ""
    exit 0
fi

case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}不支持的架构: $ARCH${NC}"
        exit 1
        ;;
esac

echo -e "${YELLOW}检测到系统:${NC} $OS/$ARCH"

# 获取最新版本
echo -e "${YELLOW}获取最新版本...${NC}"
LATEST_VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
    echo -e "${RED}无法获取最新版本，请检查仓库配置${NC}"
    exit 1
fi

echo -e "${GREEN}最新版本:${NC} $LATEST_VERSION"

# 下载 URL
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_VERSION/${BINARY_NAME}_${OS}_${ARCH}"

echo -e "${YELLOW}下载中...${NC}"
echo "URL: $DOWNLOAD_URL"

# 创建安装目录
mkdir -p "$INSTALL_DIR"

# 下载二进制文件
if curl -L -o "$INSTALL_DIR/$BINARY_NAME" "$DOWNLOAD_URL"; then
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
    echo -e "${GREEN}✅ 安装成功！${NC}"
    echo ""
    echo "安装位置: $INSTALL_DIR/$BINARY_NAME"

    # 验证安装
    if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
        "$INSTALL_DIR/$BINARY_NAME" --version
    fi

    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}  配置建议${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo "1. 添加到 Codex (~/.codex/config.toml):"
    echo ""
    echo "   [mcp_servers.xz_mcp]"
    echo "   command = \"$INSTALL_DIR/$BINARY_NAME\""
    echo ""
    echo "2. 添加到 Claude Desktop:"
    echo ""
    echo "   claude mcp add-json xz_mcp -s user '{\"type\":\"stdio\",\"command\":\"$INSTALL_DIR/$BINARY_NAME\",\"args\":[],\"env\":{}}'"
    echo ""
    echo "3. 添加到 PATH (如需全局访问):"
    echo ""
    echo "   export PATH=\"$INSTALL_DIR:\$PATH\""
    echo ""
else
    echo -e "${RED}❌ 下载失败！${NC}"
    exit 1
fi