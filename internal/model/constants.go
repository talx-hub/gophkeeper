package model

import (
	"errors"
	"time"
)

const RepoOperationTO = 2 * time.Second

var ErrNotFound = errors.New("not found")
