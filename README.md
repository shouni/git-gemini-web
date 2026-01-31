# 🤖 Git Gemini Web

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/git-gemini-web)](https://golang.org/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/git-gemini-web)](https://github.com/shouni/git-gemini-web/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 🚀 概要 (About) - WebベースのAIレビューオーケストレーター

**Git Gemini Web** は、AIコードレビューの**コアライブラリ機能**を **[Gemini Reviewer Core](https://github.com/shouni/gemini-reviewer-core)** を活用し、その機能を**Cloud Run** および **Google Cloud Tasks** を利用して**Webアプリケーション化**したプロジェクトです。

Webフォームを通じてレビュー依頼を受け付け、時間がかかるAIレビュー処理を**非同期ワーカー（Cloud Tasks）で実行するためのインターフェースとオーケストレーション**を担います。

-----

## ✨ 技術スタック (Technology Stack)

本アプリケーションは、Google Cloud RunとGoogle Cloud Tasksを組み合わせて、フロントエンドと非同期ワーカーの役割を果たします。レビューのコアロジックは外部モジュールに依存します。

| 要素 | 技術 / ライブラリ | 役割 |
| :--- | :--- | :--- |
| **言語** | **Go (Golang)** | Webサーバー（API/タスクワーカー）の開発言語。 |
| **コアレビュー機能** | **[`github.com/shouni/gemini-reviewer-core`](https://github.com/shouni/gemini-reviewer-core)** | **Git操作、AI通信、HTML変換**といった中核のレビューロジックを担う外部ライブラリです。 |
| **認証・セッション** | **`x/oauth2`** / **`gorilla/sessions`** | **Google OAuth 2.0** フローの制御と、Cookieベースのセッション管理を行います。 |
| **Webフレームワーク** | **go-chi/chi/v5** | 軽量でモジュール化されたルーティング処理。 |
| **アーキテクチャ** | **依存性注入 (DI)** / **アダプタパターン** | 高い保守性とテスト容易性を実現するためのサーバー設計基盤。 |
| **Web画面** | **`html/template`** | Goサーバー自身でレビュー依頼フォームの**HTMLテンプレートをレンダリング**し、ユーザーにフィードバックを表示します。 |
| **非同期実行** | **Google Cloud Tasks** | HTTPリクエストのタイムアウトを防ぐため、時間のかかるレビュー実行を**非同期キュー**に投入します。 |
| **デプロイ環境** | **Google Cloud Run** | Webフロントエンドと非同期ワーカーを実行するスケーラブルなサーバーレス環境。 |
| **結果保存** | **Google Cloud Storage (GCS)** | AIが出力したレビュー結果（HTML）を保存し、ユーザーにURLを提供します。 |
| **I/O抽象化** | **[`github.com/shouni/go-remote-io`](https://github.com/shouni/go-remote-io)** | GCSへのI/O操作、署名付きURLの生成処理を抽象化します。 |

-----

## 🚀 使い方 (Usage) / セットアップ

### 1\. GCPコンソールでの事前準備 (OAuth) 🔐

アプリケーションを実行する前に、Google Cloud ConsoleでOAuth認証情報を設定する必要があります。

### 2\. 必要な環境変数

実行環境には以下の環境変数を設定する必要があります。

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
| `GEMINI_API_KEY` | Google Gemini APIキー | **(設定必須)** |
| `GEMINI_MODEL` | 使用するGeminiモデル名 | `gemini-2.5-flash` |
| `SSH_KEY_PATH` | Git操作用のSSH秘密鍵パス（Secret Managerマウント推奨） | `/secrets/ssh/id_rsa` |
| `SLACK_WEBHOOK_URL` | レビュー結果のURLを通知するためのSlack Webhook URL。未設定の場合は通知をスキップします。 | `https://hooks.slack.com/services/T...` |

**認証設定 (OAuth):**

| 環境変数 | 説明 | 設定例 |
| :--- | :--- | :--- |
| `GOOGLE_CLIENT_ID` | GCPで作成したOAuthクライアントID | `xxxx.apps.googleusercontent.com` |
| `GOOGLE_CLIENT_SECRET` | GCPで作成したOAuthシークレット | `GOCSPX-xxxx...` |
| `SESSION_SECRET` | セッションデータのHMAC署名用シークレット | `openssl rand -base64 32` 等で生成 |
| `SESSION_ENCRYPT_KEY` | セッションデータのAES暗号化用シークレット | `openssl rand -base64 32` 等で生成 |
| `ALLOWED_EMAILS` / `ALLOWED_DOMAINS` | **必須:** アクセスを許可するメールアドレスまたはドメイン (例: `user@example.com,user2@example.com` / `example.com`)。**どちらか一方は設定が必要です。** | `,`で区切る |

### 3\. 必要なIAMロールの設定

本アプリケーションをGoogle Cloud RunとCloud Tasksで安全に運用するためには、各サービスアカウント（SA）に対し、**正確な権限付与**が必要です。設定が不足していると `403 Forbidden` エラーが発生します。

#### A. Cloud Run サービスアカウント (アプリケーション実行用)

*Webフロントエンドおよびワーカーとして動作するサービスアカウントです。*

| 権限（IAMロール） | 目的 |
| :--- | :--- |
| **Cloud Tasks エンキューア**<br>(`roles/cloudtasks.enqueuer`) | Webフォーム受付時に、タスクを Cloud Tasks キューに**追加**するために必要です。 |
| **サービス アカウント ユーザー**<br>(`roles/iam.serviceAccountUser`) | **重要:** タスク投入時、そのタスクを実行するID（Cloud Tasks SA）として振る舞う（ActAs）ために必要です。これがないとOIDCトークン付きのタスクを作成できません。 |
| **Storage オブジェクト管理者**<br>(`roles/storage.objectAdmin`) | AIレビュー結果のHTMLファイルを **GCS** バケットに書き込むために必要です。 |
| **Secret Manager のシークレット アクセサー**<br>(`roles/secretmanager.secretAccessor`) | `GEMINI_API_KEY` を Secret Manager から安全に取得する場合に推奨されます。 |

#### B. Cloud Tasks サービスアカウント (タスク実行ID)

*Cloud Tasks がワーカー（Cloud Run）を呼び出す際に使用するIDです。アプリケーションSAと同じものを使うことも可能ですが、セキュリティ上分けることを推奨します。*

| 権限（IAMロール） | 目的 |
| :--- | :--- |
| **Cloud Run 起動元**<br>(`roles/run.invoker`) | Cloud Tasks が、ワーカーエンドポイント (`/tasks/execute_review`) を認証付きで呼び出すために必要です。 |

#### C. デプロイ担当者 / CI/CD

*インフラ構築やデプロイを行うユーザーまたはサービスアカウントです。*

| 権限（IAMロール） | 目的 |
| :--- | :--- |
| **サービス アカウント トークン作成者**<br>(`roles/iam.serviceAccountTokenCreator`) | ローカルテストやデプロイ時に、一時的な認証トークンを生成するために必要になる場合があります。 |

-----

## 📁 Git Gemini Web プロジェクトレイアウト

```text
git-gemini-web/
├── main.go                      # エントリーポイント（Appの初期化と起動）
├── internal/
│   ├── app/                     # アプリケーションの基盤構造
│   │   └── container.go         # Container 構造体と Close メソッドの定義
│   ├── adapters/                # 外部システム連携（抽象インターフェースの実装）
│   │   └── slack_adapter.go     # Slack通知の実装
│   ├── builder/                 # DIコンテナ / 依存関係の組み立て
│   │   ├── app.go               # Containerの構築
│   │   ├── runners.go           # ReviewRunner / PublishRunner 等の生成ロジック
│   │   └── handlers.go          # 各コントローラー（auth, web, worker）のインスタンス化
│   ├── config/                  # 環境変数・バリデーション・設定管理
│   │   └── config.go            # Secret Managerマウントパス等の設定を含む
│   ├── domain/                  # ドメインモデル（純粋なデータ構造と型定義）
│   │   ├── response.go          # ReviewResult 等
│   │   └── review.go            # ReviewRequest, ReviewStatus, Outcome 等
│   ├── pipeline/                # ワークフローの指揮（Runner間を繋ぐ）
│   │   └── pipeline.go          # ReviewRunner -> PublishRunner の流れを制御
│   ├── runner/                  # ビジネスロジックの具象
│   │   ├── review_runner.go     # Git操作・Gemini API呼び出し
│   │   ├── publish_runner.go    # GCS保存・エラーレポート生成ヘルパー
│   │   └── report_builder.go    # Markdown/HTML テンプレート実行
│   └── server/                  # HTTPサーバー基盤
│       ├── server.go            # サーバーのライフサイクル（Run/Shutdown）
│       ├── router.go            # chiによるルーティングとミドルウェア設定
│       └── handlers/            # HTTPリクエストハンドラー（コントローラーの実体）
│           ├── handler.go       # ハンドラー共通構造体・基盤
│           └── handler_helpers.go # ハンドラー内の共通補助処理
└── templates/                   # UI用 HTMLテンプレート
    └── review_form.html
```

-----

## 💻 GCS連携とフィードバックの実現方法

本アプリケーションの**非同期処理**は、迅速なユーザーフィードバックを提供するために、以下のフローで動作します。

1.  **フォーム受付:**
    ユーザーがフォームを送信すると、`package handlers` のhandlerがリクエストを処理します（認証済みユーザーのみ）。
2.  **署名付き URL 生成:**
    `go-remote-io` を活用し、将来レビュー結果が保存される GCS オブジェクトへの**一時的な署名付き URL** を生成します。
3.  **タスクエンキューと即時応答:**
    レビュータスクを **Cloud Tasks** のキューに投入した後、handlerは処理完了を待たずに `HTTP 202 Accepted` で即座に応答を返し、生成した署名付き URL をユーザーに提示します。

-----

## 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。
