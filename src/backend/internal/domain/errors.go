package domain

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrReauthRequired = errors.New("reauth_required")
	ErrConflict      = errors.New("conflict")
	ErrInvalidInput  = errors.New("invalid input")
)
