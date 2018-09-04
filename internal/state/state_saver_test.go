package state_test

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"testing"

	"github.com/apoydence/cf-faas-log-cache/internal/state"
	"github.com/apoydence/cf-faas-log-cache/internal/web"
	"github.com/apoydence/onpar"
	. "github.com/apoydence/onpar/expect"
	. "github.com/apoydence/onpar/matchers"
)

type TS struct {
	*testing.T
	s             *state.Saver
	spyCapiClient *spyCapiClient
}

func TestStateSaver(t *testing.T) {
	t.Parallel()
	o := onpar.New()
	defer o.Run(t)

	o.BeforeEach(func(t *testing.T) TS {
		spyCapiClient := newSpyCapiClient()
		return TS{
			T:             t,
			s:             state.NewSaver("some-guid", spyCapiClient, log.New(ioutil.Discard, "", 0)),
			spyCapiClient: spyCapiClient,
		}
	})

	o.Spec("it saves the state as JSON in the environment variables", func(t TS) {
		err := t.s.SaveState(context.Background(), []web.Query{
			{
				Query: "some-query-1",
				Path:  "some-path-1",
			},
			{
				Query: "some-query-2",
				Path:  "some-path-2",
			},
		})
		Expect(t, err).To(BeNil())

		Expect(t, t.spyCapiClient.setEnvCtx).To(Not(BeNil()))
		_, ok := t.spyCapiClient.setEnvCtx.Deadline()
		Expect(t, ok).To(BeTrue())
		Expect(t, t.spyCapiClient.setEnvCtx.Err()).To(Not(BeNil()))
		Expect(t, t.spyCapiClient.setEnvAppGuid).To(Equal("some-guid"))
		Expect(t, t.spyCapiClient.setEnvVars["QUERIES"]).To(
			MatchJSON(`{"queries":[{"query":"some-query-1","path":"some-path-1"},{"query":"some-query-2","path":"some-path-2"}]}`),
		)
	})

	o.Spec("it returns an error if saving the env fails", func(t TS) {
		t.spyCapiClient.setEnvErr = errors.New("some-error")
		err := t.s.SaveState(context.Background(), []web.Query{
			{
				Query: "some-query-1",
				Path:  "some-path-1",
			},
			{
				Query: "some-query-2",
				Path:  "some-path-2",
			},
		})
		Expect(t, err).To(Not(BeNil()))
	})

	o.Spec("it restarts the app", func(t TS) {
		err := t.s.SaveState(context.Background(), []web.Query{
			{
				Query: "some-query-1",
				Path:  "some-path-1",
			},
			{
				Query: "some-query-2",
				Path:  "some-path-2",
			},
		})
		Expect(t, err).To(BeNil())

		Expect(t, t.spyCapiClient.restartCtx).To(Not(BeNil()))
		_, ok := t.spyCapiClient.restartCtx.Deadline()
		Expect(t, ok).To(BeTrue())
		Expect(t, t.spyCapiClient.restartCtx.Err()).To(Not(BeNil()))
		Expect(t, t.spyCapiClient.restartAppGuid).To(Equal("some-guid"))
	})

	o.Spec("it returns an error if restarting fails", func(t TS) {
		t.spyCapiClient.restartErr = errors.New("some-error")
		err := t.s.SaveState(context.Background(), []web.Query{
			{
				Query: "some-query-1",
				Path:  "some-path-1",
			},
			{
				Query: "some-query-2",
				Path:  "some-path-2",
			},
		})
		Expect(t, err).To(Not(BeNil()))
	})
}

type spyCapiClient struct {
	setEnvCtx     context.Context
	setEnvAppGuid string
	setEnvVars    map[string]string
	setEnvErr     error

	getEnvCtx     context.Context
	getEnvAppGuid string
	getEnvVars    map[string]string
	getEnvErr     error

	restartCtx     context.Context
	restartAppGuid string
	restartErr     error
}

func newSpyCapiClient() *spyCapiClient {
	return &spyCapiClient{}
}

func (s *spyCapiClient) SetEnvironmentVariables(ctx context.Context, appGuid string, vars map[string]string) error {
	s.setEnvCtx = ctx
	s.setEnvAppGuid = appGuid
	s.setEnvVars = vars
	return s.setEnvErr
}

func (s *spyCapiClient) GetEnvironmentVariables(ctx context.Context, appGuid string) (map[string]string, error) {
	s.getEnvCtx = ctx
	s.getEnvAppGuid = appGuid
	return s.getEnvVars, s.getEnvErr
}

func (s *spyCapiClient) Restart(ctx context.Context, appGuid string) error {
	s.restartCtx = ctx
	s.restartAppGuid = appGuid
	return s.restartErr
}
