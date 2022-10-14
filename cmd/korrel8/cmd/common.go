package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/korrel8/korrel8/pkg/alert"
	"github.com/korrel8/korrel8/pkg/engine"
	"github.com/korrel8/korrel8/pkg/k8s"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/loki"
	"github.com/korrel8/korrel8/pkg/templaterule"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func must[T any](v T, err error) T { check(err); return v }

func open(name string) (f *os.File) {
	if name == "-" {
		return os.Stdin
	} else {
		return must(os.Open(name))
	}
}

func restConfig() *rest.Config {
	cfg, err := config.GetConfig()
	if err == nil {
		cfg.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(100, 1000)
	}
	return must(cfg, err)
}

func k8sClient(cfg *rest.Config) client.Client {
	return must(client.New(cfg, client.Options{}))
}

// noStore returns an error if it is used.
type noStore struct{ err error }

// Defer store creation errors until the store is used. It may no be.
func needStore(store korrel8.Store, err error) korrel8.Store {
	if err != nil {
		return noStore{err}
	}
	return store
}
func (s noStore) Get(context.Context, korrel8.Query, korrel8.Result) error { return s.err }

func newEngine() *engine.Engine {
	cfg := restConfig()
	e := engine.New()
	e.AddDomain(k8s.Domain, needStore(k8s.NewStore(k8sClient(cfg))))
	e.AddDomain(alert.Domain, needStore(alert.NewStore(cfg)))
	e.AddDomain(loki.Domain, loki.NewStore(*lokiBaseURL, http.DefaultClient))

	// Load rules.
	for _, root := range *rulePaths {
		check(filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !d.Type().IsRegular() {
				return nil
			}

			if filepath.Ext(path) != ".yml" && filepath.Ext(path) != ".yaml" && filepath.Ext(path) != ".json" {
				return nil
			}

			f, err := os.Open(path)
			if err != nil {
				return err
			}

			if err := templaterule.Read(f, e); err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}

			return nil
		}))
	}

	return e
}

func jsonString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return err.Error()
	}
	return string(b)
}

type printer struct{ print func(o korrel8.Object) }

func newPrinter(w io.Writer) printer {
	switch *output {

	case "json":
		return printer{print: func(o korrel8.Object) { fmt.Fprintln(w, jsonString(o)) }}

	case "json-pretty":
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return printer{print: func(o korrel8.Object) { check(encoder.Encode(o)) }}

	case "yaml":
		return printer{print: func(o korrel8.Object) { fmt.Fprintf(w, "---\n%s", must(yaml.Marshal(&o))) }}

	default:
		check(fmt.Errorf("invalid output type: %v", *output))
		return printer{}
	}
}

func (p printer) Append(objects ...korrel8.Object) {
	for _, o := range objects {
		p.print(o)
	}
}
