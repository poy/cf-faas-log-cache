package promql_test

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/apoydence/cf-faas-log-cache/internal/promql"
	"github.com/apoydence/onpar"
	. "github.com/apoydence/onpar/expect"
	. "github.com/apoydence/onpar/matchers"
)

type TC struct {
	*testing.T
	c       *promql.Client
	spyDoer *spyDoer
}

func TestClient(t *testing.T) {
	t.Parallel()
	o := onpar.New()
	defer o.Run(t)

	o.BeforeEach(func(t *testing.T) TC {
		spyDoer := newSpyDoer()
		return TC{
			T:       t,
			c:       promql.NewClient("http://some.url", spyDoer),
			spyDoer: spyDoer,
		}
	})

	o.Spec("returns true if the query has results", func(t TC) {
		t.spyDoer.resp = &http.Response{
			StatusCode: 200,
			Body: ioutil.NopCloser(
				strings.NewReader(vectorResult()),
			),
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		hasData, err := t.c.PromQL(ctx, "some-query / other-part")
		Expect(t, err).To(BeNil())
		Expect(t, hasData).To(BeTrue())

		Expect(t, t.spyDoer.req).To(Not(BeNil()))

		Expect(t, t.spyDoer.req.Context().Err()).To(Not(BeNil()))
		_, ok := t.spyDoer.req.Context().Deadline()
		Expect(t, ok).To(BeTrue())

		Expect(t, t.spyDoer.req.URL.String()).To(Equal("http://some.url/api/v1/query?query=some-query+%2F+other-part"))
		Expect(t, t.spyDoer.req.Method).To(Equal(http.MethodGet))
	})

	o.Spec("returns false if the query has results", func(t TC) {
		t.spyDoer.resp = &http.Response{
			StatusCode: 200,
			Body: ioutil.NopCloser(
				strings.NewReader(emptyVectorResult()),
			),
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		hasData, err := t.c.PromQL(ctx, "some-query")
		Expect(t, err).To(BeNil())
		Expect(t, hasData).To(BeFalse())
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
		t.c = promql.NewClient("::invalid::", t.spyDoer)
		_, err := t.c.PromQL(context.Background(), "some-query")
		Expect(t, err).To(Not(BeNil()))
	})
}

func emptyVectorResult() string {
	return `{
  "status": "success",
  "data": {
    "resultType": "vector"
  }
}`
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
