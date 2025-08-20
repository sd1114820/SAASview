package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// Merchant 商户模型
type Merchant struct {
	ID          int       `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Timezone    string    `json:"timezone" db:"timezone"`
	Country     string    `json:"country" db:"country"`
	City        string    `json:"city" db:"city"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Order 订单模型
type Order struct {
	ID           int       `json:"id" db:"id"`
	MerchantID   int       `json:"merchant_id" db:"merchant_id"`
	OrderNumber  string    `json:"order_number" db:"order_number"`
	Amount       float64   `json:"amount" db:"amount"`
	Currency     string    `json:"currency" db:"currency"`
	Status       string    `json:"status" db:"status"`
	OrderTimeUTC time.Time `json:"order_time_utc" db:"order_time_utc"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// OrderAnalysis 订单分析模型（对应视图）
type OrderAnalysis struct {
	// 基础订单信息
	OrderID      int     `json:"order_id" db:"order_id"`
	OrderNumber  string  `json:"order_number" db:"order_number"`
	Amount       float64 `json:"amount" db:"amount"`
	Currency     string  `json:"currency" db:"currency"`
	Status       string  `json:"status" db:"status"`

	// 商户信息
	MerchantID   int    `json:"merchant_id" db:"merchant_id"`
	MerchantName string `json:"merchant_name" db:"merchant_name"`
	Timezone     string `json:"timezone" db:"timezone"`
	Country      string `json:"country" db:"country"`
	City         string `json:"city" db:"city"`

	// 时间信息（核心）
	OrderTimeUTC   time.Time `json:"order_time_utc" db:"order_time_utc"`
	OrderTimeLocal time.Time `json:"order_time_local" db:"order_time_local"`
	LocalDate      string    `json:"local_date" db:"local_date"`
	LocalHour      int       `json:"local_hour" db:"local_hour"`
	LocalDayOfWeek int       `json:"local_day_of_week" db:"local_day_of_week"`
	LocalWeekday   string    `json:"local_weekday" db:"local_weekday"`
	IsWeekend      bool      `json:"is_weekend" db:"is_weekend"`
	IsBusinessHour bool      `json:"is_business_hour" db:"is_business_hour"`

	// 时区偏移信息
	TimezoneOffset int `json:"timezone_offset" db:"timezone_offset"`
}

// TimezoneDemo 时区演示数据
type TimezoneDemo struct {
	UTCTime     string                   `json:"utc_time"`
	Description string                   `json:"description"`
	Timezones   []TimezoneConversion     `json:"timezones"`
	Summary     TimezoneDemoSummary      `json:"summary"`
}

// TimezoneConversion 时区转换信息
type TimezoneConversion struct {
	Timezone    string `json:"timezone"`
	LocalTime   string `json:"local_time"`
	LocalDate   string `json:"local_date"`
	Offset      string `json:"offset"`
	Country     string `json:"country"`
	City        string `json:"city"`
	IsNextDay   bool   `json:"is_next_day"`
	IsPrevDay   bool   `json:"is_prev_day"`
}

// TimezoneDemoSummary 时区演示汇总
type TimezoneDemoSummary struct {
	TotalTimezones int `json:"total_timezones"`
	NextDayCount   int `json:"next_day_count"`
	SameDayCount   int `json:"same_day_count"`
	PrevDayCount   int `json:"prev_day_count"`
	MinOffset      int `json:"min_offset_hours"`
	MaxOffset      int `json:"max_offset_hours"`
}

// TimezoneComparison 时区对比分析
type TimezoneComparison struct {
	UTCTime       string                    `json:"utc_time"`
	Comparisons   []TimezoneComparisonItem  `json:"comparisons"`
	Statistics    TimezoneStatistics        `json:"statistics"`
}

// TimezoneComparisonItem 时区对比项
type TimezoneComparisonItem struct {
	MerchantName   string `json:"merchant_name"`
	Timezone       string `json:"timezone"`
	LocalTime      string `json:"local_time"`
	LocalDate      string `json:"local_date"`
	Hour           int    `json:"hour"`
	DayOfWeek      string `json:"day_of_week"`
	IsWeekend      bool   `json:"is_weekend"`
	IsBusinessHour bool   `json:"is_business_hour"`
	TimeDifference string `json:"time_difference"`
}

// TimezoneStatistics 时区统计信息
type TimezoneStatistics struct {
	BusinessHourCount int     `json:"business_hour_count"`
	WeekendCount      int     `json:"weekend_count"`
	AverageHour       float64 `json:"average_hour"`
	TimezoneSpread    int     `json:"timezone_spread_hours"`
}

// AnalysisData 分析数据
type AnalysisData struct {
	Date            string                 `json:"date"`
	TotalOrders     int                    `json:"total_orders"`
	TotalAmount     float64                `json:"total_amount"`
	HourlyBreakdown []HourlyOrderBreakdown `json:"hourly_breakdown"`
	TimezoneStats   []TimezoneOrderStats   `json:"timezone_stats"`
	TopMerchants    []MerchantOrderStats   `json:"top_merchants"`
}

// HourlyOrderBreakdown 按小时订单分解
type HourlyOrderBreakdown struct {
	Hour        int     `json:"hour"`
	OrderCount  int     `json:"order_count"`
	TotalAmount float64 `json:"total_amount"`
	AvgAmount   float64 `json:"avg_amount"`
}

// TimezoneOrderStats 时区订单统计
type TimezoneOrderStats struct {
	Timezone    string  `json:"timezone"`
	Country     string  `json:"country"`
	OrderCount  int     `json:"order_count"`
	TotalAmount float64 `json:"total_amount"`
	AvgAmount   float64 `json:"avg_amount"`
}

// MerchantOrderStats 商户订单统计
type MerchantOrderStats struct {
	MerchantID   int     `json:"merchant_id"`
	MerchantName string  `json:"merchant_name"`
	Timezone     string  `json:"timezone"`
	OrderCount   int     `json:"order_count"`
	TotalAmount  float64 `json:"total_amount"`
	AvgAmount    float64 `json:"avg_amount"`
}

// NullTime 可空时间类型
type NullTime struct {
	Time  time.Time
	Valid bool
}

// Scan 实现 sql.Scanner 接口
func (nt *NullTime) Scan(value interface{}) error {
	if value == nil {
		nt.Time, nt.Valid = time.Time{}, false
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		nt.Time, nt.Valid = v, true
		return nil
	case []byte:
		nt.Valid = false
		if len(v) == 0 {
			return nil
		}
		t, err := time.Parse(time.RFC3339, string(v))
		if err != nil {
			return err
		}
		nt.Time, nt.Valid = t, true
		return nil
	case string:
		if v == "" {
			nt.Valid = false
			return nil
		}
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return err
		}
		nt.Time, nt.Valid = t, true
		return nil
	}
	return fmt.Errorf("cannot scan %T into NullTime", value)
}

// Value 实现 driver.Valuer 接口
func (nt NullTime) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return nt.Time, nil
}

// MarshalJSON 实现 JSON 序列化
func (nt NullTime) MarshalJSON() ([]byte, error) {
	if !nt.Valid {
		return []byte("null"), nil
	}
	return nt.Time.MarshalJSON()
}

// UnmarshalJSON 实现 JSON 反序列化
func (nt *NullTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		nt.Valid = false
		return nil
	}
	err := nt.Time.UnmarshalJSON(data)
	nt.Valid = err == nil
	return err
}

// String 实现 Stringer 接口
func (nt NullTime) String() string {
	if !nt.Valid {
		return "null"
	}
	return nt.Time.String()
}

// IsZero 检查是否为零值
func (nt NullTime) IsZero() bool {
	return !nt.Valid || nt.Time.IsZero()
}

// Ptr 返回时间指针，如果无效则返回 nil
func (nt NullTime) Ptr() *time.Time {
	if !nt.Valid {
		return nil
	}
	return &nt.Time
}

// NewNullTime 创建新的 NullTime
func NewNullTime(t time.Time, valid bool) NullTime {
	return NullTime{
		Time:  t,
		Valid: valid,
	}
}

// NewNullTimeFromPtr 从时间指针创建 NullTime
func NewNullTimeFromPtr(t *time.Time) NullTime {
	if t == nil {
		return NullTime{Valid: false}
	}
	return NullTime{
		Time:  *t,
		Valid: true,
	}
}