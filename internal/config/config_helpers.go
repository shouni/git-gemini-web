package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/shouni/go-utils/envutil"
	"github.com/shouni/go-utils/text"
	"github.com/shouni/go-utils/urlpath"
	"github.com/shouni/netarmor/securenet"
)

// IsSecureServiceURL は、設定されたServiceURLが安全なスキーム (HTTPS など) を使用しているかどうかを確認します。
func (c *Config) IsSecureServiceURL() bool {
	return securenet.IsSecureServiceURL(c.ServiceURL)
}

// ValidateEssentialConfig は設定バリデーションを行います。
func (c *Config) ValidateEssentialConfig() error {
	if !c.IsSecureServiceURL() {
		return fmt.Errorf("本番環境では SERVICE_URL ('%s') は HTTPS である必要があります", c.ServiceURL)
	}

	if c.GoogleClientID == "" || c.GoogleClientSecret == "" || c.SessionSecret == "" {
		return fmt.Errorf("Google OAuth 関連の設定（ClientID, ClientSecret, SessionSecret）が不足しています")
	}

	if len(c.AllowedEmails) == 0 && len(c.AllowedDomains) == 0 {
		return fmt.Errorf("許可されたメールアドレスまたはドメインが一つも設定されていません（認可リストが空です）")
	}

	if c.SessionEncryptKey == "" {
		return fmt.Errorf("SESSION_ENCRYPT_KEY が設定されていません。セキュアな運用のために必須です")
	}

	// SessionEncryptKey の長さチェック (AES要件: 16, 24, 32 bytes)
	keyLen := len(c.SessionEncryptKey)
	if keyLen != 16 && keyLen != 24 && keyLen != 32 {
		return fmt.Errorf("SESSION_ENCRYPT_KEY の長さが不正です (%d バイト)。16, 24, 32 バイトのいずれかにしてください", keyLen)
	}

	return nil
}

// StorageURI は、保存先 URI (gs://bucket/path 形式) を生成します。
func (c *Config) StorageURI(repoURL, feature string, t time.Time) string {
	now := t.Format("20060102_150405")
	repoID := urlpath.GenerateGCSKeyName(repoURL)
	safeBranchName := strings.ReplaceAll(feature, "/", "-")

	return fmt.Sprintf("gs://%s/reviews/%s/%s_%s.html",
		c.GCSBucket,
		repoID,
		now,
		safeBranchName,
	)
}

// getEnv は環境変数を取得し、存在しない場合はデフォルト値を返します。
func getEnv(key string, defaultValue string) string {
	return envutil.GetEnv(key, defaultValue)
}

// getEnvAsBool は環境変数からbool値を読み込みます。
func getEnvAsBool(key string, defaultValue bool) bool {
	return envutil.GetEnvAsBool(key, defaultValue)
}

// parseCommaSeparatedList はカンマ区切りの文字列をパースしてスライスを返します。
func parseCommaSeparatedList(value string) []string {
	return text.ParseCommaSeparatedList(value)
}
