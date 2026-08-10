# 🤖 Git Gemini Web

[![CI](https://github.com/shouni/git-gemini-web/actions/workflows/ci.yml/badge.svg)](https://github.com/shouni/git-gemini-web/actions/workflows/ci.yml)
[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Platform](https://img.shields.io/badge/Platform-Cloud%20Run-blue?logo=google-cloud)](https://cloud.google.com/run)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/git-gemini-web)](https://golang.org/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/git-gemini-web)](https://github.com/shouni/git-gemini-web/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 🚀 概要 (About)

**Git Gemini Web** は、Git リポジトリの差分を AI にレビューさせる Web アプリです。

レビューの手順そのものは [`go-review-kit`](https://github.com/shouni/go-review-kit) に、
非同期ジョブの記録とページングは [`go-job-kit`](https://github.com/shouni/go-job-kit) に委ね、
本リポジトリは**依頼の受付・認証・非同期実行・結果の保存と表示**を担います。

元々は AI コードレビューツールとして作りましたが、今はコードより、Git で管理している記事や
小説の原稿をレビューする用途で使っています。`assets/prompts/` のプロンプトを差し替えれば、
レビュー対象はコード以外にも切り替えられます。

---

## 🏗 アーキテクチャ (Architecture)

**ヘキサゴナルアーキテクチャ（Ports and Adapters）** を採用し、外部との接続はすべて
アダプターとして分離しています。

```text
フォーム受付          非同期ワーカー
─────────────        ────────────────────────────────────
ジョブID採番     →   Git 差分 → Gemini → report.json 保存
Cloud Tasks 投入     ↘ status.json 記録 / Slack 通知
受付を記録
                     履歴 /history → /history/{jobID}
```

* **非同期実行**: 重い解析を Cloud Tasks へ逃がし、Web 側のタイムアウトを回避します。
* **依存性注入**: `internal/builder` が全コンポーネントを組み立てます。通知先や保存先を
  ロジックに触れずに差し替えられます。
* **1 バイナリ 2 役**: Web と Worker を同じバイナリが兼ね、自分自身へ self-invoke します
  （[必要な IAM ロール](#2-必要なiamロールの設定)を参照）。

### 成果物の置き場所

1 ジョブ分のオブジェクトは、ジョブ ID のプレフィックス配下にまとめます。

```text
gs://{GCS_REVIEW_BUCKET}/reviews/{jobID}/
├── status.json   # 進行状況と一覧用メタ（投入時に作成）
└── report.json   # レビュー結果の全文（成功時のみ）
```

* **履歴一覧は `reviews/` 直下のプレフィックスを列挙して作ります。** 状態ファイルを
  同じ配下に置いているため、実行中・失敗・スキップのジョブも一覧に並びます。
* **一覧用のメタと結果の全文を分けています。** 一覧は 1 行につき 1 回 `status.json` を
  読むため、指摘の全文を同じファイルへ入れると読み取り量が指摘件数に比例して増えます。
* **結果は JSON で保存し、表示は `/history/{jobID}` が行います。** 整形済みの HTML を
  置いて署名付き URL で配ると、アプリの認証を迂回できてしまううえ、同じ内容の見た目が
  詳細画面と 2 系統に分かれます。
* **削除はプレフィックスの一括走査で行います。** 消す側は「そのジョブが何を作ったか」を
  知る必要がなく、成果物の種類が増えても削除処理を直さずに済みます。実行中のジョブは
  削除できません（消してもワーカーが `status.json` を書き戻して復活するためです）。

---

## 📂 プロジェクト構造 (Project Structure)

```text
git-gemini-web/
├── assets/            # 【資産】静的リソース（embed でバイナリに埋め込み）
│   ├── prompts/       #   - LLM への指示書（ファイル名がレビューモード名）
│   ├── partials/      #   - 全モード共通の出力フォーマット説明
│   ├── templates/     #   - HTML テンプレート
│   ├── static/        #   - ブラウザへ配信する CSS / JS（/static/ で公開）
│   └── assets.go      #   - embed.FS の定義
├── internal/
│   ├── adapters/      # 【接続】Gemini / Git / Slack / 結果保存 / パイプライン ACL
│   ├── app/           # 【基盤】Container による依存の保持とライフサイクル管理
│   ├── builder/       # 【構築】各コンポーネントの初期化と組み立て
│   ├── config/        # 【設定】環境変数・定数・バリデーション
│   ├── domain/        # 【中心】モデル、保存先の規約、ポート定義
│   ├── giturl/        # 【変換】リポジトリURLの解析と表示用パス
│   ├── repository/    # 【読み取り】GCS 上のレビュー履歴
│   └── server/        # 【玄関】HTTP サーバー、ルーティング、ハンドラ
└── main.go            # 【起点】起動とシグナルハンドリング
```

---

## ✨ 技術スタック (Technology Stack)

| 要素 | 技術 / ライブラリ |
| --- | --- |
| 言語 | Go |
| レビューエンジン | [`go-review-kit`](https://github.com/shouni/go-review-kit) |
| ジョブ状態・履歴ページング | [`go-job-kit`](https://github.com/shouni/go-job-kit) |
| 実行基盤 | Cloud Run / Cloud Tasks |
| 認証・セッション | OAuth 2.0（[`gcp-kit`](https://github.com/shouni/gcp-kit)） |
| I/O 抽象化 | [`go-remote-io`](https://github.com/shouni/go-remote-io)（GCS 操作） |

**AI は Vertex AI 経由で呼びます。** `go-review-kit` の `gemini.Options` は API キー方式にも
対応していますが、本アプリは `ProjectID` のみを渡す配線なので（`internal/adapters/ai.go`）、
API キー経路は使いません。切り替えるには `gemini.Options.APIKey` を渡すよう変更が要ります。

---

## ⚙️ セットアップ

### 1. 必要な環境変数

**未設定だと起動時に落ちる**のは `SERVICE_URL`（本番は HTTPS 必須）・`GOOGLE_CLIENT_ID`・
`GOOGLE_CLIENT_SECRET`・`SESSION_SECRET`・`SESSION_ENCRYPT_KEY`・`GEMINI_MODELS`・
`ALLOWED_EMAILS` または `ALLOWED_DOMAINS` です。残りは空でも起動します（機能しないだけです）。

`GEMINI_MODELS` にアプリ側の既定値を置かないのは意図的です。モデル ID が古くなるのは
Google のリリース周期であってこのリポジトリの都合ではないため、既定値があると
「デプロイ設定を変えていないのに古いモデルを指し続ける」状態に誰も気付けません。

**基本設定:**

| 環境変数 | 説明 | デフォルト値（例） |
| :--- | :--- | :--- |
| `SERVICE_URL` | アプリケーションのルート URL（末尾スラッシュなし）。**本番では HTTPS 必須** | `https://myapp.run.app` または `http://localhost:8080` |
| `PORT` | サーバーがリッスンするポート | `8080` |
| `GCP_PROJECT_ID` | GCP のプロジェクト ID | `your-gcp-project` |
| `GCP_LOCATION_ID` | Cloud Tasks キューのリージョン | `asia-northeast1` |
| `CLOUD_TASKS_QUEUE_ID` | 使用する Cloud Tasks のキュー名 | `review-queue` |
| `SERVICE_ACCOUNT_EMAIL` | タスク発行に使用するサービスアカウント | - |
| `GCS_REVIEW_BUCKET` | レビュー結果と進行状況を保存する GCS バケット名 | `your-review-archive-bucket` |
| `GEMINI_API_KEY` | 読み込むが**現在は未使用**（AI は Vertex AI 経由。「技術スタック」参照） | - |
| `GEMINI_MODELS` | 使用する Gemini モデル名。カンマ区切りで複数指定するとフォームで選択可能（先頭がデフォルト）。**アプリ側に既定値は無く、未設定だと起動時に落ちます** | **必須**（Google の最新モデル ID を確認して設定） |
| `TASK_AUDIENCE_URL` | Cloud Tasks の OIDC トークン検証に使う audience。未設定なら `SERVICE_URL` | `https://myapp.run.app` |
| `PIPELINE_TIMEOUT` | レビュー 1 件の実行時間の上限（`5m` 形式）。Cloud Tasks の dispatch deadline より短いこと。超えると起動時エラー | `5m` |
| `SSH_KEY_PATH` | SSH 形式のリポジトリ（`git@github.com:owner/repo.git`）のクローンに使う秘密鍵パス（Secret Manager マウント推奨） | `/secrets/ssh/id_rsa` |
| `SLACK_WEBHOOK_URL` | レビューの結末を通知する Slack Webhook URL。未設定なら通知をスキップ | `https://hooks.slack.com/services/T...` |

> **SSH ホストキー検証を無効化するスイッチはありません。** `Dockerfile` が GitHub の
> ホストキーを `/etc/ssh/ssh_known_hosts` へ焼き込むため通常は設定不要で、GitHub 以外を
> 対象にする場合のみ同ファイルへ追記します。

**認証設定 (OAuth):**

| 環境変数 | 説明 | 設定例 |
| :--- | :--- | :--- |
| `GOOGLE_CLIENT_ID` | OAuth クライアント ID（リダイレクト URI は `<SERVICE_URL>/auth/callback`） | `xxxx.apps.googleusercontent.com` |
| `GOOGLE_CLIENT_SECRET` | OAuth シークレット | `GOCSPX-xxxx...` |
| `SESSION_SECRET` | セッションの HMAC 署名用シークレット | `openssl rand -base64 32` |
| `SESSION_ENCRYPT_KEY` | セッションの AES 暗号化用シークレット（16/24/32 バイト） | `openssl rand -base64 32` |
| `ALLOWED_EMAILS` / `ALLOWED_DOMAINS` | アクセスを許可するメールまたはドメイン。**どちらか一方は必要**（両方空だと誰もログインできません） | `user@example.com,user2@example.com` / `example.com` |

### 2. 必要なIAMロールの設定

**SA は 1 つだけです。** 1 バイナリが Web と Worker を兼ね、自分自身へ self-invoke します
（`SERVICE_URL` が自分の URL）。Cloud Tasks 用に別の SA を用意する必要はありません。
`SERVICE_ACCOUNT_EMAIL` は OIDC トークンの**発行者**であると同時に、受信側の**許可リスト**も
兼ねるため、SA を変えるときは env も同時に変えてください。

実行 SA には、次のことができる権限が要ります。

- レビュー結果と進行状況を置く GCS バケットの読み書き
- Cloud Tasks キューへのタスク投入と、自分自身を指定した OIDC トークンの発行（ActAs）
- 自分自身の Cloud Run サービスの呼び出し
- Vertex AI の呼び出し
- 使用するシークレットの読み取り

**ロール名を列挙していないのは、粒度が環境によって変わるためです。** 決め方だけ挙げておきます。

- **GCS はバケット単位で `objectUser` を使ってください。** `objectAdmin` はオブジェクト ACL の
  操作まで許します。プロジェクトレベルで付けると、無関係なバケットにも到達します
- **シークレットはシークレット単位で付けてください。** プロジェクトレベルだと全シークレットに
  到達します

設定が不足していると `403 Forbidden` になります。

---

## 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。
