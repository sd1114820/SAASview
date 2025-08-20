-- =====================================================
-- SAAS多租户时区处理示例 - 数据库架构
-- PostgreSQL版本
-- =====================================================

-- 删除已存在的表和视图（如果存在）
DROP VIEW IF EXISTS dws_orders_analysis_view;
DROP TABLE IF EXISTS dws_orders;
DROP TABLE IF EXISTS dim_merchant;

-- =====================================================
-- 商户维度表 (dim_merchant)
-- 存储商户基本信息和时区配置
-- =====================================================
CREATE TABLE dim_merchant (
    merchant_id SERIAL PRIMARY KEY,
    merchant_name VARCHAR(100) NOT NULL,
    merchant_code VARCHAR(50) UNIQUE NOT NULL,
    country VARCHAR(50) NOT NULL,
    city VARCHAR(50) NOT NULL,
    -- 时区字段：使用标准时区名称
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
    -- 商户状态
    status VARCHAR(20) DEFAULT 'active',
    -- 创建时间（UTC）
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 为商户表添加索引
CREATE INDEX idx_merchant_code ON dim_merchant(merchant_code);
CREATE INDEX idx_merchant_timezone ON dim_merchant(timezone);

-- 添加商户表注释
COMMENT ON TABLE dim_merchant IS '商户维度表，存储商户基本信息和时区配置';
COMMENT ON COLUMN dim_merchant.timezone IS '商户所在时区，使用标准时区名称如Asia/Shanghai';

-- =====================================================
-- 订单事实表 (dws_orders)
-- 存储订单交易数据，时间统一使用UTC
-- =====================================================
CREATE TABLE dws_orders (
    order_id SERIAL PRIMARY KEY,
    order_no VARCHAR(50) UNIQUE NOT NULL,
    merchant_id INTEGER NOT NULL REFERENCES dim_merchant(merchant_id),
    -- 订单金额
    order_amount DECIMAL(15,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    -- 订单状态
    order_status VARCHAR(20) DEFAULT 'pending',
    -- 核心时间字段：统一存储UTC时间
    order_time_utc TIMESTAMP WITH TIME ZONE NOT NULL,
    -- 支付时间（UTC）
    payment_time_utc TIMESTAMP WITH TIME ZONE,
    -- 客户信息
    customer_id VARCHAR(50),
    customer_email VARCHAR(100),
    -- 订单来源
    order_source VARCHAR(50) DEFAULT 'web',
    -- 创建和更新时间
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 为订单表添加索引
CREATE INDEX idx_orders_merchant_id ON dws_orders(merchant_id);
CREATE INDEX idx_orders_time_utc ON dws_orders(order_time_utc);
CREATE INDEX idx_orders_status ON dws_orders(order_status);
CREATE INDEX idx_orders_no ON dws_orders(order_no);

-- 复合索引：商户+时间，用于分析查询
CREATE INDEX idx_orders_merchant_time ON dws_orders(merchant_id, order_time_utc);

-- 添加订单表注释
COMMENT ON TABLE dws_orders IS '订单事实表，存储订单交易数据，时间统一使用UTC';
COMMENT ON COLUMN dws_orders.order_time_utc IS '订单创建时间，统一存储为UTC时间';
COMMENT ON COLUMN dws_orders.payment_time_utc IS '支付完成时间，统一存储为UTC时间';

-- =====================================================
-- 创建更新时间触发器函数
-- =====================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为商户表添加更新时间触发器
CREATE TRIGGER update_merchant_updated_at 
    BEFORE UPDATE ON dim_merchant 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- 为订单表添加更新时间触发器
CREATE TRIGGER update_orders_updated_at 
    BEFORE UPDATE ON dws_orders 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- =====================================================
-- 数据完整性约束
-- =====================================================

-- 确保订单金额为正数
ALTER TABLE dws_orders ADD CONSTRAINT chk_order_amount_positive 
    CHECK (order_amount > 0);

-- 确保时区格式正确（基本验证）
ALTER TABLE dim_merchant ADD CONSTRAINT chk_timezone_format 
    CHECK (timezone ~ '^[A-Za-z]+/[A-Za-z_]+(/[A-Za-z_]+)?$' OR timezone = 'UTC');

-- 确保订单状态在允许范围内
ALTER TABLE dws_orders ADD CONSTRAINT chk_order_status 
    CHECK (order_status IN ('pending', 'paid', 'shipped', 'delivered', 'cancelled', 'refunded'));

-- 确保商户状态在允许范围内
ALTER TABLE dim_merchant ADD CONSTRAINT chk_merchant_status 
    CHECK (status IN ('active', 'inactive', 'suspended'));

-- 确保货币代码格式正确
ALTER TABLE dws_orders ADD CONSTRAINT chk_currency_format 
    CHECK (currency ~ '^[A-Z]{3}$');

-- 确保客户邮箱格式正确（如果提供）
ALTER TABLE dws_orders ADD CONSTRAINT chk_customer_email_format 
    CHECK (customer_email IS NULL OR customer_email ~ '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$');

-- =====================================================
-- 性能优化索引
-- =====================================================

-- 为分析查询添加额外索引
-- 按日期范围查询的索引
CREATE INDEX idx_orders_date_range ON dws_orders 
    USING BTREE (DATE(order_time_utc AT TIME ZONE 'UTC'));

-- 按金额范围查询的索引
CREATE INDEX idx_orders_amount ON dws_orders(order_amount);

-- 按货币类型查询的索引
CREATE INDEX idx_orders_currency ON dws_orders(currency);

-- 按客户查询的索引
CREATE INDEX idx_orders_customer ON dws_orders(customer_id) WHERE customer_id IS NOT NULL;

-- 按订单来源查询的索引
CREATE INDEX idx_orders_source ON dws_orders(order_source);

-- 商户表的复合索引
CREATE INDEX idx_merchant_country_city ON dim_merchant(country, city);
CREATE INDEX idx_merchant_status_timezone ON dim_merchant(status, timezone) WHERE status = 'active';

-- 用于时区分析的部分索引（只索引活跃商户）
CREATE INDEX idx_merchant_active_timezone ON dim_merchant(timezone) WHERE status = 'active';

-- 用于最近订单查询的部分索引
CREATE INDEX idx_orders_recent ON dws_orders(order_time_utc DESC, merchant_id) 
    WHERE order_time_utc >= CURRENT_TIMESTAMP - INTERVAL '30 days';

-- 用于支付完成订单的索引
CREATE INDEX idx_orders_paid ON dws_orders(payment_time_utc, merchant_id) 
    WHERE payment_time_utc IS NOT NULL;

-- =====================================================
-- 数据库统计信息更新
-- =====================================================

-- 更新表统计信息以优化查询计划
ANALYZE dim_merchant;
ANALYZE dws_orders;

-- =====================================================
-- 数据库架构创建完成
-- =====================================================

-- 数据库架构创建完成！
-- 包含表：dim_merchant（商户维度表）、dws_orders（订单事实表）
-- 已添加必要的索引、约束和触发器