package mux

import (
	"bytes"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	coap "github.com/dustin/go-coap"
)

func braceIndices(s string) ([]int, error) {
	var level, idx int
	idxs := make([]int, 0)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{':
			if level++; level == 1 {
				idx = i
			}
		case '}':
			if level--; level == 0 {
				idxs = append(idxs, idx, i-1)
			} else if level < 0 {
				return nil, fmt.Errorf("mux: unbalanced braces in %q", s)
			}
		}
	}
	if level != 0 {
		return nil, fmt.Errorf("mux: unbalanced braces in %q", s)
	}
	return idxs, nil
}

// routeRegexp stores a regexp to match a host or path and information to
// collect and validate route variables.
type routeRegexp struct {
	// The unmodified template.
	template string
	// Expanded regexp.
	regexp *regexp.Regexp
	// Reverse template.
	reverse string
	// Variable names.
	varsN []string
	// Variable regexps (validators).
	varsR []*regexp.Regexp
}

// varGroupName builds a capturing group name for the indexed variable.
func varGroupName(idx int) string {
	return "v" + strconv.Itoa(idx)
}

// newRouteRegexp parses a route template and returns a routeRegexp,
// used to match a host, a path or a query string.
//
// It will extract named variables, assemble a regexp to be matched, create
// a "reverse" template to build URLs and compile regexps to validate variable
// values used in URL building.
//
// Previously we accepted only Python-like identifiers for variable
// names ([a-zA-Z_][a-zA-Z0-9_]*), but currently the only restriction is that
// name and pattern can't be empty, and names can't contain a colon.
func newRouteRegexp(tpl string) (*routeRegexp, error) {
	// Check if it is well-formed.
	idxs, errBraces := braceIndices(tpl)
	if errBraces != nil {
		return nil, errBraces
	}
	// Backup the original.
	template := tpl
	// Now let's parse it.
	defaultPattern := "[^/]+"

	endSlash := false
	// Strip slashes from pattern, since coap messages won't have the either at the
	// beginning
	for tpl[0] == '/' {
		tpl = tpl[1:]
	}
	varsN := make([]string, len(idxs)/2)
	varsR := make([]*regexp.Regexp, len(idxs)/2)
	pattern := bytes.NewBufferString("")
	pattern.WriteByte('^')
	reverse := bytes.NewBufferString("")
	var end int
	var start int
	var err error
	end = -1
	for i := 0; i < len(idxs); i += 2 {
		// Set all values we are interested in.
		raw := tpl[end+1 : idxs[i]-1]
		end = idxs[i+1]
		start = idxs[i]
		parts := strings.SplitN(tpl[start:end], ":", 2)
		name := parts[0]
		patt := defaultPattern
		if len(parts) == 2 {
			patt = parts[1]
		}
		// Name or pattern can't be empty.
		if name == "" || patt == "" {
			return nil, fmt.Errorf("mux: missing name or pattern in %q",
				tpl[start:end])
		}
		// Build the regexp pattern.
		varIdx := i / 2
		fmt.Fprintf(pattern, "%s(?P<%s>%s)", regexp.QuoteMeta(raw), varGroupName(varIdx), patt)
		// Build the reverse template.
		fmt.Fprintf(reverse, "%s%%s", raw)

		// Append variable name and compiled pattern.
		varsN[varIdx] = name
		varsR[varIdx], err = regexp.Compile(fmt.Sprintf("^%s$", patt))
		if err != nil {
			return nil, err
		}
	}
	// Add the remaining.
	raw := tpl[end+1:]
	pattern.WriteString(regexp.QuoteMeta(raw))
	pattern.WriteByte('$')
	reverse.WriteString(raw)
	if endSlash {
		reverse.WriteByte('/')
	}
	// Compile full regexp.
	reg, errCompile := regexp.Compile(pattern.String())
	if errCompile != nil {
		return nil, errCompile
	}
	// Done!
	return &routeRegexp{
		template: template,
		regexp:   reg,
		reverse:  reverse.String(),
		varsN:    varsN,
		varsR:    varsR,
	}, nil
}

// Match matches the regexp against the URL host or path.
func (r *routeRegexp) Match(msg *coap.Message, addr *net.UDPAddr) bool {
	return r.regexp.MatchString(msg.PathString())
}

// url builds a URL part using the given values.
func (r *routeRegexp) url(values map[string]string) (string, error) {
	urlValues := make([]interface{}, len(r.varsN))
	for k, v := range r.varsN {
		value, ok := values[v]
		if !ok {
			return "", fmt.Errorf("mux: missing route variable %q", v)
		}
		urlValues[k] = value
	}
	rv := fmt.Sprintf(r.reverse, urlValues...)
	if !r.regexp.MatchString(rv) {
		// The URL is checked against the full regexp, instead of checking
		// individual variables. This is faster but to provide a good error
		// message, we check individual regexps if the URL doesn't match.
		for k, v := range r.varsN {
			if !r.varsR[k].MatchString(values[v]) {
				return "", fmt.Errorf(
					"mux: variable %q doesn't match, expected %q", values[v],
					r.varsR[k].String())
			}
		}
	}
	return rv, nil
}

// routeRegexpGroup groups the route matchers that carry variables.
type routeRegexpGroup struct {
	path *routeRegexp
}

// setMatch extracts the variables from the URL once a route matches.
func (v *routeRegexp) setMatch(msg *coap.Message, m *RouteMatch, r *Route) {
	// Store path variables.
	pathVars := v.regexp.FindStringSubmatch(msg.PathString())
	if pathVars != nil {
		subexpNames := v.regexp.SubexpNames()
		varName := 0
		for i, name := range subexpNames[1:] {
			if name != "" && name == varGroupName(varName) {
				m.Vars[v.varsN[varName]] = pathVars[i+1]
				varName++
			}
		}

	}
}
