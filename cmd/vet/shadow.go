// antha-tools/cmd/vet/shadow.go: Part of the Antha language
// Copyright (C) 2014 The Antha authors. All rights reserved.
// 
// This program is free software; you can redistribute it and/or
// modify it under the terms of the GNU General Public License
// as published by the Free Software Foundation; either version 2
// of the License, or (at your option) any later version.
// 
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
// 
// You should have received a copy of the GNU General Public License
// along with this program; if not, write to the Free Software
// Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.
// 
// For more information relating to the software or licensing issues please
// contact license@antha-lang.org or write to the Antha team c/o 
// Synthace Ltd. The London Bioscience Innovation Centre
// 1 Royal College St, London NW1 0NH UK


/*
This file contains the code to check for shadowed variables.
A shadowed variable is a variable declared in an inner scope
with the same name and type as a variable in an outer scope,
and where the outer variable is mentioned after the inner one
is declared.

(This definition can be refined; the module generates too many
false positives and is not yet enabled by default.)

For example:

	func BadRead(f *os.File, buf []byte) error {
		var err error
		for {
			n, err := f.Read(buf) // shadows the function variable 'err'
			if err != nil {
				break // causes return of wrong value
			}
			foo(buf)
		}
		return err
	}

*/

package main

import (
	"github.com/antha-lang/antha/ast"
	"github.com/antha-lang/antha/token"

	"github.com/antha-lang/antha-tools/antha/types"
)

// Span stores the minimum range of byte positions in the file in which a
// given variable (types.Object) is mentioned. It is lexically defined: it spans
// from the beginning of its first mention to the end of its last mention.
// A variable is considered shadowed (if *strictShadowing is off) only if the
// shadowing variable is declared within the span of the shadowed variable.
// In other words, if a variable is shadowed but not used after the shadowed
// variable is declared, it is inconsequential and not worth complaining about.
// This simple check dramatically reduces the nuisance rate for the shadowing
// check, at least until something cleverer comes along.
//
// One wrinkle: A "naked return" is a silent use of a variable that the Span
// will not capture, but the compilers catch naked returns of shadowed
// variables so we don't need to.
//
// Cases this gets wrong (TODO):
// - If a for loop's continuation statement mentions a variable redeclared in
// the block, we should complain about it but don't.
// - A variable declared inside a function literal can falsely be identified
// as shadowing a variable in the outer function.
//
type Span struct {
	min token.Pos
	max token.Pos
}

// contains reports whether the position is inside the span.
func (s Span) contains(pos token.Pos) bool {
	return s.min <= pos && pos < s.max
}

// growSpan expands the span for the object to contain the instance represented
// by the identifier.
func (pkg *Package) growSpan(ident *ast.Ident, obj types.Object) {
	if *strictShadowing {
		return // No need
	}
	pos := ident.Pos()
	end := ident.End()
	span, ok := pkg.spans[obj]
	if ok {
		if span.min > pos {
			span.min = pos
		}
		if span.max < end {
			span.max = end
		}
	} else {
		span = Span{pos, end}
	}
	pkg.spans[obj] = span
}

// checkShadowAssignment checks for shadowing in a short variable declaration.
func (f *File) checkShadowAssignment(a *ast.AssignStmt) {
	if !vet("shadow") {
		return
	}
	if a.Tok != token.DEFINE {
		return
	}
	if f.idiomaticShortRedecl(a) {
		return
	}
	for _, expr := range a.Lhs {
		ident, ok := expr.(*ast.Ident)
		if !ok {
			f.Badf(expr.Pos(), "invalid AST: short variable declaration of non-identifier")
			return
		}
		f.checkShadowing(ident)
	}
}

// idiomaticShortRedecl reports whether this short declaration can be ignored for
// the purposes of shadowing, that is, that any redeclarations it contains are deliberate.
func (f *File) idiomaticShortRedecl(a *ast.AssignStmt) bool {
	// Don't complain about deliberate redeclarations of the form
	//	i := i
	// Such constructs are idiomatic in range loops to create a new variable
	// for each iteration. Another example is
	//	switch n := n.(type)
	if len(a.Rhs) != len(a.Lhs) {
		return false
	}
	// We know it's an assignment, so the LHS must be all identifiers. (We check anyway.)
	for i, expr := range a.Lhs {
		lhs, ok := expr.(*ast.Ident)
		if !ok {
			f.Badf(expr.Pos(), "invalid AST: short variable declaration of non-identifier")
			return true // Don't do any more processing.
		}
		switch rhs := a.Rhs[i].(type) {
		case *ast.Ident:
			if lhs.Name != rhs.Name {
				return false
			}
		case *ast.TypeAssertExpr:
			if id, ok := rhs.X.(*ast.Ident); ok {
				if lhs.Name != id.Name {
					return false
				}
			}
		}
	}
	return true
}

// idiomaticRedecl reports whether this declaration spec can be ignored for
// the purposes of shadowing, that is, that any redeclarations it contains are deliberate.
func (f *File) idiomaticRedecl(d *ast.ValueSpec) bool {
	// Don't complain about deliberate redeclarations of the form
	//	var i, j = i, j
	if len(d.Names) != len(d.Values) {
		return false
	}
	for i, lhs := range d.Names {
		if rhs, ok := d.Values[i].(*ast.Ident); ok {
			if lhs.Name != rhs.Name {
				return false
			}
		}
	}
	return true
}

// checkShadowDecl checks for shadowing in a general variable declaration.
func (f *File) checkShadowDecl(d *ast.GenDecl) {
	if !vet("shadow") {
		return
	}
	if d.Tok != token.VAR {
		return
	}
	for _, spec := range d.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			f.Badf(spec.Pos(), "invalid AST: var GenDecl not ValueSpec")
			return
		}
		// Don't complain about deliberate redeclarations of the form
		//	var i = i
		if f.idiomaticRedecl(valueSpec) {
			return
		}
		for _, ident := range valueSpec.Names {
			f.checkShadowing(ident)
		}
	}
}

// checkShadowing checks whether the identifier shadows an identifier in an outer scope.
func (f *File) checkShadowing(ident *ast.Ident) {
	if ident.Name == "_" {
		// Can't shadow the blank identifier.
		return
	}
	obj := f.pkg.defs[ident]
	if obj == nil {
		return
	}
	// obj.Parent.Parent is the surrounding scope. If we can find another declaration
	// starting from there, we have a shadowed variable.
	shadowed := obj.Parent().Parent().LookupParent(obj.Name())
	if shadowed == nil {
		return
	}
	// Don't complain if it's shadowing a universe-declared variable; that's fine.
	if shadowed.Parent() == types.Universe {
		return
	}
	if *strictShadowing {
		// The shadowed variable must appear before this one to be an instance of shadowing.
		if shadowed.Pos() > ident.Pos() {
			return
		}
	} else {
		// Don't complain if the span of validity of the shadowed variable doesn't include
		// the shadowing variable.
		span, ok := f.pkg.spans[shadowed]
		if !ok {
			f.Badf(ident.Pos(), "internal error: no range for %s", ident.Name)
			return
		}
		if !span.contains(ident.Pos()) {
			return
		}
	}
	// Don't complain if the types differ: that implies the programmer really wants two variables.
	if types.Identical(obj.Type(), shadowed.Type()) {
		f.Badf(ident.Pos(), "declaration of %s shadows declaration at %s", obj.Name(), f.loc(shadowed.Pos()))
	}
}