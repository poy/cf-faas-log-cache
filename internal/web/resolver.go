package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"

	faas "github.com/apoydence/cf-faas"
)

type Resolver struct {
	s   StateSaver
	log *log.Logger
}

type Query struct {
	Query   string `json:"query"`
	Path    string `json:"path"`
	Context string `json:"context"`
}

type StateSaver interface {
	SaveState(context.Context, []Query) error
}

func NewResolver(s StateSaver, log *log.Logger) http.Handler {
	return &Resolver{
		s:   s,
		log: log,
	}
}

func (s *Resolver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		io.Copy(ioutil.Discard, r.Body)
		r.Body.Close()
	}()
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req faas.ConvertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"error":%q}`, err)))
		return
	}

	var (
		resp    faas.ConvertResponse
		queries []Query
	)

	for _, f := range req.Functions {
		gd, ok := f.Events["promql"]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf(`{"error":%q}`, "promql type")))
			return
		}

		for _, e := range gd {
			qs, _ := e["query"].(string)
			if qs == "" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(fmt.Sprintf(`{"error":%q}`, "invalid/missing Query")))
				return
			}

			queryContext, _ := e["context"].(string)

			q := Query{
				Query:   qs,
				Context: queryContext,
				Path:    fmt.Sprintf("/%d-prom-ql", rand.Int63()),
			}
			queries = append(queries, q)

			hf := faas.ConvertHTTPFunction{
				Handler: f.Handler,
				Events: []faas.ConvertHTTPEvent{
					{
						Method: http.MethodPost,
						Path:   q.Path,
					},
				},
			}

			resp.Functions = append(resp.Functions, hf)
		}
	}

	data, err := json.Marshal(resp)
	if err != nil {
		s.log.Panicf("failed to marshal response: %s", err)
	}

	w.Write(data)
	r.Body.Close()

	// This has to go AFTER finishing the response. Otherwise we might restart
	// the app before we have the chance to respond.
	if err := s.s.SaveState(r.Context(), queries); err != nil {
		s.log.Printf("failed to save state: %s", err)
	}
}
