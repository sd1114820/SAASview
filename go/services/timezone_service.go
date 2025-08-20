package services

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"timezone-saas-demo/database"
	"timezone-saas-demo/models"
)

// TimezoneService 时区服务
type TimezoneService struct {
	db *database.DB
}

// NewTimezoneService 创建新的时区服务
func NewTimezoneService(db *database.DB) *TimezoneService {
	return &TimezoneService{
		db: db,
	}
}

// GetMerchants 获取所有商户
func (s *TimezoneService) GetMerchants() ([]models.Merchant, error) {
	query := `
		SELECT id, name, timezone, country, city, description, created_at, updated_at
		FROM dim_merchant
		ORDER BY name
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询商户失败: %w", err)
	}
	defer rows.Close()

	var merchants []models.Merchant
	for rows.Next() {
		var merchant models.Merchant
		err := rows.Scan(
			&merchant.ID,
			&merchant.Name,
			&merchant.Timezone,
			&merchant.Country,
			&merchant.City,
			&merchant.Description,
			&merchant.CreatedAt,
			&merchant.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描商户数据失败: %w", err)
		}
		merchants = append(merchants, merchant)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历商户数据失败: %w", err)
	}

	return merchants, nil
}

// GetOrders 获取订单列表（支持时区转换）
func (s *TimezoneService) GetOrders(timezone string, limit, offset int) ([]models.OrderAnalysis, error) {
	var query string

	if timezone != "" {
		// 查询指定时区的订单
		query = `
			SELECT 
				order_id, order_number, amount, currency, status,
				merchant_id, merchant_name, timezone, country, city,
				order_time_utc, order_time_local, local_date,
				local_hour, local_day_of_week, local_weekday,
				is_weekend, is_business_hour, timezone_offset
			FROM dws_orders_analysis_view
			WHERE timezone = $1
			ORDER BY order_time_utc DESC
			LIMIT $2 OFFSET $3
		`
	} else {
		// 查询所有订单
		query = `
			SELECT 
				order_id, order_number, amount, currency, status,
				merchant_id, merchant_name, timezone, country, city,
				order_time_utc, order_time_local, local_date,
				local_hour, local_day_of_week, local_weekday,
				is_weekend, is_business_hour, timezone_offset
			FROM dws_orders_analysis_view
			ORDER BY order_time_utc DESC
			LIMIT $1 OFFSET $2
		`
	}

	var rows *sql.Rows
	var err error

	if timezone != "" {
		rows, err = s.db.Query(query, timezone, limit, offset)
	} else {
		rows, err = s.db.Query(query, limit, offset)
	}

	if err != nil {
		return nil, fmt.Errorf("查询订单失败: %w", err)
	}
	defer rows.Close()

	var orders []models.OrderAnalysis
	for rows.Next() {
		var order models.OrderAnalysis
		var localDate time.Time
		var localWeekday string

		err := rows.Scan(
			&order.OrderID,
			&order.OrderNumber,
			&order.Amount,
			&order.Currency,
			&order.Status,
			&order.MerchantID,
			&order.MerchantName,
			&order.Timezone,
			&order.Country,
			&order.City,
			&order.OrderTimeUTC,
			&order.OrderTimeLocal,
			&localDate,
			&order.LocalHour,
			&order.LocalDayOfWeek,
			&localWeekday,
			&order.IsWeekend,
			&order.IsBusinessHour,
			&order.TimezoneOffset,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描订单数据失败: %w", err)
		}

		order.LocalDate = localDate.Format("2006-01-02")
		order.LocalWeekday = strings.TrimSpace(localWeekday)
		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历订单数据失败: %w", err)
	}

	return orders, nil
}

// GetAnalysisData 获取分析数据
func (s *TimezoneService) GetAnalysisData(date string) (*models.AnalysisData, error) {
	// 解析日期
	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("日期格式错误: %w", err)
	}

	analysis := &models.AnalysisData{
		Date: date,
	}

	// 获取总订单数和总金额
	err = s.getOrderSummary(date, analysis)
	if err != nil {
		return nil, fmt.Errorf("获取订单汇总失败: %w", err)
	}

	// 获取按小时分解的数据
	err = s.getHourlyBreakdown(date, analysis)
	if err != nil {
		return nil, fmt.Errorf("获取小时分解数据失败: %w", err)
	}

	// 获取时区统计
	err = s.getTimezoneStats(date, analysis)
	if err != nil {
		return nil, fmt.Errorf("获取时区统计失败: %w", err)
	}

	// 获取顶级商户
	err = s.getTopMerchants(date, analysis)
	if err != nil {
		return nil, fmt.Errorf("获取顶级商户失败: %w", err)
	}

	return analysis, nil
}

// getOrderSummary 获取订单汇总
func (s *TimezoneService) getOrderSummary(date string, analysis *models.AnalysisData) error {
	query := `
		SELECT 
			COUNT(*) as total_orders,
			COALESCE(SUM(amount), 0) as total_amount
		FROM dws_orders_analysis_view
		WHERE local_date = $1
	`

	err := s.db.QueryRow(query, date).Scan(
		&analysis.TotalOrders,
		&analysis.TotalAmount,
	)
	if err != nil {
		return fmt.Errorf("查询订单汇总失败: %w", err)
	}

	return nil
}

// getHourlyBreakdown 获取按小时分解的数据
func (s *TimezoneService) getHourlyBreakdown(date string, analysis *models.AnalysisData) error {
	query := `
		SELECT 
			local_hour,
			COUNT(*) as order_count,
			COALESCE(SUM(amount), 0) as total_amount,
			COALESCE(AVG(amount), 0) as avg_amount
		FROM dws_orders_analysis_view
		WHERE local_date = $1
		GROUP BY local_hour
		ORDER BY local_hour
	`

	rows, err := s.db.Query(query, date)
	if err != nil {
		return fmt.Errorf("查询小时分解数据失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var breakdown models.HourlyOrderBreakdown
		err := rows.Scan(
			&breakdown.Hour,
			&breakdown.OrderCount,
			&breakdown.TotalAmount,
			&breakdown.AvgAmount,
		)
		if err != nil {
			return fmt.Errorf("扫描小时分解数据失败: %w", err)
		}
		analysis.HourlyBreakdown = append(analysis.HourlyBreakdown, breakdown)
	}

	return rows.Err()
}

// getTimezoneStats 获取时区统计
func (s *TimezoneService) getTimezoneStats(date string, analysis *models.AnalysisData) error {
	query := `
		SELECT 
			timezone,
			country,
			COUNT(*) as order_count,
			COALESCE(SUM(amount), 0) as total_amount,
			COALESCE(AVG(amount), 0) as avg_amount
		FROM dws_orders_analysis_view
		WHERE local_date = $1
		GROUP BY timezone, country
		ORDER BY total_amount DESC
	`

	rows, err := s.db.Query(query, date)
	if err != nil {
		return fmt.Errorf("查询时区统计失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var stats models.TimezoneOrderStats
		err := rows.Scan(
			&stats.Timezone,
			&stats.Country,
			&stats.OrderCount,
			&stats.TotalAmount,
			&stats.AvgAmount,
		)
		if err != nil {
			return fmt.Errorf("扫描时区统计数据失败: %w", err)
		}
		analysis.TimezoneStats = append(analysis.TimezoneStats, stats)
	}

	return rows.Err()
}

// getTopMerchants 获取顶级商户
func (s *TimezoneService) getTopMerchants(date string, analysis *models.AnalysisData) error {
	query := `
		SELECT 
			merchant_id,
			merchant_name,
			timezone,
			COUNT(*) as order_count,
			COALESCE(SUM(amount), 0) as total_amount,
			COALESCE(AVG(amount), 0) as avg_amount
		FROM dws_orders_analysis_view
		WHERE local_date = $1
		GROUP BY merchant_id, merchant_name, timezone
		ORDER BY total_amount DESC
		LIMIT 10
	`

	rows, err := s.db.Query(query, date)
	if err != nil {
		return fmt.Errorf("查询顶级商户失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var merchant models.MerchantOrderStats
		err := rows.Scan(
			&merchant.MerchantID,
			&merchant.MerchantName,
			&merchant.Timezone,
			&merchant.OrderCount,
			&merchant.TotalAmount,
			&merchant.AvgAmount,
		)
		if err != nil {
			return fmt.Errorf("扫描顶级商户数据失败: %w", err)
		}
		analysis.TopMerchants = append(analysis.TopMerchants, merchant)
	}

	return rows.Err()
}

// CompareTimezones 时区对比分析
func (s *TimezoneService) CompareTimezones(utcTimeStr string) (*models.TimezoneComparison, error) {
	// 解析UTC时间
	utcTime, err := time.Parse(time.RFC3339, utcTimeStr)
	if err != nil {
		return nil, fmt.Errorf("UTC时间格式错误: %w", err)
	}

	comparison := &models.TimezoneComparison{
		UTCTime: utcTimeStr,
	}

	// 获取所有商户的时区转换
	query := `
		SELECT 
			name as merchant_name,
			timezone,
			$1::timestamptz AT TIME ZONE timezone as local_time,
			($1::timestamptz AT TIME ZONE timezone)::date as local_date,
			EXTRACT(hour FROM $1::timestamptz AT TIME ZONE timezone)::int as hour,
			TO_CHAR($1::timestamptz AT TIME ZONE timezone, 'Day') as day_of_week,
			EXTRACT(dow FROM $1::timestamptz AT TIME ZONE timezone) IN (0, 6) as is_weekend,
			EXTRACT(hour FROM $1::timestamptz AT TIME ZONE timezone) BETWEEN 9 AND 17 as is_business_hour
		FROM dim_merchant
		ORDER BY timezone
	`

	rows, err := s.db.Query(query, utcTime)
	if err != nil {
		return nil, fmt.Errorf("查询时区对比失败: %w", err)
	}
	defer rows.Close()

	var businessHourCount, weekendCount int
	var totalHours float64
	var minHour, maxHour int = 24, -1

	for rows.Next() {
		var item models.TimezoneComparisonItem
		var localTime time.Time
		var localDate time.Time
		var dayOfWeek string

		err := rows.Scan(
			&item.MerchantName,
			&item.Timezone,
			&localTime,
			&localDate,
			&item.Hour,
			&dayOfWeek,
			&item.IsWeekend,
			&item.IsBusinessHour,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描时区对比数据失败: %w", err)
		}

		item.LocalTime = localTime.Format("2006-01-02 15:04:05")
		item.LocalDate = localDate.Format("2006-01-02")
		item.DayOfWeek = strings.TrimSpace(dayOfWeek)

		// 计算时差
		hourDiff := item.Hour - utcTime.Hour()
		if hourDiff > 12 {
			hourDiff -= 24
		} else if hourDiff < -12 {
			hourDiff += 24
		}
		item.TimeDifference = fmt.Sprintf("%+d小时", hourDiff)

		comparison.Comparisons = append(comparison.Comparisons, item)

		// 统计信息
		if item.IsBusinessHour {
			businessHourCount++
		}
		if item.IsWeekend {
			weekendCount++
		}
		totalHours += float64(item.Hour)
		if item.Hour < minHour {
			minHour = item.Hour
		}
		if item.Hour > maxHour {
			maxHour = item.Hour
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历时区对比数据失败: %w", err)
	}

	// 计算统计信息
	totalCount := len(comparison.Comparisons)
	if totalCount > 0 {
		comparison.Statistics = models.TimezoneStatistics{
			BusinessHourCount: businessHourCount,
			WeekendCount:      weekendCount,
			AverageHour:       totalHours / float64(totalCount),
			TimezoneSpread:    maxHour - minHour,
		}
	}

	return comparison, nil
}

// GetTimezoneDemo 获取时区演示数据
func (s *TimezoneService) GetTimezoneDemo() (*models.TimezoneDemo, error) {
	// 使用一个固定的UTC时间进行演示
	utcTime := time.Date(2024, 8, 19, 0, 0, 0, 0, time.UTC)
	utcTimeStr := utcTime.Format(time.RFC3339)

	demo := &models.TimezoneDemo{
		UTCTime:     utcTimeStr,
		Description: "演示同一UTC时间在全球不同时区的本地时间表现",
	}

	// 获取所有商户的时区信息
	query := `
		SELECT 
			timezone, country, city,
			$1::timestamptz AT TIME ZONE timezone as local_time,
			($1::timestamptz AT TIME ZONE timezone)::date as local_date,
			TO_CHAR($1::timestamptz AT TIME ZONE timezone, 'TZ') as offset
		FROM dim_merchant
		ORDER BY timezone
	`

	rows, err := s.db.Query(query, utcTime)
	if err != nil {
		return nil, fmt.Errorf("查询时区演示数据失败: %w", err)
	}
	defer rows.Close()

	var nextDayCount, sameDayCount, prevDayCount int
	var minOffset, maxOffset int = 24, -24
	utcDate := utcTime.Format("2006-01-02")

	for rows.Next() {
		var conversion models.TimezoneConversion
		var localTime time.Time
		var localDate time.Time
		var offset string

		err := rows.Scan(
			&conversion.Timezone,
			&conversion.Country,
			&conversion.City,
			&localTime,
			&localDate,
			&offset,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描时区演示数据失败: %w", err)
		}

		conversion.LocalTime = localTime.Format("2006-01-02 15:04:05")
		conversion.LocalDate = localDate.Format("2006-01-02")
		conversion.Offset = offset

		// 判断日期关系
		if conversion.LocalDate > utcDate {
			conversion.IsNextDay = true
			nextDayCount++
		} else if conversion.LocalDate < utcDate {
			conversion.IsPrevDay = true
			prevDayCount++
		} else {
			sameDayCount++
		}

		// 解析时区偏移（简化处理）
		if offsetHours, err := parseTimezoneOffset(offset); err == nil {
			if offsetHours < minOffset {
				minOffset = offsetHours
			}
			if offsetHours > maxOffset {
				maxOffset = offsetHours
			}
		}

		demo.Timezones = append(demo.Timezones, conversion)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历时区演示数据失败: %w", err)
	}

	// 设置汇总信息
	demo.Summary = models.TimezoneDemoSummary{
		TotalTimezones: len(demo.Timezones),
		NextDayCount:   nextDayCount,
		SameDayCount:   sameDayCount,
		PrevDayCount:   prevDayCount,
		MinOffset:      minOffset,
		MaxOffset:      maxOffset,
	}

	return demo, nil
}

// parseTimezoneOffset 解析时区偏移字符串
func parseTimezoneOffset(offset string) (int, error) {
	// 简化的时区偏移解析，实际应用中可能需要更复杂的逻辑
	if len(offset) < 3 {
		return 0, fmt.Errorf("无效的时区偏移格式: %s", offset)
	}

	// 处理 +08, -05 等格式
	sign := 1
	if offset[0] == '-' {
		sign = -1
		offset = offset[1:]
	} else if offset[0] == '+' {
		offset = offset[1:]
	}

	// 提取小时部分
	if len(offset) >= 2 {
		if hours, err := strconv.Atoi(offset[:2]); err == nil {
			return sign * hours, nil
		}
	}

	return 0, fmt.Errorf("无法解析时区偏移: %s", offset)
}

// HealthCheck 健康检查
func (s *TimezoneService) HealthCheck() error {
	// 检查数据库连接
	if err := s.db.Ping(); err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	// 检查关键表是否存在
	tables := []string{"dim_merchant", "dws_orders"}
	for _, table := range tables {
		exists, err := s.db.CheckTableExists(table)
		if err != nil {
			return fmt.Errorf("检查表 %s 失败: %w", table, err)
		}
		if !exists {
			return fmt.Errorf("表 %s 不存在", table)
		}
	}

	// 检查关键视图是否存在
	exists, err := s.db.CheckViewExists("dws_orders_analysis_view")
	if err != nil {
		return fmt.Errorf("检查视图失败: %w", err)
	}
	if !exists {
		return fmt.Errorf("分析视图不存在")
	}

	// 检查数据完整性
	merchantCount, err := s.db.GetTableRowCount("dim_merchant")
	if err != nil {
		return fmt.Errorf("获取商户数量失败: %w", err)
	}
	if merchantCount == 0 {
		return fmt.Errorf("商户表为空")
	}

	orderCount, err := s.db.GetTableRowCount("dws_orders")
	if err != nil {
		return fmt.Errorf("获取订单数量失败: %w", err)
	}
	if orderCount == 0 {
		return fmt.Errorf("订单表为空")
	}

	log.Printf("✅ 时区服务健康检查通过: %d个商户, %d个订单", merchantCount, orderCount)
	return nil
}