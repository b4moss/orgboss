package validation

import (
	"regexp"
	"strings"

	"github.com/b4m-oss/orgboss/types"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// ValidateEmail validates the format of an email address
func ValidateEmail(email string) error {
	if email == "" {
		return types.ErrInvalidInput
	}
	if !emailRegex.MatchString(email) {
		return types.ErrInvalidInput
	}
	return nil
}

// ValidateOrganizationName validates an organization name
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
