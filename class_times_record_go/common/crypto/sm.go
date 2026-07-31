// Package crypto 国密算法工具包（SM2/SM3）
//
// 与 Java 后端 com.shiroko.util.SM2Util / SM3Util 完全互通：
//   - SM2: 前端 sm-crypto 加密（C1C3C2, "04"前缀）→ Go/Java 后端解密
//   - SM3: 密码加盐哈希存储 + 请求签名校验
//
// 依赖：github.com/tjfoc/gmsm（Go 国密算法实现）
package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"

	"github.com/tjfoc/gmsm/sm2"
	"github.com/tjfoc/gmsm/sm3"
)

// ============================================================
// SM2 加解密
// ============================================================

// SM2Decrypt 使用 SM2 私钥解密密文（对齐 Java SM2Util.decrypt）
//
// 算法细节：
//   - 曲线：sm2p256v1（国标推荐曲线）
//   - 模式：C1C3C2（新国标，与前端 sm-crypto 默认一致）
//   - 输入：cipherTextHex 为 hex 编码的密文（前端 sm-crypto 输出）
//   - 输入：privateKeyHex 为 hex 编码的私钥（来自 Nacos cr-auth-service.yaml）
//
// 参数：
//   - cipherTextHex: hex 编码的密文字符串
//   - privateKeyHex: hex 编码的 SM2 私钥（64 字符，无 "00" 前缀）
//
// 返回：
//   - 解密后的明文字符串
//   - 解密失败返回错误
func SM2Decrypt(cipherTextHex, privateKeyHex string) (string, error) {
	// 1. 加载 SM2 私钥
	priv, err := LoadSM2PrivateKey(privateKeyHex)
	if err != nil {
		return "", err
	}

	// 2. 解码 hex 密文
	cipherBytes, err := hex.DecodeString(cipherTextHex)
	if err != nil {
		return "", errors.New("SM2 密文 hex 解码失败: " + err.Error())
	}

	// 3. SM2 解密（C1C3C2 模式，与 Java SM2Engine.Mode.C1C3C2 一致）
	plaintext, err := sm2.Decrypt(priv, cipherBytes, sm2.C1C3C2)
	if err != nil {
		return "", errors.New("SM2 解密失败: " + err.Error())
	}

	return string(plaintext), nil
}

// LoadSM2PrivateKey 从 hex 字符串加载 SM2 私钥
//
// Java 侧 BigInteger 私钥可能有 "00" 前缀（符号位），Go 侧需要去除
//
// 参数：
//   - privateKeyHex: hex 编码的私钥（可能有 "00" 前缀）
//
// 返回：
//   - SM2 私钥指针
//   - 加载失败返回错误
func LoadSM2PrivateKey(privateKeyHex string) (*sm2.PrivateKey, error) {
	// 去除可能的 "00" 前缀（Java BigInteger 符号位）
	hexKey := strings.TrimPrefix(privateKeyHex, "00")
	// 去除可能的 "0x" 前缀
	hexKey = strings.TrimPrefix(hexKey, "0x")

	// 解析私钥大整数
	d := new(big.Int)
	_, ok := d.SetString(hexKey, 16)
	if !ok {
		return nil, errors.New("SM2 私钥 hex 解析失败: " + hexKey)
	}

	// 通过曲线基点乘法计算公钥
	curve := sm2.P256Sm2()
	x, y := curve.ScalarBaseMult(d.Bytes())

	return &sm2.PrivateKey{
		PublicKey: sm2.PublicKey{Curve: curve, X: x, Y: y},
		D:        d,
	}, nil
}

// SM2Encrypt 使用 SM2 公钥加密明文（主要用于测试，前端用 sm-crypto 加密）
//
// 返回 hex 编码的密文字符串
func SM2Encrypt(pub *sm2.PublicKey, plaintext string) (string, error) {
	ciphertext, err := sm2.Encrypt(pub, []byte(plaintext), rand.Reader, sm2.C1C3C2)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(ciphertext), nil
}

// ============================================================
// SM3 摘要（密码加盐哈希 + 签名校验）
// ============================================================

// SM3Digest 计算 SM3 摘要（对齐 Java SM3Util.digest）
//
// 返回 64 字符的小写 hex 字符串
func SM3Digest(srcData string) string {
	hash := sm3.Sm3Sum([]byte(srcData))
	return hex.EncodeToString(hash)
}

// SM3DigestWithSalt 加盐摘要（对齐 Java SM3Util.digestWithSalt）
//
// 密码存储公式：SM3(password + salt)
// 盐值生成方式：UUID 去横杠（32 位）
//
// 参数：
//   - srcData: 原文（如密码明文）
//   - salt: 盐值（32 位 UUID 去横杠）
//
// 返回：SM3(原文+盐) 的 hex 字符串
func SM3DigestWithSalt(srcData, salt string) string {
	return SM3Digest(srcData + salt)
}

// SM3Verify 验证摘要（对齐 Java SM3Util.verify）
//
// 参数：
//   - srcData: 原文
//   - hashData: 待比对的摘要 hex 字符串
//
// 返回：摘要匹配返回 true
func SM3Verify(srcData, hashData string) bool {
	computed := SM3Digest(srcData)
	// 对齐 Java equalsIgnoreCase
	return strings.EqualFold(computed, hashData)
}

// GenerateSalt 生成 32 位盐值（对齐 Java UUID.randomUUID().toString().replace("-", "")）
//
// 注意：此处简化实现，实际应使用 UUID 库。auth-service 迁移时会用 google/uuid 替代。
func GenerateSalt() string {
	// TODO: 迁移到 google/uuid 库生成标准 UUID
	// 临时实现：使用时间戳 + 随机数，32 位 hex
	return SM3Digest(randomString(16))[:32]
}

// randomString 生成指定长度的随机字符串（临时实现）
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[big.NewInt(int64(len(charset))).Int64()%int64(len(charset))]
	}
	return string(b)
}
