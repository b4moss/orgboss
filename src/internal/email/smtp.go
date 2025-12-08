package email

import (
	"context"
	_ "embed"
	"fmt"
	"net/smtp"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"orgboss/types"
)

//go:embed templates/invitation_subject.txt
var defaultSubjectTemplate string

//go:embed templates/invitation_body.txt
var defaultBodyTemplate string

// InvitationTemplateData は招待メールテンプレートのデータ構造体
type InvitationTemplateData struct {
	Email          string
	InvitationURL  string
	Token          string
	ExpiresAt      string // フォーマット済み
	OrganizationID uint
}

// SMTPEmailSender はSMTPを使用してメールを送信する実装
type SMTPEmailSender struct {
	host                    string
	port                    int
	from                    string
	subjectTemplatePath     string
	bodyTemplatePath        string
	subjectTemplate         *template.Template
	bodyTemplate            *template.Template
}

// NewSMTPEmailSender は環境変数から設定を読み込んでSMTPEmailSenderを作成する
func NewSMTPEmailSender() *SMTPEmailSender {
	host := getEnv("SMTP_HOST", "localhost")
	portStr := getEnv("SMTP_PORT", "1025")
	port, _ := strconv.Atoi(portStr)
	from := getEnv("SMTP_FROM", "noreply@example.com")

	sender := &SMTPEmailSender{
		host: host,
		port: port,
		from: from,
	}

	// デフォルトテンプレートを読み込む
	sender.loadDefaultTemplates()

	return sender
}

// NewSMTPEmailSenderWithTemplates はテンプレートパスを指定してSMTPEmailSenderを作成する
func NewSMTPEmailSenderWithTemplates(subjectTemplatePath, bodyTemplatePath, fromOverride string) *SMTPEmailSender {
	host := getEnv("SMTP_HOST", "localhost")
	portStr := getEnv("SMTP_PORT", "1025")
	port, _ := strconv.Atoi(portStr)
	from := getEnv("SMTP_FROM", "noreply@example.com")
	if fromOverride != "" {
		from = fromOverride
	}

	sender := &SMTPEmailSender{
		host:                host,
		port:                port,
		from:                from,
		subjectTemplatePath: subjectTemplatePath,
		bodyTemplatePath:    bodyTemplatePath,
	}

	// テンプレートを読み込む
	sender.loadTemplates()

	return sender
}

// SetTemplatePaths はテンプレートパスを設定する
func (s *SMTPEmailSender) SetTemplatePaths(subjectTemplatePath, bodyTemplatePath string) {
	s.subjectTemplatePath = subjectTemplatePath
	s.bodyTemplatePath = bodyTemplatePath
	s.loadTemplates()
}

// SetFrom は差出人を設定する
func (s *SMTPEmailSender) SetFrom(from string) {
	s.from = from
}

// SendInvitation は招待メールを送信する
func (s *SMTPEmailSender) SendInvitation(ctx context.Context, invitation *types.Invitation, invitationURL string) error {
	// テンプレートデータを準備
	data := InvitationTemplateData{
		Email:          invitation.Email,
		InvitationURL:  invitationURL,
		Token:          invitation.Token,
		ExpiresAt:      invitation.ExpiresAt.Format("2006-01-02 15:04:05"),
		OrganizationID: invitation.OrganizationID,
	}

	// 件名と本文をテンプレートから生成
	subject, err := s.executeSubjectTemplate(data)
	if err != nil {
		return fmt.Errorf("failed to execute subject template: %w", err)
	}

	body, err := s.executeBodyTemplate(data)
	if err != nil {
		return fmt.Errorf("failed to execute body template: %w", err)
	}

	// メールヘッダーと本文を構築
	message := s.buildEmailMessage(invitation.Email, subject, body)

	// SMTPサーバーに接続してメールを送信
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	err = smtp.SendMail(addr, nil, s.from, []string{invitation.Email}, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// loadTemplates はテンプレートを読み込む
func (s *SMTPEmailSender) loadTemplates() {
	// 件名テンプレートの読み込み
	if s.subjectTemplatePath != "" {
		if tmpl, err := s.loadTemplateFromFile(s.subjectTemplatePath); err == nil {
			s.subjectTemplate = tmpl
		} else {
			// ファイル読み込みに失敗した場合はデフォルト件名テンプレートを使用
			s.loadDefaultSubjectTemplate()
		}
	} else {
		s.loadDefaultSubjectTemplate()
	}

	// 本文テンプレートの読み込み
	if s.bodyTemplatePath != "" {
		if tmpl, err := s.loadTemplateFromFile(s.bodyTemplatePath); err == nil {
			s.bodyTemplate = tmpl
		} else {
			// ファイル読み込みに失敗した場合はデフォルト本文テンプレートを使用
			s.loadDefaultBodyTemplate()
		}
	} else {
		s.loadDefaultBodyTemplate()
	}
}

// loadDefaultTemplates はデフォルトテンプレートを読み込む（後方互換性のため保持）
func (s *SMTPEmailSender) loadDefaultTemplates() {
	s.loadDefaultSubjectTemplate()
	s.loadDefaultBodyTemplate()
}

// loadDefaultSubjectTemplate はデフォルト件名テンプレートを読み込む
func (s *SMTPEmailSender) loadDefaultSubjectTemplate() {
	var err error
	s.subjectTemplate, err = template.New("subject").Parse(defaultSubjectTemplate)
	if err != nil {
		// パースエラーの場合はフォールバック
		s.subjectTemplate, _ = template.New("subject").Parse("組織への招待")
	}
}

// loadDefaultBodyTemplate はデフォルト本文テンプレートを読み込む
func (s *SMTPEmailSender) loadDefaultBodyTemplate() {
	var err error
	s.bodyTemplate, err = template.New("body").Parse(defaultBodyTemplate)
	if err != nil {
		// パースエラーの場合はフォールバック
		s.bodyTemplate, _ = template.New("body").Parse(`こんにちは、

あなたは組織への招待を受けました。

{{if .InvitationURL}}
以下のリンクをクリックして、パスワードを設定してください：
{{.InvitationURL}}
{{else}}
招待トークン: {{.Token}}
このトークンを使用して招待を承諾してください。
{{end}}

有効期限: {{.ExpiresAt}}

よろしくお願いいたします。`)
	}
}

// loadTemplateFromFile はファイルからテンプレートを読み込む
func (s *SMTPEmailSender) loadTemplateFromFile(path string) (*template.Template, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file %s: %w", path, err)
	}

	name := filepath.Base(path)
	tmpl, err := template.New(name).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template file %s: %w", path, err)
	}

	return tmpl, nil
}

// executeSubjectTemplate は件名テンプレートを実行する
func (s *SMTPEmailSender) executeSubjectTemplate(data InvitationTemplateData) (string, error) {
	if s.subjectTemplate == nil {
		s.loadDefaultTemplates()
	}

	var buf strings.Builder
	if err := s.subjectTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute subject template: %w", err)
	}
	return strings.TrimSpace(buf.String()), nil
}

// executeBodyTemplate は本文テンプレートを実行する
func (s *SMTPEmailSender) executeBodyTemplate(data InvitationTemplateData) (string, error) {
	if s.bodyTemplate == nil {
		s.loadDefaultTemplates()
	}

	var buf strings.Builder
	if err := s.bodyTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute body template: %w", err)
	}
	return buf.String(), nil
}

// buildInvitationEmailBody は招待メールの本文を構築する（後方互換性のため保持）
// このメソッドは非推奨です。テンプレート機能を使用してください。
func (s *SMTPEmailSender) buildInvitationEmailBody(invitation *types.Invitation, invitationURL string) string {
	if invitationURL != "" {
		// URLが提供されている場合、クリック可能なリンクを含める
		return fmt.Sprintf(`こんにちは、

あなたは組織への招待を受けました。

以下のリンクをクリックして、パスワードを設定してください：
%s

有効期限: %s

よろしくお願いいたします。`,
			invitationURL,
			invitation.ExpiresAt.Format("2006-01-02 15:04:05"),
		)
	}
	// URLが提供されていない場合、従来の形式（後方互換性のため）
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

// BuildEmailMessage はメールメッセージを構築する（テスト用に公開）
func (s *SMTPEmailSender) BuildEmailMessage(to, subject, body string) string {
	return s.buildEmailMessage(to, subject, body)
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

