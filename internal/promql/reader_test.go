package promql_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"testing"

	faaspromql "github.com/poy/cf-faas-log-cache"
	"github.com/poy/cf-faas-log-cache/internal/promql"
	"github.com/poy/cf-faas-log-cache/internal/web"
	"github.com/poy/onpar"
	. "github.com/poy/onpar/expect"
	. "github.com/poy/onpar/matchers"
)

type TR struct {
	*testing.T
	r               *promql.Reader
	spyPromQLClient *spyPromQLClient
	spyDoer         *spyDoer
}

func TestReader(t *testing.T) {
	t.Parallel()
	o := onpar.New()
	defer o.Run(t)

	o.BeforeEach(func(t *testing.T) TR {
		spyPromQLClient := newSpyPromQLClient()
		spyDoer := newSpyDoer()
		return TR{
			T:               t,
			r:               promql.NewReader(web.Query{Path: "http://some.url/some-path", Context: "some-context", Query: "some-query"}, spyPromQLClient, spyDoer, log.New(ioutil.Discard, "", 0)),
			spyPromQLClient: spyPromQLClient,
			spyDoer:         spyDoer,
		}
	})

	o.Spec("it POSTS results for a non-empty result", func(t TR) {
		expectedResult := &faaspromql.QueryResult{
			Status: "some-status",
			Data: faaspromql.RawResult{
				Result: []interface{}{1, 2},
			},
		}

		t.spyPromQLClient.result = expectedResult
		t.r.Tick()

		Expect(t, t.spyPromQLClient.ctx).To(Not(BeNil()))
		_, ok := t.spyPromQLClient.ctx.Deadline()
		Expect(t, ok).To(BeTrue())
		Expect(t, t.spyPromQLClient.ctx.Err()).To(Not(BeNil()))
		Expect(t, t.spyPromQLClient.query).To(Equal("some-query"))

		Expect(t, t.spyDoer.req).To(Not(BeNil()))
		Expect(t, t.spyDoer.req.URL.String()).To(Equal("http://some.url/some-path"))
		Expect(t, t.spyDoer.req.Method).To(Equal("POST"))

		expectedResult.Context = "some-context"
		data, err := json.Marshal(expectedResult)
		Expect(t, err).To(BeNil())

		r, err := ioutil.ReadAll(t.spyDoer.req.Body)
		Expect(t, err).To(BeNil())

		Expect(t, r).To(MatchJSON(data))
	})

	o.Spec("it does not POST for empty results", func(t TR) {
		t.spyPromQLClient.result = &faaspromql.QueryResult{
			Data: faaspromql.RawResult{},
		}
		t.r.Tick()
		Expect(t, t.spyPromQLClient.ctx).To(Not(BeNil()))
		Expect(t, t.spyDoer.req).To(BeNil())
	})

	o.Spec("it does not POST for an error", func(t TR) {
		t.spyPromQLClient.err = errors.New("some-error")
		t.spyPromQLClient.result = &faaspromql.QueryResult{
			Data: faaspromql.RawResult{
				Result: []interface{}{1, 2},
			},
		}
		t.r.Tick()
		Expect(t, t.spyPromQLClient.ctx).To(Not(BeNil()))
		Expect(t, t.spyDoer.req).To(BeNil())
	})
}

type spyPromQLClient struct {
	ctx    context.Context
	query  string
	result *faaspromql.QueryResult
	err    error
}

func newSpyPromQLClient() *spyPromQLClient {
	return &spyPromQLClient{}
}

func (s *spyPromQLClient) PromQL(ctx context.Context, query string) (*faaspromql.QueryResult, error) {
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
