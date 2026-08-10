# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## コマンド

```bash
go build ./...
go vet ./...
test -z "$(gofmt -l .)"          # CI と同じフォーマットチェック
go test -race ./...              # CI はレースディテクタ付きで実行
go test ./internal/config -run TestLoadConfig   # 単一テスト
golangci-lint run                # CI は v2.12.2 / 設定は .golangci.yml
```

`main.go` は起動時に `ValidateEssentialConfig()` を通すため、環境変数が揃っていないとローカル実行は即失敗します（HTTPS でない `SERVICE_URL`、OAuth 設定欠落、`ALLOWED_EMAILS`/`ALLOWED_DOMAINS` が両方空、`SESSION_ENCRYPT_KEY` が 16/24/32 バイト以外はすべてエラー。`http://localhost:8080` は安全なURLとして許容されます）。さらに `builder.BuildContainer` が GCS クライアントと Cloud Tasks クライアントを起動時に生成するため、実際にサーバを立ち上げるには GCP 認証情報が必要です。ロジック変更の検証は基本的に `go test` で行ってください。

必須環境変数の一覧・IAM ロールの要件は README.md にまとまっています。

## アーキテクチャ

### 1バイナリが Web と Worker を兼ねる

同じイメージが Cloud Run 上で「フォーム受付」と「非同期ワーカー」の両方を担い、区別は `internal/server/router.go` のミドルウェアだけです。

- `/auth/*` — 認証不要（OAuth ログイン）
- `/`, `/submit_review`, `/history`, `/history/{jobID}` — セッション認証 + CSRF + `http.NewCrossOriginProtection`
- `/tasks/execute_review` — Cloud Tasks からの OIDC 検証のみ（`auth.NewTaskVerifier(...).Middleware`）。audience だけでなく発行元サービスアカウントまで照合します
- `/health` — `/healthz` は Cloud Run の `*.run.app` 側で握られてコンテナまで届かないため使わないこと

受付の流れは「フォーム POST → **ジョブ ID 採番** → 保存先と閲覧先を決定 → Cloud Tasks へ enqueue → 受付を `status.json` へ記録 → 即座に詳細画面の URL を返す」（`internal/server/handlers/submit_handler.go`）。

**記録は enqueue の後です。** 積めていないジョブを履歴に出さないためで、記録の失敗は受付を失敗させません（ワーカーが `running` を書いた時点で履歴に現れます）。

### レビュー本体は go-review-kit に委譲

このリポジトリはコンテキストを組み立ててライブラリを呼ぶ実行基盤です。`internal/builder/pipeline.go` が `pipeline.Deps`（差分取得元・プロンプト・Reviewer・Publisher・Notifier）を組み立てます。

`internal/domain` の `ReviewRequest` は **意図的にライブラリの型と独立**しています。`internal/adapters/convert.go` が `domain.ReviewRequest` ↔ `review.Request` を変換する ACL です。ライブラリのモデルを domain 層やハンドラへ直接持ち込まないでください。

`internal/adapters/workflow.go` の `ReviewPipeline` が `domain.Pipeline` として公開し、`PIPELINE_TIMEOUT` の締切を被せます。**Cloud Tasks より先にアプリが諦めるための締切です** — 先を越されるとプロセスごと SIGTERM になり、失敗の記録も通知も残りません。

### ジョブ状態と履歴（go-job-kit）

`domain.JobStatus` は `jobstatus.Status` を**埋め込んで**います。入れ子のペイロードにすると保存済みの `status.json` が読めなくなるため、埋め込みを外さないでください。

配置は `internal/domain/storage_layout.go` に集約しています。状態ファイルを成果物と同じジョブプレフィックス配下へ置くのは意図的で、履歴一覧が `reviews/` 直下の列挙で作られるため、同居させることで実行中・失敗・スキップのジョブも一覧に出ます（成果物の有無で消えません）。

状態の記録はすべて `adapters/workflow.go` の `ReviewPipeline` が行います。`running` と再実行ガード（`AlreadySucceeded`）を `Run` の前に、結末（`succeeded` / `failed`）を `Run` の戻り値から記録します。

**`review.Notifier` を記録に使わないでください。** `pipeline.Run` が `(Result, *Report, error)` を返すため、レビューの中身は戻り値から取れます。`Notifier` は Slack への外向きの通知だけを担います（`adapters/slack.go`）。

**締切はレビューにだけ被せます。** `Execute` が `ctx` を上書きせず `runCtx` を別に作るのはそのためで、打ち切られた直後の記録まで期限切れの context で行うと、いちばん記録が要る場面で残りません。記録側も `context.WithoutCancel` で切り離しています。

「差分なしスキップ」は `state=succeeded` + `outcome=SKIPPED` で表します。ジョブとしては正常終了しているためで、`jobstatus.State` を拡張しないでください。

## 落とし穴

- **HTML テンプレートはページごとに別セットへパースします**（`internal/server/handlers/render.go` の `parsePageTemplates`）。どのページも本文を `{{define "content"}}` で定義しているため、`ParseFS("templates/*.html")` で 1 セットにまとめると最後にパースされたものが他を上書きします。ページを追加するときは `pageTemplates` にも足してください
- **クローン先のディレクトリ名は実行ごとに固有にします**（`adapters/git.go` の `uniqueRepoDirName`）。`go-review-kit` の既定はリポジトリ URL だけで決まるため、同じリポジトリのレビューが同時に走ると、先に終わった側の後始末（ディレクトリ削除）が実行中の側を壊します
- **ジョブ ID は `jobid.Sanitize` を通してから使います。** 末尾要素だけを取り出す正規化なので、`../../etc/passwd` は拒否ではなく `passwd` に切り詰められます（テストの期待値を書くときに注意）

### DI の組み立て順

`main.go` → `server.Run` → `builder.BuildContainer`（外部接続を確立し `app.Container` を構築、失敗時は生成済みリソースを巻き戻して Close）→ `builder.BuildHandlers` → `server.NewRouter`。`internal/builder` の各ファイルが責務ごとに分かれています（`io.go`=GCS と状態ストア、`task.go`=Cloud Tasks、`pipeline.go`=レビュー、`handlers.go`=認証/Web/Worker ハンドラ）。新しい外部依存を足すときは `app.Container` にフィールドを増やし、`builder` 側で組み立てます。

### プロンプトとレビューモード

`assets/prompts/*.md` は `embed.FS` でバイナリに埋め込まれ、**ファイル名がそのままモード名**になります（`code`, `article`, `novel`）。モードを追加するには `.md` を1枚置くだけで、フォームの選択肢・バリデーション (`assets.IsValidMode`) の両方に自動反映されます。

- 先頭行の `<!-- mode-description: ... -->` がフォームに出る説明文になります（省略時はモード名）
- テンプレートに渡るのは `.DiffContent` / `.FindingsFormat` / `.VerdictFormat`（`internal/adapters/prompt.go` の `reviewData`）
- `assets/partials/*.md` は全モード共通の出力フォーマット説明で、`prompts/` とは別ディレクトリに置くことでモード一覧に混ざらないようにしています

### 設定まわりの注意

- `GEMINI_MODEL` はカンマ区切りリストで、**先頭がデフォルト**、残りはフォームの選択肢になります。cloudbuild.yaml では値にカンマを含むため `^|^` 区切りで渡しています
- 受け付けるリポジトリ URL は `git@github.com:owner/repo.git` の SSH 形式のみ（`internal/server/handlers/handler_helpers.go` の `repoURLPattern`）
- `internal/giturl` は go-utils から取り込んだローカルパッケージ。「どこへクローンするか」「GCS のどのキーへ置くか」という本プロジェクト固有の決定に紐づくため internal に置いています

## コーディング規約

- コメント・エラーメッセージ・ログの文言は日本語で書きます（既存コードに合わせる）。コミットメッセージは簡潔な英語です
- `github.com/shouni/*` の依存は作者自身の共有ライブラリ群です。汎用ロジックはそちらに寄せ、本リポジトリ固有の判断は internal に置く方針です
- ログは `log/slog`（JSON ハンドラ）。リクエスト処理中は `slog.InfoContext` / `ErrorContext` を使います
- Go 1.26 / Dockerfile は `scratch` ベース（CGO 無効）
