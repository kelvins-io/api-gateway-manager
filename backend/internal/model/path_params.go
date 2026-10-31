package model

import (
	"fmt"
	"regexp"
	"strings"
)

// openAPIPathParam matches OpenAPI/Swagger path templates such as {id} or {petId}.
// Capture group 1 is the parameter name.
var openAPIPathParam = regexp.MustCompile(`\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// malformedPathParam catches braces that are not a valid {name} token.
var malformedPathParam = regexp.MustCompile(`\{|\}`)

// ExtractPathParams returns unique path parameter names in order of first appearance.
func ExtractPathParams(path string) []string {
	matches := openAPIPathParam.FindAllStringSubmatch(path, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]string, 0, len(matches))
	seen := map[string]struct{}{}
	for _, m := range matches {
		name := m[1]
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

// HasPathParams reports whether path contains any {name} template.
func HasPathParams(path string) bool {
	return openAPIPathParam.MatchString(path)
}

// ValidatePathTemplate ensures braces form valid {name} tokens only.
func ValidatePathTemplate(path string) error {
	without := openAPIPathParam.ReplaceAllString(path, "")
	if malformedPathParam.MatchString(without) {
		return fmt.Errorf("path %q has invalid {param} syntax; use names like {id} or {petId}", path)
	}
	return nil
}

// PathParamSetsEqual reports whether two param name lists contain the same set.
func PathParamSetsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := map[string]struct{}{}
	for _, n := range a {
		set[n] = struct{}{}
	}
	for _, n := range b {
		if _, ok := set[n]; !ok {
			return false
		}
	}
	return true
}

// ValidateAccessAndServicePathParams checks that when access paths use {xxx},
// every access path shares the same param set and service_path contains the same set.
func ValidateAccessAndServicePathParams(accessPaths []string, servicePath string) error {
	for _, p := range accessPaths {
		if err := ValidatePathTemplate(p); err != nil {
			return err
		}
	}
	if err := ValidatePathTemplate(servicePath); err != nil {
		return err
	}

	var expected []string
	hasAny := false
	for _, p := range accessPaths {
		params := ExtractPathParams(p)
		if len(params) == 0 {
			if hasAny {
				return fmt.Errorf("when some access paths use {param}, every access path must include the same {param} names")
			}
			continue
		}
		hasAny = true
		if expected == nil {
			expected = params
			continue
		}
		if !PathParamSetsEqual(expected, params) {
			return fmt.Errorf("all access paths with {param} must use the same parameter names")
		}
	}

	serviceParams := ExtractPathParams(servicePath)
	if !hasAny {
		if len(serviceParams) > 0 {
			return fmt.Errorf("service_path has {param} but access_path does not; keep them in sync")
		}
		return nil
	}
	if !PathParamSetsEqual(expected, serviceParams) {
		return fmt.Errorf("access_path has {%s} so service_path must contain the same {param} names (got service_path %q)",
			strings.Join(expected, "},{"), servicePath)
	}
	return nil
}

// ToURICaptureTemplate rewrites {name} to Kong request-transformer placeholders.
// Example: /users/{id} -> /users/$(uri_captures["id"])
func ToURICaptureTemplate(path string) string {
	return openAPIPathParam.ReplaceAllStringFunc(path, func(m string) string {
		name := m[1 : len(m)-1]
		return `$(uri_captures["` + name + `"])`
	})
}

// ToKongRegexBody converts OpenAPI {name} segments into named capture groups and
// escapes other regex metacharacters (keeps '*' as a Kong wildcard).
func ToKongRegexBody(path string) string {
	var b strings.Builder
	last := 0
	for _, loc := range openAPIPathParam.FindAllStringSubmatchIndex(path, -1) {
		b.WriteString(escapeKongRegexLiteral(path[last:loc[0]]))
		name := path[loc[2]:loc[3]]
		b.WriteString("(?<" + name + ">[^/]+)")
		last = loc[1]
	}
	b.WriteString(escapeKongRegexLiteral(path[last:]))
	return b.String()
}

func escapeKongRegexLiteral(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '.', '+', '?', '(', ')', '[', ']', '{', '}', '|', '^', '$', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// JoinServiceAccessPath joins a fixed service base path with an access path template.
// Used by OpenAPI import when access paths contain {param}.
func JoinServiceAccessPath(base, access string) string {
	base = strings.TrimSpace(base)
	access = strings.TrimSpace(access)
	if access == "" {
		if base == "" {
			return "/"
		}
		return base
	}
	if !strings.HasPrefix(access, "/") {
		access = "/" + access
	}
	base = strings.TrimRight(base, "/")
	if base == "" || base == "/" {
		return access
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	return base + access
}
