// Package redis Redis 客户端封装
//
// 对齐 Java RedisConfig + ReactiveRedisTemplate / RedisTemplate
//
// 用途：
//   - Token 黑名单查询（Gateway 和 auth-service 共用）
//   - Nonce 防重放（SignInterceptor）
//   - 用户信息缓存（UserCacheService）
//   - 绑定 token 缓存（BindTokenCache）
//
// 依赖：github.com/redis/go-redis/v9
package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config Redis 配置（对齐 Java common-redis.yaml）
type Config struct {
	Addr     string `yaml:"addr" json:"addr"`         // Redis 地址（host:port）
	Password string `yaml:"password" json:"password"` // 密码
	DB       int    `yaml:"db" json:"db"`             // 数据库编号
}

// DefaultConfig 返回默认 Redis 配置
//
// 与 Java Nacos common-redis.yaml 一致
func DefaultConfig() *Config {
	return &Config{
		Addr:     "121.196.229.10:6379",
		Password: "shiroko114514",
		DB:       0,
	}
}

// NewRedis 创建 Redis 客户端
//
// 参数：
//   - cfg: Redis 配置
//
// 返回：
//   - Redis 客户端
//   - 错误信息
func NewRedis(cfg *Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis Ping 失败: %w", err)
	}

	log.Printf("✅ Redis 连接成功: %s DB=%d", cfg.Addr, cfg.DB)
	return client, nil
}

// ============================================================
// Token 黑名单（对齐 Java TokenBlacklistService）
// ============================================================

// BlacklistPrefix Token 黑名单 key 前缀（对齐 Java TokenBlacklistService）
//
// key 格式：cr:token:blacklist:{token}
// value：任意（用 "1"），TTL = token 剩余有效期
const BlacklistPrefix = "cr:token:blacklist:"

// AddToBlacklist 将 Token 加入黑名单（对齐 Java TokenBlacklistService.addToBlacklist）
//
// 用于登出时使 Token 立即失效
//
// 参数：
//   - client: Redis 客户端
//   - token: JWT Token
//   - ttl: 剩余有效期（过期后自动从黑名单移除）
func AddToBlacklist(client *redis.Client, token string, ttl time.Duration) error {
	key := BlacklistPrefix + token
	return client.Set(context.Background(), key, "1", ttl).Err()
}

// IsBlacklisted 检查 Token 是否在黑名单中（对齐 Java TokenBlacklistService.isBlacklisted）
//
// 参数：
//   - client: Redis 客户端
//   - token: JWT Token
//
// 返回：在黑名单中返回 true
func IsBlacklisted(client *redis.Client, token string) (bool, error) {
	key := BlacklistPrefix + token
	n, err := client.Exists(context.Background(), key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// ============================================================
// Nonce 防重放（对齐 Java SignInterceptor nonce 校验）
// ============================================================

// NoncePrefix nonce 缓存 key 前缀（对齐 Java SignInterceptor.NONCE_PREFIX）
const NoncePrefix = "nonce:"

// NonceTTL nonce 缓存有效期（对齐 Java SignInterceptor.TIME_OUT = 60秒）
const NonceTTL = 60 * time.Second

// SetNonceIfAbsent 设置 nonce（防重放）
//
// 对齐 Java redisCacheService.setIfAbsent("nonce:"+nonce, timestamp, 60s)
// 使用 SETNX 语义：若 nonce 已存在返回 false（重复提交）
//
// 参数：
//   - client: Redis 客户端
//   - nonce: 随机字符串
//
// 返回：
//   - true: nonce 设置成功（首次提交）
//   - false: nonce 已存在（重复提交）
func SetNonceIfAbsent(client *redis.Client, nonce string) (bool, error) {
	key := NoncePrefix + nonce
	// SETNX 语义：key 不存在时设置，返回 true；已存在返回 false
	ok, err := client.SetNX(context.Background(), key, "1", NonceTTL).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}

// ============================================================
// 绑定 token 缓存（对齐 Java BindTokenCache）
// ============================================================

// BindTokenPrefix 绑定 token 缓存 key 前缀（对齐 Java BindTokenCache）
const BindTokenPrefix = "bind:token:"

// BindTokenTTL 绑定 token 有效期（5 分钟，对齐 Java BindTokenCache）
const BindTokenTTL = 5 * time.Minute

// SetBindToken 存储绑定 token 信息
//
// 参数：
//   - client: Redis 客户端
//   - token: 绑定 token
//   - data: 绑定信息 JSON
func SetBindToken(client *redis.Client, token, data string) error {
	key := BindTokenPrefix + token
	return client.Set(context.Background(), key, data, BindTokenTTL).Err()
}

// GetBindToken 获取绑定 token 信息
func GetBindToken(client *redis.Client, token string) (string, error) {
	key := BindTokenPrefix + token
	return client.Get(context.Background(), key).Result()
}

// MarkBindTokenUsed 标记绑定 token 已使用
func MarkBindTokenUsed(client *redis.Client, token string) error {
	key := BindTokenPrefix + token
	// 已使用则删除（对齐 Java bindTokenCache.markUsed）
	return client.Del(context.Background(), key).Err()
}

// ============================================================
// 绑定码/订阅码缓存（用于 auth-service 绑定流程）
// ============================================================

// BindCodeTTL 绑定码/token 有效期（10 分钟）
const BindCodeTTL = 10 * time.Minute

// SetKeyValue 存储键值对到 Redis（通用方法，用于绑定码/token → studentID 映射）
//
// 参数：
//   - client: Redis 客户端
//   - key: Redis key（含前缀）
//   - value: 存储值（studentID 字符串）
//   - ttl: 过期时间
func SetKeyValue(client *redis.Client, key, value string, ttl time.Duration) error {
	return client.Set(context.Background(), key, value, ttl).Err()
}

// GetKeyValue 从 Redis 读取键值（通用方法）
//
// 参数：
//   - client: Redis 客户端
//   - key: Redis key（含前缀）
//
// 返回：值字符串，未找到返回空字符串和 nil error（通过 redis.Nil 判断）
func GetKeyValue(client *redis.Client, key string) (string, error) {
	return client.Get(context.Background(), key).Result()
}

// DeleteKey 删除 Redis key（标记绑定码/token 已使用）
//
// 参数：
//   - client: Redis 客户端
//   - key: Redis key（含前缀）
func DeleteKey(client *redis.Client, key string) error {
	return client.Del(context.Background(), key).Err()
}

// IsRedisNil 判断是否为 Redis key 不存在的错误
//
// 用途：区分 key 不存在（正常业务逻辑）和真正的 Redis 错误
func IsRedisNil(err error) bool {
	return err == redis.Nil
}
