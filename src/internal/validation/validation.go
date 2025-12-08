package validation

import (
	"regexp"
	"strings"

	"orgboss/types"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// ValidateEmail はメールアドレスの形式を検証する
func ValidateEmail(email string) error {
	if email == "" {
		return types.ErrInvalidInput
	}
	if !emailRegex.MatchString(email) {
		return types.ErrInvalidInput
	}
	return nil
}

// ValidateOrganizationName は組織名を検証する
func ValidateOrganizationName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return types.ErrInvalidInput
	}
	if len(name) > 255 {
		return types.ErrInvalidInput
	}
	return nil
}
