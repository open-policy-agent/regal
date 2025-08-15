// Copyright 2020 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

// Original source at: https://github.com/open-policy-agent/opa/blob/main/ast/internal/tokens/tokens.go

package tokens

// Token represents a single Rego source code token for use by the parser.
type Token int

// All tokens must be defined here
const (
	Illegal Token = iota
	EOF
	Whitespace
	Ident
	Comment

	Package
	Import
	As
	Default
	Else
	Not
	Some
	With
	Null
	True
	False

	Number
	String

	LBrack
	RBrack
	LBrace
	RBrace
	LParen
	RParen
	Comma
	Colon

	Add
	Sub
	Mul
	Quo
	Rem
	And
	Or
	Unify
	Equal
	Assign
	In
	Neq
	Gt
	Lt
	Gte
	Lte
	Dot
	Semicolon

	Every
	Contains
	If
)

var keywords = map[string]Token{
	"package": Package,
	"import":  Import,
	"as":      As,
	"default": Default,
	"else":    Else,
	"not":     Not,
	"some":    Some,
	"with":    With,
	"null":    Null,
	"true":    True,
	"false":   False,
}

func Keyword(lit string) Token {
	if tok, ok := keywords[lit]; ok {
		return tok
	}
	return Ident
}
