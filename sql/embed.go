package sqlassets

import (
	"embed"
	"io/fs"
)

//go:embed migrations/*.sql
var raw embed.FS

var Migrations fs.FS

func init() {
	sub, err := fs.Sub(raw, "sql/migrations")
	if err != nil {
		panic(err)
	}
	Migrations = sub
}
