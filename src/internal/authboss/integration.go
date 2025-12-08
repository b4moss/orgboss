package authboss

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/aarondl/authboss/v3"
	"gorm.io/gorm"

	"github.com/b4m-oss/orgboss/types"
)

// User はAuthbossのUserモデルを拡張した構造体
// Authbossの標準Userインターフェースを実装し、organization_idとroleを追加
type User struct {
	// Authbossの標準フィールド
	ID       int64  `db:"id"`
	Email    string `db:"email"`
	Password string `db:"password"`

	// orgbossの拡張フィールド
	OrganizationID uint        `gorm:"not null;index" db:"organization_id"`
	Role           types.Role   `gorm:"not null;default:'user'" db:"role"`
	CreatedAt      sql.NullTime `db:"created_at"`
	UpdatedAt      sql.NullTime `db:"updated_at"`
	DeletedAt      sql.NullTime `gorm:"index" db:"deleted_at"`
}

// PutPID はAuthbossのUserインターフェース実装
func (u *User) PutPID(pid string) {
	u.Email = pid
}

// PutPassword はAuthbossのUserインターフェース実装
func (u *User) PutPassword(password string) {
	u.Password = password
}

// PutEmail はAuthbossのUserインターフェース実装
func (u *User) PutEmail(email string) {
	u.Email = email
}

// PutConfirmed はAuthbossのUserインターフェース実装（未使用）
func (u *User) PutConfirmed(confirmed bool) {
	// orgbossでは未使用
}

// PutLocked はAuthbossのUserインターフェース実装（未使用）
func (u *User) PutLocked(locked bool) {
	// orgbossでは未使用
}

// PutAttemptCount はAuthbossのUserインターフェース実装（未使用）
func (u *User) PutAttemptCount(attempts int) {
	// orgbossでは未使用
}

// PutLastAttempt はAuthbossのUserインターフェース実装（未使用）
func (u *User) PutLastAttempt(lastAttempt *sql.NullTime) {
	// orgbossでは未使用
}

// PutExpired はAuthbossのUserインターフェース実装（未使用）
func (u *User) PutExpired(expired *sql.NullTime) {
	// orgbossでは未使用
}

// GetPID はAuthbossのUserインターフェース実装
func (u *User) GetPID() string {
	return u.Email
}

// GetPassword はAuthbossのUserインターフェース実装
func (u *User) GetPassword() string {
	return u.Password
}

// GetEmail はAuthbossのUserインターフェース実装
// 注意: 公式のAuthboss v3のUserインターフェースにはGetEmail()メソッドがない可能性があります
// EmailはPID（Principal ID）として扱われ、GetPID()で取得します
// このメソッドはorgbossの内部実装用に保持しています
func (u *User) GetEmail() string {
	return u.Email
}

// GetConfirmed はAuthbossのUserインターフェース実装
func (u *User) GetConfirmed() bool {
	return true // orgbossでは常にtrue
}

// GetLocked はAuthbossのUserインターフェース実装
func (u *User) GetLocked() bool {
	return false // orgbossでは常にfalse
}

// GetAttemptCount はAuthbossのUserインターフェース実装
func (u *User) GetAttemptCount() int {
	return 0 // orgbossでは未使用
}

// GetLastAttempt はAuthbossのUserインターフェース実装
func (u *User) GetLastAttempt() *sql.NullTime {
	return nil // orgbossでは未使用
}

// GetExpired はAuthbossのUserインターフェース実装
func (u *User) GetExpired() *sql.NullTime {
	return nil // orgbossでは未使用
}

// SetupAuthboss はAuthbossを設定し、orgbossとの統合を行う
// 注意: Authbossの実際のAPIに合わせて実装する必要があります
// 現在はプレースホルダーとして実装されています
// 公式リポジトリ: https://github.com/aarondl/authboss
func SetupAuthboss(db *gorm.DB, ab *authboss.Authboss) error {
	// Authbossのストレージを設定
	// ここでは簡易実装として、GORMを使ったストレージを設定
	// 実際の実装では、Authbossのストレージインターフェースを実装する必要がある

	// BeforeRegisterフックを設定して、組織作成を連動
	// 注意: Authboss v3の実際のAPIに合わせて実装する必要があります
	// 公式ドキュメントを参照して、正しいフック設定方法を確認してください
	// 現在はコメントアウトしています
	// ab.Config.Core.BeforeRegister = func(ctx context.Context, r *authboss.RegisterValues) error {
	// 	// ここでorgbossのCreateOrganizationWithUserを呼び出す
	// 	// 実際の実装では、orgboss.Managerを取得して呼び出す必要がある
	// 	return nil
	// }

	return nil
}

// SetupAuthbossWithAutoLogin はAuthbossを設定し、パスワードリセット後のリダイレクト機能を有効にする
// enableAutoLoginがtrueの場合、パスワードリセット完了後にログインページにリダイレクトする設定を有効にします
// 公式リポジトリ: https://github.com/aarondl/authboss
//
// 注意: 実際のリダイレクト処理は、パスワードリセットのHTTPハンドラー側で実装する必要があります。
// RedirectToLoginAfterPasswordReset ヘルパー関数を使用して、リダイレクト処理を実装してください。
// 参考: https://github.com/aarondl/authboss/blob/v3.5.3/authboss.go#L76
func SetupAuthbossWithAutoLogin(db *gorm.DB, ab *authboss.Authboss, enableAutoLogin bool) error {
	// 基本的な設定を実行
	if err := SetupAuthboss(db, ab); err != nil {
		return err
	}

	// 自動ログイン（リダイレクト）が有効な場合の設定
	// 注意: Authboss v3では、パスワードリセット後のリダイレクト処理は、
	// HTTPハンドラーレベルで実装する必要があります。
	// RedirectToLoginAfterPasswordReset ヘルパー関数を使用して実装してください。
	if enableAutoLogin {
		// 設定は完了しました。実際のリダイレクト処理は、
		// パスワードリセットのHTTPハンドラー側で RedirectToLoginAfterPasswordReset を呼び出してください。
	}

	return nil
}

// RedirectToLoginAfterPasswordReset はパスワードリセット完了後にログインページにリダイレクトする
// この関数は、パスワードリセットのHTTPハンドラーから呼び出されます
//
// 使用例:
//   func passwordResetHandler(w http.ResponseWriter, r *http.Request) {
//       // パスワードリセット処理...
//       if err := authbossuser.RedirectToLoginAfterPasswordReset(r.Context(), w, r, ab); err != nil {
//           // エラーハンドリング
//       }
//   }
func RedirectToLoginAfterPasswordReset(ctx context.Context, w http.ResponseWriter, r *http.Request, ab *authboss.Authboss) error {
	// ログインページへのリダイレクトパスを取得
	loginPath := "/login"
	if ab.Config.Paths.Mount != "" {
		loginPath = ab.Config.Paths.Mount + loginPath
	}

	// リダイレクトオプションを設定
	ro := authboss.RedirectOptions{
		Code:         http.StatusSeeOther,
		RedirectPath: loginPath,
		Success:      "パスワードリセットが完了しました。ログインしてください。",
	}

	// リダイレクトを実行
	if ab.Config.Core.Redirector != nil {
		if err := ab.Config.Core.Redirector.Redirect(w, r, ro); err != nil {
			return err
		}
	} else {
		// Redirectorが設定されていない場合、標準的なHTTPリダイレクトを使用
		http.Redirect(w, r, loginPath, http.StatusSeeOther)
	}

	return nil
}

// ValidateOrganizationAccess は認証後のミドルウェアでorganization_idを検証する
func ValidateOrganizationAccess(ctx context.Context, user authboss.User, orgID uint, db *gorm.DB) error {
	// AuthbossのUserからEmailを取得（GetPID()を使用、EmailはPIDとして使用される）
	email := user.GetPID()
	if email == "" {
		return types.ErrUserNotFound
	}

	// データベースからUserを取得
	var orgUser types.User
	if err := db.WithContext(ctx).Where("email = ?", email).First(&orgUser).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return types.ErrUserNotFound
		}
		return err
	}

	// organization_idを検証
	if orgUser.OrganizationID != orgID {
		return types.ErrOrganizationAccessDenied
	}

	return nil
}

