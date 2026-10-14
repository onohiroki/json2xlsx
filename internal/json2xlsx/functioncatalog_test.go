package json2xlsx

import (
	"sort"
	"testing"
)

func TestFunctionCatalogMatchesKnownFuncs(t *testing.T) {
	// カタログと knownFuncs は 1:1 で対応する．
	if len(functionCatalog) != len(knownFuncs) {
		t.Fatalf("catalog has %d functions, knownFuncs has %d", len(functionCatalog), len(knownFuncs))
	}
	for _, fi := range functionCatalog {
		if !knownFuncs[fi.Name] {
			t.Errorf("catalog function %q is not in knownFuncs", fi.Name)
		}
		if fi.Category == "" {
			t.Errorf("catalog function %q has empty category", fi.Name)
		}
	}
	for name := range knownFuncs {
		if _, ok := catalogByName[name]; !ok {
			t.Errorf("knownFuncs function %q is missing from catalog", name)
		}
	}
}

func TestKnownFunctionsSorted(t *testing.T) {
	funcs := KnownFunctions()
	if len(funcs) == 0 {
		t.Fatal("KnownFunctions returned no entries")
	}
	if !sort.SliceIsSorted(funcs, func(i, j int) bool { return funcs[i].Name < funcs[j].Name }) {
		t.Error("KnownFunctions is not sorted by name")
	}
}

func TestFunctionCategories(t *testing.T) {
	cats := FunctionCategories()
	if len(cats) == 0 {
		t.Fatal("FunctionCategories returned no entries")
	}
	seen := map[string]bool{}
	for _, c := range cats {
		if seen[c] {
			t.Errorf("duplicate category %q", c)
		}
		seen[c] = true
	}
	// カタログで使うカテゴリはすべて一覧に含まれる．
	for _, fi := range functionCatalog {
		if !seen[fi.Category] {
			t.Errorf("category %q used by %q is not in FunctionCategories", fi.Category, fi.Name)
		}
	}
}

func TestFilterFunctions(t *testing.T) {
	t.Run("search substring", func(t *testing.T) {
		got := FilterFunctions("lookup", "")
		if len(got) != 3 {
			t.Fatalf("expected 3 lookup results, got %d: %v", len(got), got)
		}
		for _, fi := range got {
			if fi.Category != CategoryLookup {
				t.Errorf("%s has category %q, want %q", fi.Name, fi.Category, CategoryLookup)
			}
		}
	})

	t.Run("search case-insensitive", func(t *testing.T) {
		got := FilterFunctions("sum", "")
		if len(got) == 0 {
			t.Fatal("expected results for 'sum'")
		}
	})

	t.Run("category filter", func(t *testing.T) {
		got := FilterFunctions("", CategoryDate)
		if len(got) == 0 {
			t.Fatal("expected date functions")
		}
		for _, fi := range got {
			if fi.Category != CategoryDate {
				t.Errorf("%s has category %q, want %q", fi.Name, fi.Category, CategoryDate)
			}
		}
	})

	t.Run("no match returns empty slice", func(t *testing.T) {
		got := FilterFunctions("zzz_no_such_function", "")
		if got == nil {
			t.Fatal("expected non-nil empty slice")
		}
		if len(got) != 0 {
			t.Errorf("expected 0 results, got %d", len(got))
		}
	})

	t.Run("no filter returns all", func(t *testing.T) {
		got := FilterFunctions("", "")
		if len(got) != len(knownFuncs) {
			t.Errorf("got %d functions, want %d", len(got), len(knownFuncs))
		}
	})
}
