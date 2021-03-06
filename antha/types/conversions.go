// antha-tools/antha/types/conversions.go: Part of the Antha language
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


// This file implements typechecking of conversions.

package types

import "github.com/antha-lang/antha-tools/antha/exact"

// Conversion type-checks the conversion T(x).
// The result is in x.
func (check *checker) conversion(x *operand, T Type) {
	constArg := x.mode == constant

	var ok bool
	switch {
	case constArg && isConstType(T):
		// constant conversion
		switch t := T.Underlying().(*Basic); {
		case representableConst(x.val, check.conf, t.kind, &x.val):
			ok = true
		case x.isInteger() && isString(t):
			codepoint := int64(-1)
			if i, ok := exact.Int64Val(x.val); ok {
				codepoint = i
			}
			// If codepoint < 0 the absolute value is too large (or unknown) for
			// conversion. This is the same as converting any other out-of-range
			// value - let string(codepoint) do the work.
			x.val = exact.MakeString(string(codepoint))
			ok = true
		}
	case x.convertibleTo(check.conf, T):
		// non-constant conversion
		x.mode = value
		ok = true
	}

	if !ok {
		check.errorf(x.pos(), "cannot convert %s to %s", x, T)
		x.mode = invalid
		return
	}

	// The conversion argument types are final. For untyped values the
	// conversion provides the type, per the spec: "A constant may be
	// given a type explicitly by a constant declaration or conversion,...".
	final := x.typ
	if isUntyped(x.typ) {
		final = T
		// - For conversions to interfaces, use the argument's default type.
		// - For conversions of untyped constants to non-constant types, also
		//   use the default type (e.g., []byte("foo") should report string
		//   not []byte as type for the constant "foo").
		// - Keep untyped nil for untyped nil arguments.
		if isInterface(T) || constArg && !isConstType(T) {
			final = defaultType(x.typ)
		}
		check.updateExprType(x.expr, final, true)
	}

	x.typ = T
}

func (x *operand) convertibleTo(conf *Config, T Type) bool {
	// "x is assignable to T"
	if x.assignableTo(conf, T) {
		return true
	}

	// "x's type and T have identical underlying types"
	V := x.typ
	Vu := V.Underlying()
	Tu := T.Underlying()
	if Identical(Vu, Tu) {
		return true
	}

	// "x's type and T are unnamed pointer types and their pointer base types have identical underlying types"
	if V, ok := V.(*Pointer); ok {
		if T, ok := T.(*Pointer); ok {
			if Identical(V.base.Underlying(), T.base.Underlying()) {
				return true
			}
		}
	}

	// "x's type and T are both integer or floating point types"
	if (isInteger(V) || isFloat(V)) && (isInteger(T) || isFloat(T)) {
		return true
	}

	// "x's type and T are both complex types"
	if isComplex(V) && isComplex(T) {
		return true
	}

	// "x is an integer or a slice of bytes or runes and T is a string type"
	if (isInteger(V) || isBytesOrRunes(Vu)) && isString(T) {
		return true
	}

	// "x is a string and T is a slice of bytes or runes"
	if isString(V) && isBytesOrRunes(Tu) {
		return true
	}

	// package unsafe:
	// "any pointer or value of underlying type uintptr can be converted into a unsafe.Pointer"
	if (isPointer(Vu) || isUintptr(Vu)) && isUnsafePointer(T) {
		return true
	}
	// "and vice versa"
	if isUnsafePointer(V) && (isPointer(Tu) || isUintptr(Tu)) {
		return true
	}

	return false
}

func isUintptr(typ Type) bool {
	t, ok := typ.Underlying().(*Basic)
	return ok && t.kind == Uintptr
}

func isUnsafePointer(typ Type) bool {
	// TODO(gri): Is this (typ.Underlying() instead of just typ) correct?
	//            The spec does not say so, but gc claims it is. See also
	//            issue 6326.
	t, ok := typ.Underlying().(*Basic)
	return ok && t.kind == UnsafePointer
}

func isPointer(typ Type) bool {
	_, ok := typ.Underlying().(*Pointer)
	return ok
}

func isBytesOrRunes(typ Type) bool {
	if s, ok := typ.(*Slice); ok {
		t, ok := s.elem.Underlying().(*Basic)
		return ok && (t.kind == Byte || t.kind == Rune)
	}
	return false
}