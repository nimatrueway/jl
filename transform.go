package jl

import (
	"bytes"
	"fmt"
	"github.com/araddon/dateparse"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Context provides the current transformation context, to be used by Transformers and Stringers.
type Context struct {
	// The original string before any transformations were applied.
	Original string
	// Indicates that terminal color escape sequences should be disabled.
	DisableColor bool
	// Indicates that fields should not be truncated.
	DisableTruncate bool
}

// Transformer transforms a string and returns the result.
type Transformer interface {
	Transform(ctx *Context, input string) string
}

// TransformFunc is an adapter to allow the use of ordinary functions as Transformers.
type TransformFunc func(string) string

func (f TransformFunc) Transform(ctx *Context, input string) string {
	return f(input)
}

var (
	// UpperCase transforms the input string to upper case.
	UpperCase = TransformFunc(strings.ToUpper)
	// LowerCase transforms the input string to lower case.
	LowerCase = TransformFunc(strings.ToLower)
)

// Truncate truncates the string to the a requested number of digits.
type Truncate int

func (t Truncate) Transform(ctx *Context, input string) string {
	if ctx.DisableTruncate {
		return input
	}
	if utf8.RuneCountInString(input) <= int(t) {
		return input
	}
	return input[:t]
}

// Ellipsize replaces characters in the middle of the string with a single "…" character so that it fits within the
// requested length.
type Ellipsize int

func (remain Ellipsize) Transform(ctx *Context, input string) string {
	if ctx.DisableTruncate {
		return input
	}
	length := utf8.RuneCountInString(input)
	if length <= int(remain) {
		return input
	}
	remain -= 1 // account for the ellipsis
	chomped := length - int(remain)
	start := int(remain) / 2
	end := start + chomped
	return input[:start] + "…" + input[end:]
}

// PackageFold
type JvmClassPathFold int

func (cap JvmClassPathFold) Transform(ctx *Context, input string) string {
	if ctx.DisableTruncate {
		return input
	}
	length := utf8.RuneCountInString(input)
	if length <= int(cap) {
		return input
	}
	remaining := int(cap)
	output := ""
	parts := strings.Split(input, ".")
	className := parts[len(parts)-1]
	doCompact := false
	// class name
	if len(className) <= remaining {
		output = className
		remaining -= len(output)
	} else {
		classNameParts := strings.Split(regexp.MustCompile("(.)([A-Z]|(?:\\$+))").ReplaceAllString(className, "${1}_${2}"), "_")
		for i, v := range classNameParts {
			remainingUpperLetters := len(classNameParts) - i - 1
			if doCompact == false && (len(v)+remainingUpperLetters) > remaining {
				doCompact = true
			}
			if doCompact {
				cut := remaining - remainingUpperLetters
				output += v[:cut+1]
				remaining -= cut
			} else {
				output += v
				remaining -= len(v)
			}
		}
	}
	// packages
	for i := len(parts) - 2; i >= 0; i-- {
		if doCompact == false && (i*2)+1+len(parts[i]) >= remaining {
			doCompact = true
		}
		if doCompact {
			if remaining > 1 {
				output = string(parts[i][0]) + "." + output
				remaining -= 2
			}
		} else {
			output = parts[i] + "." + output
			remaining -= len(parts[i])
			remaining -= 1
		}
	}
	return output
}

// LeftPad pads the left side of the string with spaces so that the string becomes the requested length.
type LeftPad int

func (t LeftPad) Transform(ctx *Context, input string) string {
	spaces := int(t) - utf8.RuneCountInString(input)
	if spaces <= 0 {
		return input
	}
	buf := bytes.NewBuffer(make([]byte, 0, spaces+len(input)))
	for i := 0; i < spaces; i++ {
		buf.WriteRune(' ')
	}
	buf.WriteString(input)
	return buf.String()
}

// LeftPad pads the right side of the string with spaces so that the string becomes the requested length.
type RightPad int

func (t RightPad) Transform(ctx *Context, input string) string {
	pad := int(t) - utf8.RuneCountInString(input)
	if pad <= 0 {
		return input
	}
	buf := bytes.NewBuffer(make([]byte, 0, pad+len(input)))
	buf.WriteString(input)
	for i := 0; i < pad; i++ {
		buf.WriteRune(' ')
	}
	return buf.String()
}

// Format calls fmt.Sprintf() with the requested format string.
type Format string

func (t Format) Transform(ctx *Context, input string) string {
	return fmt.Sprintf(string(t), input)
}

type TimeFormatter string

func (t TimeFormatter) Transform(ctx *Context, input string) string {
	date, err := dateparse.ParseLocal(ctx.Original)
	if err != nil {
		return input
	}
	return date.In(time.Local).Format(string(t))
}
