package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"timezone-saas-demo/database"
	"timezone-saas-demo/services"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// APIResponse 统一的API响应格式
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// 全局变量
var (
	db             *database.DB
	timezoneService *services.TimezoneService
)

func main() {
	// 初始化数据库连接
	var err error
	db, err = database.NewConnection()
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer db.Close()

	// 初始化时区服务
	timezoneService = services.NewTimezoneService(db)

	// 设置路由
	router := setupRoutes()

	// 启动服务器
	port := getEnv("PORT", "8080")
	fmt.Printf("🚀 服务器启动在端口 %s\n", port)
	fmt.Printf("📊 API文档: http://localhost:%s/api/docs\n", port)
	fmt.Printf("🌍 时区演示: http://localhost:%s/api/timezone/demo\n", port)

	log.Fatal(http.ListenAndServe(":"+port, router))
}

// setupRoutes 设置所有路由
func setupRoutes() *mux.Router {
	router := mux.NewRouter()

	// 添加CORS中间件
	router.Use(corsMiddleware)

	// API路由
	api := router.PathPrefix("/api").Subrouter()

	// 健康检查
	api.HandleFunc("/health", healthCheckHandler).Methods("GET")

	// API文档
	api.HandleFunc("/docs", apiDocsHandler).Methods("GET")

	// 时区相关API
	api.HandleFunc("/timezone/demo", timezoneDemo).Methods("GET")
	api.HandleFunc("/timezone/merchants", getMerchants).Methods("GET")
	api.HandleFunc("/timezone/orders", getOrders).Methods("GET")
	api.HandleFunc("/timezone/analysis", getAnalysisData).Methods("GET")
	api.HandleFunc("/timezone/compare", compareTimezones).Methods("GET")

	// 静态文件服务（如果需要）
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/"))).Methods("GET")

	return router
}

// corsMiddleware CORS中间件
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// healthCheckHandler 健康检查
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	response := APIResponse{
		Success: true,
		Message: "服务运行正常",
		Data: map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
			"version":   "1.0.0",
			"service":   "timezone-saas-demo",
		},
	}
	respondJSON(w, http.StatusOK, response)
}

// apiDocsHandler API文档
func apiDocsHandler(w http.ResponseWriter, r *http.Request) {
	docs := map[string]interface{}{
		"title":       "SAAS多租户时区处理API",
		"version":     "1.0.0",
		"description": "演示如何优雅地处理多租户时区问题",
		"endpoints": map[string]interface{}{
			"/api/health":            "健康检查",
			"/api/timezone/demo":     "时区处理演示",
			"/api/timezone/merchants": "获取商户列表",
			"/api/timezone/orders":    "获取订单列表（支持时区转换）",
			"/api/timezone/analysis":  "获取分析数据（基于视图）",
			"/api/timezone/compare":   "时区对比分析",
		},
		"examples": map[string]string{
			"获取商户列表":     "/api/timezone/merchants",
			"获取订单（带时区）":   "/api/timezone/orders?timezone=Asia/Shanghai",
			"分析特定日期":     "/api/timezone/analysis?date=2024-08-19",
			"时区对比":       "/api/timezone/compare?utc_time=2024-08-19T00:00:00Z",
		},
	}

	response := APIResponse{
		Success: true,
		Message: "API文档",
		Data:    docs,
	}
	respondJSON(w, http.StatusOK, response)
}

// timezoneDemo 时区处理演示
func timezoneDemo(w http.ResponseWriter, r *http.Request) {
	demo, err := timezoneService.GetTimezoneDemo()
	if err != nil {
		response := APIResponse{
			Success: false,
			Message: "获取时区演示数据失败",
			Error:   err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, response)
		return
	}

	response := APIResponse{
		Success: true,
		Message: "时区处理演示数据",
		Data:    demo,
	}
	respondJSON(w, http.StatusOK, response)
}

// getMerchants 获取商户列表
func getMerchants(w http.ResponseWriter, r *http.Request) {
	merchants, err := timezoneService.GetMerchants()
	if err != nil {
		response := APIResponse{
			Success: false,
			Message: "获取商户列表失败",
			Error:   err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, response)
		return
	}

	response := APIResponse{
		Success: true,
		Message: fmt.Sprintf("获取到 %d 个商户", len(merchants)),
		Data:    merchants,
	}
	respondJSON(w, http.StatusOK, response)
}

// getOrders 获取订单列表
func getOrders(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	timezone := r.URL.Query().Get("timezone")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20 // 默认限制
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0 // 默认偏移
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	orders, err := timezoneService.GetOrders(timezone, limit, offset)
	if err != nil {
		response := APIResponse{
			Success: false,
			Message: "获取订单列表失败",
			Error:   err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, response)
		return
	}

	message := fmt.Sprintf("获取到 %d 条订单", len(orders))
	if timezone != "" {
		message += fmt.Sprintf("（时区: %s）", timezone)
	}

	response := APIResponse{
		Success: true,
		Message: message,
		Data:    orders,
	}
	respondJSON(w, http.StatusOK, response)
}

// getAnalysisData 获取分析数据
func getAnalysisData(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	analysis, err := timezoneService.GetAnalysisData(date)
	if err != nil {
		response := APIResponse{
			Success: false,
			Message: "获取分析数据失败",
			Error:   err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, response)
		return
	}

	response := APIResponse{
		Success: true,
		Message: fmt.Sprintf("获取 %s 的分析数据", date),
		Data:    analysis,
	}
	respondJSON(w, http.StatusOK, response)
}

// compareTimezones 时区对比分析
func compareTimezones(w http.ResponseWriter, r *http.Request) {
	utcTime := r.URL.Query().Get("utc_time")
	if utcTime == "" {
		utcTime = "2024-08-19T00:00:00Z"
	}

	comparison, err := timezoneService.CompareTimezones(utcTime)
	if err != nil {
		response := APIResponse{
			Success: false,
			Message: "时区对比分析失败",
			Error:   err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, response)
		return
	}

	response := APIResponse{
		Success: true,
		Message: fmt.Sprintf("UTC时间 %s 的全球时区对比", utcTime),
		Data:    comparison,
	}
	respondJSON(w, http.StatusOK, response)
}

// respondJSON 统一的JSON响应函数
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}