package store

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"
)

const dbSecretPrefix = "rsa-db:"

var (
	dbSecretPrivateKey *rsa.PrivateKey
	dbSecretPublicKey  *rsa.PublicKey
)

const dbSecretKeyPEM = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA4+c21AxlkdIt8CKoIVPTgPwKuSznQiXl2SDkMiPl8hy3C5Eh
G/ql4NmAq2mvgveIF0ZiWKDcxwRnIk/FMNPqbJrYhkb9vTw+3FHkg5LFYzJPtq0J
SayHM5c62/2OQ0/udrKBq5Dsu6B4RXDpaKP7sk/8ud6zczLQJLu2N2tdfOFTfw62
t5Jd2WgHaDdEXGKiP6TjDdkrLVjS0Rumjt1WdVxVVtyqQFzPMhkm7rMJ3DN33qQb
iX9Al/KY/ccpHKiqZW3AaEs8Ho1xaH9Hze9h9uPr3QYqdfP2uzcpF8Gq0vFBsvc7
gr9EH8PMssAPcYJNvm0RhNB108+xSvt9foDG/wIDAQABAoIBADinrfWg708E1O6x
buJ6GEYkYfYOt562FSGZD6F3Ux6RDOAPQA/Yi4wOBfKW307LgkVm9ePaeYfwDEN3
WSn+RHSjOdiHpWR5qZbTtN8QiYlTZIWhFoH+Jc4pdjRiIA+Tu1A+qAWijXdHOR7q
Jtwgh92YDNeYCTxGEYBQOcglJKR6V04UGbL/eREF76ak5v4Hd4vvOhqkfcLmnGPw
lp6vtmiF/nzcieSITf5mrlP4Pvt6Smn+1j1zaAkbsgrvTISCAmza8ZDQYwr1KnlE
0Wh7OEXk/E8ldeErLgetQ9Br+6bk5F17HZnwpkrKK3kfTdv/AcMJI2jYXgXnzSES
m6ro0SECgYEA9hwmdmKSdlD2LuTNxsPfj2onAsDzF/yaCMiEnCAnYBuvy4JOrWEC
nkXc3MHOz0HcROUclFJRJY4YBJbIFU7J1gd+Yc0bYIJb7j5ERp2kpibVpIu8YsV5
Ib/wPwHG2tCuWeGdkjqCf3/tDsyRFLZDQj6eGWtqFLVH1TrAMahiqGUCgYEA7Q/D
G9Qh+D57g7cxtt8evEt8ZgWlRGnvgugke4QreudGexPs+meFJI+kfQtpztB/9gC8
4vnvexHXItk98FuwlikN4MdHbhFZFlQJ7/JPHxpUAr4vPW84yT77DxRpXqPI620L
jKrhkJdImINFr3HSB8JC5PX0Ua9AmX0IwDdX8ZMCgYEAuKDjpdpK/+G63fEeAmf9
NeyvuVwgwjTpJX+wJCPnBi2fEu/9sAnf2faVOzNVv5wr769lYkvivma70+19yqZh
umPCxwIE8MC60J77v3ISC+eETL3bpMl6FvyT8eCWWp9EvP8Jo6KrNZU1tO14RW56
RJ8PIgi3+zMH4YoClv44jRUCgYAOioz5RAXhaFPDPJV8FiuYeTjkOSxuCeF7Mioq
uWzBWTZljk9W/MqZ94WrdevDl96BhIIRmisqbWm45YJ7H+SxEUucohyrj7zbNcR2
R3K7Aa5tjKTxK8Vb8tULk8Dy4TEN3955fnHfoKf/Uu4PWPf9KdlYmg2mhQ19XMIQ
qFRoqQKBgQC5FxjtAmCv6GxWGOpP6Qxr3tCyj8aXE1dNJDixB/JmRSjxJsdik7xK
6SgiZJP+f5w1FGIcL0ZikNu+navXQcqFOnFMd0BbkWdH1CbPgJtOcItIMNaZdyAI
PsML9TbGKZ89DWmvE4WuleIYyp9T26IcDwHL9uKH0QgLTnRp0RMsEA==
-----END RSA PRIVATE KEY-----`

func init() {
	block, _ := pem.Decode([]byte(dbSecretKeyPEM))
	if block == nil {
		panic("store: failed to decode embedded RSA key")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		panic(fmt.Sprintf("store: parse embedded RSA key: %v", err))
	}
	dbSecretPrivateKey = key
	dbSecretPublicKey = &key.PublicKey
}

func encryptSecretValue(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	if strings.HasPrefix(trimmed, dbSecretPrefix) {
		return trimmed, nil
	}
	cipher, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, dbSecretPublicKey, []byte(trimmed), nil)
	if err != nil {
		return "", fmt.Errorf("encrypt secret: %w", err)
	}
	return dbSecretPrefix + base64.StdEncoding.EncodeToString(cipher), nil
}

func decryptSecretValue(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	if !strings.HasPrefix(trimmed, dbSecretPrefix) {
		return trimmed, nil
	}
	raw, err := base64.StdEncoding.DecodeString(trimmed[len(dbSecretPrefix):])
	if err != nil {
		return "", fmt.Errorf("decode secret: %w", err)
	}
	plain, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, dbSecretPrivateKey, raw, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt secret: %w", err)
	}
	return string(plain), nil
}
