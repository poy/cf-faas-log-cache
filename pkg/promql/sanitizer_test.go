package promql_test

import (
	"context"
	"errors"
	"testing"

	"github.com/apoydence/cf-faas-log-cache/pkg/promql"
	"github.com/apoydence/onpar"
	. "github.com/apoydence/onpar/expect"
	. "github.com/apoydence/onpar/matchers"
)

type TS struct {
	*testing.T
	spyGuidFetcher *spyGuidFetcher
	s              *promql.Sanitizer
}

func TestSanitizer(t *testing.T) {
	t.Parallel()
	o := onpar.New()
	defer o.Run(t)

	o.BeforeEach(func(t *testing.T) TS {
		spyGuidFetcher := newSpyGuidFetcher()
		return TS{
			T:              t,
			spyGuidFetcher: spyGuidFetcher,
			s:              promql.NewSanitizer(spyGuidFetcher),
		}
	})

	o.Spec("it returns the app names replaced with their guids", func(t TS) {
		t.spyGuidFetcher.guids = map[string]string{
			// Picked letters that are throughout the string
			"s": "guid-s",
			"m": "guid-m",
		}
		result, err := t.s.Sanitize(context.Background(), `metric{source_id="s"} / metric{source_id="m"}`)
		Expect(t, err).To(BeNil())
		Expect(t, result).To(Equal(`metric{source_id="guid-s"} / metric{source_id="guid-m"}`))

		Expect(t, t.spyGuidFetcher.appNames).To(Equal([]string{"s", "m"}))
	})

	o.Spec("it returns an error for an invalid query", func(t TS) {
		_, err := t.s.Sanitize(context.Background(), `}{`)
		Expect(t, err).To(Not(BeNil()))
	})

	o.Spec("it returns an error if the fetcher fails", func(t TS) {
		t.spyGuidFetcher.err = errors.New("some-error")
		_, err := t.s.Sanitize(context.Background(), `metric{source_id="s"} / metric{source_id="m"}`)
		Expect(t, err).To(Not(BeNil()))
	})
}

type spyGuidFetcher struct {
	ctx      context.Context
	appNames []string
	guids    map[string]string
	err      error
}

func newSpyGuidFetcher() *spyGuidFetcher {
	return &spyGuidFetcher{}
}

func (s *spyGuidFetcher) GetAppGuid(ctx context.Context, appName string) (string, error) {
	s.ctx = ctx
	s.appNames = append(s.appNames, appName)
	return s.guids[appName], s.err
}
