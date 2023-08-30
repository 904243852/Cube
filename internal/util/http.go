package util

import (
	"net/url"
	"strconv"
)

type QueryParams struct {
	url.Values
}

func (p *QueryParams) GetOrDefault(key string, defaultValue string) string {
	if value := p.Get(key); value != "" {
		return value
	}
	return defaultValue
}

func (p *QueryParams) GetIntOrDefault(key string, defaultValue int) int {
	if !p.Has(key) {
		return defaultValue
	}
	if value, err := strconv.Atoi(p.Get(key)); err == nil {
		return value
	}
	return defaultValue
}
