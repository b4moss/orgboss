.PHONY: help up down build logs clean db shell restart test test-integration test-authboss-integration fmt tidy migrate migrate-seed

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
	@echo "  make test [COV=TRUE] - テストを実行（COV=TRUEでカバレッジを有効化）"
	@echo "  make test-integration [KEEP=1] - 統合テストを実行（Dockerが必要、Authbossテスト含む）"
	@echo "                                    KEEP=1を指定するとテストデータを保持"
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
# COV=TRUEを指定するとカバレッジを有効化（例: make test COV=TRUE）
# テストがあるパッケージ（orgboss）のみをカバレッジに含める
test:
	@if [ "$(COV)" = "TRUE" ] || [ "$(COV)" = "true" ] || [ "$(cov)" = "TRUE" ] || [ "$(cov)" = "true" ]; then \
		echo "カバレッジを有効化してテストを実行します（テストがあるパッケージのみ）..."; \
		cd src && docker compose exec orgboss-dev sh -c "go test -short -coverprofile=coverage.out . && go tool cover -html=coverage.out -o coverage.html"; \
	else \
		cd src && docker compose exec orgboss-dev go test -short ./...; \
	fi

# 統合テストを実行（Dockerが必要、Authbossテスト含む）
# KEEP=1を指定するとテストデータを保持（例: make test-integration KEEP=1）
test-integration:
	@if [ "$(KEEP)" = "1" ] || [ "$(keep)" = "1" ]; then \
		cd src && docker compose exec -e SKIP_CLEANUP=true orgboss-dev sh -c "go test -v -run TestIntegration ./..."; \
	else \
		cd src && docker compose exec orgboss-dev go test -v -run TestIntegration ./...; \
	fi

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

