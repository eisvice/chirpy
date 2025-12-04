package auth

import (
	"errors"
	"net/http"
	"strings"
)

func GetAPIKey(headers http.Header) (string, error) {
    apiKey := headers.Get("Authorization")
    apiKey, found := strings.CutPrefix(apiKey, "ApiKey ")
    if apiKey == "" {
        return "", errors.New("API key not found")
    }
    if !found {
        return "", errors.New("wrong API key prefix")
    }

    return apiKey, nil
}