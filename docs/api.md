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

### 请求体 (Request Body)

| 参数名 | 类型 | 必选 | 描述 |
| :--- | :--- | :--- | :--- |
| `favorites` | []string | 是 | 用户的收藏列表，作为推荐的种子数据。 |

### 请求示例

```bash
curl -X POST -H "Authorization: Bearer sk-token-alice" \
     -H "Content-Type: application/json" \
     -d '{
           "favorites": ["Bohemian Rhapsody", "Hotel California"]
         }' \
     "http://localhost:8080/api/v1/recommend/music"
```

### 响应结构 (Response)

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

## 配置说明

### 1. 用户配置 (`configs/users.yaml`)
管理用户信息和 Token。**注意：收藏列表现在通过 API 请求传递，不再配置于此。**

```yaml
users:
  - id: "user_001"
    token: "sk-token-alice"
    name: "Alice"
```
