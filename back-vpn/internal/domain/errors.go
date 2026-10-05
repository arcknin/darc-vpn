package domain

import (
	"errors"
	"fmt"
)

// ErrorCode is the stable, non-enumerating error vocabulary exposed by the API.
type ErrorCode string

const (
	ErrInvalidCredentials      ErrorCode = "invalid_credentials"
	ErrNotAuthenticated        ErrorCode = "not_authenticated"
	ErrForbidden               ErrorCode = "forbidden"
	ErrAccountBlocked          ErrorCode = "account_blocked"
	ErrTermsAcceptanceRequired ErrorCode = "terms_acceptance_required"
	ErrProfileNotFound         ErrorCode = "profile_not_found"
	ErrProfileRecoveryFailed   ErrorCode = "profile_recovery_failed"
	ErrAcceptancePersistence   ErrorCode = "acceptance_persistence_failed"
	ErrRevocationNotConfirmed  ErrorCode = "revocation_not_confirmed"
	ErrPoolExhausted           ErrorCode = "pool_exhausted"
	ErrProfileNotReady         ErrorCode = "profile_not_ready"
	ErrInvalidRequest          ErrorCode = "invalid_request"
)

// CodedError is the application error shape shared by adapters.
type CodedError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *CodedError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message == "" {
		return string(e.Code)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *CodedError) Unwrap() error { return e.Cause }

func NewCodedError(code ErrorCode, message string) *CodedError {
	return &CodedError{Code: code, Message: message}
}

func WrapCodedError(code ErrorCode, message string, cause error) *CodedError {
	return &CodedError{Code: code, Message: message, Cause: cause}
}

func CodeOf(err error) ErrorCode {
	var coded *CodedError
	if errors.As(err, &coded) && coded != nil {
		return coded.Code
	}
	return ""
}
