package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/rai/clean-modularmonolith-go/tools/spannercheck"
)

func main() {
	singlechecker.Main(spannercheck.Analyzer)
}
