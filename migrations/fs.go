package migrations

import "embed"

//go:embed db/*.sql
var FS embed.FS
