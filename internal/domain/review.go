package domain

// ReviewRequest は Web 層で受け取り、Cloud Tasks でワーカーに渡す入力モデルです。
// 外部 core の型に依存しないよう、このリポジトリ側で定義します。
type ReviewRequest struct {
	RepoURL       string `json:"repo_url"`
	BaseBranch    string `json:"base_branch"`
	FeatureBranch string `json:"feature_branch"`
	Mode          string `json:"mode"`
	ModelName     string `json:"model_name"`
	StorageURI    string `json:"storage_uri"`
	PublicURL     string `json:"public_url"`
}
