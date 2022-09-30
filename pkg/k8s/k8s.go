// package k8s is a Kubernetes implementation of the korrel8 interfaces
package k8s

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"github.com/alanconway/korrel8/internal/pkg/logging"
	"github.com/alanconway/korrel8/pkg/korrel8"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"regexp"

	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	log    = logging.Log
	Scheme = scheme.Scheme
)

var Domain = domain{}

type domain struct{}

func (d domain) String() string { return "k8s" }

// parseGVK parse name of the form: kind[.version[.group]]
// TODO is this in the k8s libs?
func parseGVK(name string) (gvk schema.GroupVersionKind) {
	parts := strings.SplitN(name, ".", 3)
	for i, f := range []*string{&gvk.Kind, &gvk.Version, &gvk.Group} {
		if len(parts) > i {
			*f = parts[i]
		}
	}
	return gvk
}

func (d domain) Class(name string) korrel8.Class {
	gvk := parseGVK(name)
	var t reflect.Type
	if gvk.Group != "" { // Fully qualified, direct lookup
		return Class{Scheme.AllKnownTypes()[gvk]}
	} else { // No version
		gvs := Scheme.PreferredVersionAllGroups()
		for _, gv := range gvs {
			if t = Scheme.KnownTypes(gv)[gvk.Kind]; t != nil {
				return Class{t}
			}
		}
	}
	return nil
}

func (d domain) KnownClasses() (classes []korrel8.Class) {
	for _, t := range Scheme.AllKnownTypes() {
		classes = append(classes, Class{t})
	}
	return classes
}

var _ korrel8.Domain = Domain // Implements interface

// TODO the Class implementation assumes all objects are pointers to the generated API struct.
// We could use scheme & GVK comparisons to generalize to untyped representations as well.

// Class ses the Go API struct type to identify a kind of resource.
type Class struct{ reflect.Type }

// ClassOf returns the Class of o, which must be a pointer to a typed API resource struct.
func ClassOf(o client.Object) Class { return Class{reflect.TypeOf(o).Elem()} }

func (c Class) String() string {
	// TODO there must be an easier way...
	for gvk, t := range Scheme.AllKnownTypes() {
		if t == c.Type {
			if gvk.Group == "" { // Core group has empty group name
				return fmt.Sprintf("%v.%v", gvk.Kind, gvk.Version)
			}
			return fmt.Sprintf("%v.%v.%v", gvk.Kind, gvk.Version, gvk.Group)
		}
	}
	return c.Type.String()
}

func (c Class) Domain() korrel8.Domain { return Domain }
func (c Class) New() korrel8.Object    { return Object{reflect.New(c.Type).Interface().(client.Object)} }

var _ korrel8.Class = Class{} // Implements interface.

type Object struct{ client.Object }

func (o Object) Native() any { return o.Object }

type Identifier struct {
	Name, Namespace string
	Class           korrel8.Class
}

func (o Object) Identifier() korrel8.Identifier {
	return types.NamespacedName{Namespace: o.GetNamespace(), Name: o.GetName()}
}

// Store implements the korrel8.Store interface over a k8s API client.
type Store struct{ c client.Client }

// NewStore creates a new store
func NewStore(c client.Client) (*Store, error) { return &Store{c: c}, nil }

// Execute a query in the form of a k8s REST URI.
// Cancel if context is canceled.
func (s *Store) Query(ctx context.Context, query string) (result []korrel8.Object, err error) {
	log.Info("query", "domain", Domain, "query", query)
	defer func() {
		if err == nil {
			b := &strings.Builder{}
			sep := ""
			for _, o := range result {
				fmt.Fprintf(b, "%v%v", sep, o.Identifier())
				sep = ","
			}
			log.Info("result    ", "objects", b.String())
		} else {
			err = fmt.Errorf("%w: executing %v query %q", err, Domain, query)
		}
	}()
	u, err := url.Parse(string(query))
	if err != nil {
		return nil, err
	}
	gvk, nsName, err := s.parseAPIPath(u)
	if err != nil {
		return nil, err
	}
	if nsName.Name != "" { // Request for single object.
		return s.getObject(ctx, gvk, nsName)
	} else {
		return s.getList(ctx, gvk, nsName.Namespace, u.Query())
	}
}

// parsing a REST URI into components then using client.Client to recreate the REST query.
//
// FIXME revisit: this is weirdly indirect - parse an API path to make a Client call which re-creates the API path.
// Should be able to use a REST client directly, but client.Client does REST client creation & caching
// and manages schema and RESTMapper stuff which I'm not sure I understand yet.
// Use a JSON query object instead?
func (s *Store) parseAPIPath(u *url.URL) (gvk schema.GroupVersionKind, nsName types.NamespacedName, err error) {
	path := k8sPathRegex.FindStringSubmatch(u.Path)
	if len(path) != pCount {
		return gvk, nsName, fmt.Errorf("invalid URI")
	}
	nsName.Namespace, nsName.Name = path[pNamespace], path[pName]
	gvr := schema.GroupVersionResource{Group: path[pGroup], Version: path[pVersion], Resource: path[pResource]}
	gvk, err = s.c.RESTMapper().KindFor(gvr)
	return gvk, nsName, err
}

func (s *Store) getObject(ctx context.Context, gvk schema.GroupVersionKind, nsName types.NamespacedName) ([]korrel8.Object, error) {
	scheme := s.c.Scheme()
	o, err := scheme.New(gvk)
	if err != nil {
		return nil, err
	}
	co, _ := o.(client.Object)
	if co == nil {
		return nil, fmt.Errorf("invalid client.Object: %T", o)
	}
	err = s.c.Get(ctx, nsName, co)
	if err != nil {
		return nil, err
	}
	return []korrel8.Object{Object{co}}, nil
}

func (s *Store) parseAPIQuery(q url.Values) (opts []client.ListOption, err error) {
	if s := q.Get("labelSelector"); s != "" {
		selector, err := labels.Parse(s)
		if err != nil {
			return nil, err
		}
		opts = append(opts, client.MatchingLabelsSelector{Selector: selector})
	}
	if s := q.Get("fieldSelector"); s != "" {
		selector, err := fields.ParseSelector(s)
		if err != nil {
			return nil, err
		}
		opts = append(opts, client.MatchingFieldsSelector{Selector: selector})
	}
	return opts, nil
}

func (s *Store) getList(ctx context.Context, gvk schema.GroupVersionKind, namespace string, query url.Values) ([]korrel8.Object, error) {
	gvk.Kind = gvk.Kind + "List"
	o, err := s.c.Scheme().New(gvk)
	if err != nil {
		return nil, err
	}
	list, _ := o.(client.ObjectList)
	if list == nil {
		return nil, fmt.Errorf("invalid list object %T", o)
	}
	opts, err := s.parseAPIQuery(query)
	if err != nil {
		return nil, err
	}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	if err := s.c.List(ctx, list, opts...); err != nil {
		return nil, err
	}
	defer func() { // Handle reflect panics.
		if r := recover(); r != nil && err == nil {
			err = fmt.Errorf("invalid list object: %T", list)
		}
	}()
	items := reflect.ValueOf(list).Elem().FieldByName("Items")
	var result []korrel8.Object
	for i := 0; i < items.Len(); i++ {
		result = append(result, Object{items.Index(i).Addr().Interface().(client.Object)})
	}
	return result, nil
}

// Parse a K8s API path into: group, version, namespace, resourcetype, name.
// See: https://kubernetes.io/docs/reference/using-api/api-concepts/
var k8sPathRegex = regexp.MustCompile(`^(?:(?:/apis/([^/]+)/)|(?:/api/))([^/]+)(?:/namespaces/([^/]+))?/([^/]+)(?:/([^/]+))?`)

// Indices for match results from k8sPathRegex
const (
	pGroup = iota + 1
	pVersion
	pNamespace
	pResource
	pName
	pCount
)
