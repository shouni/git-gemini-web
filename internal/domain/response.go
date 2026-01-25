package domain

import (
	"fmt"
	"time"
)

// ReviewStatus はレビュー結果の状態を表す型です。
type ReviewStatus string

const (
	// ReviewStatusSuccess は成功状態を示します。
	ReviewStatusSuccess ReviewStatus = "SUCCESS"
	// ReviewStatusFailure は失敗状態を示します。
	ReviewStatusFailure ReviewStatus = "FAILURE"
)

// ------------------------------

// GCSURI は、ReviewRequestからGCSのURIを構築します。
func (req ReviewRequest) GCSURI() string {
	return fmt.Sprintf("gs://%s/%s", req.GCSBucket, req.GCSPath)
}

// ReviewResult は、レビューパイプラインの最終結果を保持し、
// クライアント側（Webサーバー）へレスポンスとして返されます。
type ReviewResult struct {
	Status          ReviewStatus `json:"status"`           // ReviewStatus型を使用
	GCSURI          string       `json:"gcs_uri"`          // GCS上の結果ファイルの完全なURI
	DurationSeconds float64      `json:"duration_seconds"` // 処理の総実行時間 (秒)
	Message         string       `json:"message"`          // 処理結果の概要
}

// NewSuccessResult は、成功時のReviewResultを生成します。
func NewSuccessResult(req ReviewRequest, message string, duration time.Duration) ReviewResult {
	return ReviewResult{
		Status:          ReviewStatusSuccess,
		GCSURI:          req.GCSURI(),
		DurationSeconds: duration.Seconds(),
		Message:         message,
	}
}

// NewFailureResult は、失敗時のReviewResultを生成します。
func NewFailureResult(req ReviewRequest, err error, duration time.Duration) ReviewResult {
	return ReviewResult{
		Status:          ReviewStatusFailure,
		GCSURI:          req.GCSURI(),
		DurationSeconds: duration.Seconds(),
		Message:         fmt.Sprintf("レビューパイプラインでエラーが発生しました: %v", err),
	}
}
