// package alert implements korrel8 interfaces on prometheus alerts.
package alert

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alanconway/korrel8/pkg/korrel8"
	"github.com/prometheus/alertmanager/pkg/labels"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

var Domain = domain{}

type domain struct{}

func (d domain) String() string                { return "alert" }
func (d domain) Class(string) korrel8.Class    { return Class{} }
func (d domain) KnownClasses() []korrel8.Class { panic("not implemented") } // FIXME

var _ korrel8.Domain = Domain

type Store struct {
	api  v1.API
	host string
}

func newAlertStore(host string, rt http.RoundTripper) (*Store, error) {
	host = fmt.Sprintf("https://%s", host)
	client, err := api.NewClient(api.Config{
		Address:      host,
		RoundTripper: rt,
	})
	if err != nil {
		return nil, err
	}

	return &Store{
		api:  v1.NewAPI(client),
		host: host,
	}, nil
}

type Class struct{} // Only one class

func (c Class) Domain() korrel8.Domain         { return Domain }
func (c Class) String() string                 { return Domain.String() }
func (c Class) New() korrel8.Object            { return Alert{} }
func (c Class) Key(o korrel8.Object) any       { return o }
func (c Class) Contains(o korrel8.Object) bool { _, ok := o.(Alert); return ok }

// Alert is a 1:1 mapping of the v1.Alert type which can be used in Go templates.
type Alert struct {
	ActiveAt    time.Time
	Annotations map[string]string
	Labels      map[string]string
	State       string
	Value       string
}

func convert(a v1.Alert) Alert {
	r := Alert{
		ActiveAt:    a.ActiveAt,
		Annotations: map[string]string{},
		Labels:      map[string]string{},
		State:       string(a.State),
		Value:       a.Value,
	}

	for k, v := range a.Labels {
		r.Labels[string(k)] = string(v)
	}

	for k, v := range a.Annotations {
		r.Annotations[string(k)] = string(v)
	}

	return r
}

// Get implements the korrel8.Store interface.
// The query parameter is a PromQL label matcher expression with the wrapping
// `{` and `}` being optional, e.g.  `namespace="default",pod=~"myapp-.+"`.
func (s Store) Get(ctx context.Context, query korrel8.Query, result korrel8.Result) error {
	// TODO: allow to filter on alert state (pending/firing)?
	// TODO: support sorting order (e.g. most recent/oldest, severity)?
	// TODO: allow grouping (all alerts related to podX grouped together)?
	matchers, err := labels.ParseMatchers(strings.Trim(string(query), "\n"))
	if err != nil {
		return fmt.Errorf("failed to parse query: %q: %w", query, err)
	}

	resp, err := s.api.Alerts(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", s.host, err)
	}

	for _, a := range resp.Alerts {
		if labels.Matchers(matchers).Matches(a.Labels) {
			result.Append(convert(a))
		}
	}

	return nil
}
