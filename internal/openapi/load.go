package openapi

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/getkin/kin-openapi/openapi2"
)

func LoadSwaggerV2(path string) (*openapi2.T, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t openapi2.T
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, fmt.Errorf("invalid swagger json: %w", err)
	}
	return &t, nil
}
