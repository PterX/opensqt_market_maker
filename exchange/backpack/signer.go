package backpack

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const defaultWindow uint64 = 5000

type Signer struct {
	apiKey     string
	privateKey ed25519.PrivateKey
	window     uint64
}

func NewSigner(apiKey, secretKey string) (*Signer, error) {
	decodedKey, err := base64.StdEncoding.DecodeString(secretKey)
	if err != nil {
		return nil, fmt.Errorf("解析 Backpack SecretKey 失败: %w", err)
	}

	var privateKey ed25519.PrivateKey
	switch len(decodedKey) {
	case ed25519.SeedSize:
		privateKey = ed25519.NewKeyFromSeed(decodedKey)
	case ed25519.PrivateKeySize:
		privateKey = ed25519.PrivateKey(decodedKey)
	default:
		return nil, fmt.Errorf("Backpack SecretKey 长度无效: %d", len(decodedKey))
	}

	return &Signer{
		apiKey:     apiKey,
		privateKey: privateKey,
		window:     defaultWindow,
	}, nil
}

func (s *Signer) SignedHeaders(instruction string, query map[string]string, body map[string]any) (map[string]string, error) {
	timestamp := time.Now().UnixMilli()
	payload := s.buildPayload(instruction, query, body, timestamp)
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(s.privateKey, []byte(payload)))

	return map[string]string{
		"X-API-Key":   s.apiKey,
		"X-Signature": signature,
		"X-Timestamp": strconv.FormatInt(timestamp, 10),
		"X-Window":    strconv.FormatUint(s.window, 10),
	}, nil
}

func (s *Signer) SubscribeSignature() ([]string, error) {
	timestamp := time.Now().UnixMilli()
	payload := fmt.Sprintf("instruction=subscribe&timestamp=%d&window=%d", timestamp, s.window)
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(s.privateKey, []byte(payload)))

	return []string{
		s.apiKey,
		signature,
		strconv.FormatInt(timestamp, 10),
		strconv.FormatUint(s.window, 10),
	}, nil
}

func (s *Signer) buildPayload(instruction string, query map[string]string, body map[string]any, timestamp int64) string {
	parts := []string{fmt.Sprintf("instruction=%s", instruction)}

	if serializedQuery := serializeQuery(query); serializedQuery != "" {
		parts = append(parts, serializedQuery)
	} else if serializedBody := serializeBody(body); serializedBody != "" {
		parts = append(parts, serializedBody)
	}

	parts = append(parts,
		fmt.Sprintf("timestamp=%d", timestamp),
		fmt.Sprintf("window=%d", s.window),
	)

	return strings.Join(parts, "&")
}

func serializeQuery(query map[string]string) string {
	if len(query) == 0 {
		return ""
	}

	keys := make([]string, 0, len(query))
	for key := range query {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, query[key]))
	}

	return strings.Join(parts, "&")
}

func serializeBody(body map[string]any) string {
	if len(body) == 0 {
		return ""
	}

	keys := make([]string, 0, len(body))
	for key := range body {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, stringifyPayloadValue(body[key])))
	}

	return strings.Join(parts, "&")
}

func stringifyPayloadValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case bool:
		return strconv.FormatBool(typed)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case uint32:
		return strconv.FormatUint(uint64(typed), 10)
	case uint64:
		return strconv.FormatUint(typed, 10)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprintf("%v", typed)
	}
}
