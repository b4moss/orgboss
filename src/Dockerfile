# Dockerfile
FROM golang:1.25-alpine

WORKDIR /app

# Airのインストールに必要なパッケージをインストール
RUN apk add --no-cache git ca-certificates

# Airをインストール（PATHに追加、Go 1.24と互換性のあるバージョンを指定）
ENV PATH="${PATH}:/go/bin"
RUN go install github.com/cosmtrek/air@v1.49.0

# go.modとgo.sum（存在する場合）をコピー
# go.modが存在しない場合は初期化する
COPY go.mod* ./
COPY go.sum* ./
RUN if [ ! -f go.mod ]; then \
        go mod init orgboss; \
    fi && \
    go mod download && go mod verify

# ソースコードをコピー
COPY . .

# 開発環境ではAirを使用するため、ビルドは不要
# 本番環境用のビルドが必要な場合は別のDockerfileを作成