// antha-tools/antha/types/errors.go: Part of the Antha language
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


// This file implements various error reporters.

package types

import (
	"fmt"
	"github.com/antha-lang/antha/ast"
	"github.com/antha-lang/antha/token"
	"strings"
)

func assert(p bool) {
	if !p {
		panic("assertion failed")
	}
}

func unreachable() {
	panic("unreachable")
}

func (check *checker) sprintf(format string, args ...interface{}) string {
	for i, arg := range args {
		switch a := arg.(type) {
		case nil:
			arg = "<nil>"
		case operand:
			panic("internal error: should always pass *operand")
		case *operand:
			arg = operandString(check.pkg, a)
		case token.Pos:
			arg = check.fset.Position(a).String()
		case ast.Expr:
			arg = ExprString(a)
		case Object:
			arg = ObjectString(check.pkg, a)
		case Type:
			arg = TypeString(check.pkg, a)
		}
		args[i] = arg
	}
	return fmt.Sprintf(format, args...)
}

func (check *checker) trace(pos token.Pos, format string, args ...interface{}) {
	fmt.Printf("%s:\t%s%s\n",
		check.fset.Position(pos),
		strings.Repeat(".  ", check.indent),
		check.sprintf(format, args...),
	)
}

// dump is only needed for debugging
func (check *checker) dump(format string, args ...interface{}) {
	fmt.Println(check.sprintf(format, args...))
}

func (check *checker) err(pos token.Pos, msg string, soft bool) {
	err := Error{check.fset, pos, msg, soft}
	if check.firstErr == nil {
		check.firstErr = err
	}
	f := check.conf.Error
	if f == nil {
		panic(bailout{}) // report only first error
	}
	f(err)
}

func (check *checker) error(pos token.Pos, msg string) {
	check.err(pos, msg, false)
}

func (check *checker) errorf(pos token.Pos, format string, args ...interface{}) {
	check.err(pos, check.sprintf(format, args...), false)
}

func (check *checker) softErrorf(pos token.Pos, format string, args ...interface{}) {
	check.err(pos, check.sprintf(format, args...), true)
}

func (check *checker) invalidAST(pos token.Pos, format string, args ...interface{}) {
	check.errorf(pos, "invalid AST: "+format, args...)
}

func (check *checker) invalidArg(pos token.Pos, format string, args ...interface{}) {
	check.errorf(pos, "invalid argument: "+format, args...)
}

func (check *checker) invalidOp(pos token.Pos, format string, args ...interface{}) {
	check.errorf(pos, "invalid operation: "+format, args...)
}