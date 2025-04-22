package adminapi

// todo have proper values and more fitting types instead of any

type Filters map[string]any
type filter map[string]any

type value interface {
	int | string | bool
}
type valueOrFilter interface {
	value | filter
}

// list of all valid functions with lowercased key
var allFilters = map[string]string{
	"any":                 "Any",
	"all":                 "All",
	"containedby":         "ContainedBy",
	"containedonlyby":     "ContainedOnlyBy",
	"contains":            "Contains",
	"empty":               "Empty",
	"greaterthan":         "GreaterThan",
	"greaterthanorequals": "GreaterThanOrEquals",
	"lessthan":            "LessThan",
	"lessthanorequals":    "LessThanOrEquals",
	"not":                 "Not",
	"overlaps":            "Overlaps",
	"regexp":              "Regexp",
	"startswith":          "StartsWith",
}

func Regexp(value string) filter {
	return createFilter("Regexp", value)
}

func Not[V valueOrFilter](filter V) filter {
	return createFilter("Not", filter)
}

func Any[V valueOrFilter](values ...V) filter {
	return createFilter("Any", values)
}

func All[V valueOrFilter](values ...V) filter {
	return createFilter("All", values)
}

func Empty() filter {
	return createFilter("Empty", nil)
}

func createFilter(filterType string, value any) filter {
	return filter{
		filterType: value,
	}
}
