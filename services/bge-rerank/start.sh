#!/bin/bash
# BGE Rerank Service 启动脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "==================================="
echo "BGE Rerank Service 启动脚本"
echo "==================================="

# 检查 Python
if ! command -v python3 &> /dev/null; then
    echo "错误：未找到 Python3，请先安装 Python 3.10+"
    exit 1
fi

echo "Python 版本：$(python3 --version)"

# 检查依赖
echo ""
echo "检查依赖..."
if ! python3 -c "import fastapi" 2>/dev/null; then
    echo "安装依赖..."
    pip3 install -r requirements.txt
fi

# 检查 GPU (可选)
echo ""
if command -v nvidia-smi &> /dev/null; then
    echo "检测到 NVIDIA GPU:"
    nvidia-smi --query-gpu=name,memory.total --format=csv,noheader | head -1
    export BGE_DEVICE=cuda
else
    echo "未检测到 NVIDIA GPU，使用 CPU 模式"
    export BGE_DEVICE=cpu
fi

# 设置环境变量
export BGE_MODEL=${BGE_MODEL:-"BAAI/bge-reranker-v2-m3"}
export BGE_HOST=${BGE_HOST:-"0.0.0.0"}
export BGE_PORT=${BGE_PORT:-"8800"}

echo ""
echo "配置:"
echo "  模型：$BGE_MODEL"
echo "  设备：$BGE_DEVICE"
echo "  地址：http://$BGE_HOST:$BGE_PORT"
echo ""

# 启动服务
echo "启动 BGE Rerank 服务..."
echo "按 Ctrl+C 停止服务"
echo ""

python3 main.py
