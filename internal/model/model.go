package model

import "time"

type UserID string

type User struct {
	PasswordPHC string
	LoginHash   []byte
}
type DataID string

type DataType string

const (
	DataTypeUnspecified               DataType = "DataTypeUnspecified"
	DataTypeAuthenticationCredentials          = "DataTypeAuthenticationCredentials"
	DataTypeCard                               = "DataTypeCard"
	DataTypeBinary                             = " DataTypeBinary"
)

type Metadata struct {
	CreatedAt       time.Time
	ChunkDescriptor *ChunkDescriptor
	Description     string
	Name            string
	UserID          UserID
	DataType        DataType
	ID              DataID
}

type ChunkDescriptor struct {
	Offset uint64
	CRC32C uint32
	Last   bool
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
