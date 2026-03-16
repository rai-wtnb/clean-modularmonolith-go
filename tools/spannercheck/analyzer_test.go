package spannercheck_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/rai/clean-modularmonolith-go/tools/spannercheck"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, spannercheck.Analyzer,
		"github.com/rai/clean-modularmonolith-go/testpkg/infrastructure/persistence",
		"github.com/rai/clean-modularmonolith-go/testpkg/nonpersistence",
	)
}
