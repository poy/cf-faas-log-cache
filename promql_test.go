package faaspromql_test

import (
	"testing"

	faaspromql "github.com/apoydence/cf-faas-log-cache"
	"github.com/apoydence/onpar"
	. "github.com/apoydence/onpar/expect"
	. "github.com/apoydence/onpar/matchers"
)

func TestPromQLUnmarshal(t *testing.T) {
	t.Parallel()
	o := onpar.New()
	defer o.Run(t)

	o.Spec("vector", func(t *testing.T) {
		var r faaspromql.QueryResult
		err := faaspromql.UnmarshalJSON([]byte(`{
          "status": "success",
          "data": {
            "resultType": "vector",
            "result": [
              {
                "metric": {
                  "deployment": "some-deployment",
                  "index": "some-index"
                },
                "value": [
                  1536028324000000000,
                  71876608
                ]
              }
            ]
          }
        }`), &r)
		Expect(t, err).To(BeNil())

		Expect(t, r.Status).To(Equal("success"))
		Expect(t, r.Data.ResultType).To(Equal("vector"))
		Expect(t, r.Data.Result).To(HaveLen(1))
		Expect(t, r.Data.Result[0].(*faaspromql.Sample).Metric).To(Equal(map[string]string{
			"deployment": "some-deployment",
			"index":      "some-index",
		}))

		i, err := r.Data.Result[0].(*faaspromql.Sample).Value[0].Int64()
		Expect(t, err).To(BeNil())
		Expect(t, i).To(Equal(int64(1536028324000000000)))
		f, err := r.Data.Result[0].(*faaspromql.Sample).Value[1].Float64()
		Expect(t, err).To(BeNil())
		Expect(t, f).To(Equal(71876608.0))
	})

	o.Spec("invalid vector", func(t *testing.T) {
		var r faaspromql.QueryResult
		err := faaspromql.UnmarshalJSON([]byte(`{
          "status": "success",
          "data": {
            "resultType": "vector",
            "result": [
              {
                "metric": {
                  "deployment": ["some-deployment","invalid"]
                  "index": "some-index"
                },
                "value": [
                  1536028324000000000,
                  71876608
                ]
              }
            ]
          }
        }`), &r)
		Expect(t, err).To(Not(BeNil()))
	})

	o.Spec("matrix", func(t *testing.T) {
		var r faaspromql.QueryResult
		err := faaspromql.UnmarshalJSON([]byte(`{
          "status": "success",
          "data": {
            "resultType": "matrix",
            "result": [
              {
                "metric": {
                  "deployment": "some-deployment",
                  "index": "some-index"
                },
                "values": [
                  [
                    1536032345000000000,
                    71913472
                  ]
                ]
              }
            ]
          }
        }`), &r)
		Expect(t, err).To(BeNil())

		Expect(t, r.Status).To(Equal("success"))
		Expect(t, r.Data.ResultType).To(Equal("matrix"))
		Expect(t, r.Data.Result).To(HaveLen(1))
		Expect(t, r.Data.Result[0].(*faaspromql.Series).Metric).To(Equal(map[string]string{
			"deployment": "some-deployment",
			"index":      "some-index",
		}))

		i, err := r.Data.Result[0].(*faaspromql.Series).Values[0][0].Int64()
		Expect(t, err).To(BeNil())
		Expect(t, i).To(Equal(int64(1536032345000000000)))
		f, err := r.Data.Result[0].(*faaspromql.Series).Values[0][1].Float64()
		Expect(t, err).To(BeNil())
		Expect(t, f).To(Equal(71913472.0))
	})

	o.Spec("invalid matrix", func(t *testing.T) {
		var r faaspromql.QueryResult
		err := faaspromql.UnmarshalJSON([]byte(`{
          "status": "success",
          "data": {
            "resultType": "matrix",
            "result": [
              {
                "metric": {
                  "deployment": ["some-deployment","invalid"],
                  "index": "some-index"
                },
                "values": [
                  [
                    1536032345000000000,
                    71913472
                  ]
                ]
              }
            ]
          }
        }`), &r)
		Expect(t, err).To(Not(BeNil()))
	})

	o.Spec("invalid type", func(t *testing.T) {
		var r faaspromql.QueryResult
		err := faaspromql.UnmarshalJSON([]byte(`{
          "status": "success",
          "data": {
            "resultType": "invalid",
            "result": [
              {
                "metric": {
                  "deployment": "some-deployment",
                  "index": "some-index"
                },
                "value": [
                  1536028324000000000,
                  71876608
                ]
              }
            ]
          }
        }`), &r)
		Expect(t, err).To(Not(BeNil()))
	})
}
