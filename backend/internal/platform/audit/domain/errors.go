package domain

import "errors"

var ErrAuditSaveFailed = errors.New("audit: failed to persist audit log")
