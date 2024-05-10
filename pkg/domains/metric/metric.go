// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package metric represents Prometheus metric samples as objects.
//
// # Class
//
// There is only one class: `metric:metric`
//
// # Object
//
// A [metric sample], which includes a metric time series (name and labels), a timestamp and a value.
//
// # Query
//
// Queries are [PromQL] time series selector strings, prefixed by `metric:metric:` for example:
//
//	metric:metric:http_requests_total{environment=~"staging|testing|development",method!="GET"}
//
// # Store
//
// Prometheus is the store, store configuration:
//
//	domain: metric
//	metric: URL_OF_PROMETHEUS
//
// [PromQL]: https://prometheus.io/docs/prometheus/latest/querying/basics/#time-series-selectors
// [metric sample]: https://pkg.go.dev/github.com/prometheus/common@v0.45.0/model#Sample
package metric

// TODO: doc comment needs to show model.Sample structure or link to it properly.
// TODO: metrics are only usable as goals.

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	"github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

var (
	Domain = domain{}
	// Validate implementation of interfaces.
	_ korrel8r.Domain = Domain
	_ korrel8r.Class  = Class{}
	_ korrel8r.Query  = Query{}
	_ korrel8r.Store  = &Store{}
)

type domain struct{}

func (domain) Name() string                     { return "metric" }
func (d domain) String() string                 { return d.Name() }
func (domain) Description() string              { return "Time-series of measured values" }
func (domain) Class(name string) korrel8r.Class { return Class{} }
func (domain) Classes() []korrel8r.Class        { return []korrel8r.Class{Class{}} }
func (d domain) Query(s string) (korrel8r.Query, error) {
	_, qs, err := impl.ParseQueryString(d, s)
	return Query{PromQL: qs}, err
}

const StoreKeyMetricURL = "metric"

func (domain) Store(s any) (korrel8r.Store, error) {
	cs, err := impl.TypeAssert[config.Store](s)
	if err != nil {
		return nil, err
	}
	hc, err := k8s.NewHTTPClient(cs)
	if err != nil {
		return nil, err
	}
	return NewStore(cs[StoreKeyMetricURL], hc)
}

type Class struct{} // Singleton class

func (c Class) Domain() korrel8r.Domain { return Domain }
func (c Class) Name() string            { return Domain.Name() }
func (c Class) String() string          { return impl.ClassString(c) }
func (c Class) Description() string     { return "A set of label:value pairs identifying a time-series." }
func (c Class) New() korrel8r.Object    { var obj Object; return obj }
func (c Class) Preview(o korrel8r.Object) string {
	switch o := o.(type) {
	case *model.Sample:
		if name, ok := o.Metric["__name__"]; ok {
			return fmt.Sprintf("%v", name)
		} else {
			keys := maps.Keys(o.Metric)
			slices.Sort(keys)
			return fmt.Sprintf("%v", keys)
		}
	default:
		return fmt.Sprintf("(%T)%v", o, o)
	}
}

type Object *model.Sample

type Query struct {
	PromQL string // `json:",omitempty"`
}

func (q Query) Class() korrel8r.Class { return Class{} }
func (q Query) Data() string          { return q.PromQL }
func (q Query) String() string        { return impl.QueryString(q) }

type Store struct{ api promv1.API }

func NewStore(baseURL string, hc *http.Client) (korrel8r.Store, error) {
	c, err := api.NewClient(api.Config{Address: baseURL, Client: hc})
	if err != nil {
		return nil, err
	}
	return &Store{promv1.NewAPI(c)}, nil
}

func (s *Store) Domain() korrel8r.Domain { return Domain }

func (s *Store) Get(ctx context.Context, query korrel8r.Query, c *korrel8r.Constraint, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return err
	}
	var t time.Time
	// Evaluate time series at end of the constraint time range or now if there is no end.
	if c != nil && c.End != nil {
		t = *c.End
	} else {
		t = time.Now()
	}
	value, _, err := s.api.Query(ctx, q.PromQL, t)
	if err != nil {
		return err
	}
	if vector, ok := value.(model.Vector); ok {
		for _, v := range vector {
			result.Append(v)
		}
	} else {
		return fmt.Errorf("unexpected metric value: (%T)%#v", value, value)
	}
	return nil
}
