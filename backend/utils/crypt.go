package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"sync"
)

// secretEncPrefix 标识密文（防止二次加密与误判明文）
const secretEncPrefix = "enc:v1:"

var (
	encryptionMu  sync.RWMutex
	encryptionKey []byte // nil = 未启用加密
)

// SetEncryptionKey 设置对称加密密钥（任意长度字符串，内部派生 32 字节 AES key）。
// 在 config 初始化时调用一次；不调用/传空 = 加密关闭，全部明文（兼容老部署）。
func SetEncryptionKey(key string) {
	encryptionMu.Lock()
	defer encryptionMu.Unlock()
	if key == "" {
		encryptionKey = nil
		return
	}
	digest := sha256.Sum256([]byte(key))
	encryptionKey = digest[:]
}

func cryptoEnabled() bool {
	encryptionMu.RLock()
	defer encryptionMu.RUnlock()
	return len(encryptionKey) == 32
}

func getEncryptionKey() []byte {
	encryptionMu.RLock()
	defer encryptionMu.RUnlock()
	out := make([]byte, len(encryptionKey))
	copy(out, encryptionKey)
	return out
}

// EncryptSecret 加密敏感凭据（AES-256-GCM），输出 "enc:v1:<base64>"。
// 未启用加密或输入已有前缀/为空时原样返回。
func EncryptSecret(plain string) string {
	if plain == "" || !cryptoEnabled() {
		return plain
	}
	if hasSecretPrefix(plain) {
		// 防止对密文二次加密（如前端回显后原样保存）
		return plain
	}
	key := getEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return plain
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return plain
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return plain
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return secretEncPrefix + base64.StdEncoding.EncodeToString(sealed)
}

// DecryptSecret 解密 EncryptSecret 的输出。无前缀 = 明文（未启用加密时期写入的
// 旧数据），原样返回；解密失败（key 轮换/密文损坏）返回空串——校验场景走安全侧。
func DecryptSecret(value string) string {
	if !hasSecretPrefix(value) {
		return value
	}
	if !cryptoEnabled() {
		return ""
	}
	raw, err := base64.StdEncoding.DecodeString(value[len(secretEncPrefix):])
	if err != nil {
		return ""
	}
	key := getEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return ""
	}
	return string(plain)
}

// SecretIndex 返回凭据的确定性索引值（HMAC-SHA256 hex），用于等值查询/唯一索引——
// 随机 nonce 的密文无法做查询条件。未启用加密时返回原文（明文模式自洽）。
func SecretIndex(value string) string {
	if !cryptoEnabled() {
		return value
	}
	key := getEncryptionKey()
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

// CryptoEnabled 报告是否启用存储加密
func CryptoEnabled() bool {
	return cryptoEnabled()
}

func hasSecretPrefix(s string) bool {
	return len(s) > len(secretEncPrefix) && s[:len(secretEncPrefix)] == secretEncPrefix
}
