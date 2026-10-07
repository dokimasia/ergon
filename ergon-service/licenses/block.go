// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package licenses

import (
	"bytes"
	"regexp"
	"slices"
	"strings"

	"github.com/apache/skywalking-eyes/pkg/comments"
)

// The texts of the lines of a header, without their comment markers.
var (
	// copyrightLine matches a copyright notice: Copyright, ©, or (c), in any case.
	copyrightLine = regexp.MustCompile(`(?i)^(?:copyright\b|©|\(c\))`)

	// spdxTag matches a tag of SPDX, such as SPDX-License-Identifier: or SPDX-FileCopyrightText:.
	spdxTag = regexp.MustCompile(`^SPDX-[A-Za-z-]+:`)

	// years matches the years of a copyright notice: a year, a list such as 2024, 2026, or a range
	// such as 2020-2026.
	years = regexp.MustCompile(`\d{4}(?:\s*[-,]\s*\d{4})*`)
)

// block is the header of a file: the comment block after the preamble of its style, before any
// other line, that has a copyright notice or a tag of SPDX.
type block struct {
	// years are the years of the first copyright notice of the block that states years, or empty.
	years string

	// start is the offset of the first line of the block.
	start int

	// end is the offset after the last line of the block and the blank lines that follow it.
	end int

	// conflict is the number of the first line of the block whose text is neither a copyright
	// notice, nor a tag of SPDX, nor empty, counted from 1, and 0 for a block of such lines alone.
	conflict int
}

// line is a line of a file.
type line struct {
	// text is the line without its end.
	text string

	// start is the offset of the line.
	start int

	// end is the offset after the end of the line.
	end int
}

// find returns the header of content, the content of a file without its byte-order mark, in the
// comment style s, and reports whether content has one. after is the compiled preamble of s, or
// nil. The header is the first comment block after the preamble that has a copyright notice or a
// tag of SPDX, among the comment blocks and blank lines before the first other line, so a block
// such as the build constraint of Go may precede it. A block without a copyright notice or a tag
// of SPDX, such as the documentation of a package, is no header. The preamble starts after the
// spaces and newlines that open content, as the library matches it.
func find(content []byte, s *comments.CommentStyle, after *regexp.Regexp) (block, bool) {
	at := 0
	if after != nil {
		trimmed := bytes.TrimLeft(content, " \n")
		if loc := after.FindIndex(trimmed); loc != nil {
			at = len(content) - len(trimmed) + loc[1]
		}
	}
	lines := split(content, at)
	if at > 0 && len(lines) > 0 {
		lines = lines[1:]
	}
	for {
		for len(lines) > 0 && strings.TrimSpace(lines[0].text) == "" {
			lines = lines[1:]
		}
		texts, n := comment(lines, s)
		if n == 0 {
			return block{}, false
		}
		b := block{start: lines[0].start, end: lines[n-1].end}
		marked := false
		for i, text := range texts {
			if copyrightLine.MatchString(text) {
				if b.years == "" {
					b.years = years.FindString(text)
				}
				marked = true
			} else if spdxTag.MatchString(text) {
				marked = true
			} else if text != "" && b.conflict == 0 {
				b.conflict = bytes.Count(content[:lines[i].start], []byte("\n")) + 1
			}
		}
		lines = lines[n:]
		if !marked {
			continue
		}
		for _, l := range lines {
			if strings.TrimSpace(l.text) != "" {
				break
			}
			b.end = l.end
		}
		return b, true
	}
}

// comment returns the texts of the comment block that opens lines in the style s, without their
// markers and trimmed, and the number of lines of the block, 0 for lines that open with no comment.
// A style whose start and middle are one marker, such as //, comments each line, so its block ends
// before the first line without the marker. Any other style opens its block with a line that
// starts with its start, such as /*, and ends it with the first line that ends with its end, such
// as */. A block without its end is no comment.
func comment(lines []line, s *comments.CommentStyle) ([]string, int) {
	start, middle, end := strings.TrimSpace(s.Start), strings.TrimSpace(s.Middle), strings.TrimSpace(s.End)
	var texts []string
	if start == middle {
		for _, l := range lines {
			text, ok := strings.CutPrefix(strings.TrimSpace(l.text), start)
			if !ok {
				break
			}
			texts = append(texts, strings.TrimSpace(text))
		}
		return texts, len(texts)
	}
	if len(lines) == 0 || !strings.HasPrefix(strings.TrimSpace(lines[0].text), start) {
		return nil, 0
	}
	for i, l := range lines {
		text := strings.TrimSpace(l.text)
		if i == 0 {
			text = strings.TrimPrefix(text, start)
		}
		closed := strings.HasSuffix(text, end)
		text = strings.TrimSpace(strings.TrimSuffix(text, end))
		if middle != "" {
			text = strings.TrimSpace(strings.TrimPrefix(text, middle))
		}
		texts = append(texts, text)
		if closed {
			return texts, i + 1
		}
	}
	return nil, 0
}

// insert returns content with the header text, which ends in a blank line, after the preamble of the
// style s, as the insertion of skywalking-eyes places it. after is the compiled preamble of s, or
// nil for a style without one, whose header opens the file. For a style with a preamble, insert
// drops the spaces and newlines that open content, and puts the header on the line after the
// preamble. Without the preamble, the header opens content, behind the ensured preamble of the
// style where it has one, such as the <?php line of PHP, and before its closing text, such as ?>.
func insert(content []byte, text string, s *comments.CommentStyle, after *regexp.Regexp) []byte {
	if after == nil {
		return slices.Concat([]byte(text), content)
	}
	content = bytes.TrimLeft(content, " \n")
	loc := after.FindIndex(content)
	if loc == nil {
		if s.EnsureAfter != "" {
			return slices.Concat([]byte(s.EnsureAfter+"\n"+text+s.EnsureBefore), content)
		}
		return slices.Concat([]byte(text), content)
	}
	next := min(loc[1]+1, len(content))
	return slices.Concat(content[:loc[1]], []byte("\n"+text), content[next:])
}

// split returns the lines of content from the offset at, each with its offsets. The last line has
// no end when content does not end in a newline.
func split(content []byte, at int) []line {
	var lines []line
	for at < len(content) {
		n := bytes.IndexByte(content[at:], '\n')
		end := len(content)
		if n >= 0 {
			end = at + n + 1
		}
		lines = append(lines, line{text: string(bytes.TrimSuffix(content[at:end], []byte("\n"))), start: at, end: end})
		at = end
	}
	return lines
}
