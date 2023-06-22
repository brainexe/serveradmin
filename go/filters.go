package main

// todo have proper values and more fitting types instead of any
type value interface {
	int | string | bool
}
type valueOrFilter interface {
	value | filter
}

type filter map[string]any

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
