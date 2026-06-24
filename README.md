# 🤖 Git Gemini Web

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Platform](https://img.shields.io/badge/Platform-Cloud%20Run-blue?logo=google-cloud)](https://cloud.google.com/run)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/git-gemini-web)](https://golang.org/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/git-gemini-web)](https://github.com/shouni/git-gemini-web/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/shouni/git-gemini-web)](https://goreportcard.com/report/github.com/shouni/git-gemini-web)
[![Status](https://img.shields.io/badge/Status-Completed-brightgreen)](#)

## 🚀 概要 (About) - WebベースのAIレビュー・オーケストレーター

**Git Gemini Web** は、AIコードレビューエンジン **[Gemini Reviewer Core](https://github.com/shouni/gemini-reviewer-core)** を Google Cloud 上でスケールさせるための **Webフロントエンド兼オーケストレーター** です。

本プロジェクトは、Webフォーム経由の依頼受付、OAuth認証によるアクセス制御、および非同期ジョブ実行の管理に特化した「実行基盤」を提供します。レビューの核となるロジックは Core エンジンに完全に委譲されており、常に最新の解析アルゴリズムを安全かつスケーラブルなサーバーレス環境で実行可能です。

---

## 🏗 アーキテクチャ設計 (Architecture)

本プロジェクトは、ビジネスロジックをライブラリ化（Core）し、外部インターフェースや通知機能を独立した「アダプター」として実装する **ヘキサゴナルアーキテクチャ（Ports and Adapters）** を採用しています。また、Google Cloud のマネージドサービスを組み合わせた **サーバーレス・オーケストレーション** により、高いスケーラビリティと耐障害性を実現しています。

* **Core Logic Delegation**:
  レビューのメインワークフロー（Git Fetch → AI Analysis → Publish）は、コアライブラリである `gemini-reviewer-core` が一括管理します。本プロジェクト（Web/Worker）は、実行に必要なコンテキスト（環境変数・認証情報・イベントデータ）を整えて Core を呼び出す「実行基盤」の役割に特化しています。
* **Serverless Orchestration**:
  **Cloud Run** と **Cloud Tasks** を組み合わせた非同期実行モデルを採用しています。
    * **非同期処理**: 重い解析処理をキューイングすることで、Webフロントエンドのタイムアウトを回避します。
    * **リトライ＆流量制御**: Cloud Tasks による自動リトライや並列度の制御により、AI API のレートリミット回避や一時的なネットワークエラーへの耐性を高めています。
* **Dependency Injection & Adaptability**:
  `internal/builder` にて全てのコンポーネントを紐付け、実行環境（Local/Cloud）や用途に応じたアダプター（Slack / GCS / Local FS）を動的に注入します。これにより、ビジネスロジックを汚染することなく、通知先や保存先の柔軟な切り替えが可能です。

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

## ✨ 技術スタック (Technology Stack)

| 要素 | 技術 / ライブラリ | 役割 |
| --- | --- | --- |
| **言語** | **Go (Golang)** | 全体の開発言語。 |
| **AIバックエンド** | **Hybrid Gemini Adapter** | **Google AI Studio** または **Vertex AI** を自動切替可能。 |
| **レビューエンジン** | **[`gemini-reviewer-core`](https://github.com/shouni/gemini-reviewer-core)** | レビューの全工程を制御。 |
| **非同期実行** | **Google Cloud Tasks** | 重いレビュー処理を非同期キューで管理。 |
| **認証・セッション** | **OAuth 2.0 / Gorilla Sessions** | Googleアカウントによるアクセス制限。 |
| **I/O抽象化** | **[`github.com/shouni/go-remote-io`](https://github.com/shouni/go-remote-io)**| GCS操作と署名付きURL生成の抽象化。 |

---

## 🤖 ハイブリッドなAIバックエンド対応

本アプリは、環境変数に応じて2つのAPIバックエンドを透過的に切り替え可能です。これにより、特定のAPIのサービス停止（503 Unavailable等）時にも柔軟に対応できます。

1. **Google AI Studio (API Key方式)**: 低遅延でプロトタイプ開発や個人利用に最適。
2. **Vertex AI (GCP方式)**: エンタープライズレベルのSLA、高いクォータ制限、組織的な予算管理に対応。

---

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
| `GEMINI_API_KEY` | Google Gemini APIキー | - |
| `GEMINI_MODEL` | 使用するGeminiモデル名。カンマ区切りで複数指定した場合はフォームで選択可能（先頭がデフォルト） | `gemini-2.5-flash` |
| `SSH_KEY_PATH` | GitHub SSH URL (`git@github.com:owner/repo.git`) のクローンに使うSSH秘密鍵パス（Secret Managerマウント推奨） | `/secrets/ssh/id_rsa` |
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

---

## 📜 ライセンス (License)

* このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。

