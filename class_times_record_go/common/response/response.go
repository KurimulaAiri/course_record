// Package response 统一响应封装
//
// 对齐 Java com.shiroko.repository.dto.ResponseDTO<T>
// 字段：code / message / data / requestTime
//
// 所有 Controller/Handler 返回此结构，JSON 序列化后与 Java 后端响应格式完全一致
package response

import (
	"encoding/json"
	"net/http"
	"time"
)

// ResponseDTO 统一响应结构
//
// 对齐 Java ResponseDTO<T>：
//   - code: 状态码（200=成功，400=失败，可自定义业务码）
//   - message: 消息描述
//   - data: 业务数据（泛型，Go 用 interface{}）
//   - requestTime: 请求时间（对象构造时自动填充，格式 "yyyy-MM-dd HH:mm:ss"）
type ResponseDTO struct {
	Code        int64       `json:"code"`                  // 状态码
	Message     string      `json:"message"`               // 消息
	Data        interface{} `json:"data"`                  // 业务数据
	RequestTime string      `json:"requestTime,omitempty"` // 请求时间（构造时填充）
}

// nowFormatted 返回当前时间字符串（对齐 Java DateTimeFormatter "yyyy-MM-dd HH:mm:ss"）
func nowFormatted() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// Success 成功响应（对齐 Java ResponseDTO.success(T data)）
//
// 参数：
//   - data: 业务数据
//
// 返回：code=200, message="success" 的响应
func Success(data interface{}) *ResponseDTO {
	return &ResponseDTO{
		Code:        200,
		Message:     "success",
		Data:        data,
		RequestTime: nowFormatted(),
	}
}

// SuccessWithMessage 成功响应（对齐 Java ResponseDTO.success(String message, T data)）
func SuccessWithMessage(message string, data interface{}) *ResponseDTO {
	return &ResponseDTO{
		Code:        200,
		Message:     message,
		Data:        data,
		RequestTime: nowFormatted(),
	}
}

// Fail 失败响应（对齐 Java ResponseDTO.fail(String message)）
//
// 返回：code=400 的失败响应
func Fail(message string) *ResponseDTO {
	return &ResponseDTO{
		Code:        400,
		Message:     message,
		Data:        nil,
		RequestTime: nowFormatted(),
	}
}

// FailWithCode 失败响应（对齐 Java ResponseDTO.fail(Long code, String message)）
//
// 参数：
//   - code: 自定义错误码（如 401=未授权，500=系统错误，1001=课时不足）
//   - message: 错误消息
func FailWithCode(code int64, message string) *ResponseDTO {
	return &ResponseDTO{
		Code:        code,
		Message:     message,
		Data:        nil,
		RequestTime: nowFormatted(),
	}
}

// FailWithData 失败响应（对齐 Java ResponseDTO.fail(String message, T data)）
func FailWithData(message string, data interface{}) *ResponseDTO {
	return &ResponseDTO{
		Code:        400,
		Message:     message,
		Data:        data,
		RequestTime: nowFormatted(),
	}
}

// WriteJSON 将响应以 JSON 格式写入 HTTP ResponseWriter
//
// 参数：
//   - w: HTTP ResponseWriter
//   - resp: 响应对象
func WriteJSON(w http.ResponseWriter, resp *ResponseDTO) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(http.StatusOK) // 业务层始终返回 200，错误码在 body 中
	json.NewEncoder(w).Encode(resp)
}

// 业务错误码常量（对齐 Java ResultCode 枚举）
const (
	CodeSuccess            = 200  // 成功
	CodeFail               = 400  // 失败
	CodeUnauthorized       = 401  // 未授权
	CodeNotFound           = 404  // 未找到
	CodeServerError        = 500  // 系统错误
	CodeCourseBalanceEmpty = 1001 // 课时余额不足
	CodeCourseExpired      = 1003 // 课时已过期
)
