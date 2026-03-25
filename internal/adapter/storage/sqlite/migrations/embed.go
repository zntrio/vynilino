// Package migrations embeds the SQL migration files for the SQLite adapter.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
