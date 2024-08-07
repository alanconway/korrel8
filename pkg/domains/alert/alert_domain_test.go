// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package alert_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/domain"
	"github.com/korrel8r/korrel8r/pkg/domains/alert"
)

// FIXME store does not respect limits, remove SkipOpenshift
var fixture = domain.Fixture{Query: alert.Query{}, SkipOpenshift: true}

func TestAlertDomain(t *testing.T)      { fixture.Test(t) }
func BenchmarkAlertDomain(b *testing.B) { fixture.Benchmark(b) }
