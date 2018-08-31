package web_test

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	faas "github.com/apoydence/cf-faas"
	"github.com/apoydence/cf-faas-log-cache/internal/web"
	"github.com/apoydence/onpar"
	. "github.com/apoydence/onpar/expect"
	. "github.com/apoydence/onpar/matchers"
)

type TR struct {
	*testing.T
	recorder      *httptest.ResponseRecorder
	spyStateSaver *spyStateSaver
	s             http.Handler
}

func TestResolverRejects(t *testing.T) {
	t.Parallel()
	o := onpar.New()
	defer o.Run(t)

	o.BeforeEach(func(t *testing.T) TR {
		spyStateSaver := newSpyStateSaver()
		return TR{
			T:             t,
			recorder:      httptest.NewRecorder(),
			s:             web.NewResolver(spyStateSaver, log.New(ioutil.Discard, "", 0)),
			spyStateSaver: spyStateSaver,
		}
	})

	o.Spec("it reconfigures and restages the app", func(t TR) {
		req := httptest.NewRequest("POST", "http://some.url", strings.NewReader(`{"functions":[{"events":{"promql":[{"query":"some-query","context":"some-context"}]},"handler":{"command":"some-command","app_name":"some-app-name"}}]}`))
		ctx, _ := context.WithCancel(context.Background())
		req = req.WithContext(ctx)
		t.s.ServeHTTP(t.recorder, req)

		Expect(t, t.recorder.Code).To(Equal(http.StatusOK))

		var resp faas.ConvertResponse
		Expect(t, json.NewDecoder(t.recorder.Body).Decode(&resp)).To(BeNil())
		Expect(t, resp.Functions).To(HaveLen(1))
		Expect(t, resp.Functions[0].Handler).To(Equal(faas.ConvertHandler{
			Command: "some-command",
			AppName: "some-app-name",
		}))
		Expect(t, resp.Functions[0].Events).To(HaveLen(1))
		Expect(t, resp.Functions[0].Events[0].Path).To(Not(Equal("")))
		Expect(t, resp.Functions[0].Events[0].Method).To(Equal(http.MethodPost))

		Expect(t, t.spyStateSaver.queries).To(Equal([]web.Query{
			{Query: "some-query", Context: "some-context", Path: resp.Functions[0].Events[0].Path},
		}))
		Expect(t, t.spyStateSaver.ctx).To(Equal(req.Context()))
	})

	o.Spec("it returns a 400 for a POST missing the query", func(t TR) {
		t.s.ServeHTTP(t.recorder, httptest.NewRequest("POST", "http://some.url", strings.NewReader(`{"functions":[{"handler":{"command":"some-command"}}]}`)))

		Expect(t, t.recorder.Code).To(Equal(http.StatusBadRequest))
	})

	o.Spec("it returns a 400 for a POST with invalid JSON", func(t TR) {
		t.s.ServeHTTP(t.recorder, httptest.NewRequest("POST", "http://some.url", strings.NewReader(`invalid`)))

		Expect(t, t.recorder.Code).To(Equal(http.StatusBadRequest))
	})

	o.Spec("it returns a 405 for non POST requests", func(t TR) {
		t.s.ServeHTTP(t.recorder, httptest.NewRequest("GET", "http://some.url", nil))

		Expect(t, t.recorder.Code).To(Equal(http.StatusMethodNotAllowed))
	})
}

type spyStateSaver struct {
	ctx     context.Context
	queries []web.Query
	err     error
}

func newSpyStateSaver() *spyStateSaver {
	return &spyStateSaver{}
}

func (s *spyStateSaver) SaveState(ctx context.Context, q []web.Query) error {
	s.ctx = ctx
	s.queries = q
	return s.err
}
