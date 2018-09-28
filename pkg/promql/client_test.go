package promql_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/apoydence/cf-faas-log-cache"
	"github.com/apoydence/cf-faas-log-cache/pkg/promql"
	"github.com/apoydence/onpar"
	. "github.com/apoydence/onpar/expect"
	. "github.com/apoydence/onpar/matchers"
)

type TC struct {
	*testing.T
	c                   *promql.Client
	spyDoer             *spyDoer
	spyAppNameSanitizer *spyAppNameSanitizer
}

func TestClientPromQL(t *testing.T) {
	t.Parallel()
	o := onpar.New()
	defer o.Run(t)

	o.BeforeEach(func(t *testing.T) TC {
		spyDoer := newSpyDoer()
		spyAppNameSanitizer := newSpyAppNameSanitizer()
		return TC{
			T:                   t,
			c:                   promql.NewClient("http://some.url", spyAppNameSanitizer, spyDoer),
			spyDoer:             spyDoer,
			spyAppNameSanitizer: spyAppNameSanitizer,
		}
	})

	o.Spec("returns the results", func(t TC) {
		t.spyDoer.resp = &http.Response{
			StatusCode: 200,
			Body: ioutil.NopCloser(
				strings.NewReader(vectorResult()),
			),
		}
		t.spyAppNameSanitizer.result = "some-san-query / other-part"

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		results, err := t.c.PromQL(ctx, "some-query / other-part")
		Expect(t, err).To(BeNil())

		Expect(t, t.spyAppNameSanitizer.query).To(Equal("some-query / other-part"))
		Expect(t, t.spyAppNameSanitizer.ctx.Err()).To(Not(BeNil()))
		_, ok := t.spyAppNameSanitizer.ctx.Deadline()
		Expect(t, ok).To(BeTrue())

		Expect(t, t.spyDoer.req).To(Not(BeNil()))

		Expect(t, t.spyDoer.req.Context().Err()).To(Not(BeNil()))
		_, ok = t.spyDoer.req.Context().Deadline()
		Expect(t, ok).To(BeTrue())

		Expect(t, t.spyDoer.req.URL.String()).To(Equal("http://some.url/api/v1/query?query=some-san-query+%2F+other-part"))
		Expect(t, t.spyDoer.req.Method).To(Equal(http.MethodGet))
		Expect(t, results).To(Equal(&faaspromql.QueryResult{
			Status: "success",
			Data: faaspromql.RawResult{
				ResultType: "vector",
				Result: []interface{}{
					&faaspromql.Sample{
						Metric: map[string]string{
							"status_code": "200",
							"user_agent":  "Go-http-client/1.1",
						},
						Value: []json.Number{
							"1535779665000000000",
							"271323437",
						},
					},
				},
			},
		}))
	})

	o.Spec("returns the results for a range query", func(t TC) {
		t.spyDoer.resp = &http.Response{
			StatusCode: 200,
			Body: ioutil.NopCloser(
				strings.NewReader(vectorResult()),
			),
		}
		t.spyAppNameSanitizer.result = "some-san-query / other-part"

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		results, err := t.c.PromQLRange(ctx, "some-query / other-part", time.Unix(0, 1), time.Unix(0, 2), 5*time.Second)
		Expect(t, err).To(BeNil())

		Expect(t, t.spyAppNameSanitizer.query).To(Equal("some-query / other-part"))
		Expect(t, t.spyAppNameSanitizer.ctx.Err()).To(Not(BeNil()))
		_, ok := t.spyAppNameSanitizer.ctx.Deadline()
		Expect(t, ok).To(BeTrue())

		Expect(t, t.spyDoer.req).To(Not(BeNil()))

		Expect(t, t.spyDoer.req.Context().Err()).To(Not(BeNil()))
		_, ok = t.spyDoer.req.Context().Deadline()
		Expect(t, ok).To(BeTrue())

		Expect(t, t.spyDoer.req.URL.Scheme).To(Equal("http"))
		Expect(t, t.spyDoer.req.URL.Host).To(Equal("some.url"))
		Expect(t, t.spyDoer.req.URL.Path).To(Equal("/api/v1/query_range"))
		Expect(t, t.spyDoer.req.URL.Query()["query"]).To(Equal([]string{"some-san-query / other-part"}))
		Expect(t, t.spyDoer.req.URL.Query()["start"]).To(Equal([]string{"1"}))
		Expect(t, t.spyDoer.req.URL.Query()["end"]).To(Equal([]string{"2"}))
		Expect(t, t.spyDoer.req.URL.Query()["step"]).To(Equal([]string{"5s"}))

		Expect(t, t.spyDoer.req.Method).To(Equal(http.MethodGet))
		Expect(t, results).To(Equal(&faaspromql.QueryResult{
			Status: "success",
			Data: faaspromql.RawResult{
				ResultType: "vector",
				Result: []interface{}{
					&faaspromql.Sample{
						Metric: map[string]string{
							"status_code": "200",
							"user_agent":  "Go-http-client/1.1",
						},
						Value: []json.Number{
							"1535779665000000000",
							"271323437",
						},
					},
				},
			},
		}))
	})

	o.Spec("returns an error if the request fails", func(t TC) {
		t.spyAppNameSanitizer.err = errors.New("some-error")
		t.spyDoer.resp = &http.Response{
			StatusCode: 200,
			Body: ioutil.NopCloser(
				strings.NewReader(emptyVectorResult()),
			),
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := t.c.PromQL(ctx, "some-query")
		Expect(t, err).To(Not(BeNil()))
	})

	o.Spec("returns an error if the request fails", func(t TC) {
		t.spyDoer.err = errors.New("some-error")
		t.spyDoer.resp = &http.Response{
			StatusCode: 200,
			Body: ioutil.NopCloser(
				strings.NewReader(emptyVectorResult()),
			),
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := t.c.PromQL(ctx, "some-query")
		Expect(t, err).To(Not(BeNil()))
	})

	o.Spec("returns an error if the results are invalid", func(t TC) {
		t.spyDoer.resp = &http.Response{
			StatusCode: 200,
			Body: ioutil.NopCloser(
				strings.NewReader("invalid"),
			),
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := t.c.PromQL(ctx, "some-query")
		Expect(t, err).To(Not(BeNil()))
	})

	o.Spec("returns an error if the response code is not 200", func(t TC) {
		t.spyDoer.resp = &http.Response{
			StatusCode: 400,
			Body: ioutil.NopCloser(
				strings.NewReader(emptyVectorResult()),
			),
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := t.c.PromQL(ctx, "some-query")
		Expect(t, err).To(Not(BeNil()))
	})

	o.Spec("it returns an error for an invalid addr", func(t TC) {
		t.spyDoer.resp = &http.Response{
			StatusCode: 200,
			Body: ioutil.NopCloser(
				strings.NewReader(emptyVectorResult()),
			),
		}
		t.c = promql.NewClient("::invalid::", t.spyAppNameSanitizer, t.spyDoer)
		_, err := t.c.PromQL(context.Background(), "some-query")
		Expect(t, err).To(Not(BeNil()))
	})
}

func vectorResult() string {
	return `{
  "status": "success",
  "data": {
    "resultType": "vector",
    "result": [
      {
        "metric": {
          "status_code": "200",
          "user_agent": "Go-http-client/1.1"
        },
        "value": [
          1535779665000000000,
          271323437
        ]
      }
    ]
  }
}`
}

func emptyVectorResult() string {
	return `{
  "status": "success",
  "data": {
    "resultType": "vector"
  }
}`
}

type spyAppNameSanitizer struct {
	result string
	query  string
	err    error
	ctx    context.Context
}

func newSpyAppNameSanitizer() *spyAppNameSanitizer {
	return &spyAppNameSanitizer{}
}

func (s *spyAppNameSanitizer) Sanitize(ctx context.Context, query string) (string, error) {
	s.ctx = ctx
	s.query = query
	return s.result, s.err
}

type spyDoer struct {
	req  *http.Request
	resp *http.Response
	err  error
}

func newSpyDoer() *spyDoer {
	return &spyDoer{}
}

func (s *spyDoer) Do(r *http.Request) (*http.Response, error) {
	s.req = r

	if s.resp == nil {
		return &http.Response{
			Body: ioutil.NopCloser(bytes.NewReader(nil)),
		}, s.err
	}

	return s.resp, s.err
}
