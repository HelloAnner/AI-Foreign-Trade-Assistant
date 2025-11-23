package api

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	defaultLoginPassword   = "foreign_123_login"
	DefaultLoginPassword   = defaultLoginPassword
	defaultEncryptionKey   = "AI_FTA::crypto::v1"
	defaultIssuer          = "ai-foreign-trade-assistant"
	encryptedPrefix        = "enc:"
	rsaPrefix              = "rsa:"
	ctrPrefix              = "ctr:"
	tokenCookieName        = "fta_token"
	authHeaderBearerPrefix = "bearer "
)

// HashLoginPassword 将明文口令做 SHA-256，并返回十六进制字符串。
func HashLoginPassword(plain string) (string, error) {
	trimmed := strings.TrimSpace(plain)
	if trimmed == "" {
		return "", fmt.Errorf("登录口令不能为空")
	}
	sum := sha256.Sum256([]byte(trimmed))
	return hex.EncodeToString(sum[:]), nil
}

func decodePasswordHash(hashHex string) ([32]byte, error) {
	var buf [32]byte
	trimmed := strings.TrimSpace(hashHex)
	if trimmed == "" {
		return buf, fmt.Errorf("登录口令哈希缺失")
	}
	raw, err := hex.DecodeString(trimmed)
	if err != nil {
		return buf, fmt.Errorf("登录口令哈希无效: %w", err)
	}
	if len(raw) != len(buf) {
		return buf, fmt.Errorf("登录口令哈希长度异常")
	}
	copy(buf[:], raw)
	return buf, nil
}

// AuthConfig 聚合鉴权及加密相关配置。
type AuthConfig struct {
	PasswordHash    string
	PasswordVersion int
	EncryptionKey   string
	JWTSecret       string
	TokenTTL        time.Duration
}

// AuthManager 负责登录校验、Token 生成与敏感数据解密。
type AuthManager struct {
	passwordHash    [32]byte
	passwordVersion int
	tokenTTL        time.Duration
	signerKey       []byte
	aead            cipher.AEAD

	mu sync.RWMutex

	privateKey   *rsa.PrivateKey
	publicKeyPEM string
	keyID        string
	keyGenerated time.Time
	// encryptionKey is stored for CTR mode decryption compatibility
	encryptionKey [32]byte
}

// PublicKeyPayload 用于向前端暴露的密钥信息。
type PublicKeyPayload struct {
	KeyID     string `json:"kid"`
	Algorithm string `json:"alg"`
	PublicKey string `json:"public_key"`
	Generated string `json:"generated_at"`
}

type tokenClaims struct {
	jwt.RegisteredClaims
	PasswordVersion int `json:"pwd_version"`
}

// NewAuthManager 构建鉴权管理器。
func NewAuthManager(cfg AuthConfig) (*AuthManager, error) {
	version := cfg.PasswordVersion
	if version <= 0 {
		version = 1
	}
	passwordHash, err := func() ([32]byte, error) {
		var zero [32]byte
		hashHex := strings.TrimSpace(cfg.PasswordHash)
		if hashHex != "" {
			return decodePasswordHash(hashHex)
		}
		fallback := strings.TrimSpace(os.Getenv("FTA_LOGIN_PASSWORD"))
		if fallback == "" {
			fallback = defaultLoginPassword
			log.Printf("未配置 FTA_LOGIN_PASSWORD，使用默认口令，请尽快修改登录口令。")
		}
		hashHex, err := HashLoginPassword(fallback)
		if err != nil {
			return zero, err
		}
		return decodePasswordHash(hashHex)
	}()
	if err != nil {
		return nil, fmt.Errorf("初始化登录口令失败: %w", err)
	}

	var aead cipher.AEAD
	encryptionSecret := strings.TrimSpace(cfg.EncryptionKey)
	if encryptionSecret == "" {
		encryptionSecret = os.Getenv("FTA_ENCRYPTION_KEY")
	}
	if encryptionSecret == "" {
		encryptionSecret = defaultEncryptionKey
		log.Printf("未配置 FTA_ENCRYPTION_KEY，使用默认加密密钥，请尽快通过环境变量覆盖。")
	}
	keyDigest := sha256.Sum256([]byte(encryptionSecret))
	if block, err := aes.NewCipher(keyDigest[:]); err == nil {
		if cipher, err := cipher.NewGCM(block); err == nil {
			aead = cipher
		}
	}

	jwtSecret := strings.TrimSpace(cfg.JWTSecret)
	if jwtSecret == "" {
		jwtSecret = os.Getenv("FTA_JWT_SECRET")
	}
	if jwtSecret == "" {
		jwtSecret = fmt.Sprintf("%s::jwt", encryptionSecret)
	}
	signKey := sha256.Sum256([]byte(jwtSecret))

	tokenTTL := cfg.TokenTTL
	if tokenTTL <= 0 {
		tokenTTL = 14 * 24 * time.Hour
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("生成 RSA 密钥失败: %w", err)
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("导出公钥失败: %w", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})
	sum := sha256.Sum256(publicDER)
	keyID := fmt.Sprintf("rsa-%x", sum[:4])

	return &AuthManager{
		passwordHash:    passwordHash,
		passwordVersion: version,
		tokenTTL:        tokenTTL,
		signerKey:       signKey[:],
		aead:            aead,
		privateKey:      privateKey,
		publicKeyPEM:    string(pemBytes),
		keyID:           keyID,
		keyGenerated:    time.Now().UTC(),
		encryptionKey:   keyDigest,
	}, nil
}

// Middleware 返回 chi 兼容的鉴权中间件。
func (a *AuthManager) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				writeJSON(w, http.StatusUnauthorized, Response{OK: false, Error: "未登录或登录已过期"})
				return
			}
			if _, err := a.ValidateToken(token); err != nil {
				writeJSON(w, http.StatusUnauthorized, Response{OK: false, Error: "登录状态无效，请重新登录"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// IssueToken 根据密文校验口令并生成 JWT。
func (a *AuthManager) IssueToken(cipherText string) (string, time.Time, error) {
	if a == nil {
		return "", time.Time{}, fmt.Errorf("鉴权模块未初始化")
	}
	if len(cipherText) > 0 {
		log.Printf("[DEBUG] IssueToken - 收到密文前缀: %s... (总长度: %d)", cipherText[:min(20, len(cipherText))], len(cipherText))
	} else {
		log.Printf("[DEBUG] IssueToken - 收到空密文")
	}
	plain, err := a.decryptCipher(cipherText)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("解密登录口令失败: %w", err)
	}
	log.Printf("[DEBUG] IssueToken - 解密后明文: %q", plain)
	trimmedPlain := strings.TrimSpace(plain)
	inputHash := sha256.Sum256([]byte(trimmedPlain))
	a.mu.RLock()
	expected := a.passwordHash
	version := a.passwordVersion
	a.mu.RUnlock()
	log.Printf("[DEBUG] IssueToken - 输入密码哈希: %x", inputHash)
	log.Printf("[DEBUG] IssueToken - 期望密码哈希: %x", expected)
	log.Printf("[DEBUG] IssueToken - 密码版本: %d", version)
	if subtle.ConstantTimeCompare(expected[:], inputHash[:]) != 1 {
		log.Printf("[DEBUG] IssueToken - ❌ 密码不匹配")
		return "", time.Time{}, fmt.Errorf("口令不正确")
	}
	log.Printf("[DEBUG] IssueToken - ✅ 密码匹配成功")
	return a.generateToken(version)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ValidateToken 校验 token 并返回声明。
func (a *AuthManager) ValidateToken(tokenStr string) (*tokenClaims, error) {
	if a == nil {
		return nil, fmt.Errorf("鉴权模块未初始化")
	}
	claims := &tokenClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("签名算法不匹配")
		}
		return a.signerKey, nil
	})
	if err != nil {
		return nil, err
	}
	a.mu.RLock()
	expectedVersion := a.passwordVersion
	a.mu.RUnlock()
	if claims.PasswordVersion != expectedVersion {
		return nil, fmt.Errorf("登录状态已失效")
	}
	return claims, nil
}

// PublicKey 返回当前 RSA 公钥信息。
func (a *AuthManager) PublicKey() *PublicKeyPayload {
	if a == nil || a.privateKey == nil {
		return nil
	}
	return &PublicKeyPayload{
		KeyID:     a.keyID,
		Algorithm: "RSA-OAEP-256",
		PublicKey: a.publicKeyPEM,
		Generated: a.keyGenerated.Format(time.RFC3339),
	}
}

// UpdatePassword 在内存中热更新登录口令哈希与版本。
func (a *AuthManager) UpdatePassword(hashHex string, version int) error {
	if a == nil {
		return fmt.Errorf("鉴权模块未初始化")
	}
	decoded, err := decodePasswordHash(hashHex)
	if err != nil {
		return err
	}
	if version <= 0 {
		version = 1
	}
	a.mu.Lock()
	a.passwordHash = decoded
	a.passwordVersion = version
	a.mu.Unlock()
	return nil
}

// DecryptField 若字符串带有 rsa: enc: ctr: 前缀则进行解密。
func (a *AuthManager) DecryptField(value string) (string, error) {
	if a == nil {
		return value, fmt.Errorf("鉴权模块未初始化")
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return value, nil
	}
	if strings.HasPrefix(trimmed, rsaPrefix) || strings.HasPrefix(trimmed, encryptedPrefix) || strings.HasPrefix(trimmed, ctrPrefix) {
		return a.decryptCipher(trimmed)
	}
	return value, nil
}

// EncryptField 使用 AES-GCM 对字段加密，返回 enc: 前缀的密文。
func (a *AuthManager) EncryptField(value string) (string, error) {
	if a == nil || a.aead == nil {
		return value, nil
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.HasPrefix(trimmed, encryptedPrefix) {
		return value, nil
	}
	nonce := make([]byte, a.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("生成随机数失败: %w", err)
	}
	cipher := a.aead.Seal(nil, nonce, []byte(trimmed), nil)
	buf := make([]byte, len(nonce)+len(cipher))
	copy(buf, nonce)
	copy(buf[len(nonce):], cipher)
	encoded := base64.StdEncoding.EncodeToString(buf)
	return encryptedPrefix + encoded, nil
}

// DecryptJSONFields 解密 JSON 中的敏感字段。
func (a *AuthManager) DecryptJSONFields(raw []byte, fields []string) ([]byte, error) {
	if a == nil || len(raw) == 0 {
		return raw, nil
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}
	changed := false
	processAll := len(fields) == 0
	if processAll {
		fields = make([]string, 0, len(payload))
		for key := range payload {
			fields = append(fields, key)
		}
	}
	for _, field := range fields {
		val, ok := payload[field].(string)
		if !ok || strings.TrimSpace(val) == "" {
			continue
		}
		decrypted, err := a.DecryptField(val)
		if err != nil {
			return nil, fmt.Errorf("%s 解密失败: %w", field, err)
		}
		if decrypted != val {
			payload[field] = decrypted
			changed = true
		}
	}
	if !changed {
		return raw, nil
	}
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(payload); err != nil {
		return nil, fmt.Errorf("重编码 JSON 失败: %w", err)
	}
	return bytes.TrimSpace(buf.Bytes()), nil
}

func (a *AuthManager) decryptCipher(value string) (string, error) {
	if a == nil {
		return "", fmt.Errorf("鉴权模块未初始化")
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("密文为空")
	}
	switch {
	case strings.HasPrefix(trimmed, rsaPrefix):
		return a.decryptRSA(strings.TrimPrefix(trimmed, rsaPrefix))
	case strings.HasPrefix(trimmed, encryptedPrefix):
		if a.aead == nil {
			return "", fmt.Errorf("服务器未启用 AES 解密")
		}
		return a.decryptAES(strings.TrimPrefix(trimmed, encryptedPrefix))
	case strings.HasPrefix(trimmed, ctrPrefix):
		return a.decryptCTR(strings.TrimPrefix(trimmed, ctrPrefix))
	default:
		if a.privateKey != nil {
			return a.decryptRSA(trimmed)
		}
		if a.aead != nil {
			return a.decryptAES(trimmed)
		}
		return "", fmt.Errorf("未知的密文格式")
	}
}

func (a *AuthManager) decryptAES(payload string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("Base64 解码失败: %w", err)
	}
	nonceSize := a.aead.NonceSize()
	if len(raw) <= nonceSize {
		return "", fmt.Errorf("密文长度不合法")
	}
	nonce := raw[:nonceSize]
	data := raw[nonceSize:]
	plain, err := a.aead.Open(nil, nonce, data, nil)
	if err != nil {
		return "", fmt.Errorf("AES-GCM 解密失败: %w", err)
	}
	return string(plain), nil
}

func (a *AuthManager) decryptRSA(payload string) (string, error) {
	if a.privateKey == nil {
		return "", fmt.Errorf("服务端未初始化 RSA")
	}
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("Base64 解码失败: %w", err)
	}
	plain, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, a.privateKey, raw, nil)
	if err != nil {
		return "", fmt.Errorf("RSA-OAEP 解密失败: %w", err)
	}
	return string(plain), nil
}

// decryptCTR decrypts AES-256-CTR encrypted ciphertext from crypto-js
// Format: base64(IV[16 bytes] + Ciphertext)
func (a *AuthManager) decryptCTR(payload string) (string, error) {
	// CTR mode shares the same key as AES-GCM (from EncryptionKey)
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("Base64 解码失败: %w", err)
	}

	// Extract IV (16 bytes for CTR mode)
	ivSize := 16
	if len(raw) <= ivSize {
		return "", fmt.Errorf("CTR 密文长度不合法")
	}

	iv := raw[:ivSize]
	ciphertext := raw[ivSize:]

	// Create AES cipher in CTR mode using the stored encryption key
	block, err := aes.NewCipher(a.encryptionKey[:])
	if err != nil {
		return "", fmt.Errorf("创建 AES cipher 失败: %w", err)
	}

	// Create CTR stream
	stream := cipher.NewCTR(block, iv)

	// Decrypt
	plaintext := make([]byte, len(ciphertext))
	stream.XORKeyStream(plaintext, ciphertext)

	return string(plaintext), nil
}

func (a *AuthManager) generateToken(version int) (string, time.Time, error) {
	if a == nil {
		return "", time.Time{}, fmt.Errorf("鉴权模块未初始化")
	}
	expiresAt := time.Now().Add(a.tokenTTL)
	claims := tokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    defaultIssuer,
			Subject:   "operator",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		PasswordVersion: version,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(a.signerKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("生成 token 失败: %w", err)
	}
	return tokenStr, expiresAt, nil
}

func extractToken(r *http.Request) string {
	if r == nil {
		return ""
	}
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(authHeader), authHeaderBearerPrefix) {
		return strings.TrimSpace(authHeader[len(authHeaderBearerPrefix):])
	}
	if cookie, err := r.Cookie(tokenCookieName); err == nil {
		return strings.TrimSpace(cookie.Value)
	}
	return ""
}
