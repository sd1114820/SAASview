# SAAS多租户时区处理解决方案

> 🌍 通过数据库视图优雅解决多租户时区混乱问题的完整示例

## 📋 项目概述

在SAAS多租户系统中，不同商户分布在全球各个时区，传统的时区处理方式往往导致：
- 查询逻辑复杂，每次都需要手动转换时区
- 数据分析师需要深入了解时区转换细节
- 容易出现时区转换错误，影响数据准确性
- 代码重复，维护成本高

本项目通过**数据库视图**的方式，将复杂的时区转换逻辑封装在数据层，为上层应用提供统一、简洁的数据接口。

## 🎯 核心价值

### ✅ 解决的问题
- **简化查询**：分析师无需关心复杂的时区转换逻辑
- **统一标准**：所有时区相关字段通过视图统一提供
- **数据一致性**：避免不同查询中时区转换的不一致
- **职责分离**：数据工程师负责视图，分析师专注业务分析

### 🚀 技术亮点
- PostgreSQL 强大的时区支持
- 视图封装复杂时区转换逻辑
- Go 语言现代化 API 设计
- Docker 容器化部署
- 完整的示例数据和查询场景

## 🏗️ 项目结构

```
timezone-saas-demo/
├── sql/                          # PostgreSQL 相关文件
│   ├── 01_schema.sql            # 数据库架构（表结构）
│   ├── 02_sample_data.sql       # 示例数据插入
│   ├── 03_analysis_view.sql     # 核心分析视图
│   └── 04_query_examples.sql    # 查询示例
├── go/                          # Go 应用程序
│   ├── main.go                  # 主程序入口
│   ├── models/                  # 数据模型
│   │   └── models.go
│   ├── database/                # 数据库连接
│   │   └── database.go
│   ├── services/                # 业务服务
│   │   └── timezone_service.go
│   ├── Dockerfile              # Go 应用容器化
│   ├── go.mod                  # Go 模块依赖
│   └── .dockerignore
├── docs/                        # 文档目录
├── docker-compose.yml           # Docker 编排配置
└── README.md                    # 项目说明文档
```

## 🚀 快速开始

### 前置要求
- Docker & Docker Compose
- Git

### 1. 克隆项目
```bash
git clone <repository-url>
cd timezone-saas-demo
```

### 2. 启动服务
```bash
# 启动基础服务（PostgreSQL + Go API）
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f
```

### 3. 验证部署
```bash
# 健康检查
curl http://localhost:8080/api/health

# 查看API文档
curl http://localhost:8080/api/docs
```

### 4. 可选服务
```bash
# 启动 pgAdmin（数据库管理工具）
docker-compose --profile tools up -d pgadmin
# 访问: http://localhost:5050 (admin@example.com / admin)

# 启动 Redis 缓存
docker-compose --profile cache up -d redis
```

## 📊 核心功能演示

### 1. 时区演示
```bash
# 查看同一UTC时间在全球不同时区的表现
curl "http://localhost:8080/api/timezone/demo"
```

### 2. 商户管理
```bash
# 获取所有商户及其时区信息
curl "http://localhost:8080/api/timezone/merchants"
```

### 3. 订单查询（时区转换）
```bash
# 获取订单列表（使用商户本地时区）
curl "http://localhost:8080/api/timezone/orders"

# 转换到指定时区查看订单
curl "http://localhost:8080/api/timezone/orders?timezone=Asia/Shanghai"
curl "http://localhost:8080/api/timezone/orders?timezone=America/New_York"
```

### 4. 数据分析
```bash
# 获取特定日期的分析数据
curl "http://localhost:8080/api/timezone/analysis?date=2024-08-19"

# 时区对比分析
curl "http://localhost:8080/api/timezone/compare?utc_time=2024-08-19T00:00:00Z"
```

## 🗄️ 数据库设计

### 核心表结构

#### 商户维度表 (dim_merchant)
```sql
CREATE TABLE dim_merchant (
    merchant_id SERIAL PRIMARY KEY,
    merchant_name VARCHAR(100) NOT NULL,
    merchant_code VARCHAR(50) UNIQUE NOT NULL,
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',  -- 商户时区
    country VARCHAR(50) NOT NULL,
    city VARCHAR(50) NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
```

#### 订单事实表 (dws_orders)
```sql
CREATE TABLE dws_orders (
    order_id SERIAL PRIMARY KEY,
    order_no VARCHAR(50) UNIQUE NOT NULL,
    merchant_id INTEGER NOT NULL REFERENCES dim_merchant(merchant_id),
    order_amount DECIMAL(15,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    order_status VARCHAR(20) DEFAULT 'pending',
    order_time_utc TIMESTAMPTZ NOT NULL,  -- 统一UTC时间存储
    payment_time_utc TIMESTAMPTZ,
    customer_id VARCHAR(50),
    customer_email VARCHAR(100),
    order_source VARCHAR(50) DEFAULT 'web',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
```

### 🎯 核心分析视图

```sql
CREATE OR REPLACE VIEW dws_orders_analysis_view AS
WITH t AS (
  SELECT
    -- 事实字段（做统一别名，兼容 Go）
    o.order_id,
    o.order_no                         AS order_number,
    o.order_amount                     AS amount,
    o.currency,
    o.order_status                     AS status,

    -- 商户字段（兼容 Go：timezone 列名）
    m.merchant_id,
    m.merchant_name,
    m.country,
    m.city,
    m.timezone,                        -- 保留列名为 timezone，方便 Go 直接使用

    -- 原始 UTC
    o.order_time_utc,
    o.payment_time_utc,

    -- 本地时间（timestamp without time zone）
    (o.order_time_utc   AT TIME ZONE m.timezone) AS order_time_local,
    (o.payment_time_utc AT TIME ZONE m.timezone) AS payment_time_local,

    -- 本地日期（兼容 Go：local_date）
    (o.order_time_utc AT TIME ZONE m.timezone)::date AS local_date
  FROM dws_orders o
  JOIN dim_merchant m ON m.merchant_id = o.merchant_id
)
SELECT
  t.*,

  -- 维度拆解（整点、周几等）
  EXTRACT(HOUR FROM t.order_time_local)::int       AS local_hour,
  EXTRACT(DOW  FROM t.order_time_local)::int       AS local_day_of_week,   -- 0=周日, 1=周一, ...
  TO_CHAR(t.order_time_local, 'FMDay')             AS local_weekday,       -- 英文周名，首字母大写，FM去空格

  -- 是否周末 / 是否工作时间（示例：周一~周五且 09:00-18:59）
  CASE WHEN EXTRACT(DOW FROM t.order_time_local) IN (0,6) THEN TRUE ELSE FALSE END AS is_weekend,
  CASE
    WHEN EXTRACT(DOW FROM t.order_time_local) BETWEEN 1 AND 5
     AND EXTRACT(HOUR FROM t.order_time_local) BETWEEN 9 AND 18
    THEN TRUE ELSE FALSE
  END AS is_business_hour,

  -- 时区偏移（单位：秒；可自行换算小时）
  -- 计算：本地时间 - UTC 本地化时间（两者都是 timestamp），得到偏移量
  EXTRACT(EPOCH FROM (t.order_time_local - (t.order_time_utc AT TIME ZONE 'UTC')))::int AS timezone_offset
FROM t;
```

## 📈 使用效果对比

### 传统方式（复杂）
```sql
-- 分析师需要手动处理时区转换
SELECT 
    m.merchant_name,
    COUNT(*) as order_count,
    (o.order_time_utc AT TIME ZONE m.timezone)::date as local_date,
    EXTRACT(hour FROM o.order_time_utc AT TIME ZONE m.timezone) as local_hour
FROM dws_orders o
JOIN dim_merchant m ON o.merchant_id = m.merchant_id
WHERE (o.order_time_utc AT TIME ZONE m.timezone)::date = '2024-08-19'
GROUP BY m.merchant_name, local_date, local_hour
ORDER BY local_hour;
```

### 使用视图（简洁）
```sql
-- 分析师只需关注业务逻辑
SELECT 
    merchant_name,
    COUNT(*) as order_count,
    local_date,
    local_hour
FROM dws_orders_analysis_view
WHERE local_date = '2024-08-19'
GROUP BY merchant_name, local_date, local_hour
ORDER BY local_hour;
```

## 🔧 开发指南

### 本地开发

#### 1. 数据库开发
```bash
# 仅启动数据库
docker-compose up -d postgres

# 连接数据库
psql -h localhost -U postgres -d timezone_demo

# 执行SQL脚本
psql -h localhost -U postgres -d timezone_demo -f sql/01_schema.sql
psql -h localhost -U postgres -d timezone_demo -f sql/02_sample_data.sql
psql -h localhost -U postgres -d timezone_demo -f sql/03_analysis_view.sql
```

#### 2. Go应用开发
```bash
cd go

# 安装依赖
go mod tidy

# 设置环境变量
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=timezone_demo

# 运行应用
go run main.go
```

#### 3. 开发模式启动
```bash
# 使用开发配置启动（支持热重载）
docker-compose --profile dev up -d app-dev
```

### API 接口文档

| 接口 | 方法 | 描述 | 示例 |
|------|------|------|------|
| `/api/health` | GET | 健康检查 | `curl localhost:8080/api/health` |
| `/api/docs` | GET | API文档 | `curl localhost:8080/api/docs` |
| `/api/timezone/demo` | GET | 时区演示 | `curl localhost:8080/api/timezone/demo` |
| `/api/timezone/merchants` | GET | 商户列表 | `curl localhost:8080/api/timezone/merchants` |
| `/api/timezone/orders` | GET | 订单列表 | `curl "localhost:8080/api/timezone/orders?timezone=Asia/Shanghai&limit=10"` |
| `/api/timezone/analysis` | GET | 分析数据 | `curl "localhost:8080/api/timezone/analysis?date=2024-08-19"` |
| `/api/timezone/compare` | GET | 时区对比 | `curl "localhost:8080/api/timezone/compare?utc_time=2024-08-19T00:00:00Z"` |

## 📚 学习要点

### 1. PostgreSQL 时区处理
- `TIMESTAMPTZ` 类型的使用
- `AT TIME ZONE` 语法进行时区转换
- 时区相关函数：`EXTRACT()`, `TO_CHAR()`

### 2. 数据库视图设计
- 封装复杂逻辑，简化上层查询
- 提供统一的数据接口
- 便于维护和优化

### 3. Go 语言实践
- 结构化项目组织
- 数据库连接池管理
- RESTful API 设计
- 错误处理和日志记录

### 4. 容器化部署
- 多阶段构建优化镜像大小
- 健康检查和服务依赖
- 环境变量配置管理

## 🔍 故障排除

### 常见问题

#### 1. 数据库连接失败
```bash
# 检查数据库状态
docker-compose logs postgres

# 检查网络连接
docker-compose exec app ping postgres
```

#### 2. 时区数据问题
```sql
-- 检查时区设置
SHOW timezone;

-- 查看可用时区
SELECT name FROM pg_timezone_names WHERE name LIKE 'Asia%' LIMIT 10;
```

#### 3. API 服务异常
```bash
# 查看应用日志
docker-compose logs app

# 进入容器调试
docker-compose exec app sh
```

### 性能优化

#### 1. 数据库索引
```sql
-- 为常用查询字段添加索引
CREATE INDEX idx_orders_merchant_time ON dws_orders(merchant_id, order_time_utc);
CREATE INDEX idx_orders_utc_time ON dws_orders(order_time_utc);
```

#### 2. 连接池配置
```go
// 在 database.go 中调整连接池参数
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```


---

**💡 提示**: 这个示例项目展示了如何通过数据库视图优雅地解决多租户时区问题。在实际生产环境中，还需要考虑更多因素，如缓存策略、监控告警、数据备份等。

**🌟 如果这个项目对你有帮助，请给个 Star！**