-- =====================================================
-- 修复版：订单分析视图（PostgreSQL）
-- 依赖：dws_orders(order_time_utc/payment_time_utc 为 timestamptz)
-- 命名对齐 Go 查询使用的列名
-- =====================================================

DROP VIEW IF EXISTS dws_orders_analysis_view;

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

-- =====================================================
-- 视图创建完成
-- =====================================================

-- 视图创建完成！
-- 该视图提供了完整的时区转换功能，字段命名与Go代码完全对齐
-- 支持的分析维度：本地时间、日期、小时、星期、工作时间判断等
-- 时区偏移以秒为单位，可根据需要转换为小时