package errs

import (
	"errors"
	"fmt"
)

type GatewayError struct {
	Code      int
	Message   string // 对外展示的错误信息
	Cause     error  // 原始错误
	Retriable bool
}

func (e *GatewayError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *GatewayError) Unwrap() error {
	return e.Cause
}

func As(err error) *GatewayError {
	var ge *GatewayError
	if errors.As(err, &ge) {
		return ge
	}
	return nil
}

func New(code int, msg string, retriable bool, cause error) *GatewayError {
	return &GatewayError{
		Code:      code,
		Message:   msg,
		Cause:     cause,
		Retriable: retriable,
	}
}
