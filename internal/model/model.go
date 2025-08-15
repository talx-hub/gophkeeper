package model

type UserID string

type User struct {
	Login        string
	PasswordHash []byte
	UUID         UserID
}
