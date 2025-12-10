package auth

import (
	"fmt"
	"net/http"
	"strings"
)

func GetAPIKey(headers http.Header) (string, error) {
	header := headers.Get("Authorization")
	if header == "" {
		return "", fmt.Errorf("no authorization credentials found")
	}

	// verify header starts with "ApiKey"
	if !strings.HasPrefix(header, "ApiKey") {
		return "", fmt.Errorf("invalid authorization header")
	}

	key := strings.TrimSpace(strings.TrimPrefix(header, "ApiKey"))
	return key, nil
}