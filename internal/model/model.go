package model

type UserID string

type User struct {
	Login        string
	UUID         UserID
	PasswordHash []byte
}
