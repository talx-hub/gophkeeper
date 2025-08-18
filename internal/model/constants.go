package model

import (
	"errors"
	"time"
)

const RepoOperationTO = 2 * time.Second

const ContextKeyUserID = "userID"

var ErrNotFound = errors.New("not found")
