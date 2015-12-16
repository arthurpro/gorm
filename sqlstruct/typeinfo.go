package sqlstruct

// basic type introspection for automatic query value unmarshalling
//

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// mostly a verbatim copy of encoding/json/encode.go
type byName []field

func (x byName) Len() int      { return len(x) }
func (x byName) Swap(i, j int) { x[i], x[j] = x[j], x[i] }
func (x byName) Less(i, j int) bool {
	if x[i].name != x[j].name {
		return x[i].name < x[j].name
	}
	if len(x[i].index) != len(x[j].index) {
		return len(x[i].index) < len(x[j].index)
	}
	if x[i].tag != x[j].tag {
		return x[i].tag
	}
	return byIndex(x).Less(i, j)
}

// byIndex sorts field by index sequence.
type byIndex []field

func (x byIndex) Len() int      { return len(x) }
func (x byIndex) Swap(i, j int) { x[i], x[j] = x[j], x[i] }
func (x byIndex) Less(i, j int) bool {
	for k, xik := range x[i].index {
		if k >= len(x[j].index) {
			return false
		}
		if xik != x[j].index[k] {
			return xik < x[j].index[k]
		}
	}
	return len(x[i].index) < len(x[j].index)
}

type tagOptions string

func (o tagOptions) contains(opt string) bool {
	if len(o) == 0 {
		return false
	}
	s := string(o)
	for s != "" {
		var next string
		i := strings.Index(s, ",")
		if i >= 0 {
			s, next = s[:i], s[i+1:]
		}
		if s == opt {
			return true
		}
		s = next
	}
	return false
}

// index is a slice of field indices - it specifies parent/current
// field index
type field struct {
	ctx   string // containing struct name, empty for top-level struct
	name  string
	fname string // field's name (as found in the struct)
	tag   bool
	index []int
	typ   reflect.Type
}

func (f field) String() string {
	return fmt.Sprintf(`%s("%s"); tagged? %t, indices: [%v], typ: %v`,
		f.ctx, f.name, f.tag, f.index, f.typ)
}

func (f field) ColName() string {
	if f.name != f.fname {
		return fmt.Sprintf(`"%s"."%s" as "%s"`, f.ctx, f.fname, f.name)
	}
	return fmt.Sprintf(`"%s"."%s"`, f.ctx, f.name)
}

// parseTag splits a struct field's sql tag into its name and
// comma-separated options.
func parseTag(tag string) (string, tagOptions) {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx], tagOptions(tag[idx+1:])
	}
	return tag, tagOptions("")
}

func typeFields(t reflect.Type) []field {
	// Anonymous fields to explore at the current level and the next.
	current := []field{}
	next := []field{{typ: t}}

	// Count of queued names for current level and the next.
	count := map[reflect.Type]int{}
	nextCount := map[reflect.Type]int{}

	// Types already visited at an earlier level.
	visited := map[reflect.Type]bool{}

	// Fields found.
	var fields []field

	for len(next) > 0 {
		current, next = next, current[:0]
		count, nextCount = nextCount, map[reflect.Type]int{}

		for _, f := range current {
			if visited[f.typ] {
				continue
			}
			visited[f.typ] = true

			// Scan f.typ for fields to include.
			for i := 0; i < f.typ.NumField(); i++ {
				sf := f.typ.Field(i)
				if sf.PkgPath != "" { // unexported
					continue
				}

				// FIXME(ap): skip fields that have no sql tag
				// to enable to mix structs from various domains (i.e. xml + sql)
				// maybe skip in sqlstruct.Columns()?
				tag := sf.Tag.Get("sql")
				if tag == "-" { // || tag == "" {
					continue
				}
				name, _ := parseTag(tag)
				index := make([]int, len(f.index)+1)
				copy(index, f.index)
				index[len(f.index)] = i

				ft := sf.Type
				if ft.Name() == "" && ft.Kind() == reflect.Ptr {
					// Follow pointer.
					ft = ft.Elem()
				}

				// Record found field and index sequence.
				if name != "" || !sf.Anonymous || ft.Kind() != reflect.Struct {
					tagged := name != ""
					if name == "" {
						name = sf.Name
					}
					fields = append(fields, field{f.typ.Name(), name, sf.Name, tagged, index, ft})
					if count[f.typ] > 1 {
						// If there were multiple instances, add a second,
						// so that the annihilation code will see a duplicate.
						// It only cares about the distinction between 1 or 2,
						// so don't bother generating any more copies.
						fields = append(fields, fields[len(fields)-1])
					}
					continue
				}

				// Record new anonymous struct to explore in next round.
				nextCount[ft]++
				if nextCount[ft] == 1 {
					next = append(next, field{name: ft.Name(), index: index, typ: ft})
				}
			}
		}
	}

	sort.Sort(byName(fields))

	// Remove fields with annihilating name collisions
	// and also fields shadowed by fields with explicit JSON tags.
	name := ""
	out := fields[:0]
	for _, f := range fields {
		if f.name != name {
			name = f.name
			out = append(out, f)
			continue
		}
		if n := len(out); n > 0 && out[n-1].name == name && (!out[n-1].tag || f.tag) {
			out = out[:n-1]
		}
	}
	fields = out

	sort.Sort(byIndex(fields))

	return fields
}
