package promql

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

type Client struct {
	addr string
	d    Doer
}

func NewClient(addr string, d Doer) *Client {
	return &Client{
		addr: addr,
		d:    d,
	}
}

func (c *Client) PromQL(ctx context.Context, query string) (bool, error) {
	req, err := http.NewRequest(http.MethodGet, c.addr+"/api/v1/query", nil)
	if err != nil {
		return false, err
	}
	v := req.URL.Query()
	v.Set("query", query)
	req.URL.RawQuery = v.Encode()

	ctx, _ = context.WithTimeout(ctx, 5*time.Second)
	req = req.WithContext(ctx)

	resp, err := c.d.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to make PromQL request: %s", err)
	}

	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return false, fmt.Errorf("unexpected status code %d getting PromQL results: %s", resp.StatusCode, body)
	}

	var result struct {
		Data struct {
			Result []map[string]interface{} `json:"result"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to parse PromQL results: %s", err)
	}

	return len(result.Data.Result) > 0, nil
}
