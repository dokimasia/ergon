// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package render

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"text/template"
)

// The names of the functions that every template calls.
const (
	// wordsFunc writes a list as words of the shell, escaped for make.
	wordsFunc = "words"

	// makeFunc escapes a value for make.
	makeFunc = "make"

	// yamlFunc writes a scalar, or a list of scalars, of YAML.
	yamlFunc = "yaml"
)

// functions are the functions of every template.
var functions = template.FuncMap{
	wordsFunc: words,
	makeFunc:  escape,
	yamlFunc:  scalar,
}

// plainWord matches a word that a shell takes as it is, without quotes.
var plainWord = regexp.MustCompile(`^[A-Za-z0-9_@%+=:,./^-]+$`)

// plainScalar matches a scalar that YAML reads as a string without quotes, unless it is a word of
// [keywords] or a number that starts with a full stop: words of letters, digits, '_', '.', '/' and
// '-', separated by single spaces, whose first starts with a letter, '_', '.' or '/'.
var plainScalar = regexp.MustCompile(`^[A-Za-z_./][A-Za-z0-9_./-]*( [A-Za-z0-9_./-]+)*$`)

// keywords are the plain scalars that YAML 1.1 or YAML 1.2 reads as a boolean, as null, or as a
// float, in lowercase.
var keywords = []string{"y", "yes", "n", "no", "true", "false", "on", "off", "null", ".inf", ".nan"}

// makeEscapes escapes the characters that make reads in the value of a variable: $ starts a
// reference, and # starts a comment.
var makeEscapes = strings.NewReplacer("$", "$$", "#", `\#`)

// words returns the elements of list, a slice or an array of strings, as the words of a command of
// the shell in the value of a variable of make, separated by spaces. A word of letters, digits and
// the characters _ @ % + = : , . / ^ - is as it is, and any other word is in single quotes, in which
// each single quote of the word closes the quotes, follows escaped by a backslash, and opens them
// again. Each word has $ and # escaped for make. It returns an error that wraps
// [ErrInvalidTemplate] for a list that is not a slice or an array of strings.
func words(list any) (string, error) {
	v := reflect.ValueOf(list)
	if (v.Kind() != reflect.Slice && v.Kind() != reflect.Array) || v.Type().Elem().Kind() != reflect.String {
		return "", fmt.Errorf("%w: words of a %T", ErrInvalidTemplate, list)
	}
	quoted := make([]string, 0, v.Len())
	for i := range v.Len() {
		w := v.Index(i).String()
		if !plainWord.MatchString(w) {
			w = "'" + strings.ReplaceAll(w, "'", `'\''`) + "'"
		}
		quoted = append(quoted, makeEscapes.Replace(w))
	}
	return strings.Join(quoted, " "), nil
}

// escape returns value, a string, with $ and # escaped for the value of a variable of make. It
// returns an error that wraps [ErrInvalidTemplate] for a value that is not a string.
func escape(value any) (string, error) {
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.String {
		return "", fmt.Errorf("%w: make of a %T", ErrInvalidTemplate, value)
	}
	return makeEscapes.Replace(v.String()), nil
}

// scalar returns value as YAML that reads back as the same value: a boolean as true or false, an
// integer in decimal, a string as it is when YAML 1.1 and YAML 1.2 read it as that string, and
// otherwise in double quotes with the escapes of Go, which YAML accepts, and a slice or an array of
// these as a sequence in flow style. It returns an error that wraps [ErrInvalidTemplate] for a value
// of any other kind.
func scalar(value any) (string, error) {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Bool:
		return strconv.FormatBool(v.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10), nil
	case reflect.String:
		s := v.String()
		lower := strings.ToLower(s)
		number := len(s) > 1 && s[0] == '.' && '0' <= s[1] && s[1] <= '9'
		if plainScalar.MatchString(s) && !slices.Contains(keywords, lower) && !number {
			return s, nil
		}
		return strconv.Quote(s), nil
	case reflect.Slice, reflect.Array:
		items := make([]string, 0, v.Len())
		for i := range v.Len() {
			item, err := scalar(v.Index(i).Interface())
			if err != nil {
				return "", err
			}
			items = append(items, item)
		}
		return "[" + strings.Join(items, ", ") + "]", nil
	default:
		return "", fmt.Errorf("%w: yaml of a %T", ErrInvalidTemplate, value)
	}
}
