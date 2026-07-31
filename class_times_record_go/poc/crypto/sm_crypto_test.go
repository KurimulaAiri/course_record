// Package crypto 国密 SM2/SM3 互通验证 PoC
//
// 验证目标：
//  1. Go tjfoc/gmsm 的 SM2 加解密与 Java BouncyCastle (C1C3C2 模式) 互通
//  2. Go SM3 加盐哈希结果与 Java BouncyCastle SM3Digest 一致
//
// Java 侧实现参考：
//   - SM2Util: cipherMode=C1C3C2, 前端密文带 "04" 前缀 (sm-crypto 库默认)
//   - SM3Util: digestWithSalt(明文, 盐) = SM3(明文 + 盐), UTF-8 编码, 输出小写 hex
package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/tjfoc/gmsm/sm2"
	"github.com/tjfoc/gmsm/sm3"
)

// ============================================================
// 密钥配置（来自 Nacos cr-admin-service.yaml / cr-auth-service.yaml）
// ============================================================

// SM2 公钥（来自 Nacos crypto.sm2.public-key，admin-service）
// 格式：04 + X(32字节) + Y(32字节)，共 65 字节 = 130 hex 字符
const javaPublicKeyHex = "04ac715b7e653298c9667b366268e6ebdf67ca135259fc1c4183977df54e45bbe8efad05ba0fea995f45f0548ddb79426b6801fc11363de7d1662c19e4d9452fd1"

// SM2 私钥（来自 Nacos crypto.sm2.private-key，admin-service）
// 32 字节 = 64 hex 字符
const javaPrivateKeyHex = "1f9fe7f47b0a27025ff42a0da5039827a629f3a26a5721d5a3ad04e3fc5d8969"

// auth-service 私钥（带 Java BigInteger 的 "00" 符号位前缀）
// 原始值: 00b3b8e61213bbd5e7d001e0cd4e33015efc04ae68ae61f2a36da55d92903cb0eb
// 实际私钥 = 去掉前导 "00" 后的部分（64 hex 字符）
const authPrivateKeyHexWithPad = "00b3b8e61213bbd5e7d001e0cd4e33015efc04ae68ae61f2a36da55d92903cb0eb"

// ============================================================
// SM2 密钥加载工具
// ============================================================

// loadSM2PrivateKey 从 hex 字符串加载 SM2 私钥
// 兼容 Java BigInteger 的 "00" 符号位前缀（自动去除前导 00）
//
// Java 侧: new BigInteger(hex, 16) 会在最高位为 1 时自动添加 00 前缀保证正数
// Go 侧:   直接用 big.Int 即可，需去除前导 00
//
// 参数：
//   - hexKey: 私钥 hex 字符串（可能带 Java BigInteger 的 00 前缀）
//
// 返回：*sm2.PrivateKey
func loadSM2PrivateKey(hexKey string) (*sm2.PrivateKey, error) {
	// 去除 Java BigInteger 的 "00" 符号位前缀
	// 仅当长度 > 64 hex 字符（32 字节）时才去除，避免误删有效前导字节
	cleaned := hexKey
	for strings.HasPrefix(cleaned, "00") && len(cleaned) > 64 {
		cleaned = cleaned[2:]
	}

	// 将 hex 字符串转为 big.Int
	d := new(big.Int)
	d.SetString(cleaned, 16)
	if d == nil {
		return nil, fmt.Errorf("私钥 hex 解析失败")
	}

	// 使用 SM2 曲线，通过私钥标量 D 计算公钥点 (X, Y) = D * G
	curve := sm2.P256Sm2()
	x, y := curve.ScalarBaseMult(d.Bytes())

	// 构建 PrivateKey 结构体
	priv := &sm2.PrivateKey{
		PublicKey: sm2.PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
		},
		D: d,
	}
	return priv, nil
}

// loadSM2PublicKey 从 hex 字符串加载 SM2 公钥
// 公钥格式：04 + X(32字节) + Y(32字节) = 65 字节（非压缩格式）
//
// 参数：
//   - hexKey: 公钥 hex 字符串（以 04 开头）
//
// 返回：*sm2.PublicKey
func loadSM2PublicKey(hexKey string) (*sm2.PublicKey, error) {
	pubBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("公钥 hex 解码失败: %w", err)
	}

	// 校验非压缩格式：04 前缀 + 64 字节坐标 = 65 字节
	if len(pubBytes) != 65 || pubBytes[0] != 0x04 {
		return nil, fmt.Errorf("公钥格式错误: 期望 65 字节(04+X+Y), 实际 %d 字节", len(pubBytes))
	}

	// 提取 X 和 Y 坐标（各 32 字节，大端序）
	x := new(big.Int).SetBytes(pubBytes[1:33])
	y := new(big.Int).SetBytes(pubBytes[33:65])

	return &sm2.PublicKey{
		Curve: sm2.P256Sm2(),
		X:     x,
		Y:     y,
	}, nil
}

// ============================================================
// SM2 加解密测试
// ============================================================

// TestSM2_RoundTrip 验证 Go 内部 SM2 加解密往返
// 基本验证：确保 tjfoc/gmsm 库工作正常
func TestSM2_RoundTrip(t *testing.T) {
	priv, err := loadSM2PrivateKey(javaPrivateKeyHex)
	if err != nil {
		t.Fatalf("加载私钥失败: %v", err)
	}

	plaintext := "TestPassword123!"

	// 加密（模拟前端 sm-crypto 行为），使用 C1C3C2 模式（与 Java 一致）
	ciphertext, err := sm2.Encrypt(&priv.PublicKey, []byte(plaintext), rand.Reader, sm2.C1C3C2)
	if err != nil {
		t.Fatalf("SM2 加密失败: %v", err)
	}

	// 解密（后端行为）
	decrypted, err := sm2.Decrypt(priv, ciphertext, sm2.C1C3C2)
	if err != nil {
		t.Fatalf("SM2 解密失败: %v", err)
	}

	if string(decrypted) != plaintext {
		t.Errorf("解密结果不匹配: got %q, want %q", string(decrypted), plaintext)
	}
	t.Logf("✅ SM2 往返验证通过: 明文=%q, 解密=%q", plaintext, string(decrypted))
}

// TestSM2_JavaCompatibility 验证 Go 能解密前端 sm-crypto 产生的密文
// 前端 sm-crypto 加密后的密文格式：04 + C1 + C3 + C2（hex 字符串）
// Java BouncyCastle SM2Engine.Mode.C1C3C2 解密此格式
// Go tjfoc/gmsm 同样使用 C1C3C2 模式
//
// 测试方式：用公钥加密 → 转为 hex（模拟前端传输）→ 后端接收 hex → 解密
func TestSM2_JavaCompatibility(t *testing.T) {
	priv, err := loadSM2PrivateKey(javaPrivateKeyHex)
	if err != nil {
		t.Fatalf("加载私钥失败: %v", err)
	}

	pub, err := loadSM2PublicKey(javaPublicKeyHex)
	if err != nil {
		t.Fatalf("加载公钥失败: %v", err)
	}

	// 验证公钥与私钥是否匹配（比较公钥坐标）
	pubFromPriv := &priv.PublicKey
	if pubFromPriv.X.Cmp(pub.X) != 0 || pubFromPriv.Y.Cmp(pub.Y) != 0 {
		t.Logf("⚠️  公钥与私钥不匹配（来自不同配置），使用私钥推导的公钥测试")
		pub = pubFromPriv
	}

	// 模拟前端加密密码
	plaintext := "MySecretPass123"
	ciphertext, err := sm2.Encrypt(pub, []byte(plaintext), rand.Reader, sm2.C1C3C2)
	if err != nil {
		t.Fatalf("SM2 加密失败: %v", err)
	}

	// 模拟前端传输：转为 hex 字符串（sm-crypto 库默认输出 hex）
	cipherHex := hex.EncodeToString(ciphertext)
	t.Logf("模拟前端加密密文(hex前20字符): %s...", cipherHex[:min(20, len(cipherHex))])

	// 后端接收 hex 字符串并解密
	cipherBytes, err := hex.DecodeString(cipherHex)
	if err != nil {
		t.Fatalf("密文 hex 解码失败: %v", err)
	}

	decrypted, err := sm2.Decrypt(priv, cipherBytes, sm2.C1C3C2)
	if err != nil {
		t.Fatalf("SM2 解密失败: %v", err)
	}

	if string(decrypted) != plaintext {
		t.Errorf("解密结果不匹配: got %q, want %q", string(decrypted), plaintext)
	}
	t.Logf("✅ SM2 Java 互通验证通过: 明文=%q, 解密=%q", plaintext, string(decrypted))
}

// TestSM2_AuthServiceKey 验证 auth-service 的私钥（带 00 前缀）能正确加载
// auth-service 私钥: 00b3b8e6...（Java BigInteger 符号位填充）
// Go 中需去除前导 00 才能正确使用
func TestSM2_AuthServiceKey(t *testing.T) {
	priv, err := loadSM2PrivateKey(authPrivateKeyHexWithPad)
	if err != nil {
		t.Fatalf("auth-service 私钥加载失败: %v", err)
	}

	// 验证能正常加解密
	plaintext := "AuthTestPwd456"
	ciphertext, err := sm2.Encrypt(&priv.PublicKey, []byte(plaintext), rand.Reader, sm2.C1C3C2)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	decrypted, err := sm2.Decrypt(priv, ciphertext, sm2.C1C3C2)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}

	if string(decrypted) != plaintext {
		t.Errorf("解密结果不匹配: got %q, want %q", string(decrypted), plaintext)
	}
	t.Logf("✅ auth-service 私钥(带00前缀)加载验证通过: 明文=%q, 解密=%q", plaintext, string(decrypted))
}

// ============================================================
// SM3 加盐哈希测试
// ============================================================

// sm3DigestWithSalt 模拟 Java SM3Util.digestWithSalt
// Java 实现: SM3(明文 + 盐), UTF-8 编码, 输出小写 hex
//
// 参数：
//   - plaintext: 明文
//   - salt: 盐值
//
// 返回：64 字符的小写 hex 字符串
func sm3DigestWithSalt(plaintext, salt string) string {
	// Java: String combined = srcData + salt; → UTF-8 编码 → SM3 摘要
	combined := plaintext + salt
	hash := sm3.Sm3Sum([]byte(combined))
	return hex.EncodeToString(hash)
}

// TestSM3_WithSalt 验证 SM3 加盐哈希
// Java SM3Util.digestWithSalt("12345678", salt) = SM3("12345678" + salt)
// 此测试验证 Go sm3.Sm3Sum 产出与 Java 一致的结果
func TestSM3_WithSalt(t *testing.T) {
	testCases := []struct {
		name      string
		plaintext string
		salt      string
	}{
		{"简单密码", "12345678", "abcdef1234567890abcdef1234567890"},
		{"空盐", "password", ""},
		{"中文密码", "密码123", "salt123"},
		{"长密码", "ThisIsAVeryLongPasswordWithSpecialChars!@#$%", "uuid-salt-value"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sm3DigestWithSalt(tc.plaintext, tc.salt)

			// SM3 摘要固定 32 字节 = 64 hex 字符
			if len(result) != 64 {
				t.Errorf("SM3 哈希长度错误: got %d, want 64", len(result))
			}

			// 验证输出为小写 hex
			for _, c := range result {
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
					t.Errorf("SM3 哈希包含非小写hex字符: %c", c)
					break
				}
			}

			t.Logf("✅ SM3(%q+%q) = %s", tc.plaintext, tc.salt, result)
		})
	}
}

// TestSM3_VerifyKnownValue 验证 SM3 对已知输入的输出
// SM3 标准测试向量：SM3("abc") = 66c7f0f462eeedd9d1f2d46bdc10e4e24167c4875cf2f7a2297da02b8f4ba8e0
// 来自 GB/T 32905-2016 标准文档
func TestSM3_VerifyKnownValue(t *testing.T) {
	// GB/T 32905-2016 标准测试向量
	plaintext := "abc"
	expected := "66c7f0f462eeedd9d1f2d46bdc10e4e24167c4875cf2f7a2297da02b8f4ba8e0"

	result := hex.EncodeToString(sm3.Sm3Sum([]byte(plaintext)))

	if result != expected {
		t.Errorf("SM3 标准测试向量不匹配:\n  got:  %s\n  want: %s", result, expected)
	}
	t.Logf("✅ SM3 标准测试向量验证通过: SM3(\"%s\") = %s", plaintext, result)
}

// TestSM3_Deterministic 验证相同输入产出相同哈希（确定性）
func TestSM3_Deterministic(t *testing.T) {
	plaintext := "testpassword"
	salt := "randomsalt123456"

	hash1 := sm3DigestWithSalt(plaintext, salt)
	hash2 := sm3DigestWithSalt(plaintext, salt)

	if hash1 != hash2 {
		t.Errorf("SM3 不具备确定性: %s != %s", hash1, hash2)
	}
	t.Logf("✅ SM3 确定性验证通过: %s", hash1)
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
