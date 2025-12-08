package email

import (
	"context"
	"fmt"
	"net/smtp"
	"os"
	"strconv"

	"orgboss/types"
)

// SMTPEmailSender はSMTPを使用してメールを送信する実装
type SMTPEmailSender struct {
	host string
	port int
	from string
}

// NewSMTPEmailSender は環境変数から設定を読み込んでSMTPEmailSenderを作成する
func NewSMTPEmailSender() *SMTPEmailSender {
	host := getEnv("SMTP_HOST", "localhost")
	portStr := getEnv("SMTP_PORT", "1025")
	port, _ := strconv.Atoi(portStr)
	from := getEnv("SMTP_FROM", "noreply@example.com")

	return &SMTPEmailSender{
		host: host,
		port: port,
		from: from,
	}
}

// SendInvitation は招待メールを送信する
func (s *SMTPEmailSender) SendInvitation(ctx context.Context, invitation *types.Invitation) error {
	// メールの件名と本文を構築
	subject := "組織への招待"
	body := s.buildInvitationEmailBody(invitation)

	// メールヘッダーと本文を構築
	message := s.buildEmailMessage(invitation.Email, subject, body)

	// SMTPサーバーに接続してメールを送信
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	err := smtp.SendMail(addr, nil, s.from, []string{invitation.Email}, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// buildInvitationEmailBody は招待メールの本文を構築する
func (s *SMTPEmailSender) buildInvitationEmailBody(invitation *types.Invitation) string {
	// 実際のアプリケーションでは、招待URLを構築する必要があります
	// ここではテスト用のシンプルな形式を使用します
	return fmt.Sprintf(`こんにちは、

あなたは組織への招待を受けました。

招待トークン: %s
有効期限: %s

このトークンを使用して招待を承諾してください。

よろしくお願いいたします。`,
		invitation.Token,
		invitation.ExpiresAt.Format("2006-01-02 15:04:05"),
	)
}

// buildEmailMessage はメールメッセージを構築する
func (s *SMTPEmailSender) buildEmailMessage(to, subject, body string) string {
	message := fmt.Sprintf("From: %s\r\n", s.from)
	message += fmt.Sprintf("To: %s\r\n", to)
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "Content-Type: text/plain; charset=UTF-8\r\n"
	message += "\r\n"
	message += body
	return message
}

// getEnv は環境変数を取得し、デフォルト値を返す
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

