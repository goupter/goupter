#!/bin/bash
# Goupter 安装脚本
# 用法: curl -sSL https://raw.githubusercontent.com/goupter/goupter/main/install.sh | bash

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 项目信息
REPO="goupter/goupter"
BINARY_NAME="goupter"

# 打印信息
info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# 检测操作系统
detect_os() {
    case "$(uname -s)" in
        Linux*)     echo "linux";;
        Darwin*)    echo "darwin";;
        MINGW*|MSYS*|CYGWIN*) echo "windows";;
        *)          error "不支持的操作系统: $(uname -s)";;
    esac
}

# 检测架构
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)   echo "amd64";;
        aarch64|arm64)  echo "arm64";;
        armv7l)         echo "arm";;
        *)              error "不支持的架构: $(uname -m)";;
    esac
}

# 获取最新版本
get_latest_version() {
    if command -v curl &> /dev/null; then
        curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    elif command -v wget &> /dev/null; then
        wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    else
        error "需要 curl 或 wget 来下载文件"
    fi
}

# 下载文件
download_file() {
    local url=$1
    local output=$2
    
    if command -v curl &> /dev/null; then
        curl -sSL -o "$output" "$url"
    elif command -v wget &> /dev/null; then
        wget -qO "$output" "$url"
    else
        error "需要 curl 或 wget 来下载文件"
    fi
}

# 主安装流程
main() {
    info "开始安装 Goupter CLI..."
    
    # 检测系统信息
    OS=$(detect_os)
    ARCH=$(detect_arch)
    info "检测到系统: ${OS}_${ARCH}"
    
    # 获取最新版本
    info "获取最新版本..."
    VERSION=$(get_latest_version)
    if [ -z "$VERSION" ]; then
        error "无法获取最新版本信息"
    fi
    info "最新版本: ${VERSION}"
    
    # 构建下载 URL
    if [ "$OS" = "windows" ]; then
        ARCHIVE_EXT="zip"
    else
        ARCHIVE_EXT="tar.gz"
    fi
    
    ARCHIVE_NAME="${BINARY_NAME}_${VERSION#v}_${OS}_${ARCH}.${ARCHIVE_EXT}"
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE_NAME}"
    
    info "下载地址: ${DOWNLOAD_URL}"
    
    # 创建临时目录
    TMP_DIR=$(mktemp -d)
    trap "rm -rf ${TMP_DIR}" EXIT
    
    # 下载归档文件
    info "正在下载..."
    ARCHIVE_PATH="${TMP_DIR}/${ARCHIVE_NAME}"
    download_file "$DOWNLOAD_URL" "$ARCHIVE_PATH"
    
    # 解压文件
    info "正在解压..."
    cd "$TMP_DIR"
    if [ "$ARCHIVE_EXT" = "zip" ]; then
        unzip -q "$ARCHIVE_PATH"
    else
        tar -xzf "$ARCHIVE_PATH"
    fi
    
    # 确定安装目录
    if [ -w "/usr/local/bin" ]; then
        INSTALL_DIR="/usr/local/bin"
    elif [ -w "$HOME/.local/bin" ]; then
        INSTALL_DIR="$HOME/.local/bin"
        mkdir -p "$INSTALL_DIR"
    else
        INSTALL_DIR="$HOME/bin"
        mkdir -p "$INSTALL_DIR"
    fi
    
    # 安装二进制文件
    info "安装到 ${INSTALL_DIR}..."
    if [ "$OS" = "windows" ]; then
        BINARY_FILE="${BINARY_NAME}.exe"
    else
        BINARY_FILE="${BINARY_NAME}"
    fi
    
    if [ ! -f "$BINARY_FILE" ]; then
        error "未找到二进制文件: $BINARY_FILE"
    fi
    
    # 移动并设置权限
    mv "$BINARY_FILE" "${INSTALL_DIR}/"
    chmod +x "${INSTALL_DIR}/${BINARY_FILE}"
    
    # 验证安装
    info "验证安装..."
    if command -v "$BINARY_NAME" &> /dev/null; then
        info "✅ 安装成功！"
        echo ""
        "$BINARY_NAME" --version
        echo ""
        info "运行 '${BINARY_NAME} --help' 查看使用帮助"
    else
        warn "⚠️  安装完成，但 ${BINARY_NAME} 未在 PATH 中"
        warn "请将 ${INSTALL_DIR} 添加到 PATH 环境变量"
        echo ""
        echo "添加以下内容到你的 shell 配置文件 (~/.bashrc, ~/.zshrc 等):"
        echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    fi
}

# 运行主函数
main
