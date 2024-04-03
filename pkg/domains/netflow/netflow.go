package netflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/korrel8r/korrel8r/internal/pkg/loki"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	"github.com/korrel8r/korrel8r/pkg/openshift"
	"golang.org/x/exp/maps"
)

var (
	// Verify implementing interfaces.
	_ korrel8r.Domain     = Domain
	_ openshift.Converter = Domain
	_ korrel8r.Store      = &store{}
	_ korrel8r.Store      = &stackStore{}
	_ korrel8r.Query      = Query{}
	_ korrel8r.Class      = Class{}
	_ korrel8r.Previewer  = Class{}
)

// Domain for log records produced by openshift-logging.
//
// There are several possible log store configurations:
// - Default LokiStack store on current Openshift cluster: `{}`
// - Remote LokiStack: `{ "lokiStack": "https://url-of-lokistack"}`
// - Plain Loki store: `{ "loki": "https://url-of-loki"}`
var Domain = domain{}

type domain struct{}

func (domain) Name() string                     { return "netflow" }
func (d domain) String() string                 { return d.Name() }
func (domain) Description() string              { return "Network flows from source nodes to destination nodes." }
func (domain) Class(name string) korrel8r.Class { return Class{} }
func (domain) Classes() []korrel8r.Class        { return []korrel8r.Class{Class{}} }
func (d domain) Query(s string) (korrel8r.Query, error) {
	_, s, err := impl.ParseQueryString(d, s)
	if err != nil {
		return nil, err
	}
	return Query{logQL: s}, nil
}

const (
	StoreKeyLoki      = "loki"
	StoreKeyLokiStack = "lokiStack"
)

func (domain) Store(sc korrel8r.StoreConfig) (korrel8r.Store, error) {
	hc, err := k8s.NewHTTPClient()
	if err != nil {
		return nil, err
	}

	loki, lokiStack := sc[StoreKeyLoki], sc[StoreKeyLokiStack]
	switch {

	case loki != "" && lokiStack != "":
		return nil, fmt.Errorf("can't set both loki and lokiStack URLs")

	case loki != "":
		u, err := url.Parse(loki)
		if err != nil {
			return nil, err
		}
		return NewPlainLokiStore(u, hc)

	case lokiStack != "":
		u, err := url.Parse(lokiStack)
		if err != nil {
			return nil, err
		}
		return NewLokiStackStore(u, hc)

	default:
		return nil, fmt.Errorf("must set one of loki or lokiStack URLs")
	}
}

// There is only a single class, named "netflow	".
type Class struct{}

func (c Class) Domain() korrel8r.Domain { return Domain }
func (c Class) Name() string            { return "network" }
func (c Class) String() string          { return korrel8r.ClassName(c) }
func (c Class) Description() string     { return "A set of label:value pairs identifying a netflow." }

func (c Class) New() korrel8r.Object { return NewObject(&loki.Entry{}) }

// Preview extracts the message from a Viaq log record.
func (c Class) Preview(x korrel8r.Object) (line string) { return Preview(x) }

// Preview extracts the message from a Viaq log record.
func Preview(x korrel8r.Object) (line string) {
    if m := x.(Object)["SrcK8S_Namespace"]; m != nil {
		s, _ := m.(string)
		message := "Network Flows from :" + s
        if m = x.(Object)["DstK8S_Namespace"]; m != nil {
            d, _ := m.(string)
			message = message + " to : " + d
        }
        return message
	}
	return ""
}

// Object is a map holding netflow entries
type Object map[string]any

func NewObject(entry *loki.Entry) Object {
	var label_object, o Object
	o = make(map[string]any)
	_ = json.Unmarshal([]byte(entry.Line), &o)
	if entry.Labels != nil {
		label_object = make(map[string]any)
		for k, v := range entry.Labels {
			label_object[k] = v
		}
		maps.Copy(o, label_object)
	}
	return o
}

func (o *Object) UnmarshalJSON(line []byte) error {
	if err := json.Unmarshal([]byte(line), (*map[string]any)(o)); err != nil {
		*o = map[string]any{"message": line}
	}
	return nil
}

// Query is a LogQL query string
type Query struct {
	logQL string // `json:",omitempty"`
}

func NewQuery(logQL string) korrel8r.Query { return Query{logQL: logQL} }

func (q Query) Class() korrel8r.Class { return Class{} }
func (q Query) Query() string         { return q.logQL }
func (q Query) String() string        { return korrel8r.QueryName(q) }

// NewLokiStackStore returns a store that uses a LokiStack observatorium-style URLs.
func NewLokiStackStore(base *url.URL, h *http.Client) (korrel8r.Store, error) {
	return &stackStore{store: store{loki.New(h, base)}}, nil
}

// NewPlainLokiStore returns a store that uses plain Loki URLs.
func NewPlainLokiStore(base *url.URL, h *http.Client) (korrel8r.Store, error) {
	return &store{loki.New(h, base)}, nil
}

type store struct{ *loki.Client }

func (store) Domain() korrel8r.Domain { return Domain }
func (s *store) Get(ctx context.Context, query korrel8r.Query, c *korrel8r.Constraint, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return err
	}
	return s.Client.Get(q.logQL, c, func(e *loki.Entry) { result.Append(NewObject(e)) })
}

type stackStore struct{ store }

func (stackStore) Domain() korrel8r.Domain { return Domain }
func (s *stackStore) Get(ctx context.Context, query korrel8r.Query, c *korrel8r.Constraint, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return err
	}

	return s.Client.GetStack(q.logQL, "network", c, func(e *loki.Entry) { result.Append(NewObject(e)) })
}

func (domain) QueryToConsoleURL(query korrel8r.Query) (*url.URL, error) {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return nil, err
	}
	v := url.Values{}
	v.Add("q", q.logQL+"|json")
	v.Add("tenant", "network")
	return &url.URL{Path: "/netflow-traffic", RawQuery: v.Encode()}, nil
}

func (domain) ConsoleURLToQuery(u *url.URL) (korrel8r.Query, error) {
	q := u.Query().Get("q")
	return NewQuery(q), nil
}
