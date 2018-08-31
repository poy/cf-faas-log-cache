package promql

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/apoydence/cf-faas-log-cache/internal/web"
)

type Reader struct {
	c   PromQLClient
	d   Doer
	log *log.Logger
	q   web.Query
}

type PromQLClient interface {
	PromQL(
		ctx context.Context,
		query string,
	) (bool, error)
}

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

func NewReader(
	q web.Query,
	c PromQLClient,
	d Doer,
	log *log.Logger,
) *Reader {
	return &Reader{
		q:   q,
		c:   c,
		d:   d,
		log: log,
	}
}

func (r *Reader) Tick() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	hasData, err := r.c.PromQL(ctx, r.q.Query)
	if err != nil {
		r.log.Printf("failed to make PromQL query: %s", err)
		return
	}

	if !hasData {
		return
	}

	req, err := http.NewRequest("POST", r.q.Path, strings.NewReader(r.q.Context))
	if err != nil {
		r.log.Panicf("failed to parse request: %s", err)
	}

	resp, err := r.d.Do(req)
	if err != nil {
		r.log.Printf("failed to make POST: %s", err)
		return
	}

	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		r.log.Printf("POST returned unexected status code %d: %s", resp.StatusCode, body)
		return
	}

	r.log.Print("successfully made POST")
}
