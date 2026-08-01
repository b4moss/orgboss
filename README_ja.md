# orgboss

[English](README.md) | **日本語**

Authboss 連携の組織マルチテナント管理パッケージ

[![Tests](https://github.com/b4m-oss/orgboss/actions/workflows/test.yml/badge.svg?branch=develop)](https://github.com/b4m-oss/orgboss/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/b4m-oss/orgboss/graph/badge.svg?branch=develop)](https://codecov.io/gh/b4m-oss/orgboss)

## 機能

- **組織・ユーザー管理**: トランザクションにより組織とユーザーをアトミックに作成
- **招待システム**: メールによる組織への招待（トークン認証・有効期限管理）
- **ロールベースアクセス制御**: manager（全権限）と user（読み取り専用）のロール
- **カスタムハンドラ**: `RoleChecker`、`DeletionHandler`、`EmailSender` インターフェースで独自ロジックを実装可能
- **フックシステム**: 組織作成・ユーザー削除などの前後処理を拡張可能
- **Authboss 連携**: [Authboss](https://github.com/aarondl/authboss) 認証パッケージとの統合
- **柔軟なストレージ**: テスト向けインメモリ、本番向け PostgreSQL（GORM）

## 要件

- Go 1.25 以上
- PostgreSQL（本番利用時）
- SMTP サーバー（メール招待時）

## インストール

```bash
go get github.com/b4m-oss/orgboss
```

## クイックスタート

```go
package main

import (
    "context"
    "github.com/b4m-oss/orgboss"
)

func main() {
    ctx := context.Background()
    
    // デフォルト設定でマネージャーを作成
    manager := orgboss.NewManager(nil)
    
    // 最初のユーザー付きで組織を作成（自動的に manager ロールが付与される）
    org, user, err := manager.CreateOrganizationWithUser(
        ctx,
        "My Organization",
        "manager@example.com",
    )
    if err != nil {
        panic(err)
    }
    
    // ユーザーには自動的に manager ロールが付与される
    // org.ID と user.ID が利用可能
}
```

## 設定

`Config` で orgboss の挙動をカスタマイズできます。

```go
import "time"

config := &orgboss.Config{
    InvitationExpiryDuration:        24 * time.Hour,
    DefaultRole:                     orgboss.RoleUser,
    EnableBulkInvite:                true,
    MaxBulkInviteCount:              100,
    InvitationBaseURL:               "http://localhost:8080",
    InvitationRedirectPath:          "/reset-password",
    EnableAutoLoginAfterPasswordReset: true,
}

manager := orgboss.NewManager(config)
```

### 設定オプション

- `InvitationExpiryDuration`: 招待の有効期限（デフォルト: 24 時間）
- `DefaultRole`: 新規ユーザーのデフォルトロール（デフォルト: `RoleUser`）
- `EnableBulkInvite`: 一括招待の有効化（デフォルト: `true`）
- `MaxBulkInviteCount`: 一括招待あたりの最大件数（デフォルト: 100）
- `InvitationBaseURL`: 招待リンクのベース URL（メール招待時に必須）
- `InvitationRedirectPath`: 招待受諾後のリダイレクトパス（デフォルト: `/reset-password`）
- `EnableAutoLoginAfterPasswordReset`: パスワードリセット後の自動ログイン（デフォルト: `true`）

## ストレージ

デフォルトではテスト向けのインメモリ実装を使います。本番では `Storage` インターフェースを実装し、`NewManagerWithStorage` を使ってください。

### PostgreSQL ストレージの利用

```go
import (
    "github.com/b4m-oss/orgboss/internal/database"
    "github.com/b4m-oss/orgboss/internal/storage"
)

// PostgreSQL に接続
db, err := database.Connect()
if err != nil {
    panic(err)
}

// マイグレーションを実行
err = database.Migrate(db)
if err != nil {
    panic(err)
}

// PostgreSQL ストレージを作成
postgresStorage := storage.NewPostgresStorage(db)

// PostgreSQL ストレージ付きでマネージャーを作成
config := orgboss.DefaultConfig()
config.EmailSender = email.NewSMTPEmailSender() // 招待に必須
manager := orgboss.NewManagerWithStorage(config, postgresStorage)
```

## 使用例

### ユーザーの招待

```go
// 単一ユーザーを招待
invitation, err := manager.InviteUser(ctx, org.ID, "user@example.com")
if err != nil {
    panic(err)
}

// 複数ユーザーを一括招待
emails := []string{"user1@example.com", "user2@example.com", "user3@example.com"}
invitations, err := manager.InviteUsers(ctx, org.ID, emails)
if err != nil {
    panic(err)
}
```

### 招待の受諾

```go
// 招待を受諾（ランダムパスワードでユーザーを作成）
user, err := manager.AcceptInvitation(ctx, invitation.Token)
if err != nil {
    panic(err)
}

// パスワードを更新（招待ステータスが accepted になる）
err = manager.UpdatePassword(ctx, user.ID, org.ID, "newpassword123")
if err != nil {
    panic(err)
}
```

### 権限チェック

```go
// ユーザーが操作を実行する権限を持つか確認
err := manager.CheckPermission(ctx, userID, orgID, "update")
if err != nil {
    // 権限なし
}

// 組織へのアクセスを検証
err := manager.ValidateOrganizationAccess(ctx, userID, orgID)
if err != nil {
    // アクセス拒否
}
```

## カスタマイズ

### カスタム RoleChecker

```go
type CustomRoleChecker struct{}

func (c *CustomRoleChecker) HasPermission(role orgboss.Role, action string) bool {
    switch role {
    case orgboss.RoleManager:
        return true
    case orgboss.RoleUser:
        return action == "read" || action == "update"
    default:
        return false
    }
}

config := orgboss.DefaultConfig()
config.RoleChecker = &CustomRoleChecker{}
manager := orgboss.NewManager(config)
```

### カスタム DeletionHandler

```go
type CustomDeletionHandler struct {
    storage orgboss.Storage
}

func (h *CustomDeletionHandler) DeleteUser(ctx context.Context, user *orgboss.User) error {
    // 独自の削除ロジック（例: メールマスク、匿名化）
    user.Email = "deleted@example.com"
    return h.storage.UpdateUser(ctx, user)
}

func (h *CustomDeletionHandler) DeleteOrganization(ctx context.Context, org *orgboss.Organization) error {
    // 独自の組織削除ロジック
    return h.storage.DeleteOrganization(ctx, org.ID)
}

func (h *CustomDeletionHandler) SetStorage(storage orgboss.Storage) {
    h.storage = storage
}

config := orgboss.DefaultConfig()
config.DeletionHandler = &CustomDeletionHandler{}
manager := orgboss.NewManager(config)
```

### カスタム EmailSender

```go
import "github.com/b4m-oss/orgboss/internal/email"

// 組み込みの SMTP メール送信を利用
smtpSender := email.NewSMTPEmailSender()
// 環境変数で設定:
// SMTP_HOST=localhost
// SMTP_PORT=1025
// SMTP_FROM=noreply@example.com

config := orgboss.DefaultConfig()
config.EmailSender = smtpSender
config.InvitationBaseURL = "http://localhost:8080"
manager := orgboss.NewManager(config)
```

### フックの利用

```go
manager := orgboss.NewManager(nil)

// 組織作成前のフック
manager.Hooks().BeforeOrganizationCreate = func(ctx context.Context, data interface{}) error {
    // 組織作成前の独自処理
    return nil
}

// 組織作成後のフック
manager.Hooks().AfterOrganizationCreate = func(ctx context.Context, data interface{}) error {
    org := data.(*orgboss.Organization)
    // 組織作成後の独自処理
    return nil
}
```

## Authboss 連携

orgboss は認証に [Authboss v3](https://github.com/aarondl/authboss) と統合できます。

### セットアップ

```go
import (
    "github.com/aarondl/authboss/v3"
    authbossuser "github.com/b4m-oss/orgboss/internal/authboss"
    "gorm.io/gorm"
)

// orgboss 連携で Authboss をセットアップ
ab := &authboss.Authboss{}
err := authbossuser.SetupAuthboss(db, ab)
if err != nil {
    panic(err)
}

// パスワードリセット後の自動ログインを有効化
err = authbossuser.SetupAuthbossWithAutoLogin(db, ab, true)
if err != nil {
    panic(err)
}
```

### User モデルの拡張

orgboss は Authboss の User モデルに `organization_id` と `role` を追加します。`authbossuser.User` は Authboss の User インターフェースを実装しています。

## 開発

### 前提条件

- Go 1.25 以上
- Docker および Docker Compose
- PostgreSQL（Docker Compose 経由）

### テストの実行

```bash
# 単体テスト
make test

# 統合テスト（Docker が必要）
make test-integration

# カバレッジ付きテスト
make test COV=TRUE
```

### 開発環境

```bash
# 開発環境を起動（PostgreSQL, Mailpit）
make up

# データベースマイグレーション
make migrate

# 開発環境を停止
make down
```

### 利用可能な Make コマンド

- `make up` - 開発環境を起動
- `make down` - 開発環境を停止
- `make test` - 単体テストを実行
- `make test-integration` - 統合テストを実行
- `make fmt` - コードをフォーマット
- `make tidy` - 依存関係を整理
- `make migrate` - データベースマイグレーションを実行

## バージョン

現在のバージョン: **0.3.0-.rc1**

## 注意: 本番環境では使用しないでください

このモジュールはまだ安定版ではありません。

## ライセンス

MIT License - 詳細は [LICENSE](LICENSE) を参照してください。
