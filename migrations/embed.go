// Package migrations embeds the goose SQL migration files.
package migrations

import "embed"

// FS holds the embedded *.sql migrations (applied by store.Migrate).
//
//go:embed *.sql
var FS embed.FS
