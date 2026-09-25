# 統合テストガイド

## 概要

このプロジェクトには、実際のPostgreSQLデータベースを使用した統合テストが含まれています。統合テストは、Docker Composeで起動したPostgreSQLコンテナ（orgboss-db）に接続して実行されます。

## 前提条件

- DockerとDocker Composeがインストールされていること
- Go 1.24以上がインストールされていること

## 統合テストの実行

### 1. Docker Composeでデータベースを起動

```bash
make up
```

### 2. 統合テストを実行

```bash
# 通常の統合テスト（テスト後にデータを削除）
make test-integration

# テストデータを保持する統合テスト（データ確認用）
make test-integration-keep-data
```

### すべてのテストを実行（単体テスト + 統合テスト）

```bash
# 単体テスト（統合テストはスキップ）
make test

# 統合テスト
make test-integration
```

### 統合テストをスキップして単体テストのみ実行

```bash
cd src
go test -short ./...
```

## データベースマイグレーション

### 開発環境でのマイグレーション

```bash
make migrate
```

### マイグレーション + シーディング

```bash
make migrate-seed
```

## 統合テストの内容

統合テストでは以下の機能をテストしています：

1. **CreateOrganizationWithUser**: 組織とユーザーの作成
2. **InviteUser**: ユーザー招待機能
3. **AcceptInvitation**: 招待承諾機能
4. **DeleteUser**: ユーザー削除機能
5. **DeleteOrganization**: 組織削除機能
6. **シードデータを使用したテスト**: 事前に投入されたデータを使用したテスト

## テストデータ

統合テストでは、`internal/seed/seed.go`で定義されたシードデータを使用します：

- 2つの組織（テスト組織1、テスト組織2）
- 3人のユーザー（manager1、user1、manager2）
- 3つの招待（pending、expired、accepted）

## Authboss統合

`internal/authboss/integration.go`でAuthbossとの統合を実装しています。統合テストでは、実際のAuthbossインスタンスを使用したテストは含まれていませんが、統合のための構造体と関数を提供しています。

## 注意事項

- 統合テストを実行する前に、`make up`でDocker Composeを起動している必要があります
- 統合テストは既存のPostgreSQLコンテナ（orgboss-db）に接続します
- 各テストの実行後、テストデータは自動的にクリーンアップされます（デフォルト動作）
- `make test-integration-keep-data`を使用すると、テストデータが保持されます（データ確認用）
- テストは並列実行を避けるため、`-parallel 1`オプションを使用することを推奨します

## トラブルシューティング

### データベース接続エラーが発生する場合

1. Docker Composeが起動していることを確認：
   ```bash
   docker compose ps
   ```

2. データベースコンテナが正常に動作していることを確認：
   ```bash
   make db
   ```

3. 環境変数が正しく設定されていることを確認（compose.ymlを参照）

