-- =====================================================
-- SAAS多租户时区处理示例 - 查询示例
-- 展示视图使用效果，对比传统方式与视图方式
-- =====================================================

-- =====================================================
-- 第一部分：问题演示 - 传统方式的复杂性
-- =====================================================

-- === 第一部分：传统方式的复杂查询 ===
-- 问题：分析师需要手写复杂的JOIN和时区转换逻辑

-- 传统方式：分析师想查询"巴西商户在当地时间2024年8月19日的销售额"
-- 需要写这样复杂的SQL：
SELECT 
    m.merchant_name,
    m.timezone,
    SUM(o.order_amount) as daily_sales,
    COUNT(o.order_id) as order_count
FROM dws_orders o
JOIN dim_merchant m ON o.merchant_id = m.merchant_id
WHERE m.merchant_name LIKE '%圣保罗%'
  AND DATE(timezone(m.timezone, o.order_time_utc)) = '2024-08-19'  -- 复杂的时区转换
  AND o.order_status IN ('paid', 'shipped', 'delivered')
GROUP BY m.merchant_name, m.timezone;

-- 传统方式问题：
-- 1. 查询复杂，容易出错
-- 2. 每个分析师可能写出不同的时区转换逻辑
-- 3. 性能较差，重复计算时区转换

-- =====================================================
-- 第二部分：视图方式的简洁性
-- =====================================================

-- === 第二部分：使用视图的简洁查询 ===
-- 解决方案：直接使用预处理好的本地时间字段

-- 使用视图：同样的需求变得极其简单
SELECT 
    merchant_name,
    merchant_timezone,
    SUM(order_amount) as daily_sales,
    COUNT(order_id) as order_count
FROM dws_orders_analysis_view  -- 直接使用视图
WHERE merchant_name LIKE '%圣保罗%'
  AND order_date_local = '2024-08-19'  -- 直接使用本地日期
  AND order_status IN ('paid', 'shipped', 'delivered')
GROUP BY merchant_name, merchant_timezone;

-- 视图方式优势：
-- 1. 查询简洁，不易出错
-- 2. 所有分析师使用统一的时区转换逻辑
-- 3. 性能更好，时区转换逻辑被优化

-- =====================================================
-- 第三部分：实际业务场景查询示例
-- =====================================================

-- === 第三部分：实际业务场景查询示例 ===

-- 场景1：全球销售日报 - 按商户本地日期统计
SELECT 
    order_date_local,
    country,
    merchant_timezone,
    COUNT(order_id) as order_count,
    SUM(order_amount) as total_sales,
    AVG(order_amount) as avg_order_value,
    COUNT(DISTINCT merchant_id) as active_merchants
FROM dws_orders_analysis_view
WHERE order_date_local BETWEEN '2024-08-17' AND '2024-08-19'
  AND order_status IN ('paid', 'shipped', 'delivered')
GROUP BY order_date_local, country, merchant_timezone
ORDER BY order_date_local DESC, total_sales DESC;

-- 场景2：时区对比分析 - 同一UTC时间在不同时区的表现
SELECT 
    order_time_utc,
    merchant_name,
    country,
    merchant_timezone,
    order_time_local,
    order_date_local,
    business_time_period,
    day_type,
    order_amount,
    currency
FROM dws_orders_analysis_view
WHERE order_time_utc = '2024-08-19 00:00:00+00'  -- 同一UTC时间
ORDER BY order_time_local;

-- 场景3：营业时间分析 - 各时区的营业高峰时段
SELECT 
    merchant_timezone,
    country,
    business_time_period,
    COUNT(order_id) as order_count,
    SUM(order_amount) as total_sales,
    AVG(order_amount) as avg_order_value
FROM dws_orders_analysis_view
WHERE order_status IN ('paid', 'shipped', 'delivered')
GROUP BY merchant_timezone, country, business_time_period
ORDER BY merchant_timezone, total_sales DESC;

-- 场景4：工作日vs周末分析
SELECT 
    country,
    merchant_timezone,
    day_type,
    COUNT(order_id) as order_count,
    SUM(order_amount) as total_sales,
    AVG(order_amount) as avg_order_value,
    AVG(payment_processing_minutes) as avg_processing_time
FROM dws_orders_analysis_view
WHERE order_status IN ('paid', 'shipped', 'delivered')
  AND payment_processing_minutes IS NOT NULL
GROUP BY country, merchant_timezone, day_type
ORDER BY country, merchant_timezone, day_type;

-- 场景5：跨日期边界问题展示 - 同一UTC时间在不同时区可能是不同日期
SELECT 
    order_time_utc,
    merchant_name,
    merchant_timezone,
    order_time_local,
    order_date_local,
    CASE 
        WHEN DATE(order_time_utc) != order_date_local THEN '跨日期'
        ELSE '同日期'
    END as date_boundary_status
FROM dws_orders_analysis_view
WHERE order_time_utc IN (
    '2024-08-19 23:30:00+00',  -- UTC深夜
    '2024-08-20 02:00:00+00'   -- UTC凌晨
)
ORDER BY order_time_utc, merchant_timezone;

-- =====================================================
-- 第四部分：高级分析查询
-- =====================================================

-- 高级场景1：时区热力图数据 - 24小时销售热力图数据（按商户本地时间）
SELECT 
    merchant_timezone,
    order_hour_local,
    COUNT(order_id) as order_count,
    SUM(order_amount) as hourly_sales,
    AVG(order_amount) as avg_order_value
FROM dws_orders_analysis_view
WHERE order_status IN ('paid', 'shipped', 'delivered')
GROUP BY merchant_timezone, order_hour_local
ORDER BY merchant_timezone, order_hour_local;

-- 高级场景2：同期对比分析（本地时间维度） - 各时区相同本地时间的表现
WITH timezone_performance AS (
    SELECT 
        merchant_timezone,
        order_hour_local,
        COUNT(order_id) as order_count,
        SUM(order_amount) as total_sales
    FROM dws_orders_analysis_view
    WHERE order_status IN ('paid', 'shipped', 'delivered')
      AND order_date_local = '2024-08-19'
    GROUP BY merchant_timezone, order_hour_local
)
SELECT 
    order_hour_local,
    COUNT(DISTINCT merchant_timezone) as active_timezones,
    SUM(order_count) as global_orders,
    SUM(total_sales) as global_sales,
    AVG(total_sales) as avg_timezone_sales
FROM timezone_performance
GROUP BY order_hour_local
ORDER BY order_hour_local;

-- 高级场景3：商户活跃度分析（基于本地时间） - 基于本地营业时间
SELECT 
    merchant_name,
    merchant_timezone,
    country,
    COUNT(DISTINCT order_date_local) as active_days,
    COUNT(DISTINCT order_hour_local) as active_hours,
    COUNT(order_id) as total_orders,
    SUM(order_amount) as total_sales,
    MIN(order_time_local) as first_order_local,
    MAX(order_time_local) as last_order_local
FROM dws_orders_analysis_view
WHERE order_status IN ('paid', 'shipped', 'delivered')
GROUP BY merchant_name, merchant_timezone, country
ORDER BY total_sales DESC;

-- =====================================================
-- 第五部分：数据质量检查
-- =====================================================

-- 检查1：时区转换一致性验证
SELECT 
    merchant_timezone,
    COUNT(*) as record_count,
    COUNT(DISTINCT DATE(order_time_local)) as unique_local_dates,
    MIN(order_time_local) as earliest_local_time,
    MAX(order_time_local) as latest_local_time
FROM dws_orders_analysis_view
GROUP BY merchant_timezone
ORDER BY merchant_timezone;

-- 检查2：跨日期边界统计
SELECT 
    '跨日期边界订单统计' as check_type,
    COUNT(*) as total_orders,
    SUM(CASE WHEN DATE(order_time_utc) != order_date_local THEN 1 ELSE 0 END) as cross_date_orders,
    ROUND(
        100.0 * SUM(CASE WHEN DATE(order_time_utc) != order_date_local THEN 1 ELSE 0 END) / COUNT(*), 
        2
    ) as cross_date_percentage
FROM dws_orders_analysis_view;

-- 检查3：时区覆盖度统计
SELECT 
    '时区覆盖统计' as summary_type,
    COUNT(DISTINCT merchant_timezone) as unique_timezones,
    COUNT(DISTINCT country) as unique_countries,
    COUNT(DISTINCT merchant_id) as unique_merchants,
    COUNT(*) as total_orders
FROM dws_orders_analysis_view;

-- =====================================================
-- 数据验证查询（从示例数据迁移）
-- =====================================================

-- 显示插入的商户数据统计
SELECT 
    '商户数据统计' as info,
    COUNT(*) as merchant_count,
    COUNT(DISTINCT timezone) as timezone_count
FROM dim_merchant;

-- 显示插入的订单数据统计
SELECT 
    '订单数据统计' as info,
    COUNT(*) as order_count,
    COUNT(DISTINCT merchant_id) as merchant_with_orders,
    MIN(order_time_utc) as earliest_order,
    MAX(order_time_utc) as latest_order,
    SUM(order_amount) as total_amount
FROM dws_orders;

-- 显示各时区的商户和订单分布
SELECT 
    m.timezone,
    m.country,
    COUNT(DISTINCT m.merchant_id) as merchant_count,
    COUNT(o.order_id) as order_count,
    COALESCE(SUM(o.order_amount), 0) as total_amount
FROM dim_merchant m
LEFT JOIN dws_orders o ON m.merchant_id = o.merchant_id
GROUP BY m.timezone, m.country
ORDER BY m.timezone;

-- 验证视图创建成功
SELECT 
    'dws_orders_analysis_view' as view_name,
    COUNT(*) as total_records,
    COUNT(DISTINCT merchant_timezone) as timezone_count,
    MIN(order_date_local) as earliest_local_date,
    MAX(order_date_local) as latest_local_date
FROM dws_orders_analysis_view;

-- 展示视图的强大功能 - 时区转换效果对比
-- 同一UTC时间在不同时区的本地时间对比
SELECT 
    order_time_utc,
    merchant_name,
    country,
    merchant_timezone,
    order_time_local,
    order_date_local,
    business_time_period,
    day_type
FROM dws_orders_analysis_view
WHERE order_time_utc = '2024-08-19 00:00:00+00'
ORDER BY merchant_timezone;

-- =====================================================
-- 查询示例演示完成
-- =====================================================

-- 核心价值体现：
-- 1. 查询复杂度大幅降低
-- 2. 时区处理逻辑统一且正确
-- 3. 分析师可专注业务逻辑而非技术细节
-- 4. 数据一致性得到保障
-- 5. 查询性能得到优化