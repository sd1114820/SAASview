package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

// DB 数据库连接包装器
type DB struct {
	*sql.DB
}

// Config 数据库配置
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	Timezone string
}

// NewConnection 创建新的数据库连接
func NewConnection() (*DB, error) {
	config := getConfigFromEnv()
	
	// 构建连接字符串
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s timezone=%s",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.DBName,
		config.SSLMode,
		config.Timezone,
	)

	log.Printf("正在连接数据库: %s:%d/%s", config.Host, config.Port, config.DBName)

	// 打开数据库连接
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开数据库连接失败: %w", err)
	}

	// 配置连接池
	db.SetMaxOpenConns(25)                 // 最大打开连接数
	db.SetMaxIdleConns(5)                  // 最大空闲连接数
	db.SetConnMaxLifetime(5 * time.Minute) // 连接最大生存时间
	db.SetConnMaxIdleTime(1 * time.Minute) // 连接最大空闲时间

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	log.Println("✅ 数据库连接成功")

	return &DB{DB: db}, nil
}

// getConfigFromEnv 从环境变量获取配置
func getConfigFromEnv() Config {
	config := Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnvAsInt("DB_PORT", 5432),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		DBName:   getEnv("DB_NAME", "timezone_demo"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
		Timezone: getEnv("DB_TIMEZONE", "UTC"),
	}

	// 如果密码为空，尝试从文件读取（Docker secrets）
	if config.Password == "" {
		if passwordFile := getEnv("DB_PASSWORD_FILE", ""); passwordFile != "" {
			if password, err := os.ReadFile(passwordFile); err == nil {
				config.Password = string(password)
			}
		}
	}

	return config
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt 获取环境变量并转换为整数
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// Close 关闭数据库连接
func (db *DB) Close() error {
	log.Println("正在关闭数据库连接...")
	return db.DB.Close()
}

// Ping 测试数据库连接
func (db *DB) Ping() error {
	return db.DB.Ping()
}

// GetStats 获取数据库连接统计信息
func (db *DB) GetStats() sql.DBStats {
	return db.DB.Stats()
}

// BeginTx 开始事务
func (db *DB) BeginTx() (*sql.Tx, error) {
	return db.DB.Begin()
}

// ExecWithRetry 带重试的执行
func (db *DB) ExecWithRetry(query string, args ...interface{}) (sql.Result, error) {
	var result sql.Result
	var err error
	
	for i := 0; i < 3; i++ {
		result, err = db.Exec(query, args...)
		if err == nil {
			return result, nil
		}
		
		log.Printf("执行SQL失败 (尝试 %d/3): %v", i+1, err)
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	
	return result, fmt.Errorf("执行SQL失败，已重试3次: %w", err)
}

// QueryWithRetry 带重试的查询
func (db *DB) QueryWithRetry(query string, args ...interface{}) (*sql.Rows, error) {
	var rows *sql.Rows
	var err error
	
	for i := 0; i < 3; i++ {
		rows, err = db.Query(query, args...)
		if err == nil {
			return rows, nil
		}
		
		log.Printf("查询SQL失败 (尝试 %d/3): %v", i+1, err)
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	
	return rows, fmt.Errorf("查询SQL失败，已重试3次: %w", err)
}

// QueryRowWithRetry 带重试的单行查询
func (db *DB) QueryRowWithRetry(query string, args ...interface{}) *sql.Row {
	// sql.Row 不会返回连接错误，所以直接返回
	return db.QueryRow(query, args...)
}

// HealthCheck 健康检查
func (db *DB) HealthCheck() error {
	// 检查连接
	if err := db.Ping(); err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	// 检查基本查询
	var result int
	err := db.QueryRow("SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("数据库查询失败: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("数据库查询结果异常: 期望1，得到%d", result)
	}

	// 检查时区设置
	var timezone string
	err = db.QueryRow("SHOW timezone").Scan(&timezone)
	if err != nil {
		return fmt.Errorf("获取数据库时区失败: %w", err)
	}

	log.Printf("数据库时区: %s", timezone)
	return nil
}

// GetVersion 获取数据库版本
func (db *DB) GetVersion() (string, error) {
	var version string
	err := db.QueryRow("SELECT version()").Scan(&version)
	if err != nil {
		return "", fmt.Errorf("获取数据库版本失败: %w", err)
	}
	return version, nil
}

// CheckTableExists 检查表是否存在
func (db *DB) CheckTableExists(tableName string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = $1
		)
	`
	err := db.QueryRow(query, tableName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("检查表存在性失败: %w", err)
	}
	return exists, nil
}

// CheckViewExists 检查视图是否存在
func (db *DB) CheckViewExists(viewName string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.views 
			WHERE table_schema = 'public' 
			AND table_name = $1
		)
	`
	err := db.QueryRow(query, viewName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("检查视图存在性失败: %w", err)
	}
	return exists, nil
}

// GetTableRowCount 获取表行数
func (db *DB) GetTableRowCount(tableName string) (int, error) {
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("获取表行数失败: %w", err)
	}
	return count, nil
}

// ExecuteScript 执行SQL脚本文件
func (db *DB) ExecuteScript(scriptPath string) error {
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("读取脚本文件失败: %w", err)
	}

	_, err = db.Exec(string(content))
	if err != nil {
		return fmt.Errorf("执行脚本失败: %w", err)
	}

	log.Printf("✅ 成功执行脚本: %s", scriptPath)
	return nil
}

// LogStats 记录数据库连接统计信息
func (db *DB) LogStats() {
	stats := db.GetStats()
	log.Printf("数据库连接统计: 打开=%d, 使用中=%d, 空闲=%d, 等待=%d",
		stats.OpenConnections,
		stats.InUse,
		stats.Idle,
		stats.WaitCount,
	)
}