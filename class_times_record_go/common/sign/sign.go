// Package sign 请求签名工具
//
// 对齐 Java com.shiroko.interceptor.SignInterceptor 的签名算法
//
// 签名规则（与前端 stableStringify 完全一致）：
//  1. 收集参与签名的参数：URL Query + Body JSON + timestamp + nonce
//  2. 过滤 null 和空白值
//  3. 按 key 字典序排序
//  4. 复杂对象（Map/Array）用 JSON 序列化（key 排序，对齐 Java JSONWriter.SortMapEntriesByKeys）
//  5. 拼接 "k1=v1&k2=v2&..."
//  6. 末尾拼接 APP_SECRET
//  7. SM3(rawData) 输出小写 hex
//
// APP_SECRET 必须与前端和 Java SignInterceptor 完全一致
package sign

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/tjfoc/gmsm/sm3"
)

// APP_SECRET 签名密钥（对齐 Java SignInterceptor.APP_SECRET）
//
// 必须与前端 src/utils/sign.ts 和 Java SignInterceptor 完全一致
const APP_SECRET = "SHIROKO_SM3_SALT_2026"

// SignRequest 生成请求签名（对齐 Java SignInterceptor 签名算法）
//
// 参数：
//   - queryParams: URL query 参数 map（每个 key 取第一个值，对齐 Java v[0]）
//   - bodyParams: JSON body 解析后的 map（仅支持对象格式，对齐 Java parseObject）
//   - timestamp: 时间戳字符串（毫秒）
//   - nonce: 随机字符串
//
// 返回：SM3 签名 hex 字符串（小写）
func SignRequest(queryParams, bodyParams map[string]interface{}, timestamp, nonce string) string {
	// 1. 合并所有参与签名的参数
	allParams := make(map[string]interface{})

	// 收集 Query 参数（对齐 Java request.getParameterMap 取 v[0]）
	for k, v := range queryParams {
		allParams[k] = v
	}

	// 收集 Body 参数（对齐 Java JSON.parseObject(bodyString) 后 putAll）
	for k, v := range bodyParams {
		allParams[k] = v
	}

	// 系统级参数（对齐 Java allParams.put("timestamp", ...), put("nonce", ...)）
	allParams["timestamp"] = timestamp
	allParams["nonce"] = nonce

	// 2. 按 key 字典序排序（对齐 Java Map.Entry.comparingByKey）
	keys := make([]string, 0, len(allParams))
	for k := range allParams {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 3. 拼接字符串 stringA（对齐 Java stringA 构造逻辑）
	var parts []string
	for _, k := range keys {
		v := allParams[k]
		if v == nil {
			continue // 对齐 Java null 过滤
		}

		var valueStr string
		switch val := v.(type) {
		case map[string]interface{}, []interface{}:
			// 复杂对象转 JSON，key 排序（对齐 Java JSON.toJSONString(value, SortMapEntriesByKeys)）
			sortedBytes, _ := json.Marshal(sortedJSON(v))
			valueStr = string(sortedBytes)
		case string:
			// 对齐 Java StrUtil.isNotBlank 过滤空白字符串
			if strings.TrimSpace(val) == "" {
				continue
			}
			valueStr = val
		case float64:
			// JSON 解析的数字默认是 float64
			valueStr = formatNumber(val)
		case int, int64, int32:
			valueStr = fmt.Sprintf("%v", val)
		case bool:
			valueStr = fmt.Sprintf("%v", val)
		default:
			valueStr = fmt.Sprintf("%v", val)
			if valueStr == "" {
				continue
			}
		}

		parts = append(parts, k+"="+valueStr)
	}
	stringA := strings.Join(parts, "&")

	// 4. 末尾拼接 APP_SECRET 并计算 SM3（对齐 Java rawData = stringA + APP_SECRET）
	rawData := stringA + APP_SECRET
	hash := sm3.Sm3Sum([]byte(rawData))
	return hex.EncodeToString(hash)
}

// VerifyRequest 验证请求签名
//
// 参数：
//   - queryParams: URL query 参数
//   - bodyParams: JSON body 参数
//   - timestamp: 时间戳字符串
//   - nonce: 随机字符串
//   - clientSign: 客户端传来的签名（x-sign header）
//
// 返回：签名匹配返回 true
func VerifyRequest(queryParams, bodyParams map[string]interface{}, timestamp, nonce, clientSign string) bool {
	if clientSign == "" {
		return false
	}
	computed := SignRequest(queryParams, bodyParams, timestamp, nonce)
	// 对齐 Java SM3Util.verify 的 equalsIgnoreCase
	return strings.EqualFold(computed, clientSign)
}

// sortedJSON 递归排序 JSON 结构的 key（对齐 Java FastJSON2 SortMapEntriesByKeys）
//
// 作用：确保复杂对象序列化时 key 顺序固定，与 Java/前端 stableStringify 一致
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
		// 数组递归排序每个元素
		for i, item := range val {
			val[i] = sortedJSON(item)
		}
		return val
	default:
		return v
	}
}

// formatNumber 格式化数字（去除 float64 多余的小数位）
//
// JSON 解析时数字默认是 float64，如 1 会被解析为 1.0
// 签名时需还原为整数形式，与前端/Java 保持一致
func formatNumber(f float64) string {
	// 如果是整数，去除小数部分
	if f == float64(int64(f)) {
		return fmt.Sprintf("%d", int64(f))
	}
	return fmt.Sprintf("%v", f)
}
