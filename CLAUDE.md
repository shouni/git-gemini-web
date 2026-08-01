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

必須環境変数の一覧・IAM ロールの要件は README.md に表形式でまとまっています。

## アーキテクチャ

### 1バイナリが Web と Worker を兼ねる

同じイメージが Cloud Run 上で「フォーム受付」と「非同期ワーカー」の両方を担い、区別は `internal/server/router.go` のミドルウェアだけです。

- `/auth/*` — 認証不要（OAuth ログイン）
- `/`, `/submit_review` — セッション認証 + CSRF + `http.NewCrossOriginProtection`
- `/tasks/execute_review` — Cloud Tasks からの OIDC 検証のみ（`TaskOIDCVerificationMiddleware`）。audience だけでなく発行元サービスアカウント (`AllowedTaskServiceAccounts`) まで照合します
- `/health` — `/healthz` は Cloud Run の `*.run.app` 側で握られてコンテナまで届かないため使わないこと

フローは 「フォーム POST → 保存先 URI 決定 → **署名付き URL を先に生成** → Cloud Tasks へ enqueue → 即座に URL を返す」（`internal/server/handlers/submit_handler.go`）。ワーカーは後から同じ URI に成果物を書くため、ユーザーは生成完了を待たずに URL を受け取れます。この順序を崩さないでください。

### レビュー本体は gemini-reviewer-core に委譲

このリポジトリはコンテキストを組み立てて Core を呼ぶ実行基盤です。`internal/builder/pipeline.go` が Core の `runner.ReviewRunner`（Prompt + Git + AI）と `runner.PublishRunner`（GCS 書き込み + 通知）を組み立て、`workflow.New` でつないでいます。

`internal/domain` の `ReviewRequest` は **意図的に Core の型と独立**しています。`internal/adapters/workflow.go` の `CoreWorkflowAdapter` が ACL として `domain.ReviewRequest` → `coreports.ReviewRequest` を変換します。Core のモデルを domain 層やハンドラに直接持ち込まないでください。

### DI の組み立て順

`main.go` → `server.Run` → `builder.BuildContainer`（外部接続を確立し `app.Container` を構築、失敗時は生成済みリソースを巻き戻して Close）→ `builder.BuildHandlers` → `server.NewRouter`。`internal/builder` の各ファイルが責務ごとに分かれています（`io.go`=GCS、`task.go`=Cloud Tasks、`pipeline.go`=Core、`handlers.go`=認証/Web/Worker ハンドラ）。新しい外部依存を足すときは `app.Container` にフィールドを増やし、`builder` 側で組み立てます。

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

- コメント・エラーメッセージ・ログの文言は日本語で書きます（既存コードに合わせる）
- `github.com/shouni/*` の依存は作者自身の共有ライブラリ群です。汎用ロジックはそちらに寄せ、本リポジトリ固有の判断は internal に置く方針です
- ログは `log/slog`（JSON ハンドラ）。リクエスト処理中は `slog.InfoContext` / `ErrorContext` を使います
- Go 1.26 / Dockerfile は `scratch` ベース（CGO 無効）
