// antha-tools/antha/types/ordering.go: Part of the Antha language
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


// This file implements resolveOrder.

package types

import (
	"github.com/antha-lang/antha/ast"
	"sort"
)

// resolveOrder computes the order in which package-level objects
// must be type-checked.
//
// Interface types appear first in the list, sorted topologically
// by dependencies on embedded interfaces that are also declared
// in this package, followed by all other objects sorted in source
// order.
//
// TODO(gri) Consider sorting all types by dependencies here, and
// in the process check _and_ report type cycles. This may simplify
// the full type-checking phase.
//
func (check *checker) resolveOrder() []Object {
	var ifaces, others []Object

	// collect interface types with their dependencies, and all other objects
	for obj := range check.objMap {
		if ityp := check.interfaceFor(obj); ityp != nil {
			ifaces = append(ifaces, obj)
			// determine dependencies on embedded interfaces
			for _, f := range ityp.Methods.List {
				if len(f.Names) == 0 {
					// Embedded interface: The type must be a (possibly
					// qualified) identifier denoting another interface.
					// Imported interfaces are already fully resolved,
					// so we can ignore qualified identifiers.
					if ident, _ := f.Type.(*ast.Ident); ident != nil {
						embedded := check.pkg.scope.Lookup(ident.Name)
						if check.interfaceFor(embedded) != nil {
							check.objMap[obj].addDep(embedded)
						}
					}
				}
			}
		} else {
			others = append(others, obj)
		}
	}

	// final object order
	var order []Object

	// sort interface types topologically by dependencies,
	// and in source order if there are no dependencies
	sort.Sort(inSourceOrder(ifaces))
	if debug {
		for _, obj := range ifaces {
			assert(check.objMap[obj].mark == 0)
		}
	}
	for _, obj := range ifaces {
		check.appendInPostOrder(&order, obj)
	}

	// sort everything else in source order
	sort.Sort(inSourceOrder(others))

	return append(order, others...)
}

// interfaceFor returns the AST interface denoted by obj, or nil.
func (check *checker) interfaceFor(obj Object) *ast.InterfaceType {
	tname, _ := obj.(*TypeName)
	if tname == nil {
		return nil // not a type
	}
	d := check.objMap[obj]
	if d == nil {
		check.dump("%s: %s should have been declared", obj.Pos(), obj.Name())
		unreachable()
	}
	if d.typ == nil {
		return nil // invalid AST - ignore (will be handled later)
	}
	ityp, _ := d.typ.(*ast.InterfaceType)
	return ityp
}

func (check *checker) appendInPostOrder(order *[]Object, obj Object) {
	d := check.objMap[obj]
	if d.mark != 0 {
		// We've already seen this object; either because it's
		// already added to order, or because we have a cycle.
		// In both cases we stop. Cycle errors are reported
		// when type-checking types.
		return
	}
	d.mark = 1

	for _, obj := range orderedSetObjects(d.deps) {
		check.appendInPostOrder(order, obj)
	}

	*order = append(*order, obj)
}