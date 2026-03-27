#!/bin/bash
set -e

echo "=== myRAG Web 前端安装 ==="

cd "$(dirname "$0")"

# 检查 Node.js
if ! command -v node &> /dev/null; then
    echo "错误：需要安装 Node.js"
    echo "请访问 https://nodejs.org/ 下载安装"
    exit 1
fi

echo "Node.js 版本：$(node --version)"

# 安装依赖
echo "安装依赖..."
npm install

echo ""
echo "=== 安装完成 ==="
echo ""
echo "开发模式运行："
echo "  npm run dev"
echo ""
echo "生产构建："
echo "  npm run build"
echo ""
