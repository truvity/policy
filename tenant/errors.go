package tenant

import (
	"fmt"
	"sort"
)

// collector gathers every refusal of one pass.
type collector struct {
	company string
	errs    Errors
}

func (c *collector) add(file, path string, code Code, format string, args ...any) {
	c.errs = append(c.errs, Error{
		Company: c.company,
		File:    file,
		Path:    path,
		Code:    code,
		Reason:  fmt.Sprintf(format, args...),
	})
}

// result is nil when nothing was refused, so a caller can compare it to nil.
func (c *collector) result() error {
	if len(c.errs) == 0 {
		return nil
	}

	c.sort()

	return c.errs
}

// sort puts the refusals in file order (the order of [sections], then any file
// the contract does not name), keeping the order rows were found in within a
// file, so that two runs over one bundle print the same lines.
func (c *collector) sort() {
	sort.SliceStable(c.errs, func(i, j int) bool {
		return fileRank(c.errs[i].File) < fileRank(c.errs[j].File)
	})
}

func fileRank(file string) int {
	for i := range sections {
		if sections[i].file == file {
			return i
		}
	}

	return len(sections)
}

// at formats a list position as it appears in a path: "grants[2]".
func at(list string, i int) string { return fmt.Sprintf("%s[%d]", list, i) }
