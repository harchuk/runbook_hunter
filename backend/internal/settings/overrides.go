package settings

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/config"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/store"
	"gorm.io/gorm"
)

const globalOverrideKey = "global_settings"

type Service struct {
	repo    *store.Repository
	baseCfg config.Config
	key     []byte
}

func NewService(repo *store.Repository, baseCfg config.Config, keyB64 string) (*Service, error) {
	var key []byte
	if strings.TrimSpace(keyB64) != "" {
		decoded, err := base64.StdEncoding.DecodeString(keyB64)
		if err != nil {
			return nil, fmt.Errorf("decode settings crypto key: %w", err)
		}
		if len(decoded) != 32 {
			return nil, fmt.Errorf("settings crypto key must be 32 bytes (base64)")
		}
		key = decoded
	}
	return &Service{repo: repo, baseCfg: baseCfg, key: key}, nil
}

func (s *Service) GetOverrides(ctx context.Context) (map[string]any, error) {
	override, err := s.repo.GetOverride(ctx, globalOverrideKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(err.Error(), "record not found") {
			return map[string]any{}, nil
		}
		return nil, err
	}
	payload := override.Value
	if override.Encrypted {
		decrypted, err := s.decrypt(payload)
		if err != nil {
			return nil, err
		}
		payload = decrypted
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(payload), &out); err != nil {
		return nil, err
	}
	if out == nil {
		return map[string]any{}, nil
	}
	return out, nil
}

func (s *Service) PutOverrides(ctx context.Context, values map[string]any) error {
	blob, err := json.Marshal(values)
	if err != nil {
		return err
	}
	encryptedValue, err := s.encrypt(string(blob))
	if err != nil {
		return err
	}
	return s.repo.SetOverride(ctx, globalOverrideKey, encryptedValue, true)
}

func (s *Service) EffectiveConfig(ctx context.Context) (config.Config, error) {
	base := map[string]any{}
	baseBlob, err := json.Marshal(s.baseCfg)
	if err != nil {
		return config.Config{}, err
	}
	if err := json.Unmarshal(baseBlob, &base); err != nil {
		return config.Config{}, err
	}

	overrides, err := s.GetOverrides(ctx)
	if err != nil {
		return config.Config{}, err
	}
	merged := deepMerge(base, overrides)

	mergedBlob, err := json.Marshal(merged)
	if err != nil {
		return config.Config{}, err
	}
	var out config.Config
	if err := json.Unmarshal(mergedBlob, &out); err != nil {
		return config.Config{}, err
	}
	if err := out.Validate(); err != nil {
		return config.Config{}, err
	}
	return out, nil
}

func (s *Service) ResetAll(ctx context.Context) error {
	return s.repo.DeleteOverride(ctx, globalOverrideKey)
}

func (s *Service) ResetKeys(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	overrides, err := s.GetOverrides(ctx)
	if err != nil {
		return err
	}
	for _, key := range keys {
		deletePath(overrides, key)
	}
	return s.PutOverrides(ctx, overrides)
}

func deepMerge(base, override map[string]any) map[string]any {
	result := make(map[string]any, len(base))
	for key, value := range base {
		result[key] = value
	}
	for key, value := range override {
		if valueMap, ok := value.(map[string]any); ok {
			if currentMap, exists := result[key].(map[string]any); exists {
				result[key] = deepMerge(currentMap, valueMap)
				continue
			}
		}
		result[key] = value
	}
	return result
}

func deletePath(values map[string]any, path string) {
	parts := strings.Split(path, ".")
	current := values
	for i, part := range parts {
		if i == len(parts)-1 {
			delete(current, part)
			return
		}
		next, ok := current[part].(map[string]any)
		if !ok {
			return
		}
		current = next
	}
}

// Security notes: AES-256-GCM protects UI overrides at rest.
func (s *Service) encrypt(plaintext string) (string, error) {
	if len(s.key) != 32 {
		return "", fmt.Errorf("settings crypto key is not configured")
	}
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *Service) decrypt(encoded string) (string, error) {
	if len(s.key) != 32 {
		return "", fmt.Errorf("settings crypto key is not configured")
	}
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce := ciphertext[:nonceSize]
	data := ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
