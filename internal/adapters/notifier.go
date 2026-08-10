package adapters

import (
	"context"
	"errors"

	"github.com/shouni/go-review-kit/review"
)

// MultiNotifier は、複数の Notifier へ同じ通知を配る review.Notifier です。
//
// 進行状況の記録（StatusRecorder）と Slack 投稿はどちらもレビューの結末を受け取りますが、
// 片方が失敗しても片方は行うべきものです。順に呼び、失敗はまとめて返します。
type MultiNotifier []review.Notifier

var _ review.Notifier = (MultiNotifier)(nil)

// Notify は、登録されたすべての Notifier を順に呼びます。
// 途中で失敗しても残りを呼び、発生したエラーをまとめて返します。
func (m MultiNotifier) Notify(ctx context.Context, n review.Notification) error {
	var errs []error
	for _, notifier := range m {
		if notifier == nil {
			continue
		}
		if err := notifier.Notify(ctx, n); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
