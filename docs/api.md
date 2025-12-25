# 接口文档 (API Documentation)

本系统提供基于 HTTP 的推荐服务。所有的 API 请求都需要通过 Bearer Token 进行鉴权。

## 基础信息

- **Base URL**: `http://localhost:8080/api/v1`
- **Content-Type**: `application/json`

## 鉴权 (Authentication)

系统使用 **Bearer Token** 机制。Token 定义在 `configs/users.yaml` 文件的 `token` 字段中。

**Header:**
```http
Authorization: Bearer <your_token>
```

示例: `Authorization: Bearer sk-token-alice`

---

## 推荐接口 (Recommendation)

获取个性化的推荐结果。

**Endpoint:**
`POST /recommend/:scene`

### 路径参数 (Path Parameters)

| 参数名 | 类型 | 必选 | 描述 |
| :--- | :--- | :--- | :--- |
| `scene` | string | 是 | 推荐场景标识（如 `music`, `movie`），对应 `pipelines.json` 中的配置键。 |

### 查询参数 (Query Parameters)

| 参数名 | 类型 | 必选 | 描述 |
| :--- | :--- | :--- | :--- |
| `async`| boolean | 否 | 设置为 `true` 时，启用异步模式。服务器将立即返回一个任务ID，并开始在后台处理推荐请求。如果省略或为 `false`，则为同步模式。|

### 请求体 (Request Body)

| 参数名 | 类型 | 必选 | 描述 |
| :--- | :--- | :--- | :--- |
| `favorites` | []string | 是 | 用户的收藏列表，作为推荐的种子数据。 |

### 请求示例

#### 同步请求 (Synchronous Request)

```bash
curl -X POST -H "Authorization: Bearer sk-token-alice" \
     -H "Content-Type: application/json" \
     -d '{
           "favorites": ["Bohemian Rhapsody", "Hotel California"]
         }' \
     "http://localhost:8080/api/v1/recommend/music"
```

#### 异步请求 (Asynchronous Request)

```bash
curl -X POST -H "Authorization: Bearer sk-token-alice" \
     -H "Content-Type: application/json" \
     -d '{
           "favorites": ["写给黄淮", "可能否", "童话镇"]
         }' \
     "http://localhost:8080/api/v1/recommend/music?async=true"
```

### 响应结构 (Response)

#### 同步响应 (Synchronous Response)
请求成功后，立即返回推荐结果。
```json
{
  "scene": "music",
  "items": [
    {
      "id": "歌曲名称",
      "name": "歌曲名称",
      "score": 0,
      "source": "召回源名称 (e.g., xinhuo_recall_1)",
      "meta_data": null
    },
    ...
  ]
}
```

#### 异步响应 (Asynchronous Response)
请求成功后，立即返回 `202 Accepted` 和一个任务 ID。
```json
{
  "task_id": "9b8a3e1f-6a7c-4b5d-9f3a-3e1d9c2b8a7d"
}
```
后续需要通过 [获取异步任务结果](#获取异步任务结果-get-task-result) 接口查询最终结果。


### 错误响应

**400 Bad Request**
```json
{
  "error": "invalid request body: ..."
}
```

**401 Unauthorized**
```json
{
  "error": "invalid token"
}
```

**404 Not Found**
```json
{
  "error": "scene 'xxx' not supported"
}
```

**500 Internal Server Error**
```json
{
  "error": "recommendation failed: <detail>"
}
```

---

## 获取异步任务结果 (Get Task Result)

根据任务 ID 查询异步推荐任务的状态和结果。

**Endpoint:**
`GET /recommend/result/:task_id`

### 路径参数 (Path Parameters)

| 参数名 | 类型 | 必选 | 描述 |
| :--- | :--- | :--- | :--- |
| `task_id` | string | 是 | 异步请求返回的任务 ID。 |

### 请求示例

```bash
curl -X GET -H "Authorization: Bearer sk-token-alice" \
     "http://localhost:8080/api/v1/recommend/result/9b8a3e1f-6a7c-4b5d-9f3a-3e1d9c2b8a7d"
```

### 响应结构 (Response)

响应结构根据任务的当前状态而变化。

**状态: `pending` 或 `processing`**
任务正在等待处理或正在处理中。
```json
{
  "status": "processing"
}
```

**状态: `completed`**
任务成功完成，`data` 字段包含完整的推荐结果。
```json
{
  "status": "completed",
  "data": {
    "scene": "music",
    "items": [
      {
        "id": "歌曲A",
        "name": "歌曲A",
        "score": 0.9,
        "source": "llm_recall",
        "meta_data": null
      }
    ]
  }
}
```

**状态: `failed`**
任务处理失败，`error` 字段包含失败原因。
```json
{
  "status": "failed",
  "error": "recommendation failed: some internal error"
}
```

### 错误响应

**404 Not Found**
如果 `task_id` 不存在。
```json
{
  "error": "task not found"
}
```

---

## 配置说明

### 1. 用户配置 (`configs/users.yaml`)
管理用户信息和 Token。**注意：收藏列表现在通过 API 请求传递，不再配置于此。**

```yaml
users:
  - id: "user_001"
    token: "sk-token-alice"
    name: "Alice"
```
