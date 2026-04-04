// Package bind is the entry point for gomobile bind.
// It re-exports types from the mobile package for gomobile to generate bindings.
package bind

import (
	_ "github.com/alantheprice/ledit/mobile"
)

// This package exists to provide a stable import path for gomobile bind.
// The actual types are defined in the mobile package.