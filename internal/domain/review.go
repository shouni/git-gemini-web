package domain

import "time"

// ReviewRequest は、AIコードレビューを開始するために必要なすべての入力パラメータを保持します。
// これは Cloud Tasks のペイロードとして JSON でエンコードされます。
type ReviewRequest struct {
	RepoURL       string `json:"repo_url"`       // GitリポジトリのURL (例: ssh://git@github.com/user/repo)
	BaseBranch    string `json:"base_branch"`    // 比較元となるブランチ名 (例: main)
	FeatureBranch string `json:"feature_branch"` // 比較対象となるブランチ名 (例: develop)
	Mode          string `json:"mode"`           // レビューモード (例: "detail" または "release")
	ModelName     string `json:"model_name"`     // モデル名

	// 以下は Runner が結果を保存するために必要な情報
	GCSBucket string `json:"gcs_bucket"` // 結果を保存するGCSバケット名
	GCSPath   string `json:"gcs_path"`   // GCSバケット内のオブジェクトパス
}

// ReviewProcessOutcome は、ReviewRunner の実行結果（中間状態）を保持します。
type ReviewProcessOutcome struct {
	StartTime      time.Time
	StepName       string
	ReviewMarkdown string
	IsSkipped      bool
	Error          error
}

// ErrorReportParams は、エラーレポートのためのパラメータを保持します。
type ErrorReportParams struct {
	OriginalErr error
	Req         ReviewRequest
	Duration    time.Duration
	StepName    string
}
