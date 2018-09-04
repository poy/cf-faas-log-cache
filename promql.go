package faaspromql

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	faas "github.com/apoydence/cf-faas"
)

type Handler interface {
	Handle(QueryResult) error
}

type HandlerFunc func(QueryResult) error

func (f HandlerFunc) Handle(r QueryResult) error {
	return f(r)
}

func Start(h Handler) {
	faas.Start(faas.HandlerFunc(func(req faas.Request) (faas.Response, error) {
		var r QueryResult
		if err := UnmarshalJSON(req.Body, &r); err != nil {
			return faas.Response{}, err
		}

		if err := h.Handle(r); err != nil {
			return faas.Response{}, err
		}

		return faas.Response{
			StatusCode: http.StatusOK,
		}, nil
	}))
}

type QueryResult struct {
	Status  string    `json:"status"`
	Data    RawResult `json:"data"`
	Context string    `json:"context"`
}

type RawResult struct {
	ResultType string `json:"resultType"`

	// Result will be *Sample or *Series
	Result []interface{} `json:"-"`

	// RawData is for unmarshal only. Don't use.
	RawResult []json.RawMessage `json:"result,omitempty"`
}

type Sample struct {
	Metric map[string]string `json:"metric"`
	Value  []json.Number     `json:"value"`
}

type Series struct {
	Metric map[string]string `json:"metric"`
	Values [][]json.Number   `json:"values"`
}

func UnmarshalJSON(data []byte, r *QueryResult) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	if err := dec.Decode(r); err != nil {
		return err
	}

	var results []interface{}
	for _, s := range r.Data.RawResult {
		var dst interface{}
		switch r.Data.ResultType {
		case "vector":
			dst = &Sample{}
		case "matrix":
			dst = &Series{}
		default:
			return fmt.Errorf("unknown ResultType: %s", r.Data.ResultType)
		}

		if err := json.Unmarshal(s, dst); err != nil {
			return err
		}
		results = append(results, dst)
	}
	r.Data.RawResult = nil
	r.Data.Result = results

	return nil
}
