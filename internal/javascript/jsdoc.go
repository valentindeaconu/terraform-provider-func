package javascript

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

var (
	jsdocRegEx            = regexp.MustCompile(`\/\*\*((?:.|\n)*?)\*\/\n(?:\$\()?(.*)`)
	jsdocBeginRegEx       = regexp.MustCompile(`^\s?\*\s?`)
	jsdocTagRegEx         = regexp.MustCompile(`^@(\w+)`)
	jsdocTypeRegEx        = regexp.MustCompile(`\{(.*)\}`)
	jsdocParamNameRegEx   = regexp.MustCompile(`\}\s+(\w+)\s+\-`)
	jsdocDescriptionRegEx = regexp.MustCompile(`([^}]+)$`)
	wsRegEx               = regexp.MustCompile(`\s+`)
)

// javaScriptArgumentMetadata holds metadata for a JavaScript argument.
type javaScriptArgumentMetadata struct {
	name        string
	typ         string
	description string
}

// javaScriptReturnMetadata holds metadata for a JavaScript return.
type javaScriptReturnMetadata struct {
	typ         string
	description string
}

// JavaScriptFunctionMetadata holds metadata for a JavaScript function.
type JavaScriptFunctionMetadata struct {
	summary     string
	description string
	params      []*javaScriptArgumentMetadata
	returns     *javaScriptReturnMetadata
}

// parseScriptJSDoc parses JSDoc from a JavaScript script file.
func parseScriptJSDoc(src string) (map[string]*JavaScriptFunctionMetadata, error) {
	matches := jsdocRegEx.FindAllStringSubmatch(src, -1)

	res := make(map[string]*JavaScriptFunctionMetadata, len(matches))

	for _, match := range matches {
		jsdoc := match[1]
		fnSignature := match[2]

		fnHash := removeWhitespaceFromString(fnSignature)

		md, err := parseJSDoc(jsdoc)
		if err != nil {
			return nil, err
		}

		res[fnHash] = md
	}

	return res, nil
}

// parseJSDoc parses a JSDoc string.
func parseJSDoc(doc string) (*JavaScriptFunctionMetadata, error) {
	lines := strings.Split(doc, "\n")

	var buf bytes.Buffer

	params := make([]*javaScriptArgumentMetadata, 0)
	var returns *javaScriptReturnMetadata = nil

	for _, line := range lines {
		// Replace "*" and adjacent whitespace from the beginning of the line
		line = jsdocBeginRegEx.ReplaceAllString(line, "")

		// Remove other whitespace
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check for tags
		if strings.HasPrefix(line, "@") {
			tag, line := regExFindAndDelete(jsdocTagRegEx, line, "")

			switch tag {
			case "param":
				var (
					paramName        string
					paramDescription string
					paramType        string
				)
				paramName, line = regExFindAndDelete(jsdocParamNameRegEx, line, "}")
				paramDescription, line = regExFindAndDelete(jsdocDescriptionRegEx, line, "")
				paramType, _ = regExFindAndDelete(jsdocTypeRegEx, line, "")

				params = append(params, &javaScriptArgumentMetadata{
					name:        paramName,
					typ:         paramType,
					description: paramDescription,
				})
			case "returns":
				var (
					returnDescription string
					returnType        string
				)
				returnDescription, line = regExFindAndDelete(jsdocDescriptionRegEx, line, "")
				returnType, _ = regExFindAndDelete(jsdocTypeRegEx, line, "")

				returns = &javaScriptReturnMetadata{
					typ:         returnType,
					description: returnDescription,
				}
			default:
				return nil, fmt.Errorf("unknown tag: %s", tag)
			}
		} else {
			// Everything else goes into the description buffer
			buf.WriteString(line)
			buf.WriteRune('\n')
		}
	}

	allDescription := buf.String()

	// First line of the description is the summary, everything else is
	// the description itself.
	parts := strings.SplitN(allDescription, "\n", 2)

	var (
		summary     string = ""
		description string = ""
	)

	if len(parts) == 1 {
		summary = parts[0]
	} else if len(parts) == 2 {
		summary = parts[0]
		description = parts[1]
	}

	return &JavaScriptFunctionMetadata{
		summary:     summary,
		description: description,
		params:      params,
		returns:     returns,
	}, nil
}

// regExFindAndDelete find the match of regex in a given string
// then removes the match from the string.
func regExFindAndDelete(re *regexp.Regexp, s string, putback string) (string, string) {
	match := re.FindStringSubmatch(s)
	return strings.TrimSpace(match[1]), re.ReplaceAllString(s, putback)
}

// removeWhitespaceFromString removes any whitespace character from
// a given string keeping everything else as it is.
//
// Example: "this is an example" => "thisisanexample".
func removeWhitespaceFromString(s string) string {
	return wsRegEx.ReplaceAllString(s, "")
}
