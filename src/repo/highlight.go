// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package repo

import (
	"bytes"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
)

func Highlight(name, input string) (string, string, error) {
	var buf, css bytes.Buffer

	lexer := lexers.Match(name)
	if lexer == nil {
		lexer = lexers.Fallback
	}

	formatter := html.New(html.WithClasses(true))

	iter, err := lexer.Tokenise(nil, input)
	if err != nil {
		return "", "", err
	}

	if err := formatter.Format(&buf, goitStyle, iter); err != nil {
		return "", "", err
	}

	if err := formatter.WriteCSS(&css, goitStyle); err != nil {
		return "", "", err
	}

	return buf.String(), css.String(), nil
}

var goitStyle = styles.Register(chroma.MustNewStyle("goit", chroma.StyleEntries{
	chroma.Background:            "#888888",
	chroma.Comment:               "italic #666666",
	chroma.CommentPreproc:        "noinherit #8ec07c",
	chroma.CommentPreprocFile:    "noinherit #b8bb26",
	chroma.GenericDeleted:        "#d65d0e",
	chroma.GenericEmph:           "italic",
	chroma.GenericError:          "bold bg:#fb4934",
	chroma.GenericHeading:        "bold #fabd2f",
	chroma.GenericInserted:       "#b8bb26",
	chroma.GenericOutput:         "#504945",
	chroma.GenericPrompt:         "#ebdbb2",
	chroma.GenericStrong:         "bold",
	chroma.GenericSubheading:     "bold #fabd2f",
	chroma.GenericTraceback:      "bold bg:#fb4934",
	chroma.GenericUnderline:      "underline",
	chroma.Keyword:               "#fb4934",
	chroma.KeywordNamespace:      "#d3869b",
	chroma.KeywordType:           "#fabd2f",
	chroma.LiteralNumber:         "#d3869b",
	chroma.LiteralString:         "#b8bb26",
	chroma.LiteralStringEscape:   "#d3869b",
	chroma.LiteralStringInterpol: "#8ec07c",
	chroma.LiteralStringRegex:    "#fe8019",
	chroma.LiteralStringSymbol:   "#83a598",
	chroma.Name:                  "#ebdbb2",
	chroma.NameAttribute:         "#fabd2f",
	chroma.NameBuiltin:           "#fabd2f",
	chroma.NameClass:             "#fabd2f",
	chroma.NameConstant:          "#d3869b",
	chroma.NameEntity:            "#fabd2f",
	chroma.NameException:         "#fb4934",
	chroma.NameFunction:          "#fabd2f",
	chroma.NameLabel:             "#fb4934",
	chroma.NameTag:               "#8ec07c",
	chroma.NameVariable:          "#83a598",
	chroma.Operator:              "#8ec07c",
}))
