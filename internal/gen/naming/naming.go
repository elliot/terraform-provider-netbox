// Package naming converts between the identifier conventions of the NetBox
// OpenAPI document, the openapi-generator Go client and Terraform.
package naming

import (
	"strings"
	"unicode"
)

var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true, "default": true,
	"defer": true, "else": true, "fallthrough": true, "for": true, "func": true, "go": true, "goto": true,
	"if": true, "import": true, "interface": true, "map": true, "package": true, "range": true,
	"return": true, "select": true, "struct": true, "switch": true, "type": true, "var": true,
}

// GoField converts an OpenAPI property name to the exported Go field name that
// openapi-generator 7.x emits: leading underscores are stripped, the name is
// split on "_", and every segment is capitalised with the remainder lowered
// ("primary_ip4" -> "PrimaryIp4", "_depth" -> "Depth", "a_terminations" -> "ATerminations").
func GoField(prop string) string {
	prop = strings.TrimLeft(prop, "_")
	var b strings.Builder
	for _, seg := range strings.Split(prop, "_") {
		if seg == "" {
			continue
		}
		b.WriteString(strings.ToUpper(seg[:1]))
		b.WriteString(strings.ToLower(seg[1:]))
	}
	return b.String()
}

// GoMethod converts an operationId to the generated Go method name
// ("dcim_sites_partial_update" -> "DcimSitesPartialUpdate").
func GoMethod(operationID string) string { return GoField(operationID) }

// GoParam is the constructor parameter name for a property (lower camel, with a
// trailing underscore for Go keywords: "type" -> "type_").
func GoParam(prop string) string {
	f := GoField(prop)
	if f == "" {
		return f
	}
	p := strings.ToLower(f[:1]) + f[1:]
	if goKeywords[p] {
		p += "_"
	}
	return p
}

// GoType converts a component schema name to the generated Go type name:
// segments split on "_" are capitalised but otherwise keep their case
// ("IPAddress" -> "IPAddress", "BriefX_Request" -> "BriefXRequest").
func GoType(schema string) string {
	var b strings.Builder
	for _, seg := range strings.Split(schema, "_") {
		if seg == "" {
			continue
		}
		b.WriteString(strings.ToUpper(seg[:1]))
		b.WriteString(seg[1:])
	}
	return b.String()
}

// GoIdent converts a snake_case Terraform name to an exported Go identifier
// with conventional acronym casing preserved as given ("ip_address" -> "IpAddress").
func GoIdent(snake string) string { return GoField(snake) }

// Snake converts a path segment or CamelCase name to snake_case
// ("ip-addresses" -> "ip_addresses", "IPAddress" -> "ip_address").
func Snake(s string) string {
	s = strings.ReplaceAll(s, "-", "_")
	if !strings.ContainsAny(s, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return s
	}
	var out []rune
	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 && runes[i-1] != '_' && (unicode.IsLower(runes[i-1]) || unicode.IsDigit(runes[i-1]) ||
				(i+1 < len(runes) && unicode.IsLower(runes[i+1]) && unicode.IsUpper(runes[i-1]))) {
				out = append(out, '_')
			}
			out = append(out, unicode.ToLower(r))
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}

// Singular converts a plural snake_case collection name to its singular form
// using the small rule set that suffices for NetBox's API path segments.
func Singular(plural string) string {
	parts := strings.Split(plural, "_")
	last := parts[len(parts)-1]
	switch {
	case last == "chassis":
		// uncountable
	case strings.HasSuffix(last, "ies"):
		last = strings.TrimSuffix(last, "ies") + "y"
	case strings.HasSuffix(last, "sses"), strings.HasSuffix(last, "xes"), strings.HasSuffix(last, "ches"), strings.HasSuffix(last, "shes"):
		last = strings.TrimSuffix(last, "es")
	case strings.HasSuffix(last, "s"):
		last = strings.TrimSuffix(last, "s")
	}
	parts[len(parts)-1] = last
	return strings.Join(parts, "_")
}

// Camel converts snake_case to lowerCamelCase.
func Camel(snake string) string {
	f := GoField(snake)
	if f == "" {
		return f
	}
	return strings.ToLower(f[:1]) + f[1:]
}

// Title returns a human title from snake_case ("ip_address" -> "Ip Address").
func Title(snake string) string {
	parts := strings.Split(snake, "_")
	for i, p := range parts {
		if p != "" {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}
