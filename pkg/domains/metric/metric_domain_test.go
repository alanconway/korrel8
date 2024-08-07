// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package metric_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/domain"
	"github.com/korrel8r/korrel8r/pkg/domains/metric"
)

// FIXME store does not respect limits, remove SkipOpenshift
var fixture = domain.Fixture{Query: metric.Query(`{namespace="kube-system"}`), SkipOpenshift: true}

func TestMetricDomain(t *testing.T)      { fixture.Test(t) }
func BenchmarkMetricDomain(b *testing.B) { fixture.Benchmark(b) }
