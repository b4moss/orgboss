package orgboss

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/b4m-oss/orgboss/internal/email"
	"github.com/b4m-oss/orgboss/internal/storage"
	"github.com/b4m-oss/orgboss/internal/validation"
)

// Manager はorgbossのコア機能を提供する
type Manager struct {
	config  *Config
	hooks   *Hooks
	storage Storage
}

// NewManager は新しいManagerを作成する
func NewManager(config *Config) *Manager {
	if config == nil {
		config = DefaultConfig()
	}
	stor := storage.NewInMemoryStorage()
	
	// デフォルト実装を設定
	if config.RoleChecker == nil {
		config.RoleChecker = NewDefaultRoleChecker()
	}
	if config.DeletionHandler == nil {
		config.DeletionHandler = NewDefaultDeletionHandler(stor)
	} else {
		// DeletionHandlerのストレージを更新（DefaultConfigで作成されたストレージとManagerで使用するストレージを統一）
		config.DeletionHandler.SetStorage(stor)
	}

	// EmailSenderがSMTPEmailSenderの場合、テンプレート設定を適用
	applyEmailTemplateConfig(config)
	
	return &Manager{
		config:  config,
		hooks:   NewHooks(),
		storage: stor,
	}
}

// NewManagerWithStorage は指定されたストレージで新しいManagerを作成する
func NewManagerWithStorage(config *Config, storage Storage) *Manager {
	if config == nil {
		config = DefaultConfig()
	}
	
	// デフォルト実装を設定
	if config.RoleChecker == nil {
		config.RoleChecker = NewDefaultRoleChecker()
	}
	if config.DeletionHandler == nil {
		config.DeletionHandler = NewDefaultDeletionHandler(storage)
	} else {
		// DeletionHandlerのストレージを更新
		config.DeletionHandler.SetStorage(storage)
	}

	// EmailSenderがSMTPEmailSenderの場合、テンプレート設定を適用
	applyEmailTemplateConfig(config)
	
	return &Manager{
		config:  config,
		hooks:   NewHooks(),
		storage: storage,
	}
}

// CreateOrganizationWithUser はOrganizationとUserを同時に作成する
func (m *Manager) CreateOrganizationWithUser(ctx context.Context, orgName string, userEmail string) (*Organization, *User, error) {
	// バリデーション
	if err := validation.ValidateOrganizationName(orgName); err != nil {
		return nil, nil, err
	}
	if err := validation.ValidateEmail(userEmail); err != nil {
		return nil, nil, err
	}

	// BeforeOrganizationCreateフック実行
	if err := m.executeHook(m.hooks.BeforeOrganizationCreate, ctx, nil); err != nil {
		return nil, nil, err
	}

	// トランザクション開始（インメモリ実装のため、実際のトランザクションはなし）
	// Organization作成
	org, err := m.createOrganization(ctx, orgName)
	if err != nil {
		return nil, nil, err
	}

	// ランダムパスワードを生成してハッシュ化
	randomPassword, err := generateRandomPassword()
	if err != nil {
		return nil, nil, err
	}
	hashedPassword, err := hashPassword(randomPassword)
	if err != nil {
		return nil, nil, err
	}

	// User作成（role=manager, ランダムパスワード）
	user, err := m.createUserWithPassword(ctx, userEmail, org.ID, RoleManager, hashedPassword)
	if err != nil {
		return nil, nil, err
	}

	// トランザクションコミット（インメモリ実装のため、実際のコミットはなし）

	// AfterOrganizationCreateフック実行
	if err := m.executeHook(m.hooks.AfterOrganizationCreate, ctx, org); err != nil {
		return nil, nil, err
	}

	return org, user, nil
}

// InviteUser はユーザーを招待する
func (m *Manager) InviteUser(ctx context.Context, orgID uint, email string) (*Invitation, error) {
	// バリデーション
	if err := validation.ValidateEmail(email); err != nil {
		return nil, err
	}

	// BeforeInviteフック実行
	if err := m.executeHook(m.hooks.BeforeInvite, ctx, nil); err != nil {
		return nil, err
	}

	// トークン生成（ハッシュ）
	token, err := m.GenerateToken()
	if err != nil {
		return nil, err
	}

	// 有効期限計算（Config.InvitationExpiryDuration）
	expiresAt := m.CalculateExpiry()

	// Invitation作成（status=pending）
	invitation, err := m.createInvitation(ctx, email, orgID, token, expiresAt)
	if err != nil {
		return nil, err
	}

	// メール送信（EmailSender.SendInvitation）
	if m.config.EmailSender == nil {
		return nil, ErrEmailSendFailed
	}
	// 招待URLを生成
	invitationURL := m.GetInvitationURL(token)
	if err := m.config.EmailSender.SendInvitation(ctx, invitation, invitationURL); err != nil {
		return nil, ErrEmailSendFailed
	}

	// AfterInviteフック実行
	if err := m.executeHook(m.hooks.AfterInvite, ctx, invitation); err != nil {
		return nil, err
	}

	return invitation, nil
}

// InviteUsers は複数のユーザーを招待する（バルク招待）
func (m *Manager) InviteUsers(ctx context.Context, orgID uint, emails []string) ([]*Invitation, error) {
	// 招待数上限チェック（MaxBulkInviteCount）
	if len(emails) > m.config.MaxBulkInviteCount {
		return nil, ErrBulkInviteLimitExceeded
	}

	// 各メールアドレスに対してInviteUserを実行
	invitations := make([]*Invitation, 0, len(emails))
	for _, email := range emails {
		invitation, err := m.InviteUser(ctx, orgID, email)
		if err != nil {
			// 一部のメールアドレスでエラーが発生しても、他の招待は処理される
			continue
		}
		invitations = append(invitations, invitation)
	}

	return invitations, nil
}

// AcceptInvitation は招待を承諾する
// この時点ではランダムパスワードを生成してハッシュ化して保存するが、
// Invitationのstatusはpendingのまま（パスワード更新時にacceptedになる）
func (m *Manager) AcceptInvitation(ctx context.Context, token string) (*User, error) {
	// トークン検証（Invitation検索）
	invitation, err := m.storage.GetInvitationByToken(ctx, token)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// 有効期限チェック
	if m.IsExpired(invitation.ExpiresAt) {
		return nil, ErrInvitationExpired
	}

	// 既にacceptedまたはrejectedの場合、エラーが返される
	if invitation.Status == InvitationStatusAccepted {
		return nil, ErrInvitationAlreadyAccepted
	}
	if invitation.Status == InvitationStatusRejected {
		return nil, ErrInvitationAlreadyRejected
	}

	// トランザクション開始（インメモリ実装のため、実際のトランザクションはなし）

	// ランダムパスワードを生成してハッシュ化
	randomPassword, err := generateRandomPassword()
	if err != nil {
		return nil, err
	}
	hashedPassword, err := hashPassword(randomPassword)
	if err != nil {
		return nil, err
	}

	// User作成（organization_id, role=user, ランダムパスワード）
	// Invitationのstatusはpendingのまま（パスワード更新時にacceptedになる）
	user, err := m.createUserWithPassword(ctx, invitation.Email, invitation.OrganizationID, RoleUser, hashedPassword)
	if err != nil {
		return nil, err
	}

	// トランザクションコミット（インメモリ実装のため、実際のコミットはなし）

	return user, nil
}

// RejectInvitation は招待を拒否する
func (m *Manager) RejectInvitation(ctx context.Context, token string) error {
	// トークン検証（Invitation検索）
	invitation, err := m.storage.GetInvitationByToken(ctx, token)
	if err != nil {
		return ErrInvalidToken
	}

	// 有効期限チェック
	if m.IsExpired(invitation.ExpiresAt) {
		return ErrInvitationExpired
	}

	// Invitation更新（status=rejected）
	invitation.Status = InvitationStatusRejected
	if err := m.storage.UpdateInvitation(ctx, invitation); err != nil {
		return err
	}

	return nil
}

// DeleteUser はユーザーを削除する
func (m *Manager) DeleteUser(ctx context.Context, userID uint, orgID uint) error {
	// User取得
	user, err := m.storage.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	// 権限チェック（ValidateOrganizationAccess）
	// userIDとorgIDで自分自身または同じ組織のメンバーであることを確認
	if err := m.ValidateOrganizationAccess(ctx, userID, orgID); err != nil {
		return err
	}

	// organization_id一致確認
	if user.OrganizationID != orgID {
		return ErrOrganizationAccessDenied
	}

	// BeforeUserDeleteフック実行
	if err := m.executeHook(m.hooks.BeforeUserDelete, ctx, user); err != nil {
		return err
	}

	// ロール判定
	if user.Role == RoleManager {
		// manager退会の場合：DeleteOrganization実行
		if err := m.DeleteOrganization(ctx, orgID); err != nil {
			return err
		}
	} else {
		// user退会の場合：DeleteUser実行（DeletionHandler）
		if m.config.DeletionHandler == nil {
			return ErrPermissionDenied
		}
		if err := m.config.DeletionHandler.DeleteUser(ctx, user); err != nil {
			return err
		}
	}

	// AfterUserDeleteフック実行
	if err := m.executeHook(m.hooks.AfterUserDelete, ctx, user); err != nil {
		return err
	}

	return nil
}

// DeleteOrganization は組織を削除する
func (m *Manager) DeleteOrganization(ctx context.Context, orgID uint) error {
	// Organization取得
	org, err := m.storage.GetOrganization(ctx, orgID)
	if err != nil {
		return err
	}

	// 関連User取得（エラーが返されても続行）
	users, _ := m.storage.GetUsersByOrganizationID(ctx, orgID)

	// DeletionHandlerが設定されていない場合はエラー
	if m.config.DeletionHandler == nil {
		return ErrPermissionDenied
	}

	// 関連User削除（論理/物理/マスク、DeletionHandler）
	for _, user := range users {
		if err := m.config.DeletionHandler.DeleteUser(ctx, user); err != nil {
			return err
		}
	}

	// Organization削除（論理/物理、DeletionHandler）
	if err := m.config.DeletionHandler.DeleteOrganization(ctx, org); err != nil {
		return err
	}

	return nil
}

// ResendInvitation は招待を再送信する
func (m *Manager) ResendInvitation(ctx context.Context, invitationID uint, orgID uint, userID uint) error {
	// 権限チェック（managerのみ）
	if err := m.CheckPermission(ctx, userID, orgID, "update"); err != nil {
		return err
	}

	// Invitation検索（invitationID, orgID）
	invitation, err := m.storage.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return ErrInvitationNotFound
	}

	// organization_id一致確認
	if invitation.OrganizationID != orgID {
		return ErrOrganizationAccessDenied
	}

	// トークン再生成（オプション）
	token, err := m.GenerateToken()
	if err != nil {
		return err
	}

	// 有効期限更新
	expiresAt := m.CalculateExpiry()

	// Invitation更新（status=pending, expiresAt更新）
	invitation.Status = InvitationStatusPending
	invitation.ExpiresAt = expiresAt
	invitation.Token = token
	if err := m.storage.UpdateInvitation(ctx, invitation); err != nil {
		return err
	}

	// メール再送信（EmailSender.SendInvitation）
	if m.config.EmailSender == nil {
		return ErrEmailSendFailed
	}
	// 招待URLを生成
	invitationURL := m.GetInvitationURL(token)
	if err := m.config.EmailSender.SendInvitation(ctx, invitation, invitationURL); err != nil {
		return ErrEmailSendFailed
	}

	return nil
}

// UpdatePassword はユーザーのパスワードを更新する
// パスワード更新時に、該当するInvitationをacceptedにする
func (m *Manager) UpdatePassword(ctx context.Context, userID uint, orgID uint, newPassword string) error {
	// User取得
	user, err := m.storage.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	// 権限チェック（自分のみ更新可能）
	if err := m.ValidateOrganizationAccess(ctx, userID, orgID); err != nil {
		return err
	}

	// organization_id一致確認
	if user.OrganizationID != orgID {
		return ErrOrganizationAccessDenied
	}

	// パスワードをハッシュ化
	hashedPassword, err := hashPassword(newPassword)
	if err != nil {
		return err
	}

	// Userのパスワードを更新
	user.Password = hashedPassword
	user.UpdatedAt = time.Now()
	if err := m.storage.UpdateUser(ctx, user); err != nil {
		return err
	}

	// 該当するInvitationを探してacceptedにする
	invitations, err := m.storage.GetInvitationsByOrganizationID(ctx, orgID)
	if err != nil {
		return err
	}

	for _, invitation := range invitations {
		if invitation.Email == user.Email && invitation.Status == InvitationStatusPending {
			invitation.Status = InvitationStatusAccepted
			if err := m.storage.UpdateInvitation(ctx, invitation); err != nil {
				return err
			}
			break
		}
	}

	return nil
}

// UpdateProfile はユーザーのプロフィールを更新する
// userIDは更新対象のユーザーID、orgIDは組織ID
// 自分のみ更新可能（userIDとorgIDで自分自身を確認）
func (m *Manager) UpdateProfile(ctx context.Context, userID uint, orgID uint, updates map[string]interface{}) error {
	// User取得
	user, err := m.storage.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	// 権限チェック（自分のみ更新可能）
	// userIDとorgIDで自分自身を確認
	if err := m.ValidateOrganizationAccess(ctx, userID, orgID); err != nil {
		return err
	}

	// organization_id一致確認
	if user.OrganizationID != orgID {
		return ErrOrganizationAccessDenied
	}

	// 自分のみ更新可能なため、userIDで指定されたユーザーがorgIDの組織に属していることを確認
	// これは既にValidateOrganizationAccessで確認済み

	// User更新
	// updatesマップからフィールドを更新（簡易実装）
	// 実際の実装では、フィールドごとのバリデーションが必要
	user.UpdatedAt = time.Now()
	if err := m.storage.UpdateUser(ctx, user); err != nil {
		return err
	}

	// Authboss通知（現時点では実装なし、将来的に追加）

	return nil
}

// UpdateOrganization は組織を更新する
func (m *Manager) UpdateOrganization(ctx context.Context, orgID uint, userID uint, updates map[string]interface{}) error {
	// 権限チェック（managerのみ）
	if err := m.CheckPermission(ctx, userID, orgID, "update"); err != nil {
		return err
	}

	// Organization取得
	org, err := m.storage.GetOrganization(ctx, orgID)
	if err != nil {
		return err
	}

	// BeforeOrganizationUpdateフック実行
	if err := m.executeHook(m.hooks.BeforeOrganizationUpdate, ctx, org); err != nil {
		return err
	}

	// organization_id一致確認（既にCheckPermissionで確認済みだが、念のため）
	user, err := m.storage.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	if user.OrganizationID != orgID {
		return ErrOrganizationAccessDenied
	}

	// Organization更新（組織名の重複は許可）
	// updatesマップからフィールドを更新（簡易実装）
	// 実際の実装では、フィールドごとのバリデーションが必要
	if name, ok := updates["name"].(string); ok {
		org.Name = name
	}
	org.UpdatedAt = time.Now()
	if err := m.storage.UpdateOrganization(ctx, org); err != nil {
		return err
	}

	// AfterOrganizationUpdateフック実行
	if err := m.executeHook(m.hooks.AfterOrganizationUpdate, ctx, org); err != nil {
		return err
	}

	return nil
}

// ValidateOrganizationAccess はユーザーが組織にアクセスできるか検証する
func (m *Manager) ValidateOrganizationAccess(ctx context.Context, userID uint, orgID uint) error {
	// User検索（userID, organizationID）
	user, err := m.storage.GetUser(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	// organization_id一致確認
	if user.OrganizationID != orgID {
		return ErrOrganizationAccessDenied
	}

	return nil
}

// CheckPermission は権限をチェックする
func (m *Manager) CheckPermission(ctx context.Context, userID uint, orgID uint, action string) error {
	// User検索
	user, err := m.storage.GetUser(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	// organization_id一致確認
	if user.OrganizationID != orgID {
		return ErrOrganizationAccessDenied
	}

	// ロール判定（RoleChecker）
	if m.config.RoleChecker == nil {
		return ErrPermissionDenied
	}

	if !m.config.RoleChecker.HasPermission(user.Role, action) {
		return ErrPermissionDenied
	}

	return nil
}

// executeHook はフックを実行するヘルパーメソッド
func (m *Manager) executeHook(hook HookFunc, ctx context.Context, data interface{}) error {
	if hook != nil {
		return hook(ctx, data)
	}
	return nil
}

// createOrganization はOrganizationを作成するヘルパーメソッド
func (m *Manager) createOrganization(ctx context.Context, name string) (*Organization, error) {
	// デフォルトではランダム文字列をSignatureとして使用
	signature, err := generateRandomSignature()
	if err != nil {
		return nil, err
	}
	
	org := &Organization{
		Name:      name,
		Signature: signature,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := m.storage.CreateOrganization(ctx, org); err != nil {
		return nil, err
	}
	return org, nil
}

// generateRandomSignature はランダムなSignatureを生成する（12バイト、24文字の16進数）
func generateRandomSignature() (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// createUser はUserを作成するヘルパーメソッド（パスワードなし）
func (m *Manager) createUser(ctx context.Context, email string, orgID uint, role Role) (*User, error) {
	return m.createUserWithPassword(ctx, email, orgID, role, "")
}

// createUserWithPassword はUserを作成するヘルパーメソッド（パスワード付き）
func (m *Manager) createUserWithPassword(ctx context.Context, email string, orgID uint, role Role, hashedPassword string) (*User, error) {
	user := &User{
		Email:          email,
		Password:       hashedPassword,
		OrganizationID: orgID,
		Role:           role,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := m.storage.CreateUser(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// generateRandomPassword はランダムなパスワードを生成する（32バイト、64文字の16進数）
func generateRandomPassword() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// hashPassword はパスワードをbcryptでハッシュ化する
func hashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// createInvitation はInvitationを作成するヘルパーメソッド
func (m *Manager) createInvitation(ctx context.Context, email string, orgID uint, token string, expiresAt time.Time) (*Invitation, error) {
	invitation := &Invitation{
		Email:          email,
		OrganizationID: orgID,
		Token:          token,
		ExpiresAt:      expiresAt,
		Status:         InvitationStatusPending,
		CreatedAt:      time.Now(),
	}
	if err := m.storage.CreateInvitation(ctx, invitation); err != nil {
		return nil, err
	}
	return invitation, nil
}

// GenerateToken はトークンを生成する
func (m *Manager) GenerateToken() (string, error) {
	bytes := make([]byte, 32) // 256ビットのランダムなトークン
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CalculateExpiry は有効期限を計算する
func (m *Manager) CalculateExpiry() time.Time {
	return time.Now().Add(m.config.InvitationExpiryDuration)
}

// IsExpired は有効期限切れかチェックする
func (m *Manager) IsExpired(expiresAt time.Time) bool {
	return time.Now().After(expiresAt)
}

// GetInvitationURL は招待URLを生成する
func (m *Manager) GetInvitationURL(token string) string {
	if m.config.InvitationBaseURL == "" {
		return ""
	}
	return fmt.Sprintf("%s/invite/%s", m.config.InvitationBaseURL, token)
}

// ValidateInvitationTokenAndGetRedirectURL はトークンを検証し、リダイレクト先URLを返す
func (m *Manager) ValidateInvitationTokenAndGetRedirectURL(ctx context.Context, token string) (string, error) {
	// トークン検証（Invitation検索）
	invitation, err := m.storage.GetInvitationByToken(ctx, token)
	if err != nil {
		return "", ErrInvalidToken
	}

	// 有効期限チェック
	if m.IsExpired(invitation.ExpiresAt) {
		return "", ErrInvitationExpired
	}

	// 既にacceptedまたはrejectedの場合、エラーが返される
	if invitation.Status == InvitationStatusAccepted {
		return "", ErrInvitationAlreadyAccepted
	}
	if invitation.Status == InvitationStatusRejected {
		return "", ErrInvitationAlreadyRejected
	}

	// リダイレクト先URLを生成
	redirectPath := m.config.InvitationRedirectPath
	if redirectPath == "" {
		redirectPath = "/reset-password"
	}
	redirectURL := fmt.Sprintf("%s?token=%s", redirectPath, token)

	return redirectURL, nil
}

// applyEmailTemplateConfig はEmailSenderがSMTPEmailSenderの場合、テンプレート設定を適用する
func applyEmailTemplateConfig(config *Config) {
	if config.EmailSender == nil {
		return
	}

	// SMTPEmailSenderに型アサーション
	smtpSender, ok := config.EmailSender.(*email.SMTPEmailSender)
	if !ok {
		return
	}

	// テンプレートパスが設定されている場合、適用
	if config.InvitationEmailSubjectTemplatePath != "" || config.InvitationEmailBodyTemplatePath != "" {
		smtpSender.SetTemplatePaths(config.InvitationEmailSubjectTemplatePath, config.InvitationEmailBodyTemplatePath)
	}

	// 差出人が設定されている場合、適用
	if config.InvitationEmailFrom != "" {
		smtpSender.SetFrom(config.InvitationEmailFrom)
	}
}

