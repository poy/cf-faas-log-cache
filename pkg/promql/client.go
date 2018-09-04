package promql

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/apoydence/cf-faas-log-cache"
)

type Client struct {
	addr string
	s    AppNameSanitizer
	d    Doer
}

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

type AppNameSanitizer interface {
	Sanitize(ctx context.Context, query string) (string, error)
}

func NewClient(addr string, s AppNameSanitizer, d Doer) *Client {
	return &Client{
		addr: addr,
		d:    d,
		s:    s,
	}
}

func (c *Client) PromQL(ctx context.Context, query string) (*faaspromql.QueryResult, error) {
	sctx, _ := context.WithTimeout(ctx, 5*time.Second)
	query, err := c.s.Sanitize(sctx, query)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, c.addr+"/api/v1/query", nil)
	if err != nil {
		return nil, err
	}
	v := req.URL.Query()
	v.Set("query", query)
	req.URL.RawQuery = v.Encode()

	rctx, _ := context.WithTimeout(ctx, 5*time.Second)
	req = req.WithContext(rctx)

	resp, err := c.d.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make PromQL request: %s", err)
	}

	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d getting PromQL results: %s", resp.StatusCode, body)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %s", err)
	}

	var result faaspromql.QueryResult
	if err := faaspromql.UnmarshalJSON(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse QueryResult: %s", err)
	}

	return &result, nil
}
