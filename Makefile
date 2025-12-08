.PHONY: help up down build logs clean db shell restart test test-integration test-integration-keep-data test-authboss-integration test-authboss-integration-keep-data fmt tidy migrate migrate-seed

# デフォルトターゲット
help:
	@echo "利用可能なコマンド:"
	@echo "  make up          - 開発環境を起動"
	@echo "  make down        - 開発環境を停止"
	@echo "  make build       - イメージをビルド"
	@echo "  make logs        - ログを表示"
	@echo "  make clean       - コンテナとボリュームを削除"
	@echo "  make db          - Postgresに接続"
	@echo "  make shell       - アプリコンテナのシェルに入る"
	@echo "  make restart     - 開発環境を再起動"
	@echo "  make test        - テストを実行"
	@echo "  make test-integration - 統合テストを実行（Dockerが必要）"
	@echo "  make test-integration-keep-data - 統合テストを実行（テストデータを保持）"
	@echo "  make test-authboss-integration - Authboss統合テストを実行（Dockerが必要）"
	@echo "  make test-authboss-integration-keep-data - Authboss統合テストを実行（テストデータを保持）"
	@echo "  make fmt         - コードをフォーマット"
	@echo "  make tidy        - 依存関係を整理"
	@echo "  make migrate     - データベースマイグレーション実行"
	@echo "  make migrate-seed - マイグレーションとシーディング実行"

# 開発環境を起動
up:
	cd src && docker compose up -d

# 開発環境を停止
down:
	cd src && docker compose down

# イメージをビルド
build:
	cd src && docker compose build

# ログを表示
logs:
	cd src && docker compose logs -f

# コンテナとボリュームを削除
clean:
	cd src && docker compose down -v
	docker volume prune -f

# Postgresに接続
db:
	cd src && docker compose exec orgboss-db psql -U orgboss -d orgboss

# アプリコンテナのシェルに入る
shell:
	cd src && docker compose exec orgboss-dev sh

# 開発環境を再起動
restart: down up

# テストを実行
test:
	cd src && docker compose exec orgboss-dev go test -short ./...

# 統合テストを実行（Dockerが必要）
test-integration:
	cd src && docker compose exec orgboss-dev go test -v -run TestIntegration ./...

# 統合テストを実行（テストデータを保持）
test-integration-keep-data:
	cd src && docker compose exec -e SKIP_CLEANUP=true orgboss-dev sh -c "go test -v -run TestIntegration ./..."

# Authboss統合テストを実行（Dockerが必要）
test-authboss-integration:
	cd src && docker compose exec orgboss-dev go test -v -run TestAuthbossIntegration ./...

# Authboss統合テストを実行（テストデータを保持）
test-authboss-integration-keep-data:
	cd src && docker compose exec -e SKIP_CLEANUP=true orgboss-dev sh -c "go test -v -run TestAuthbossIntegration ./..."

# コードをフォーマット
fmt:
	cd src && docker compose exec orgboss-dev go fmt ./...

# 依存関係を整理
tidy:
	cd src && docker compose exec orgboss-dev go mod tidy

# データベースマイグレーション実行
migrate:
	cd src && docker compose exec orgboss-dev go run cmd/migrate/main.go

# マイグレーションとシーディング実行
migrate-seed:
	cd src && docker compose exec orgboss-dev go run cmd/migrate/main.go -seed

