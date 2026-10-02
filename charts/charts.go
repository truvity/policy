// Package charts carries the library chart, so that the tests of every chart
// that depends on it render the library as it stands in this tree.
//
// The embed is not decoration. The chart tests shell out to `helm`, and Go's
// test cache keys on the files a test package READS, not on what a
// subprocess reads: without it, editing a library template would leave a
// cached PASS in place and every chart that uses it would go unchecked.
package charts

import "embed"

// Library is the library chart `service-lib`, rooted at `service-lib/`. It
// is the source of truth: a chart depends on it through
// `file://…/charts/service-lib` and `helm dependency build` resolves it.
//
//go:embed all:service-lib
var Library embed.FS
