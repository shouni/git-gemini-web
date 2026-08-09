# 🤖 Git Gemini Web

[![CI](https://github.com/shouni/git-gemini-web/actions/workflows/ci.yml/badge.svg)](https://github.com/shouni/git-gemini-web/actions/workflows/ci.yml)
[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Platform](https://img.shields.io/badge/Platform-Cloud%20Run-blue?logo=google-cloud)](https://cloud.google.com/run)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/git-gemini-web)](https://golang.org/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/git-gemini-web)](https://github.com/shouni/git-gemini-web/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Status](https://img.shields.io/badge/Status-Completed-brightgreen)](#)

## 🚀 概要 (About)

**Git Gemini Web** は、AIレビューエンジン **[Gemini Reviewer Core](https://github.com/shouni/gemini-reviewer-core)** をベースにした、Webベースのレビュー・オーケストレーターです。

レビューの核となるロジックは Core に委譲し、本プロジェクトは依頼の受付・認証・非同期実行を担う**実行基盤**に徹しています。

元々はAIコードレビューツールとして作りましたが、今はコードレビューより、Gitリポジトリで管理している記事や小説の原稿をレビューする用途で使っています。`assets/prompts/` のプロンプトを差し替えれば、レビュー対象はコード以外にも切り替えられます。

---

## 🏗 アーキテクチャ設計 (Architecture)

**ヘキサゴナルアーキテクチャ（Ports and Adapters）** を採用し、外部との接続はすべてアダプターとして分離しています。

```
Web フォーム / OAuth → Cloud Tasks → Worker → Core（Git → AI → 公開）→ GCS / Slack
```

* **非同期実行**: 重い解析を Cloud Tasks へ逃がし、Web 側のタイムアウトを回避します。リトライと並列度の制御で、AI API のレートリミットや一時的なエラーにも耐えます。
* **依存性注入**: `internal/builder` が全コンポーネントを組み立て、実行環境（Local/Cloud）に応じたアダプター（Slack / GCS / Local FS）を注入します。通知先や保存先をロジックに触れずに差し替えられます。
* **1 バイナリ 2 役**: Web と Worker を同じバイナリが兼ね、自分自身へ self-invoke します（「必要なIAMロールの設定」参照）。

---

## 📂 プロジェクト構造 (Project Structure)

```text
git-gemini-web/
├── assets/            # 【資産】静的リソース（Go バイナリに embed で埋め込み）
│   ├── prompts/       #   - LLM への指示書（Markdown テンプレート）
│   ├── templates/     #   - Web 表示用の HTML テンプレート群
│   └── assets.go      #   - embed.FS の定義（Prompts / Templates）
├── internal/
│   ├── adapters/      # 【接続】外部（AI API, Slack, Git）との通信を担う実装
│   ├── app/           # 【基盤】Container による依存関係の保持とライフサイクル管理
│   ├── builder/       # 【構築】各コンポーネントの初期化・インスタンス組み立て
│   ├── config/        # 【設定】環境変数・定数・バリデーションの管理
│   ├── domain/        # 【中心】ビジネスルール、モデル定義、抽象インターフェース (Ports)
│   └── server/        # 【玄関】HTTP サーバー、ルーティング、ハンドラ実装
├── docs/              # 【記録】設計ドキュメントや動作イメージ画像
└── main.go            # 【起点】アプリの起動、シグナルハンドリング
```

---

## ✨ 技術スタック (Technology Stack)

| 要素 | 技術 / ライブラリ |
| --- | --- |
| 言語 | Go |
| レビューエンジン | [`gemini-reviewer-core`](https://github.com/shouni/gemini-reviewer-core) |
| 実行基盤 | Cloud Run / Cloud Tasks |
| 認証・セッション | OAuth 2.0 / Gorilla Sessions |
| I/O抽象化 | [`go-remote-io`](https://github.com/shouni/go-remote-io)（GCS操作と署名付きURL生成） |

**AI は Vertex AI 経由で呼びます。** Core の `ai.GeminiAdapter` は API キー方式にも対応
していますが、本アプリは `ProjectID` のみを渡す配線なので（`internal/adapters/ai.go`）、
API キー経路は使いません。切り替えたい場合は `GeminiOptions.APIKey` を渡すよう変更が要ります。

---

## ⚙️ セットアップ

### 1\. 必要な環境変数

**未設定だと起動時に落ちる**のは `SERVICE_URL`（本番は HTTPS 必須）・`GOOGLE_CLIENT_ID` ・
`GOOGLE_CLIENT_SECRET`・`SESSION_SECRET`・`SESSION_ENCRYPT_KEY`・`ALLOWED_EMAILS` または
`ALLOWED_DOMAINS` の 6 つです。残りは空でも起動します（機能しないだけです）。

**基本設定:**

| 環境変数 | 説明 | デフォルト値（例） |
| :--- | :--- | :--- |
| `SERVICE_URL` | アプリケーションのルートURL (末尾スラッシュなし)。**本番環境ではHTTPS (`https://...`) が必須です。** | `https://myapp.run.app` または `http://localhost:8080` |
| `PORT` | サーバーがリッスンするポート | `8080` |
| `GCP_PROJECT_ID` | GCPのプロジェクトID | `your-gcp-project` |
| `GCP_LOCATION_ID` | Cloud Tasks キューのリージョン | `asia-northeast1` |
| `CLOUD_TASKS_QUEUE_ID` | 使用するCloud Tasksのキュー名 | `review-queue` |
| `SERVICE_ACCOUNT_EMAIL` | タスク発行に使用するサービスアカウント | - |
| `GCS_REVIEW_BUCKET` | レビュー結果（HTML）を保存するGCSバケット名 | `your-review-archive-bucket` |
| `GEMINI_API_KEY` | 読み込むが**現在は未使用**（AI は Vertex AI 経由。「技術スタック」参照） | - |
| `GEMINI_MODEL` | 使用するGeminiモデル名。カンマ区切りで複数指定した場合はフォームで選択可能（先頭がデフォルト） | `gemini-3.6-flash` |
| `TASK_AUDIENCE_URL` | Cloud Tasks の OIDC トークン検証に使う audience。未設定なら `SERVICE_URL` を使う | `https://myapp.run.app` |
| `PIPELINE_TIMEOUT` | レビュー1件の実行時間の上限（`5m` 形式）。Cloud Tasks の dispatch deadline (10分) より短いこと。超えると起動時エラー | `5m` |
| `SSH_KEY_PATH` | GitHub SSH URL (`git@github.com:owner/repo.git`) のクローンに使うSSH秘密鍵パス（Secret Managerマウント推奨） | `/secrets/ssh/id_rsa` |
| `SLACK_WEBHOOK_URL` | レビュー結果(成功時のURL、スキップ・失敗時はその内容)を通知するためのSlack Webhook URL。未設定の場合は通知をスキップします。 | `https://hooks.slack.com/services/T...` |

> **SSH ホストキー検証を無効化するスイッチはありません**（`SKIP_HOST_KEY_CHECK` は
> `gemini-reviewer-core` v1.11.x で廃止）。`Dockerfile` が GitHub のホストキーを
> `/etc/ssh/ssh_known_hosts` へ焼き込むため通常は設定不要で、GitHub 以外を対象にする
> 場合のみ同ファイルへ追記します。

**認証設定 (OAuth):**

| 環境変数 | 説明 | 設定例 |
| :--- | :--- | :--- |
| `GOOGLE_CLIENT_ID` | GCPで作成したOAuthクライアントID（リダイレクトURIは `<SERVICE_URL>/auth/callback`） | `xxxx.apps.googleusercontent.com` |
| `GOOGLE_CLIENT_SECRET` | GCPで作成したOAuthシークレット | `GOCSPX-xxxx...` |
| `SESSION_SECRET` | セッションデータのHMAC署名用シークレット | `openssl rand -base64 32` 等で生成 |
| `SESSION_ENCRYPT_KEY` | セッションデータのAES暗号化用シークレット | `openssl rand -base64 32` 等で生成 |
| `ALLOWED_EMAILS` / `ALLOWED_DOMAINS` | アクセスを許可するメールアドレスまたはドメイン。**どちらか一方は必要**（両方空だと誰もログインできません） | `user@example.com,user2@example.com` / `example.com` |

### 2\. 必要なIAMロールの設定

本アプリケーションを Cloud Run と Cloud Tasks で動かすには、実行サービスアカウントに
いくつかの権限が要ります。設定が不足していると `403 Forbidden` になります。

**SA は 1 つだけです。** 1 バイナリが Web と Worker を兼ね、**自分自身へ self-invoke します**
（`SERVICE_URL` が自分の URL）。Cloud Tasks 用に別の SA を用意する必要はありません。
`SERVICE_ACCOUNT_EMAIL` は OIDC トークンの**発行者**であると同時に受信側の**許可リスト**
（`AllowedTaskServiceAccounts`）も兼ねるため、SA を変えるときは env も同時に変えてください。

実行 SA には、次のことができる権限が要ります。

- レビュー結果を置く GCS バケットの読み書き
- Cloud Tasks キューへのタスク投入と、自分自身を指定した OIDC トークンの発行（ActAs）
- 署名付き URL の生成（秘密鍵を持たないため IAM の SignBlob 経由）
- 自分自身の Cloud Run サービスの呼び出し
- Vertex AI の呼び出し
- 使用するシークレットの読み取り

**ロール名を列挙していないのは、粒度が環境によって変わるためです。** 決め方だけ挙げておきます。

- **GCS はバケット単位で、`objectUser` を使ってください。** `objectAdmin` はオブジェクト ACL の
  操作まで許します。プロジェクトレベルで付けると、無関係なバケットにも到達します
- **シークレットはシークレット単位で付けてください。** プロジェクトレベルだと全シークレットに
  到達します

---

## 📜 ライセンス (License)

* このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。

