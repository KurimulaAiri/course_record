// Package main Go Gateway 启动入口
//
// 替代 Java GatewayApplication.java
// 启动方式：go run ./gateway
//
// 对比 Java 启动：
//
//	java -Dspring.profiles.active=dev -jar gateway/target/gateway-1.0-SNAPSHOT.jar
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/kurimula-airi/course_record_go/gateway/internal"
)

// main Gateway 启动入口
// 职责：
//  1. 解析命令行参数（端口）
//  2. 加载默认配置（与 Java application-dev.yml 对齐）
//  3. 应用环境变量覆盖（生产部署使用）
//  4. 创建 Gateway 实例并启动 HTTP 服务
func main() {
	// 命令行参数：-port 指定监听端口，默认 9999（与 Java Gateway 一致）
	port := flag.Int("port", 9999, "Gateway 监听端口")
	flag.Parse()

	// 加载配置（默认本地开发配置，生产环境通过环境变量覆盖）
	cfg := internal.DefaultConfig()
	cfg.Port = *port

	// 环境变量覆盖（生产部署时使用）
	// GATEWAY_PORT：覆盖监听端口
	if envPort := os.Getenv("GATEWAY_PORT"); envPort != "" {
		log.Printf("使用环境变量 GATEWAY_PORT=%s", envPort)
	}
	// REDIS_ADDR：覆盖 Redis 地址（如生产环境 Redis 集群地址）
	if envRedisAddr := os.Getenv("REDIS_ADDR"); envRedisAddr != "" {
		cfg.Redis.Addr = envRedisAddr
		log.Printf("使用环境变量 REDIS_ADDR=%s", envRedisAddr)
	}

	// 创建 Gateway 实例（初始化 JWT 工具、Redis 客户端、路由表）
	gw := internal.NewGateway(cfg)

	// 启动 HTTP 服务
	addr := ":" + strconv.Itoa(cfg.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      gw,
		ReadTimeout:  30 * time.Second,  // 读取超时，防止慢速攻击
		WriteTimeout: 30 * time.Second,  // 写入超时
		IdleTimeout:  120 * time.Second, // 空闲连接超时
	}

	// 启动日志（便于调试和验证配置）
	log.Printf("🚀 Go Gateway 启动中，监听 %s", addr)
	log.Printf("   路由配置：")
	for _, route := range cfg.Routes {
		log.Printf("   - %s/** → %s (StripPrefix=%d)", route.Prefix, route.URI, route.StripPrefix)
	}
	log.Printf("   Redis: %s", cfg.Redis.Addr)
	log.Printf("   公开路径: %d 个", len(internal.PublicPaths()))

	// 启动服务（阻塞）
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("❌ Gateway 启动失败: %v", err)
	}
}
