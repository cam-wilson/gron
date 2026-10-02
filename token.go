package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

// A token is a chunk of text from a statement with a type
type token struct {
	text string
	typ  tokenTyp
}

// A tokenTyp identifies what kind of token something is
type tokenTyp int

const (
	// A bare word is a unquoted key; like 'foo' in json.foo = 1;
	typBare tokenTyp = iota

	// Numeric key; like '2' in json[2] = "foo";
	typNumericKey

	// A quoted key; like 'foo bar' in json["foo bar"] = 2;
	typQuotedKey

	// Punctuation types
	typDot    // .
	typLBrace // [
	typRBrace // ]
	typEquals // =
	typSemi   // ;
	typComma  // ,

	// Value types
	typString      // "foo"
	typNumber      // 4
	typTrue        // true
	typFalse       // false
	typNull        // null
	typEmptyArray  // []
	typEmptyObject // {}

	// Ignored token
	typIgnored

	// Error token
	typError
)

// a sprintFn adds color to its input
type sprintFn func(...interface{}) string

// mapping of token types to the appropriate color sprintFn
var sprintFns = newSprintFns()

func newSprintFns() map[tokenTyp]sprintFn {
	return map[tokenTyp]sprintFn{
		typBare:        bareColor.SprintFunc(),
		typNumericKey:  numColor.SprintFunc(),
		typQuotedKey:   strColor.SprintFunc(),
		typLBrace:      braceColor.SprintFunc(),
		typRBrace:      braceColor.SprintFunc(),
		typString:      strColor.SprintFunc(),
		typNumber:      numColor.SprintFunc(),
		typTrue:        boolColor.SprintFunc(),
		typFalse:       boolColor.SprintFunc(),
		typNull:        boolColor.SprintFunc(),
		typEmptyArray:  braceColor.SprintFunc(),
		typEmptyObject: braceColor.SprintFunc(),
	}
}

func refreshSprintFns() {
	sprintFns = newSprintFns()
}

func setTokenColors(enabled bool) {
	for _, c := range []interface {
		EnableColor()
		DisableColor()
	}{strColor, braceColor, bareColor, numColor, boolColor} {
		if enabled {
			c.EnableColor()
			continue
		}
		c.DisableColor()
	}
	refreshSprintFns()
}

// isValue returns true if the token is a valid value type
func (t token) isValue() bool {
	switch t.typ {
	case typString, typNumber, typTrue, typFalse, typNull, typEmptyArray, typEmptyObject:
		return true
	default:
		return false
	}
}

// isPunct returns true if the token is a punctuation type
func (t token) isPunct() bool {
	switch t.typ {
	case typDot, typLBrace, typRBrace, typEquals, typSemi, typComma:
		return true
	default:
		return false
	}
}

// format returns the formatted version of the token text
func (t token) format() string {
	if t.typ == typEquals {
		return " " + t.text + " "
	}
	return t.text
}

// formatColor returns the colored formatted version of the token text
func (t token) formatColor() string {
	text := t.text
	if t.typ == typEquals {
		text = " " + text + " "
	}
	if t.typ == typString {
		text = terminalSafeString(text)
	}
	fn, ok := sprintFns[t.typ]
	if ok {
		return fn(text)
	}
	return text

}

func terminalSafeString(text string) string {
	parts := strings.SplitAfter(text, `\u001B`)
	if len(parts) == 1 {
		return text
	}
	for i := 1; i < len(parts); i++ {
		if isTerminalControl(parts[i]) {
			continue
		}
		parts[i] = strings.ToLower(parts[i])
	}
	return strings.Join(parts, "")
}

func isTerminalControl(text string) bool {
	if len(text) < 2 || text[0] != '[' {
		return false
	}
	return unicode.IsDigit(rune(text[1])) || text[2] == ';'
}

// valueTokenFromInterface takes any valid value and
// returns a value token to represent it
func valueTokenFromInterface(v interface{}) token {
	switch vv := v.(type) {

	case map[string]interface{}:
		return token{"{}", typEmptyObject}
	case []interface{}:
		return token{"[]", typEmptyArray}
	case json.Number:
		return token{vv.String(), typNumber}
	case string:
		return token{quoteString(vv), typString}
	case bool:
		if vv {
			return token{"true", typTrue}
		}
		return token{"false", typFalse}
	case nil:
		return token{"null", typNull}
	default:
		return token{"", typError}
	}
}

// quoteString takes a string and returns a quoted and
// escaped string valid for use in gron output
func quoteString(s string) string {

	out := &bytes.Buffer{}
	// bytes.Buffer never returns errors on these methods.
	// errors are explicitly ignored to keep the linter
	// happy. A price worth paying so that the linter
	// remains useful.
	_ = out.WriteByte('"')

	for _, r := range s {
		switch r {
		case '\\':
			_, _ = out.WriteString(`\\`)
		case '"':
			_, _ = out.WriteString(`\"`)
		case '\b':
			_, _ = out.WriteString(`\b`)
		case '\f':
			_, _ = out.WriteString(`\f`)
		case '\n':
			_, _ = out.WriteString(`\n`)
		case '\r':
			_, _ = out.WriteString(`\r`)
		case '\t':
			_, _ = out.WriteString(`\t`)
		// \u2028 and \u2029 are separator runes that are not valid
		// in javascript strings so they must be escaped.
		// See http://timelessrepo.com/json-isnt-a-javascript-subset
		case '\u2028':
			_, _ = out.WriteString(`\u2028`)
		case '\u2029':
			_, _ = out.WriteString(`\u2029`)
		default:
			// Any other control runes must be escaped
			if unicode.IsControl(r) {
				_, _ = fmt.Fprintf(out, `\u%04X`, r)
			} else {
				// Unescaped rune
				_, _ = out.WriteRune(r)
			}
		}
	}

	_ = out.WriteByte('"')
	return out.String()

}
