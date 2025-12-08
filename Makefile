.PHONY: help up down build logs clean db shell restart test fmt tidy

# デフォルトターゲット
help:
	@echo "利用可能なコマンド:"
	@echo "  make up      - 開発環境を起動"
	@echo "  make down    - 開発環境を停止"
	@echo "  make build   - イメージをビルド"
	@echo "  make logs    - ログを表示"
	@echo "  make clean   - コンテナとボリュームを削除"
	@echo "  make db      - Postgresに接続"
	@echo "  make shell   - アプリコンテナのシェルに入る"
	@echo "  make restart - 開発環境を再起動"
	@echo "  make test    - テストを実行"
	@echo "  make fmt     - コードをフォーマット"
	@echo "  make tidy    - 依存関係を整理"

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
	cd src && docker compose exec orgboss-dev go test ./...

# コードをフォーマット
fmt:
	cd src && docker compose exec orgboss-dev go fmt ./...

# 依存関係を整理
tidy:
	cd src && docker compose exec orgboss-dev go mod tidy

