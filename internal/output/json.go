package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// JSON writes data as indented JSON to stdout.
func JSON(data interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}
	return nil
}
