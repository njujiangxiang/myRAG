#!/bin/bash
# BGE Rerank 测试脚本

set -e

BASE_URL="${BGE_URL:-http://localhost:8800}"

echo "==================================="
echo "BGE Rerank Service 测试"
echo "==================================="
echo ""

# 健康检查
echo "1. 健康检查..."
HEALTH_RESPONSE=$(curl -s "$BASE_URL/health")
echo "响应：$HEALTH_RESPONSE"
echo ""

# Rerank 测试
echo "2. Rerank 测试..."
RERANK_RESPONSE=$(curl -s -X POST "$BASE_URL/rerank" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "人工智能发展",
    "documents": [
      "机器学习是人工智能的一个重要分支",
      "今天天气很好，适合出去玩",
      "深度学习在图像识别领域取得了巨大成功",
      "我喜欢吃苹果和香蕉"
    ],
    "top_n": 2
  }')

echo "请求:"
echo '  query: "人工智能发展"'
echo '  documents: ["机器学习是人工智能...", "今天天气很好...", "深度学习在图像识别...", "我喜欢吃苹果..."]'
echo '  top_n: 2'
echo ""
echo "响应:"
echo "$RERANK_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$RERANK_RESPONSE"
echo ""

# 解析结果
echo "3. 结果分析:"
echo "$RERANK_RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
print(f'返回结果数：{len(data.get(\"results\", []))}')
for i, r in enumerate(data.get('results', [])):
    print(f'  #{i+1}: index={r[\"index\"]}, score={r[\"score\"]:.3f}, text={r.get(\"text\", \"\")[:50]}...')
" 2>/dev/null || echo "（无法解析 JSON）"

echo ""
echo "==================================="
echo "测试完成"
echo "==================================="
