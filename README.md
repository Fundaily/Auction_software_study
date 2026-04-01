# 拍卖系统 API 文档

这是一个基于 Go 语言和 Gin 框架构建的拍卖系统 API，提供完整的拍卖流程管理功能。

## 项目概述

- **框架**: Gin Web Framework
- **数据库**: MySQL (通过 GORM ORM)
- **认证**: JWT Token
- **实时通信**: WebSocket

## 快速开始

### 环境要求
- Go 1.22+
- MySQL 5.7+

### 安装和运行

1. 克隆项目并进入目录
2. 配置数据库连接信息
   ```bash
   cp config.yaml.example config.yaml
   # 编辑 config.yaml 文件，设置数据库连接信息
   ```

3. 安装依赖
   ```bash
   go mod tidy
   ```

4. 运行服务
   ```bash
   go run ./cmd/server
   ```

服务默认运行在 `http://localhost:8080`

## API 接口文档

### 认证接口

#### 用户注册
- **URL**: `POST /api/register`
- **认证**: 无需认证
- **请求格式**:
  ```json
  {
    "username": "string (2-64字符)",
    "password": "string (6-128字符)",
    "display_name": "string (可选)"
  }
  ```
- **响应格式**:
  ```json
  {
    "id": "用户ID",
    "username": "用户名"
  }
  ```

#### 用户登录
- **URL**: `POST /api/login`
- **认证**: 无需认证
- **请求格式**:
  ```json
  {
    "username": "string",
    "password": "string"
  }
  ```
- **响应格式**:
  ```json
  {
    "token": "JWT Token",
    "user_id": "用户ID",
    "username": "用户名",
    "is_admin": "是否管理员",
    "expires_in": "过期时间(秒)"
  }
  ```

### 拍卖相关接口

#### 获取拍卖列表
- **URL**: `GET /api/auctions`
- **认证**: 无需认证
- **查询参数**:
  - `status`: 过滤状态 (scheduled, active, ended, settled, cancelled)
- **响应格式**: 拍卖对象数组

#### 获取单个拍卖详情
- **URL**: `GET /api/auctions/:id`
- **认证**: 无需认证
- **响应格式**: 包含商品信息的完整拍卖对象

#### 获取拍卖出价记录
- **URL**: `GET /api/auctions/:id/bids`
- **认证**: 无需认证
- **响应格式**: 出价记录数组

#### WebSocket 实时竞价
- **URL**: `GET /api/auctions/:id/ws`
- **认证**: 无需认证
- **协议**: WebSocket
- **消息格式**: 实时竞价和状态更新

### 商品管理接口 (需要认证)

#### 创建商品
- **URL**: `POST /api/items`
- **认证**: JWT Token
- **请求格式**:
  ```json
  {
    "title": "商品标题",
    "description": "商品描述",
    "category": "分类"
  }
  ```

#### 获取我的商品列表
- **URL**: `GET /api/me/items`
- **认证**: JWT Token
- **响应格式**: 用户拥有的商品数组

#### 上传商品图片
- **URL**: `POST /api/items/:id/images`
- **认证**: JWT Token
- **请求类型**: `multipart/form-data`
- **参数**: `file` - 图片文件

### 竞价接口 (需要认证)

#### 提交出价
- **URL**: `POST /api/auctions/:id/bids`
- **认证**: JWT Token
- **请求格式**:
  ```json
  {
    "amount_cents": "出价金额(分)"
  }
  ```
- **防刷机制**: 同一用户在同一拍卖中，出价间隔至少800ms，突发限制5次

### 支付接口 (需要认证)

#### 创建支付订单
- **URL**: `POST /api/auctions/:id/payments`
- **认证**: JWT Token
- **响应格式**: 支付订单信息

#### 获取支付详情
- **URL**: `GET /api/payments/:id`
- **认证**: JWT Token

#### 确认支付
- **URL**: `POST /api/payments/:id/confirm`
- **认证**: JWT Token

### 评价接口 (需要认证)

#### 创建评价
- **URL**: `POST /api/reviews`
- **认证**: JWT Token
- **请求格式**:
  ```json
  {
    "target_user_id": "被评价用户ID",
    "rating": "评分(1-5)",
    "comment": "评价内容"
  }
  ```

#### 获取用户评价
- **URL**: `GET /api/users/:id/reviews`
- **认证**: 无需认证

### 管理员接口

#### 创建拍卖 (管理员)
- **URL**: `POST /api/admin/auctions`
- **认证**: JWT Token + 管理员权限
- **请求格式**:
  ```json
  {
    "item_id": "商品ID",
    "rules_text": "拍卖规则",
    "start_at": "开始时间(RFC3339)",
    "end_at": "结束时间(RFC3339)",
    "starting_price_cents": "起拍价(分)",
    "min_increment_cents": "最小加价(分)",
    "extend_seconds": "自动延时秒数",
    "extend_threshold_sec": "延时触发阈值(秒)",
    "initial_status": "初始状态"
  }
  ```

#### 更新拍卖 (管理员)
- **URL**: `PATCH /api/admin/auctions/:id`
- **认证**: JWT Token + 管理员权限

#### 取消拍卖 (管理员)
- **URL**: `POST /api/admin/auctions/:id/cancel`
- **认证**: JWT Token + 管理员权限

#### 商品审核 (管理员)
- **URL**: `GET /api/admin/items/pending` - 获取待审核商品列表
- **URL**: `POST /api/admin/items/:id/review` - 审核商品

### 统计接口

#### 拍卖统计
- **URL**: `GET /api/auctions/:id/stats`
- **认证**: 无需认证

#### 全局统计 (管理员)
- **URL**: `GET /api/admin/stats/summary`
- **认证**: JWT Token + 管理员权限

#### 导出拍卖报告 (管理员)
- **URL**: `GET /api/admin/auctions/:id/export.csv`
- **认证**: JWT Token + 管理员权限
- **响应格式**: CSV 文件

### Webhook 接口

#### 支付回调
- **URL**: `POST /api/webhooks/payment`
- **认证**: 可选 (通过 X-Webhook-Secret 头)
- **用途**: 接收第三方支付平台回调

## 数据格式说明

### 时间格式
- 所有时间字段使用 RFC3339 格式 (例如: `2024-01-01T10:00:00Z`)

### 金额格式
- 所有金额字段以分为单位 (整数)
- 例如: 100元 = 10000分

### 拍卖状态
- `scheduled`: 已安排，未开始
- `active`: 进行中
- `ended`: 已结束
- `settled`: 已结算
- `cancelled`: 已取消

### 商品状态
- `pending`: 待审核
- `approved`: 已审核通过
- `rejected`: 已拒绝

## 认证机制

### JWT Token 使用
在需要认证的接口中，需要在请求头中添加：
```
Authorization: Bearer {token}
```

### 管理员权限
部分接口需要管理员权限，用户需要设置 `is_admin=true` 才能访问。

## 配置说明

### 配置文件 (config.yaml)
```yaml
addr: ":8080"                    # 服务端口
mysql_dsn: "数据库连接字符串"       # MySQL 连接信息
jwt_secret: "JWT密钥"            # JWT 签名密钥
upload_dir: "./uploads"          # 文件上传目录
static_url: "/static/uploads"    # 静态文件访问URL
admin_user: "admin"              # 管理员用户名
admin_pass: "admin123"           # 管理员密码
bid_rate_burst: 5                # 出价突发限制
bid_rate_every_ms: 800           # 出价最小间隔(毫秒)
```

## 错误处理

所有 API 接口返回标准化的错误响应：
```json
{
  "error": "错误描述信息"
}
```

### 常见 HTTP 状态码
- `200`: 成功
- `400`: 请求参数错误
- `401`: 未认证
- `403`: 权限不足
- `404`: 资源不存在
- `409`: 资源冲突
- `500`: 服务器内部错误

## 开发说明

### 项目结构
```
cmd/server/          # 服务入口
internal/
  auth/              # 认证相关
  config/            # 配置管理
  database/          # 数据库操作
  handler/           # HTTP 处理器
  middleware/        # 中间件
  models/            # 数据模型
  service/           # 业务逻辑
  ws/                # WebSocket 处理
```

### 数据库迁移
项目使用 GORM AutoMigrate 自动创建数据库表结构。

### 测试
运行测试：
```bash
go test ./...
```

## 部署说明

### 生产环境部署
1. 设置生产环境配置文件
2. 构建可执行文件：`go build -o auction ./cmd/server`
3. 使用进程管理工具 (如 systemd, supervisor) 运行服务

### 环境变量
支持通过环境变量覆盖配置：
- `AUCTION_ADDR`: 服务端口
- `AUCTION_MYSQL_DSN`: 数据库连接字符串
- `AUCTION_JWT_SECRET`: JWT 密钥

## 技术支持

如有问题或建议，请联系开发团队。