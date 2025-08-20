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
	ID            DataID
	UserID        UserID
	DataType      DataType
	Name          string
	Description   string
	CreatedAt     time.Time
	ChunkMetadata *ChunkMetadata
}

type ChunkMetadata struct {
	Offset       uint64
	ObjectSize   uint64
	Last         bool
	CRC32C       uint32
	ObjectSHA256 []byte
}
