package auth

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{2,31}$`)

func NormalizeAndValidateRegistration(email, username, password string) (string, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	username = strings.TrimSpace(username)

	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || len(email) > 254 {
		return "", "", errors.New("enter a valid email address")
	}
	if !usernamePattern.MatchString(username) {
		return "", "", errors.New("username must be 3–32 characters and use letters, numbers, dots, dashes, or underscores")
	}
	if utf8.RuneCountInString(password) < 8 {
		return "", "", errors.New("password must contain at least 8 characters")
	}
	if len(password) > 256 {
		return "", "", errors.New("password is too long")
	}

	return email, username, nil
}
