package utils

import (
	"strings"
	"testing"
)

// 加密关闭时所有函数必须是原样透传（兼容未配置 ENCRYPTION_KEY 的既有部署）
func TestEncryptSecretDisabledPassthrough(t *testing.T) {
	SetEncryptionKey("")
	plain := "my-secret-token"
	if got := EncryptSecret(plain); got != plain {
		t.Fatalf("disabled mode must passthrough encrypt, got %q", got)
	}
	if got := DecryptSecret(plain); got != plain {
		t.Fatalf("disabled mode must passthrough decrypt, got %q", got)
	}
	if got := SecretIndex(plain); got != plain {
		t.Fatalf("disabled mode index must be the plaintext itself, got %q", got)
	}
}

// 启用加密后的完整往返：密文带前缀、与明文不同、解密还原、索引确定性且不等于明文
func TestEncryptSecretRoundTrip(t *testing.T) {
	SetEncryptionKey("unit-test-key")
	defer SetEncryptionKey("")

	plain := "my-secret-token"
	enc := EncryptSecret(plain)
	if !strings.HasPrefix(enc, secretEncPrefix) {
		t.Fatalf("ciphertext must carry %q prefix, got %q", secretEncPrefix, enc)
	}
	if enc == plain {
		t.Fatal("ciphertext must differ from plaintext")
	}
	if got := DecryptSecret(enc); got != plain {
		t.Fatalf("round trip mismatch: want %q got %q", plain, got)
	}

	// 确定性索引：同明文同索引、索引不含明文
	idx1 := SecretIndex(plain)
	idx2 := SecretIndex(plain)
	if idx1 != idx2 {
		t.Fatalf("index must be deterministic: %q vs %q", idx1, idx2)
	}
	if strings.Contains(idx1, plain) {
		t.Fatal("index must not contain the plaintext")
	}
}

// 防重入：对密文再次加密必须原样返回（前端回显密文后原样保存的场景）
func TestEncryptSecretNoDoubleEncryption(t *testing.T) {
	SetEncryptionKey("unit-test-key")
	defer SetEncryptionKey("")

	enc := EncryptSecret("secret")
	if again := EncryptSecret(enc); again != enc {
		t.Fatalf("encrypting ciphertext must be a no-op, got %q", again)
	}
}

// 密钥轮换/密文损坏时解密必须返回空串（校验场景走安全侧，宁可拒绝不可放行）
func TestDecryptSecretFailsClosed(t *testing.T) {
	SetEncryptionKey("key-a")
	enc := EncryptSecret("secret")

	SetEncryptionKey("key-b")
	if got := DecryptSecret(enc); got != "" {
		t.Fatalf("decryption with rotated key must return empty string, got %q", got)
	}

	SetEncryptionKey("")
	defer SetEncryptionKey("")
	if got := DecryptSecret(enc); got != "" {
		t.Fatalf("decryption with encryption disabled must return empty string, got %q", got)
	}
}

// 短密文或畸形密文不得触发切片越界 panic，必须 fail-closed 返回空串
func TestDecryptSecretShortOrMalformedInputs(t *testing.T) {
	SetEncryptionKey("unit-test-key")
	defer SetEncryptionKey("")

	cases := []string{
		secretEncPrefix,
		secretEncPrefix + "AAAA",
		secretEncPrefix + "!!!invalid-base64!!!",
	}
	for _, tc := range cases {
		if got := DecryptSecret(tc); got != "" {
			t.Fatalf("malformed ciphertext %q must return empty string, got %q", tc, got)
		}
	}
}
