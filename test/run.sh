#!/bin/bash

# 设置颜色
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo "Starting Recommendation Engine Integration Test..."

# 1. 在后台启动服务
echo "Step 1: Starting server..."
go run ./cmd/recommend > server.log 2>&1 &
SERVER_PID=$!
echo "Server PID: $SERVER_PID"

# 确保脚本退出时清理服务进程
cleanup() {
    echo "Stopping server..."
    kill $SERVER_PID
    rm -f server.log
}
trap cleanup EXIT

# 等待服务启动
echo "Waiting for server to launch..."
for i in {1..10}; do
    if grep -q "Listening and serving HTTP" server.log; then
        echo -e "${GREEN}Server is up!${NC}"
        break
    fi
    sleep 1
done

# 2. 发送测试请求 (POST /api/v1/recommend/music)
echo "Step 2: Sending recommendation request..."
RESPONSE=$(curl -s -X POST \
  -H "Authorization: Bearer sk-token-alice" \
  -H "Content-Type: application/json" \
  -d @test/payload.json \
  "http://localhost:8080/api/v1/recommend/music")

# 3. 验证结果
echo "Step 3: Verifying response..."
echo "Response: $RESPONSE"

# 简单验证：检查是否包含 "scene":"music" 和 "items"
if echo "$RESPONSE" | grep -q '"scene":"music"' && echo "$RESPONSE" | grep -q '"items":\['; then
    echo -e "${GREEN}TEST PASSED: Valid response received.${NC}"
    
    # 可选：检查 items 数量是否足够 (需要 jq)
    if command -v jq &> /dev/null; then
        COUNT=$(echo "$RESPONSE" | jq '.items | length')
        echo "Items count: $COUNT"
        if [ "$COUNT" -gt 0 ]; then
             echo -e "${GREEN}TEST PASSED: Returned $COUNT items.${NC}"
        else
             echo -e "${RED}TEST FAILED: Returned 0 items.${NC}"
             exit 1
        fi
    fi
else
    echo -e "${RED}TEST FAILED: Invalid response format.${NC}"
    cat server.log
    exit 1
fi

echo "All tests passed successfully."
exit 0
