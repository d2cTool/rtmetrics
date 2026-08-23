package noosexit_test

import (
	"testing"

	"github.com/d2cTool/rtmetrics/cmd/staticlint/noosexit"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), noosexit.Analyzer, "a")
}
