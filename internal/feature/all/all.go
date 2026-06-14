// Package all blank-imports every UI feature package so their init() self-
// registrations run. web.Register imports this (blank) once; adding a feature
// means adding one import line here and nothing in web.go.
package all

import (
	_ "github.com/alexradunet/balaur/internal/feature/taskcards"
)
