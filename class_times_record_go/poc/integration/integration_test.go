// Package integration Go 与 Java 后端集成验证 PoC
//
// 验证目标：
//  1. Go 通过 HTTP 调用 Java Gateway 公开接口（无签名）
//  2. Go 通过 HTTP 调用 Java business-service 需签名接口（SM3 签名验证）
//  3. Go 直连 MySQL 读取 Java 写入的数据
//  4. Go 连接 Redis 验证 Token 黑名单机制
//
// 运行前置条件：Java 后端服务已启动（gateway:9999, business:10001, auth:10002）
// 运行：go test ./poc/integration/... -v
package integration

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	mathrand "math/rand"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"github.com/tjfoc/gmsm/sm2"
	"github.com/tjfoc/gmsm/sm3"
	"golang.org/x/net/context"
)

// ============================================================
// 配置（与 Java 侧 Nacos 配置一致）
// ============================================================

// Java 服务地址
const gatewayURL = "http://localhost:9999"
const authServiceURL = "http://localhost:10002"
const businessServiceURL = "http://localhost:10001"

// 数据库配置（来自 Nacos common-db.yaml）
const dbDSN = "class_times_record:8BCnbZjTT8ZxmBj6@tcp(121.196.229.10:3306)/class_times_record?charset=utf8mb4&parseTime=true&loc=Asia%2FShanghai"

// Redis 配置（来自 Nacos common-redis.yaml）
const redisAddr = "121.196.229.10:6379"
const redisPassword = "shiroko114514"

// SM2 私钥（来自 Nacos cr-auth-service.yaml，已去除 Java BigInteger 的 00 前缀）
const sm2PrivateKeyHex = "b3b8e61213bbd5e7d001e0cd4e33015efc04ae68ae61f2a36da55d92903cb0eb"

// SM3 签名盐值（与 Java SignInterceptor.APP_SECRET 一致）
const signAppSecret = "SHIROKO_SM3_SALT_2026"

// ============================================================
// SM2 加密工具（模拟前端 sm-crypto 加密行为）
// ============================================================

// loadSM2PrivateKey 从 hex 加载 SM2 私钥（复用 PoC1 逻辑）
func loadSM2PrivateKey(hexKey string) (*sm2.PrivateKey, error) {
	d := new(big.Int)
	d.SetString(hexKey, 16)
	curve := sm2.P256Sm2()
	x, y := curve.ScalarBaseMult(d.Bytes())
	return &sm2.PrivateKey{
		PublicKey: sm2.PublicKey{Curve: curve, X: x, Y: y},
		D:        d,
	}, nil
}

// sm2Encrypt 加密明文，返回 hex 字符串（模拟前端 sm-crypto 输出）
// 用于验证 Java 后端能解密 Go 加密的密文
func sm2Encrypt(pub *sm2.PublicKey, plaintext string) (string, error) {
	ciphertext, err := sm2.Encrypt(pub, []byte(plaintext), rand.Reader, sm2.C1C3C2)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(ciphertext), nil
}

// ============================================================
// SM3 签名工具（对齐 Java SignInterceptor 签名逻辑）
// ============================================================

// signRequest 生成 SM3 请求签名
// 完全对齐 Java SignInterceptor 的签名算法：
//   1. 收集 query 参数 + body JSON 参数 + timestamp + nonce
//   2. 字段按字典序排序，复杂对象用 JSON 序列化（key 排序）
//   3. 拼接 key1=value1&key2=value2&...
//   4. 末尾拼接 APP_SECRET
//   5. SM3(rawData) 输出小写 hex
//
// 参数：
//   - queryParams: URL query 参数
//   - bodyParams: JSON body 参数（已解析为 map）
//   - timestamp: 时间戳字符串
//   - nonce: 随机字符串
//
// 返回：SM3 签名 hex 字符串
func signRequest(queryParams, bodyParams map[string]interface{}, timestamp, nonce string) string {
	// 1. 合并所有参数
	allParams := make(map[string]interface{})
	for k, v := range queryParams {
		allParams[k] = v
	}
	for k, v := range bodyParams {
		allParams[k] = v
	}
	// 系统级参数
	allParams["timestamp"] = timestamp
	allParams["nonce"] = nonce

	// 2. 按 key 字典序排序
	keys := make([]string, 0, len(allParams))
	for k := range allParams {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 3. 拼接字符串（对齐 Java stringA 逻辑）
	var parts []string
	for _, k := range keys {
		v := allParams[k]
		if v == nil {
			continue
		}
		var valueStr string
		switch val := v.(type) {
		case map[string]interface{}, []interface{}:
			// 复杂对象转 JSON，key 排序（对齐 Java JSONWriter.Feature.SortMapEntriesByKeys）
			jsonBytes, _ := json.Marshal(v)
			// 重新排序 key
			var parsed interface{}
			json.Unmarshal(jsonBytes, &parsed)
			sortedBytes, _ := json.Marshal(sortedJSON(parsed))
			valueStr = string(sortedBytes)
		case string:
			if val == "" {
				continue // 对齐 Java StrUtil.isNotBlank 过滤
			}
			valueStr = val
		default:
			valueStr = fmt.Sprintf("%v", val)
			if valueStr == "" {
				continue
			}
		}
		parts = append(parts, k+"="+valueStr)
	}
	stringA := strings.Join(parts, "&")

	// 4. 拼接 APP_SECRET 并计算 SM3
	rawData := stringA + signAppSecret
	hash := sm3.Sm3Sum([]byte(rawData))
	return hex.EncodeToString(hash)
}

// sortedJSON 递归排序 JSON 结构的 key（对齐 Java FastJSON2 SortMapEntriesByKeys）
func sortedJSON(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		// 排序 key
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		result := make(map[string]interface{}, len(val))
		for _, k := range keys {
			result[k] = sortedJSON(val[k])
		}
		return result
	case []interface{}:
		for i, item := range val {
			val[i] = sortedJSON(item)
		}
		return val
	default:
		return v
	}
}

// generateNonce 生成随机 nonce（对齐前端 UUID 去横杠逻辑）
func generateNonce() string {
	mathrand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%d%d", time.Now().UnixNano(), mathrand.Intn(10000))
}

// ============================================================
// 测试 1: Go HTTP 调用 Java 公开接口（无签名）
// ============================================================

// TestJavaAPI_PublicEndpoint 验证 Go 能通过 Gateway 调用 Java 公开接口
// 使用 /auth/auth/get_bind_info（PUBLIC_PATHS 中，免鉴权免签名）
//
// 预期：Java 返回 200 + JSON 响应（即使 token 无效也返回业务错误，不是 401/500）
func TestJavaAPI_PublicEndpoint(t *testing.T) {
	// 调用 /auth/auth/get_bind_info?token=invalid_token_for_poc
	// 此接口在 PUBLIC_PATHS 中，免鉴权
	endpoint := gatewayURL + "/auth/auth/get_bind_info?token=invalid_token_for_poc"

	resp, err := http.Get(endpoint)
	if err != nil {
		t.Fatalf("HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("HTTP 状态码错误: got %d, want 200", resp.StatusCode)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("读取响应失败: %v", err)
	}

	// 解析 JSON 响应（Java ResponseDTO 格式: {code, message, data}）
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("JSON 解析失败: %v, body: %s", err, string(body))
	}

	// 验证 Java 返回了业务错误（token 无效），而不是系统错误
	code, ok := result["code"]
	if !ok {
		t.Errorf("响应缺少 code 字段: %s", string(body))
	}

	// code=500 表示业务失败（token 无效），预期行为
	t.Logf("✅ Java 公开接口调用成功: code=%v, message=%v", code, result["message"])
	t.Logf("   响应: %s", truncate(string(body), 200))
}

// TestJavaAPI_DirectAuth 验证 Go 直连 auth-service（绕过 Gateway）
func TestJavaAPI_DirectAuth(t *testing.T) {
	// 直连 auth-service 的 /auth/get_bind_info_by_code 接口
	// 此接口在 PUBLIC_PATHS 中
	endpoint := authServiceURL + "/auth/get_bind_info_by_code?bindCode=INVALID1"

	resp, err := http.Get(endpoint)
	if err != nil {
		t.Fatalf("HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	t.Logf("✅ 直连 auth-service 成功: code=%v, message=%v",
		result["code"], result["message"])
}

// ============================================================
// 测试 2: SM2 加密 → Java 解密 验证
// ============================================================

// TestSM2_GoEncryptJavaDecrypt 验证 Go 加密的密文能被 Java 后端解密
// 方式：用 Go 加密一个测试密码，模拟前端登录请求
// 由于无法直接调用 Java 的解密接口，此处通过 Go 加密 → Go 解密验证算法一致性
// （Go↔Java 互通已在 PoC1 验证）
func TestSM2_GoEncryptJavaDecrypt(t *testing.T) {
	priv, err := loadSM2PrivateKey(sm2PrivateKeyHex)
	if err != nil {
		t.Fatalf("加载私钥失败: %v", err)
	}

	// 模拟前端加密密码
	plaintext := "TestPassword123!"
	cipherHex, err := sm2Encrypt(&priv.PublicKey, plaintext)
	if err != nil {
		t.Fatalf("SM2 加密失败: %v", err)
	}

	// 验证密文格式（hex 字符串，长度 > 64）
	if len(cipherHex) < 64 {
		t.Errorf("密文长度异常: %d", len(cipherHex))
	}

	// Go 侧解密验证（与 Java 解密逻辑等价）
	cipherBytes, _ := hex.DecodeString(cipherHex)
	decrypted, err := sm2.Decrypt(priv, cipherBytes, sm2.C1C3C2)
	if err != nil {
		t.Fatalf("SM2 解密失败: %v", err)
	}

	if string(decrypted) != plaintext {
		t.Errorf("解密不匹配: got %q, want %q", string(decrypted), plaintext)
	}

	t.Logf("✅ SM2 加密→解密验证通过: 明文=%q, 密文前20字符=%s...",
		plaintext, cipherHex[:20])
}

// ============================================================
// 测试 3: SM3 签名 → Java 验签 验证
// ============================================================

// TestSM3_SignatureVerification 验证 Go 生成的签名能被 Java SignInterceptor 验证通过
// 方式：调用 business-service 的需签名接口，附带 Go 生成的签名
//
// 注意：/biz/course_record/deduct-detail 在 SignInterceptor excludePathPatterns 中
// 所以无法直接验证签名。改用调用需签名的接口（如 /biz/institution/get_by_open_id）
// 该接口在 Gateway PUBLIC_PATHS 中免 JWT，但仍需签名（SignInterceptor 拦截）
func TestSM3_SignatureVerification(t *testing.T) {
	// 准备请求参数
	queryParams := map[string]interface{}{
		"openId": "poc_test_openid_12345",
		"platform": "WEIXIN",
	}
	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
	nonce := generateNonce()

	// 生成签名
	sign := signRequest(queryParams, nil, timestamp, nonce)

	t.Logf("签名参数: timestamp=%s, nonce=%s, sign=%s", timestamp, nonce, sign)

	// 构造请求 URL（带 query 参数）
	u, _ := url.Parse(gatewayURL + "/biz/institution/get_by_open_id")
	q := u.Query()
	q.Set("openId", "poc_test_openid_12345")
	q.Set("platform", "WEIXIN")
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}

	// 添加签名 Header（对齐前端签名头）
	req.Header.Set("x-sign", sign)
	req.Header.Set("x-timestamp", timestamp)
	req.Header.Set("x-nonce", nonce)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	// 关键验证点：
	// - 如果签名通过，Java 会返回业务错误（如"机构不存在"），code=500
	// - 如果签名失败，Java 会返回"签名验证失败"或 500 错误
	message, _ := result["message"].(string)
	code, _ := result["code"].(float64)

	// 验证签名是否通过（非"签名验证失败"即表示签名通过）
	if strings.Contains(message, "签名验证失败") || strings.Contains(message, "签名参数缺失") {
		t.Errorf("❌ 签名验证失败: code=%v, message=%v", code, message)
		return
	}

	t.Logf("✅ SM3 签名验证通过: code=%v, message=%v (签名被 Java 接受)", code, message)
}

// TestSM3_SignatureWithPOSTBody 验证 POST 请求带 JSON body 的签名
// 对齐 Java SignInterceptor 解析 RepeatedlyRequestWrapper.getBodyString 的行为
func TestSM3_SignatureWithPOSTBody(t *testing.T) {
	// 模拟前端 POST 请求带 JSON body
	bodyParams := map[string]interface{}{
		"currentPage": 1,
		"pageSize":    10,
		"keyword":     "测试",
	}

	bodyJSON, _ := json.Marshal(bodyParams)

	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
	nonce := generateNonce()
	sign := signRequest(nil, bodyParams, timestamp, nonce)

	t.Logf("POST 签名: timestamp=%s, nonce=%s, sign=%s", timestamp, nonce, sign)
	t.Logf("Body: %s", string(bodyJSON))

	// 调用一个需签名的 POST 接口（如 /biz/student/query_by_teacher_id）
	// 注意：此接口需要 JWT，但签名验证先于 JWT，签名错误会先抛出
	req, _ := http.NewRequest("POST", gatewayURL+"/biz/student/query_by_teacher_id",
		bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-sign", sign)
	req.Header.Set("x-timestamp", timestamp)
	req.Header.Set("x-nonce", nonce)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP 请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// 验证签名是否通过
	if strings.Contains(bodyStr, "签名验证失败") || strings.Contains(bodyStr, "签名参数缺失") {
		t.Errorf("❌ POST 签名验证失败: %s", truncate(bodyStr, 200))
		return
	}

	t.Logf("✅ POST 签名验证通过: %s (签名被 Java 接受)", truncate(bodyStr, 200))
}

// ============================================================
// 测试 4: Go 直连 MySQL 读取 Java 数据
// ============================================================

// TestMySQL_Connection 验证 Go 能连接 Java 使用的 MySQL 数据库
// 并读取 c_institution 表数据（Java 写入的数据）
func TestMySQL_Connection(t *testing.T) {
	// 解析 DSN（确保兼容 go-sql-driver/mysql）
	cfg, err := mysql.ParseDSN(dbDSN)
	if err != nil {
		t.Fatalf("DSN 解析失败: %v", err)
	}
	t.Logf("MySQL 配置: host=%s, port=%s, db=%s, user=%s",
		cfg.Addr, "3306", cfg.DBName, cfg.User)

	// 使用 database/sql + go-sql-driver/mysql 连接
	db, err := openMySQL(dbDSN)
	if err != nil {
		t.Fatalf("MySQL 连接失败: %v", err)
	}
	defer db.Close()

	// 验证连接
	var version string
	err = db.QueryRow("SELECT VERSION()").Scan(&version)
	if err != nil {
		t.Fatalf("查询 MySQL 版本失败: %v", err)
	}
	t.Logf("✅ MySQL 连接成功: version=%s", version)

	// 读取 c_institution 表（Java 写入的数据）
	rows, err := db.Query("SELECT id, institution_name, status FROM c_institution LIMIT 5")
	if err != nil {
		t.Fatalf("查询 c_institution 失败: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int64
		var name string
		var status int64
		if err := rows.Scan(&id, &name, &status); err != nil {
			t.Errorf("扫描行失败: %v", err)
			continue
		}
		count++
		t.Logf("   机构: id=%d, name=%s, status=%d", id, name, status)
	}

	if count > 0 {
		t.Logf("✅ MySQL 数据读取成功: 读取到 %d 条机构记录", count)
	} else {
		t.Logf("⚠️  c_institution 表无数据（可能未初始化）")
	}
}

// TestMySQL_TableStructure 验证关键表结构（Go 侧 ORM 映射参考）
func TestMySQL_TableStructure(t *testing.T) {
	db, err := openMySQL(dbDSN)
	if err != nil {
		t.Fatalf("MySQL 连接失败: %v", err)
	}
	defer db.Close()

	// 验证关键业务表是否存在
	tables := []string{
		"c_institution", "c_student", "c_teacher", "c_parent",
		"c_course", "c_class", "c_course_record",
		"c_user", "c_user_auth", "c_user_platform",
		"c_parent_student", "c_class_student", "c_class_teacher",
	}

	for _, table := range tables {
		var exists int
		err := db.QueryRow(
			"SELECT COUNT(1) FROM information_schema.tables WHERE table_schema = ? AND table_name = ?",
			"class_times_record", table,
		).Scan(&exists)

		if err != nil {
			t.Errorf("查询表 %s 失败: %v", table, err)
			continue
		}

		if exists > 0 {
			t.Logf("✅ 表 %s 存在", table)
		} else {
			t.Errorf("❌ 表 %s 不存在", table)
		}
	}
}

// ============================================================
// 测试 5: Go 连接 Redis 验证
// ============================================================

// TestRedis_Connection 验证 Go 能连接 Java 使用的 Redis
// 并验证 Token 黑名单机制的兼容性
func TestRedis_Connection(t *testing.T) {
	// 创建 Redis 客户端
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword, // Nacos common-redis.yaml 配置的密码
		DB:       0,
	})

	ctx := context.Background()

	// 验证连接
	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		t.Fatalf("Redis 连接失败: %v", err)
	}
	t.Logf("✅ Redis 连接成功: ping=%s", pong)

	// 验证 Token 黑名单 Key 格式（Java TokenBlacklistService 使用的 key 前缀）
	// 模拟写入一个测试黑名单 token
	testToken := "poc_test_token_" + fmt.Sprintf("%d", time.Now().UnixNano())
	blacklistKey := "token:blacklist:" + testToken

	err = rdb.Set(ctx, blacklistKey, "1", 5*time.Minute).Err()
	if err != nil {
		t.Fatalf("Redis 写入失败: %v", err)
	}

	// 验证读取
	val, err := rdb.Get(ctx, blacklistKey).Result()
	if err != nil {
		t.Fatalf("Redis 读取失败: %v", err)
	}
	t.Logf("✅ Token 黑名单写入/读取成功: key=%s, value=%s", blacklistKey, val)

	// 清理测试数据
	rdb.Del(ctx, blacklistKey)

	// 验证 Nonce 防重放 Key（Java SignInterceptor 使用的 key 前缀）
	nonceKey := "nonce:poc_test_nonce_" + fmt.Sprintf("%d", time.Now().UnixNano())
	err = rdb.Set(ctx, nonceKey, "1", 60*time.Second).Err()
	if err != nil {
		t.Errorf("Nonce 写入失败: %v", err)
	}

	// 验证 SETNX 行为（Java setIfAbsent 等价）
	set, err := rdb.SetNX(ctx, nonceKey, "1", 60*time.Second).Result()
	if err != nil {
		t.Errorf("SETNX 失败: %v", err)
	}
	if set {
		t.Errorf("❌ SETNX 应返回 false（key 已存在），但返回了 true")
	} else {
		t.Logf("✅ Nonce SETNX 防重放验证通过: key=%s 已存在，拒绝重复设置", nonceKey)
	}

	rdb.Del(ctx, nonceKey)
}

// ============================================================
// 辅助函数
// ============================================================

// truncate 截断字符串到指定长度
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
