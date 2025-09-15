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
	CreatedAt     time.Time
	ChunkMetadata *ChunkMetadata
	UserID        UserID
	Name          string
	Description   string
	ID            DataID
	DataType      DataType
}

type ChunkMetadata struct {
	ObjectSHA256 []byte
	Offset       uint64
	ObjectSize   uint64
	CRC32C       uint32
	Last         bool
}

// ObjectLocator — непрозрачный идентификатор местоположения объекта
// в объектном хранилище (например, "pg://...", "s3://...").
type ObjectLocator string

// MetaLoc связывает доменные метаданные и локатор объекта.
// Используется при выборках списков объектов.
type MetaLoc struct {
	Locator ObjectLocator
	Meta    Metadata
}
