// Copyright 2006-2019 WebPKI.org (http://webpki.org).
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jsoncanonicalizer

import (
	"container/list"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
)

type nameValueType struct {
	name    string
	sortKey []uint16
	value   string
}

var asciiEscapes = []byte{'\\', '"', 'b', 'f', 'n', 'r', 't'}
var binaryEscapes = []byte{'\\', '"', '\b', '\f', '\n', '\r', '\t'}
var literals = []string{"true", "false", "null"}

//nolint:gocognit,funlen // Vendored upstream RFC 8785 canonicalizer; keep snapshot structurally intact for provenance.
func Transform(jsonData []byte) (result []byte, e error) {
	jsonDataLength := len(jsonData)
	index := 0

	var parseElement func() string
	var parseSimpleType func() string
	var parseQuotedString func() string
	var parseObject func() string
	var parseArray func() string

	var globalError error

	checkError := func(err error) {
		if globalError == nil {
			globalError = err
		}
	}

	setError := func(msg string) {
		checkError(errors.New(msg))
	}

	isWhiteSpace := func(c byte) bool {
		return c == 0x20 || c == 0x0a || c == 0x0d || c == 0x09
	}

	nextChar := func() byte {
		if index < jsonDataLength {
			c := jsonData[index]
			if c > 0x7f {
				setError("Unexpected non-ASCII character")
			}
			index++
			return c
		}
		setError("Unexpected EOF reached")
		return '"'
	}

	scan := func() byte {
		for {
			c := nextChar()
			if isWhiteSpace(c) {
				continue
			}
			return c
		}
	}

	scanFor := func(expected byte) {
		c := scan()
		if c != expected {
			setError("Expected '" + string(expected) + "' but got '" + string(c) + "'")
		}
	}

	getUEscape := func() rune {
		start := index
		nextChar()
		nextChar()
		nextChar()
		nextChar()
		if globalError != nil {
			return 0
		}
		u16, err := strconv.ParseUint(string(jsonData[start:index]), 16, 16)
		checkError(err)
		if u16 > 0xffff {
			setError("Invalid Unicode escape range")
			return 0
		}
		return rune(u16)
	}

	testNextNonWhiteSpaceChar := func() byte {
		save := index
		c := scan()
		index = save
		return c
	}

	decorateString := func(rawUTF8 string) string {
		var quotedString strings.Builder
		quotedString.WriteByte('"')
	coreLoop:
		for _, c := range []byte(rawUTF8) {
			for i, esc := range binaryEscapes {
				if esc == c {
					quotedString.WriteByte('\\')
					quotedString.WriteByte(asciiEscapes[i])
					continue coreLoop
				}
			}
			if c < 0x20 {
				quotedString.WriteString(fmt.Sprintf("\\u%04x", c))
			} else {
				quotedString.WriteByte(c)
			}
		}
		quotedString.WriteByte('"')
		return quotedString.String()
	}

	parseQuotedString = func() string {
		var rawString strings.Builder
	coreLoop:
		for globalError == nil {
			var c byte
			if index < jsonDataLength {
				c = jsonData[index]
				index++
			} else {
				nextChar()
				break
			}
			if c == '"' {
				break
			}
			if c < ' ' {
				setError("Unterminated string literal")
			} else if c == '\\' {
				c = nextChar()
				if c == 'u' {
					firstUTF16 := getUEscape()
					if utf16.IsSurrogate(firstUTF16) {
						if nextChar() != '\\' || nextChar() != 'u' {
							setError("Missing surrogate")
						} else {
							rawString.WriteRune(utf16.DecodeRune(firstUTF16, getUEscape()))
						}
					} else {
						rawString.WriteRune(firstUTF16)
					}
				} else if c == '/' {
					rawString.WriteByte('/')
				} else {
					for i, esc := range asciiEscapes {
						if esc == c {
							rawString.WriteByte(binaryEscapes[i])
							continue coreLoop
						}
					}
					setError("Unexpected escape: \\" + string(c))
				}
			} else {
				rawString.WriteByte(c)
			}
		}
		return rawString.String()
	}

	parseSimpleType = func() string {
		var token strings.Builder
		index--
		for globalError == nil {
			c := testNextNonWhiteSpaceChar()
			if c == ',' || c == ']' || c == '}' {
				break
			}
			c = nextChar()
			if isWhiteSpace(c) {
				break
			}
			token.WriteByte(c)
		}
		if token.Len() == 0 {
			setError("Missing argument")
		}
		value := token.String()
		for _, literal := range literals {
			if literal == value {
				return literal
			}
		}
		ieeeF64, err := strconv.ParseFloat(value, 64)
		checkError(err)
		value, err = NumberToJSON(ieeeF64)
		checkError(err)
		return value
	}

	parseElement = func() string {
		switch scan() {
		case '{':
			return parseObject()
		case '"':
			return decorateString(parseQuotedString())
		case '[':
			return parseArray()
		default:
			return parseSimpleType()
		}
	}

	parseArray = func() string {
		var arrayData strings.Builder
		arrayData.WriteByte('[')
		next := false
		for globalError == nil && testNextNonWhiteSpaceChar() != ']' {
			if next {
				scanFor(',')
				arrayData.WriteByte(',')
			} else {
				next = true
			}
			arrayData.WriteString(parseElement())
		}
		scan()
		arrayData.WriteByte(']')
		return arrayData.String()
	}

	lexicographicallyPrecedes := func(sortKey []uint16, e *list.Element) bool {
		oldSortKey := e.Value.(nameValueType).sortKey
		minLength := len(oldSortKey)
		if minLength > len(sortKey) {
			minLength = len(sortKey)
		}
		for q := 0; q < minLength; q++ {
			diff := int(sortKey[q]) - int(oldSortKey[q])
			if diff < 0 {
				return true
			}
			if diff > 0 {
				return false
			}
		}
		if len(sortKey) < len(oldSortKey) {
			return true
		}
		if len(sortKey) == len(oldSortKey) {
			setError("Duplicate key: " + e.Value.(nameValueType).name)
		}
		return false
	}

	parseObject = func() string {
		nameValueList := list.New()
		next := false
	coreLoop2:
		for globalError == nil && testNextNonWhiteSpaceChar() != '}' {
			if next {
				scanFor(',')
			}
			next = true
			scanFor('"')
			rawUTF8 := parseQuotedString()
			if globalError != nil {
				break
			}
			sortKey := utf16.Encode([]rune(rawUTF8))
			scanFor(':')
			nameValue := nameValueType{rawUTF8, sortKey, parseElement()}
			for e := nameValueList.Front(); e != nil; e = e.Next() {
				if lexicographicallyPrecedes(sortKey, e) {
					nameValueList.InsertBefore(nameValue, e)
					continue coreLoop2
				}
			}
			nameValueList.PushBack(nameValue)
		}
		scan()
		var objectData strings.Builder
		objectData.WriteByte('{')
		next = false
		for e := nameValueList.Front(); e != nil; e = e.Next() {
			if next {
				objectData.WriteByte(',')
			}
			next = true
			nameValue := e.Value.(nameValueType)
			objectData.WriteString(decorateString(nameValue.name))
			objectData.WriteByte(':')
			objectData.WriteString(nameValue.value)
		}
		objectData.WriteByte('}')
		return objectData.String()
	}

	var transformed string
	if testNextNonWhiteSpaceChar() == '[' {
		scan()
		transformed = parseArray()
	} else {
		scanFor('{')
		transformed = parseObject()
	}
	for index < jsonDataLength {
		if !isWhiteSpace(jsonData[index]) {
			setError("Improperly terminated JSON object")
			break
		}
		index++
	}
	return []byte(transformed), globalError
}
