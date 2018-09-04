package promql

import (
	"context"
	"fmt"
	"regexp"
)

type Sanitizer struct {
	f GuidFetcher
}

type GuidFetcher interface {
	GetAppGuid(ctx context.Context, appName string) (string, error)
}

func NewSanitizer(f GuidFetcher) *Sanitizer {
	return &Sanitizer{
		f: f,
	}
}

func (s *Sanitizer) Sanitize(ctx context.Context, query string) (string, error) {
	names, err := Parse(query)
	if err != nil {
		return "", fmt.Errorf("failed to parse PromQL query %s: %s", query, err)
	}
	m := map[string]string{}
	for _, name := range names {
		guid, err := s.f.GetAppGuid(ctx, name)
		if err != nil {
			return "", fmt.Errorf("failed to fetch guid for %s: %s", name, err)
		}
		m[name] = guid
	}

	for k, v := range m {
		sanitizeRE, err := regexp.Compile(`source_id\s*\=\s*["']` + k + `["']`)
		if err != nil {
			return "", fmt.Errorf("failed to create regex: %s", err)
		}
		query = sanitizeRE.ReplaceAllString(query, `source_id="`+v+`"`)
	}

	return query, nil
}
