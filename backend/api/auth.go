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
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	defaultLoginPassword   = "foreign_123_login"
	defaultEncryptionKey   = "AI_FTA::crypto::v1"
	defaultIssuer          = "ai-foreign-trade-assistant"
	encryptedPrefix        = "enc:"
	rsaPrefix              = "rsa:"
	tokenCookieName        = "fta_token"
	authHeaderBearerPrefix = "bearer "
)

// AuthConfig 聚合鉴权及加密相关配置。
type AuthConfig struct {
	Password      string
	EncryptionKey string
	JWTSecret     string
	TokenTTL      time.Duration
}

// AuthManager 负责登录校验、Token 生成与敏感数据解密。
type AuthManager struct {
	passwordHash [32]byte
	tokenTTL     time.Duration
	signerKey    []byte
	aead         cipher.AEAD

	privateKey   *rsa.PrivateKey
	publicKeyPEM string
	keyID        string
	keyGenerated time.Time
}

// PublicKeyPayload 用于向前端暴露的密钥信息。
type PublicKeyPayload struct {
	KeyID     string `json:"kid"`
	Algorithm string `json:"alg"`
	PublicKey string `json:"public_key"`
	Generated string `json:"generated_at"`
}

// NewAuthManager 构建鉴权管理器。
func NewAuthManager(cfg AuthConfig) (*AuthManager, error) {
	password := strings.TrimSpace(cfg.Password)
	if password == "" {
		password = defaultLoginPassword
		log.Printf("未配置 FTA_LOGIN_PASSWORD，使用默认口令，请尽快通过环境变量覆盖。")
	}
	passwordHash := sha256.Sum256([]byte(password))

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
		passwordHash: passwordHash,
		tokenTTL:     tokenTTL,
		signerKey:    signKey[:],
		aead:         aead,
		privateKey:   privateKey,
		publicKeyPEM: string(pemBytes),
		keyID:        keyID,
		keyGenerated: time.Now().UTC(),
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
	plain, err := a.decryptCipher(cipherText)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("解密登录口令失败: %w", err)
	}
	inputHash := sha256.Sum256([]byte(strings.TrimSpace(plain)))
	if subtle.ConstantTimeCompare(a.passwordHash[:], inputHash[:]) != 1 {
		return "", time.Time{}, fmt.Errorf("口令不正确")
	}
	return a.generateToken()
}

// ValidateToken 校验 token 并返回声明。
func (a *AuthManager) ValidateToken(tokenStr string) (*jwt.RegisteredClaims, error) {
	if a == nil {
		return nil, fmt.Errorf("鉴权模块未初始化")
	}
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("签名算法不匹配")
		}
		return a.signerKey, nil
	})
	if err != nil {
		return nil, err
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

// DecryptField 若字符串带有前缀则进行解密。
func (a *AuthManager) DecryptField(value string) (string, error) {
	if a == nil {
		return value, fmt.Errorf("鉴权模块未初始化")
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return value, nil
	}
	if strings.HasPrefix(trimmed, rsaPrefix) || strings.HasPrefix(trimmed, encryptedPrefix) {
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

func (a *AuthManager) generateToken() (string, time.Time, error) {
	if a == nil {
		return "", time.Time{}, fmt.Errorf("鉴权模块未初始化")
	}
	expiresAt := time.Now().Add(a.tokenTTL)
	claims := jwt.RegisteredClaims{
		Issuer:    defaultIssuer,
		Subject:   "operator",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
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
