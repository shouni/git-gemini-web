package config

import (
	"fmt"

	"github.com/shouni/go-utils/envutil"
	"github.com/shouni/go-utils/text"
	"github.com/shouni/netarmor/securenet"
)

// IsSecureServiceURL は。サービス通信に安全なスキーム (HTTPS など) を使用しているかどうかを確認します。
func (c *Config) IsSecureServiceURL(rawURL string) bool {
	return securenet.IsSecureServiceURL(rawURL)
}

// ValidateEssentialConfig は設定バリデーションを行います。
func ValidateEssentialConfig(cfg *Config) error {
	if !cfg.IsSecureServiceURL(cfg.ServiceURL) {
		return fmt.Errorf("security error: SERVICE_URL ('%s') must be HTTPS in production", cfg.ServiceURL)
	}

	if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" || cfg.SessionSecret == "" {
		return fmt.Errorf("configuration error: OAuth settings are missing")
	}

	if len(cfg.AllowedEmails) == 0 && len(cfg.AllowedDomains) == 0 {
		return fmt.Errorf("configuration error: authorization lists are empty")
	}

	if cfg.GeminiAPIKey == "" {
		return fmt.Errorf("configuration error: GEMINI_API_KEY is not set")
	}

	if cfg.SessionEncryptKey == "" {
		return fmt.Errorf("SESSION_ENCRYPT_KEY が設定されていません。セキュアな運用のために必須です")
	}

	// SessionEncryptKey の長さチェック (AES要件: 16, 24, 32 bytes)
	keyLen := len([]byte(cfg.SessionEncryptKey))
	if keyLen != 16 && keyLen != 24 && keyLen != 32 {
		return fmt.Errorf("SESSION_ENCRYPT_KEY の長さが不正です (%d バイト)。16, 24, 32 バイトのいずれかにしてください", keyLen)
	}

	// SessionSecret の空チェック
	if cfg.SessionSecret == "" {
		return fmt.Errorf("SESSION_SECRET が設定されていません")
	}

	return nil
}

// getEnv は環境変数を取得し、存在しない場合はデフォルト値を返します。
func getEnv(key string, defaultValue string) string {
	return envutil.GetEnv(key, defaultValue)
}

// getEnvAsBool は環境変数からbool値を読み込みます。
func getEnvAsBool(key string, defaultValue bool) bool {
	return envutil.GetEnvAsBool(key, defaultValue)
}

// parseCommaSeparatedList は環境変数からbool値を読み込みます。
func parseCommaSeparatedList(key string) []string {
	return text.ParseCommaSeparatedList(key)
}
