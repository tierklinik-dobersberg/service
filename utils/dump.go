package utils

import (
	"encoding/json"
	"io"
	"os"
)

// DumpTo dumps x as a pretty printed JSON to w.
func DumpTo(x interface{}, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")

	return enc.Encode(x)
}

// Dump dumps x as a pretty printed JSON to os.Stderr.
func Dump(x interface{}) error {
	return DumpTo(x, os.Stderr)
}
