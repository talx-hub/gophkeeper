package model

import "time"

type UserID string
type DataID int64

type User struct {
	Login        string
	UUID         UserID
	PasswordHash []byte
}

type DataType int

const (
	DataTypeUnspecified DataType = iota
	DataTypeAuthenticationCredentials
	DataTypeCard
	DataTypeBinary
)

type Metadata struct {
	ID          DataID
	UserID      UserID
	DataType    DataType
	Name        string
	Description string
	CreatedAt   time.Time
	TotalSize   uint64
}
