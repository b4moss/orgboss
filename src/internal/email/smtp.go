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

	"github.com/b4m-oss/orgboss/types"
)

//go:embed templates/invitation_subject.txt
var defaultSubjectTemplate string

//go:embed templates/invitation_body.txt
var defaultBodyTemplate string

// InvitationTemplateData is the data structure for invitation email templates
type InvitationTemplateData struct {
	Email          string
	InvitationURL  string
	Token          string
	ExpiresAt      string // Formatted
	OrganizationID uint
}

// SMTPEmailSender is an implementation that sends emails using SMTP
type SMTPEmailSender struct {
	host                    string
	port                    int
	from                    string
	subjectTemplatePath     string
	bodyTemplatePath        string
	subjectTemplate         *template.Template
	bodyTemplate            *template.Template
}

// NewSMTPEmailSender creates an SMTPEmailSender by loading settings from environment variables
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

	// Load default templates
	sender.loadDefaultTemplates()

	return sender
}

// NewSMTPEmailSenderWithTemplates creates an SMTPEmailSender with specified template paths
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

	// Load templates
	sender.loadTemplates()

	return sender
}

// SetTemplatePaths sets the template paths
func (s *SMTPEmailSender) SetTemplatePaths(subjectTemplatePath, bodyTemplatePath string) {
	s.subjectTemplatePath = subjectTemplatePath
	s.bodyTemplatePath = bodyTemplatePath
	s.loadTemplates()
}

// SetFrom sets the From address
func (s *SMTPEmailSender) SetFrom(from string) {
	s.from = from
}

// SendInvitation sends an invitation email
func (s *SMTPEmailSender) SendInvitation(ctx context.Context, invitation *types.Invitation, invitationURL string) error {
	// Prepare template data
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

// loadTemplates loads templates
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

// loadDefaultTemplates loads default templates (kept for backward compatibility)
func (s *SMTPEmailSender) loadDefaultTemplates() {
	s.loadDefaultSubjectTemplate()
	s.loadDefaultBodyTemplate()
}

// loadDefaultSubjectTemplate loads the default subject template
func (s *SMTPEmailSender) loadDefaultSubjectTemplate() {
	var err error
	s.subjectTemplate, err = template.New("subject").Parse(defaultSubjectTemplate)
	if err != nil {
		// パースエラーの場合はフォールバック
		s.subjectTemplate, _ = template.New("subject").Parse("組織への招待")
	}
}

// loadDefaultBodyTemplate loads the default body template
func (s *SMTPEmailSender) loadDefaultBodyTemplate() {
	var err error
	s.bodyTemplate, err = template.New("body").Parse(defaultBodyTemplate)
	if err != nil {
		// パースエラーの場合はフォールバック
		s.bodyTemplate, _ = template.New("body").Parse(`こんにちは、{{.Email}}

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

// loadTemplateFromFile loads a template from a file
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

// executeSubjectTemplate executes the subject template
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

// executeBodyTemplate executes the body template
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

// buildInvitationEmailBody builds the body of an invitation email (kept for backward compatibility)
// This method is deprecated. Please use the template functionality instead.
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

// BuildEmailMessage builds an email message (exposed for testing)
func (s *SMTPEmailSender) BuildEmailMessage(to, subject, body string) string {
	return s.buildEmailMessage(to, subject, body)
}

// buildEmailMessage builds an email message
func (s *SMTPEmailSender) buildEmailMessage(to, subject, body string) string {
	message := fmt.Sprintf("From: %s\r\n", s.from)
	message += fmt.Sprintf("To: %s\r\n", to)
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "Content-Type: text/plain; charset=UTF-8\r\n"
	message += "\r\n"
	message += body
	return message
}

// getEnv gets an environment variable and returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

