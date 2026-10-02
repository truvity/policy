package charts_test

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/truvity/policy/charts"
)

// Every template a library defines lands in Helm's ONE global template
// namespace, shared with every chart that includes it and every sub-chart of
// those. A name without the library's prefix is a name that can silently
// replace, or be replaced by, a chart's own, and Helm says nothing: the last
// definition wins. An invalid chart (one that collides) renders, and renders
// something else; so the prefix is held here rather than reviewed.
func TestEveryDefinedTemplateCarriesTheLibrarysPrefix(t *testing.T) {
	define := regexp.MustCompile(`{{-?\s*define\s+"([^"]+)"`)

	var defined int
	err := fs.WalkDir(charts.Library, "service-lib/templates", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, err := charts.Library.ReadFile(p)
		if err != nil {
			return err
		}
		for _, m := range define.FindAllStringSubmatch(string(raw), -1) {
			defined++
			if !strings.HasPrefix(m[1], "service-lib.") {
				t.Errorf("%s defines %q, which is outside the service-lib. namespace", p, m[1])
			}
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if defined == 0 {
		t.Fatal("the library defines no template, so the prefix was not checked")
	}
}
