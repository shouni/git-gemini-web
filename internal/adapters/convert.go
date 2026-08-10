package adapters

import (
	"github.com/shouni/go-review-kit/review"

	"github.com/shouni/git-gemini-web/internal/domain"
)

// toReviewRequest は domain.ReviewRequest をライブラリの review.Request へ変換します。
func toReviewRequest(req domain.ReviewRequest) review.Request {
	return review.Request{
		JobID:      req.JobID,
		RepoURL:    req.RepoURL,
		Base:       req.BaseBranch,
		Head:       req.FeatureBranch,
		Mode:       req.Mode,
		Model:      req.ModelName,
		StorageURI: req.StorageURI,
		PublicURL:  req.PublicURL,
	}
}

// fromReviewRequest は review.Request を domain.ReviewRequest へ戻します。
// ライブラリ側から渡ってくる通知を、こちらのモデルで扱うために使います。
func fromReviewRequest(req review.Request) domain.ReviewRequest {
	return domain.ReviewRequest{
		JobID:         req.JobID,
		RepoURL:       req.RepoURL,
		BaseBranch:    req.Base,
		FeatureBranch: req.Head,
		Mode:          req.Mode,
		ModelName:     req.Model,
		StorageURI:    req.StorageURI,
		PublicURL:     req.PublicURL,
	}
}
