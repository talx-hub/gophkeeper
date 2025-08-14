package model

type User struct {
	Login        string
	PasswordHash []byte
	UUID         string
}
