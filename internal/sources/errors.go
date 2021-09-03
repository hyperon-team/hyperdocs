package sources

import "errors"

var (
	// ErrSymbolNotFound indicates that queried symbol is not found
	ErrSymbolNotFound = errors.New("symbol not found")
)
