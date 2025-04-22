package adminapi

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// ParseQuery parses a string query (e.g. "hostname=11111") and returns a Filters map.
//
// Example forms:
//
//	"hostname=11111"                               => map: {"hostname": 11111}
//	"hostname=regexp(foo.*) game_world=any(1 2 3)" => map: {"hostname": {"Regexp": "foo.*"}, "game_world": {"Any": [1, 2, 3]}}
//	"hostname=Not(Empty())"                        => map: {"hostname": {"Not": {"Empty": nil}}}
func ParseQuery(query string) (Filters, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query must not be empty")
	}
	parts, err := splitPairs(query)
	if err != nil {
		return nil, err
	}

	filters := make(Filters)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		keyVal := strings.SplitN(part, "=", 2)
		if len(keyVal) != 2 || strings.TrimSpace(keyVal[0]) == "" {
			return nil, fmt.Errorf("invalid expression: %s", part)
		}
		key := strings.TrimSpace(keyVal[0])
		valStr := strings.TrimSpace(keyVal[1])

		val, err := parseValue(valStr)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", part, err)
		}
		filters[key] = val
	}
	return filters, nil
}

// splitPairs splits a string into key=value chunks at spaces, but never inside nested parens or quotes
func splitPairs(s string) ([]string, error) {
	var res []string
	start := 0
	depth := 0
	inQuotes := rune(0)
	for i, r := range s {
		switch {
		case (r == '\'' || r == '"') && (i == 0 || s[i-1] != '\\'):
			switch inQuotes {
			case 0:
				inQuotes = r
			case r:
				inQuotes = 0
			}
		case inQuotes != 0:
			// do nothing
		case r == '(':
			depth++
		case r == ')':
			depth--
			if depth < 0 {
				return nil, fmt.Errorf("unmatched ) found")
			}
		case unicode.IsSpace(r) && depth == 0:
			if start < i {
				res = append(res, s[start:i])
			}
			start = i + 1
		}
	}
	if depth != 0 {
		return nil, fmt.Errorf("unmatched ( found")
	}
	if start < len(s) {
		res = append(res, s[start:])
	}
	return res, nil
}

// parseValue parses any individual left-hand side after the '='. It handles integers,
// floats, booleans, quoted strings, and function-based filters like Regexp(...).
func parseValue(s string) (any, error) {
	s = strings.TrimSpace(s)
	// Recognize quoted strings
	if l := len(s); l >= 2 && ((s[0] == '"' && s[l-1] == '"') || (s[0] == '\'' && s[l-1] == '\'')) {
		return s[1 : l-1], nil
	}

	// Try int
	if i, err := strconv.Atoi(s); err == nil {
		return i, nil
	}

	// Try float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, nil
	}

	// Try bool
	switch s {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}

	// Detect function filters: functionName(...)
	idx := strings.Index(s, "(")
	if idx > 0 && s[len(s)-1] == ')' {
		fn := s[:idx]
		argsBody := s[idx+1 : len(s)-1]

		// Convert "ReGEXP" -> "regexp" for the lookup
		fnLower := strings.ToLower(strings.TrimSpace(fn))
		canonicalFn, ok := allFilters[fnLower]
		if !ok {
			return nil, fmt.Errorf("invalid filter function: %s", fn)
		}

		// Parse arguments (handle empty arg like Empty())
		//goland:noinspection GoPreferNilSlice
		argVals := []any{}
		if strings.TrimSpace(argsBody) != "" {
			argParts, err := splitPairs(argsBody)
			if err != nil {
				return nil, err
			}
			for _, ap := range argParts {
				val, err := parseValue(ap)
				if err != nil {
					return nil, err
				}
				argVals = append(argVals, val)
			}
		}

		// Single arg: Not(x), Regexp(y), etc.
		if len(argVals) == 1 {
			return filter{canonicalFn: argVals[0]}, nil
		}
		// Multi arg: Any(1 2 3), All(a b c)
		return filter{canonicalFn: argVals}, nil
	}

	// If not a filter, treat as simple string
	return s, nil
}
