#!/bin/bash

# ==================================================
# 全功能测试脚本 (同步 + 异步)
# ==================================================

# --- 配置 ---
PORT="8080"
TOKEN="sk-token-alice"
SCENE="music"
PAYLOAD_FILE="test/payload.json"
MAX_POLL_ATTEMPTS=12 # 最大轮询次数
POLL_INTERVAL=10   # 每次轮询间隔 (秒)

# --- 颜色定义 ---
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

# --- 检查依赖 ---
if ! command -v jq &> /dev/null;
then
    echo -e "${RED}Error: 'jq' is not installed. Please install it to run this test.${NC}"
    exit 1
fi
echo -e "${GREEN}jq is installed. Proceeding...${NC}"

# --- 启动和清理 ---
echo "Starting Recommendation Engine All-in-one Test..."

echo "Step 1: Starting server in background..."
go run ./cmd/recommend > server.log 2>&1 &
SERVER_PID=$!
echo "Server PID: $SERVER_PID"

cleanup() {
    echo -e "\n${YELLOW}Cleaning up...${NC}"
    echo "Stopping server (PID: $SERVER_PID)..."
    kill $SERVER_PID
    rm -f server.log
    echo "Cleanup complete."
}
trap cleanup EXIT

echo "Waiting for server to launch..."
for i in {1..10}; do
    if grep -q "Starting HTTP server on port" server.log; then
        echo -e "${GREEN}Server is up!${NC}"
        break
    fi
    sleep 1
    if [ $i -eq 10 ]; then
        echo -e "${RED}TEST FAILED: Server failed to start in time.${NC}"
        cat server.log
        exit 1
    fi
done

# ==================================================
# 测试一: 同步模式 (回归测试)
# ==================================================
echo -e "\n${YELLOW}--- TEST 1: Sync Mode (Regression Test) ---${NC}"
SYNC_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d @$PAYLOAD_FILE \
  "http://localhost:$PORT/api/v1/recommend/$SCENE")

SYNC_HTTP_CODE=$(echo "$SYNC_RESPONSE" | tail -n1)
SYNC_BODY=$(echo "$SYNC_RESPONSE" | sed '$d')

echo "Sync Response Body: $SYNC_BODY"
echo "Sync HTTP Code: $SYNC_HTTP_CODE"

if [ "$SYNC_HTTP_CODE" -ne 200 ]; then
    echo -e "${RED}TEST 1 FAILED: Expected HTTP 200, got $SYNC_HTTP_CODE.${NC}"
    exit 1
fi

if echo "$SYNC_BODY" | jq -e '.items | length > 0' > /dev/null; then
    echo -e "${GREEN}TEST 1 PASSED: Sync request successful with items returned.${NC}"
else
    echo -e "${RED}TEST 1 FAILED: Sync response is invalid or contains no items.${NC}"
    exit 1
fi

# ==================================================
# 测试二: 异步模式
# ==================================================
echo -e "\n${YELLOW}--- TEST 2: Async Mode ---${NC}"

# --- 2.1: 触发异步任务 ---
echo "Step 2.1: Triggering async task..."
ASYNC_INIT_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d @$PAYLOAD_FILE \
  "http://localhost:$PORT/api/v1/recommend/$SCENE?async=true")

ASYNC_INIT_HTTP_CODE=$(echo "$ASYNC_INIT_RESPONSE" | tail -n1)
ASYNC_INIT_BODY=$(echo "$ASYNC_INIT_RESPONSE" | sed '$d')

echo "Async Init Response Body: $ASYNC_INIT_BODY"
echo "Async Init HTTP Code: $ASYNC_INIT_HTTP_CODE"

if [ "$ASYNC_INIT_HTTP_CODE" -ne 202 ]; then
    echo -e "${RED}TEST 2.1 FAILED: Expected HTTP 202, got $ASYNC_INIT_HTTP_CODE.${NC}"
    exit 1
fi

TASK_ID=$(echo "$ASYNC_INIT_BODY" | jq -r .task_id)
if [ -z "$TASK_ID" ] || [ "$TASK_ID" == "null" ]; then
    echo -e "${RED}TEST 2.1 FAILED: task_id not found in response.${NC}"
    exit 1
fi
echo "Received Task ID: $TASK_ID"

# --- 2.2: 轮询任务结果 ---
echo "Step 2.2: Polling for result..."
for (( i=1; i<=MAX_POLL_ATTEMPTS; i++ )); do
    echo "Attempt $i/$MAX_POLL_ATTEMPTS: Polling task '$TASK_ID'..."
    POLL_RESPONSE=$(curl -s -H "Authorization: Bearer $TOKEN" "http://localhost:$PORT/api/v1/recommend/result/$TASK_ID")
    echo "Poll Response: $POLL_RESPONSE"

    STATUS=$(echo "$POLL_RESPONSE" | jq -r .status)
    echo "Current Status: $STATUS"

    if [ "$STATUS" == "completed" ]; then
        echo -e "${GREEN}TEST 2.2 PASSED: Task completed successfully.${NC}"
        # 验证结果数据
        if echo "$POLL_RESPONSE" | jq -e '.data.items | length > 0' > /dev/null; then
             echo -e "${GREEN}TEST 2.2 PASSED: Result data contains items.${NC}"
             break # 成功，跳出循环
        else
             echo -e "${RED}TEST 2.2 FAILED: Completed task but result data is invalid or empty.${NC}"
             exit 1
        fi
    elif [ "$STATUS" == "failed" ]; then
        echo -e "${RED}TEST 2.2 FAILED: Task failed during processing.${NC}"
        exit 1
    fi

    if [ $i -eq $MAX_POLL_ATTEMPTS ]; then
        echo -e "${RED}TEST 2.2 FAILED: Polling timed out after $MAX_POLL_ATTEMPTS attempts.${NC}"
        exit 1
    fi

    sleep $POLL_INTERVAL
done


# --- 总结 ---
echo -e "\n${GREEN}====== ALL TESTS PASSED SUCCESSFULLY ======${NC}"
exit 0
