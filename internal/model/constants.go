package model

import (
	"errors"
	"time"
)

const RepoOperationTO = 2 * time.Second

type TypeContextKeyUserID string

const ContextKeyUserID = TypeContextKeyUserID("userID")

var ErrNotFound = errors.New("not found")

var KeyLoggerError = "err"
