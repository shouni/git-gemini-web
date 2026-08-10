// Package repository は、GCS 上のレビュー履歴の読み取りを担います。
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"path"
	"strings"

	"github.com/shouni/go-job-kit/cache"
	"github.com/shouni/go-job-kit/jobstatus"
	"github.com/shouni/go-job-kit/paging"
	"github.com/shouni/go-remote-io/remoteio"
	"github.com/shouni/go-review-kit/review"
	"github.com/shouni/go-utils/jobid"

	"github.com/shouni/git-gemini-web/internal/domain"
)

// loadConcurrency は 1 ページ分のメタデータを取得する際の同時実行数です。
const loadConcurrency = 10

// maxReportBytes は report.json の読み取り上限です。
// AI の出力はスキーマで制約されていますが、壊れたオブジェクトを丸ごとメモリへ載せない
// ための歯止めです。
const maxReportBytes = 8 << 20 // 8 MiB

// History は、GCS 上のレビュー履歴を読み取ります。
type History struct {
	reader remoteio.InputReader
	store  domain.StatusStore
	layout domain.StorageLayout
	ids    *cache.IDList
	logger *slog.Logger
}

var _ domain.HistoryRepository = (*History)(nil)

// NewHistory は History を構築します。
func NewHistory(reader remoteio.InputReader, store domain.StatusStore, layout domain.StorageLayout) *History {
	return &History{
		reader: reader,
		store:  store,
		layout: layout,
		ids:    cache.NewIDList(cache.DefaultIDListTTL),
		logger: slog.Default().With("collection", "reviews"),
	}
}

// List は、新しい順に page ページ目を返します。
func (h *History) List(ctx context.Context, page, perPage int) (domain.HistoryPage, error) {
	jobIDs, err := h.listJobIDs(ctx)
	if err != nil {
		return domain.HistoryPage{}, fmt.Errorf("レビュー履歴の一覧取得に失敗しました: %w", err)
	}

	// ID に埋め込まれた時刻で並べます。ID の辞書順に頼らないのは、採番の接頭辞が
	// 変わったり別サービス採番の ID が混ざったりすると、時刻より先に接頭辞の差が
	// 効いて日付を無視した並びになるためです。
	items, meta, err := paging.LoadPage(ctx, jobIDs, page, perPage, h.loadStatus,
		paging.WithSortKey(jobid.SortKey),
		paging.WithConcurrency(loadConcurrency),
		paging.WithLogger(h.logger),
	)
	if err != nil {
		return domain.HistoryPage{}, fmt.Errorf("レビュー履歴の読み込みに失敗しました: %w", err)
	}

	return domain.HistoryPage{Items: items, Meta: meta}, nil
}

// Get は、1 件分の進行状況とレビュー結果全文を返します。
func (h *History) Get(ctx context.Context, jobID string) (domain.ReviewDetail, error) {
	status, err := h.store.Get(ctx, jobID)
	if err != nil {
		return domain.ReviewDetail{}, err
	}

	detail := domain.ReviewDetail{Status: status}
	if !status.HasReport() {
		// 実行中・失敗・スキップは成果物を持ちません。進行状況だけで詳細を組み立てます。
		return detail, nil
	}

	report, err := h.loadReport(ctx, status.ReportURI)
	if err != nil {
		// 結果本体が読めなくても、進行状況までは見せます。何が起きたかを画面から
		// 追えるほうが、404 を返すより手掛かりが残るためです。
		h.logger.WarnContext(ctx, "レビュー結果の読み込みに失敗しました",
			"job_id", jobID, "uri", status.ReportURI, "error", err)
		return detail, nil
	}

	detail.Report = &report
	return detail, nil
}

// Invalidate は、ジョブ ID 一覧のキャッシュを捨てます。
func (h *History) Invalidate() {
	h.ids.Invalidate(h.layout.ReviewPrefixURI())
}

// loadStatus は 1 件分の進行状況を読み取ります。
//
// 読めなかった ID も一覧からは落とさず、ジョブ ID だけの行として残します。
// 一覧から消えると、投入したはずのレビューを画面から追えなくなるためです。
func (h *History) loadStatus(ctx context.Context, jobID string) (domain.JobStatus, error) {
	status, err := h.store.Get(ctx, jobID)
	if err != nil {
		h.logger.WarnContext(ctx, "進行状況の読み込みに失敗しました", "job_id", jobID, "error", err)
		return domain.JobStatus{Status: jobstatus.Status{JobID: jobID}}, nil
	}
	return status, nil
}

// listJobIDs はプレフィックス直下のジョブ ID を集めます。
//
// 区切り文字を指定して、ジョブ 1 件を 1 エントリとして受け取ります。指定しないと配下の
// オブジェクトが全件返るため、1 ジョブにつきファイル数分の結果を受け取ったうえで、
// 呼び出し側でジョブ ID の重複を潰すことになります。
func (h *History) listJobIDs(ctx context.Context) ([]string, error) {
	prefix := h.layout.ReviewPrefixURI()

	return h.ids.Load(ctx, prefix, func(ctx context.Context) ([]string, error) {
		var jobIDs []string
		err := h.reader.List(ctx, prefix, func(gcsPath string) error {
			// 疑似ディレクトリだけを拾います。プレフィックス直下に置かれたオブジェクトは
			// ジョブではないため対象外です。
			if !strings.HasSuffix(gcsPath, "/") {
				return nil
			}
			if jobID := path.Base(strings.TrimSuffix(gcsPath, "/")); jobID != "" {
				jobIDs = append(jobIDs, jobID)
			}
			return nil
		}, remoteio.WithDelimiter("/"))
		if err != nil {
			return nil, err
		}
		return jobIDs, nil
	})
}

// loadReport は report.json を読み取ります。
func (h *History) loadReport(ctx context.Context, uri string) (review.Report, error) {
	rc, err := h.reader.Open(ctx, uri)
	if err != nil {
		return review.Report{}, fmt.Errorf("レビュー結果を開けませんでした: %w", err)
	}
	defer func() { _ = rc.Close() }()

	body, err := io.ReadAll(io.LimitReader(rc, maxReportBytes))
	if err != nil {
		return review.Report{}, fmt.Errorf("レビュー結果の読み取りに失敗しました: %w", err)
	}

	var report review.Report
	if err := json.Unmarshal(body, &report); err != nil {
		return review.Report{}, fmt.Errorf("レビュー結果の解釈に失敗しました: %w", err)
	}
	return report, nil
}
