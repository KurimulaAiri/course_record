// Package internal Go Gateway 内部实现
//
// 配置定义，与 Java application-dev.yml / Nacos cr-gateway.yaml 对齐
package internal

// ============================================================
// 配置结构体
// ============================================================

// Config Gateway 全局配置
type Config struct {
	// Gateway 监听端口（Java 侧为 9999）
	Port int `yaml:"port" json:"port" default:"9999"`

	// JWT 配置
	JWT JWTConfig `yaml:"jwt" json:"jwt"`

	// Redis 配置（与 Java common-redis.yaml 一致）
	Redis RedisConfig `yaml:"redis" json:"redis"`

	// 路由配置（与 Java application-dev.yml / Nacos cr-gateway.yaml 一致）
	Routes []RouteConfig `yaml:"routes" json:"routes"`
}

// JWTConfig JWT 配置
// 密钥与 Java JwtUtils.java / JwtAuthFilter.java 中硬编码的 SECRET_KEY 一致
type JWTConfig struct {
	// HMAC-SHA256 密钥（Java 侧硬编码，Go 侧保持一致以保证 Token 互通）
	SecretKey string `yaml:"secretKey" json:"secretKey"`
	// Access Token 过期时间（毫秒），Java 默认 5 分钟
	AccessExpiration int64 `yaml:"accessExpiration" json:"accessExpiration"`
	// Refresh Token 过期时间（毫秒），Java 默认 7 天
	RefreshExpiration int64 `yaml:"refreshExpiration" json:"refreshExpiration"`
}

// RedisConfig Redis 连接配置
type RedisConfig struct {
	Addr     string `yaml:"addr" json:"addr"`
	Password string `yaml:"password" json:"password"`
	DB       int    `yaml:"db" json:"db"`
}

// RouteConfig 路由配置
// 对应 Java Spring Cloud Gateway 的 route 配置
type RouteConfig struct {
	// 路由 ID（如 cr-auth-service）
	ID string `yaml:"id" json:"id"`
	// 目标服务地址（如 http://localhost:10002 或 lb://cr-auth-service）
	URI string `yaml:"uri" json:"uri"`
	// 路径前缀（如 /auth/**）
	Prefix string `yaml:"prefix" json:"prefix"`
	// 去除前缀层数（Java StripPrefix=1，Go 在转发时手动去除）
	StripPrefix int `yaml:"stripPrefix" json:"stripPrefix"`
}

// ============================================================
// 默认配置（与 Java 侧一致）
// ============================================================

// DefaultConfig 返回默认配置（本地开发环境）
// 与 Java application-dev.yml + common-redis.yaml 对齐
func DefaultConfig() *Config {
	return &Config{
		Port: 9999,
		JWT: JWTConfig{
			// 与 Java JwtUtils.java SECRET_KEY 完全一致（硬编码）
			SecretKey:        "shiroko_project_secret_key_at_least_32_chars_long",
			AccessExpiration: 5 * 60 * 1000,      // 5 分钟
			RefreshExpiration: 7 * 24 * 60 * 60 * 1000, // 7 天
		},
		Redis: RedisConfig{
			// 来自 Nacos common-redis.yaml
			Addr:     "121.196.229.10:6379",
			Password: "shiroko114514",
			DB:       0,
		},
		// 路由配置与 application-dev.yml 一致
		Routes: []RouteConfig{
			{
				ID:          "cr-auth-service",
				URI:         "http://localhost:10002",
				Prefix:      "/auth",
				StripPrefix: 1,
			},
			{
				ID:          "cr-business-service",
				URI:         "http://localhost:10001",
				Prefix:      "/biz",
				StripPrefix: 1,
			},
			{
				ID:          "cr-admin-service",
				URI:         "http://localhost:10003",
				Prefix:      "/admin",
				StripPrefix: 1,
			},
		},
	}
}
