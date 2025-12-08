package email

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSMTPEmailSender_LoadDefaultTemplates(t *testing.T) {
	sender := NewSMTPEmailSender()
	
	assert.NotNil(t, sender, "SMTPEmailSenderが作成される")
	assert.NotNil(t, sender.subjectTemplate, "件名テンプレートが読み込まれる")
	assert.NotNil(t, sender.bodyTemplate, "本文テンプレートが読み込まれる")
}

func TestSMTPEmailSender_ExecuteDefaultTemplates(t *testing.T) {
	sender := NewSMTPEmailSender()
	
	data := InvitationTemplateData{
		Email:          "test@example.com",
		InvitationURL:  "http://example.com/invite/token123",
		Token:          "token123",
		ExpiresAt:      "2024-01-01 12:00:00",
		OrganizationID: 1,
	}

	subject, err := sender.executeSubjectTemplate(data)
	require.NoError(t, err, "件名テンプレートの実行に成功する")
	assert.Equal(t, "組織への招待", subject, "デフォルト件名が正しい")

	body, err := sender.executeBodyTemplate(data)
	require.NoError(t, err, "本文テンプレートの実行に成功する")
	assert.Contains(t, body, "こんにちは、", "本文に挨拶が含まれる")
	assert.Contains(t, body, "http://example.com/invite/token123", "本文に招待URLが含まれる")
	assert.Contains(t, body, "2024-01-01 12:00:00", "本文に有効期限が含まれる")
}

func TestSMTPEmailSender_ExecuteDefaultTemplates_WithoutURL(t *testing.T) {
	sender := NewSMTPEmailSender()
	
	data := InvitationTemplateData{
		Email:          "test@example.com",
		InvitationURL:  "",
		Token:          "token123",
		ExpiresAt:      "2024-01-01 12:00:00",
		OrganizationID: 1,
	}

	body, err := sender.executeBodyTemplate(data)
	require.NoError(t, err, "本文テンプレートの実行に成功する")
	assert.Contains(t, body, "招待トークン: token123", "本文にトークンが含まれる")
	assert.NotContains(t, body, "http://example.com/invite/token123", "URLがない場合はURLが含まれない")
}

func TestSMTPEmailSender_SetTemplatePaths_CustomTemplates(t *testing.T) {
	// 一時ディレクトリを作成
	tmpDir := t.TempDir()
	
	// カスタム件名テンプレートファイルを作成
	subjectPath := filepath.Join(tmpDir, "subject.txt")
	err := os.WriteFile(subjectPath, []byte("カスタム件名: {{.Email}}への招待"), 0644)
	require.NoError(t, err)

	// カスタム本文テンプレートファイルを作成
	bodyPath := filepath.Join(tmpDir, "body.txt")
	err = os.WriteFile(bodyPath, []byte("こんにちは、{{.Email}}さん\n\n組織ID: {{.OrganizationID}}への招待です。"), 0644)
	require.NoError(t, err)

	sender := NewSMTPEmailSender()
	sender.SetTemplatePaths(subjectPath, bodyPath)

	data := InvitationTemplateData{
		Email:          "test@example.com",
		InvitationURL:  "http://example.com/invite/token123",
		Token:          "token123",
		ExpiresAt:      "2024-01-01 12:00:00",
		OrganizationID: 42,
	}

	subject, err := sender.executeSubjectTemplate(data)
	require.NoError(t, err, "カスタム件名テンプレートの実行に成功する")
	assert.Equal(t, "カスタム件名: test@example.comへの招待", subject, "カスタム件名が正しい")

	body, err := sender.executeBodyTemplate(data)
	require.NoError(t, err, "カスタム本文テンプレートの実行に成功する")
	assert.Contains(t, body, "test@example.comさん", "カスタム本文にメールアドレスが含まれる")
	assert.Contains(t, body, "組織ID: 42への招待です。", "カスタム本文に組織IDが含まれる")
}

func TestSMTPEmailSender_SetTemplatePaths_NonExistentFile(t *testing.T) {
	sender := NewSMTPEmailSender()
	
	// 存在しないファイルパスを設定
	sender.SetTemplatePaths("/nonexistent/subject.txt", "/nonexistent/body.txt")

	// デフォルトテンプレートが使用されることを確認
	assert.NotNil(t, sender.subjectTemplate, "件名テンプレートがデフォルトで読み込まれる")
	assert.NotNil(t, sender.bodyTemplate, "本文テンプレートがデフォルトで読み込まれる")

	data := InvitationTemplateData{
		Email:          "test@example.com",
		InvitationURL:  "http://example.com/invite/token123",
		Token:          "token123",
		ExpiresAt:      "2024-01-01 12:00:00",
		OrganizationID: 1,
	}

	subject, err := sender.executeSubjectTemplate(data)
	require.NoError(t, err, "デフォルト件名テンプレートの実行に成功する")
	assert.Equal(t, "組織への招待", subject, "デフォルト件名が使用される")
}

func TestSMTPEmailSender_SetFrom(t *testing.T) {
	sender := NewSMTPEmailSender()
	
	originalFrom := sender.from
	sender.SetFrom("custom@example.com")
	
	assert.Equal(t, "custom@example.com", sender.from, "差出人が更新される")
	assert.NotEqual(t, originalFrom, sender.from, "元の差出人と異なる")
}

func TestNewSMTPEmailSenderWithTemplates(t *testing.T) {
	// 一時ディレクトリを作成
	tmpDir := t.TempDir()
	
	// カスタムテンプレートファイルを作成
	subjectPath := filepath.Join(tmpDir, "subject.txt")
	err := os.WriteFile(subjectPath, []byte("カスタム件名"), 0644)
	require.NoError(t, err)

	bodyPath := filepath.Join(tmpDir, "body.txt")
	err = os.WriteFile(bodyPath, []byte("カスタム本文"), 0644)
	require.NoError(t, err)

	sender := NewSMTPEmailSenderWithTemplates(subjectPath, bodyPath, "custom@example.com")
	
	assert.NotNil(t, sender, "SMTPEmailSenderが作成される")
	assert.Equal(t, "custom@example.com", sender.from, "差出人が設定される")
	assert.NotNil(t, sender.subjectTemplate, "件名テンプレートが読み込まれる")
	assert.NotNil(t, sender.bodyTemplate, "本文テンプレートが読み込まれる")

	data := InvitationTemplateData{
		Email:          "test@example.com",
		InvitationURL:  "",
		Token:          "token123",
		ExpiresAt:      "2024-01-01 12:00:00",
		OrganizationID: 1,
	}

	subject, err := sender.executeSubjectTemplate(data)
	require.NoError(t, err)
	assert.Equal(t, "カスタム件名", subject, "カスタム件名が使用される")

	body, err := sender.executeBodyTemplate(data)
	require.NoError(t, err)
	assert.Equal(t, "カスタム本文", body, "カスタム本文が使用される")
}

func TestSMTPEmailSender_buildEmailMessage(t *testing.T) {
	sender := NewSMTPEmailSender()
	sender.SetFrom("test@example.com")
	
	message := sender.buildEmailMessage("recipient@example.com", "テスト件名", "テスト本文")
	
	assert.Contains(t, message, "From: test@example.com", "差出人が含まれる")
	assert.Contains(t, message, "To: recipient@example.com", "宛先が含まれる")
	assert.Contains(t, message, "Subject: テスト件名", "件名が含まれる")
	assert.Contains(t, message, "テスト本文", "本文が含まれる")
	assert.Contains(t, message, "Content-Type: text/plain; charset=UTF-8", "Content-Typeが含まれる")
}

// Test edge cases for template execution
func TestSMTPEmailSender_TemplateVariables(t *testing.T) {
	sender := NewSMTPEmailSender()
	
	data := InvitationTemplateData{
		Email:          "user@example.com",
		InvitationURL:  "http://example.com/invite/abc123",
		Token:          "abc123",
		ExpiresAt:      "2024-12-31 23:59:59",
		OrganizationID: 999,
	}

	subject, err := sender.executeSubjectTemplate(data)
	require.NoError(t, err)
	assert.NotEmpty(t, subject, "件名が空でない")

	body, err := sender.executeBodyTemplate(data)
	require.NoError(t, err)
	assert.Contains(t, body, "user@example.com", "メールアドレスが含まれる")
	assert.Contains(t, body, "abc123", "トークンが含まれる")
	assert.Contains(t, body, "2024-12-31 23:59:59", "有効期限が含まれる")
}

