// antha-tools/antha/types/typexpr.go: Part of the Antha language
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


// This file implements type-checking of identifiers and type expressions.

package types

import (
	"github.com/antha-lang/antha/ast"
	"github.com/antha-lang/antha/token"
	"sort"
	"strconv"

	"github.com/antha-lang/antha-tools/antha/exact"
)

// ident type-checks identifier e and initializes x with the value or type of e.
// If an error occurred, x.mode is set to invalid.
// For the meaning of def and path, see check.typ, below.
//
func (check *checker) ident(x *operand, e *ast.Ident, def *Named, path []*TypeName) {
	x.mode = invalid
	x.expr = e

	obj := check.scope.LookupParent(e.Name)
	if obj == nil {
		if e.Name == "_" {
			check.errorf(e.Pos(), "cannot use _ as value or type")
		} else {
			check.errorf(e.Pos(), "undeclared name: %s", e.Name)
		}
		return
	}
	check.recordUse(e, obj)

	check.objDecl(obj, def, path)
	typ := obj.Type()
	assert(typ != nil)

	switch obj := obj.(type) {
	case *PkgName:
		check.errorf(e.Pos(), "use of package %s not in selector", obj.name)
		return

	case *Const:
		// The constant may be dot-imported. Mark it as used so that
		// later we can determine if the corresponding dot-imported
		// package was used. Same applies for other objects, below.
		// (This code is only used for dot-imports. Without them, we
		// would only have to mark Vars.)
		obj.used = true
		check.addDeclDep(obj)
		if typ == Typ[Invalid] {
			return
		}
		if obj == universeIota {
			if check.iota == nil {
				check.errorf(e.Pos(), "cannot use iota outside constant declaration")
				return
			}
			x.val = check.iota
		} else {
			x.val = obj.val
		}
		assert(x.val != nil)
		x.mode = constant

	case *TypeName:
		obj.used = true
		x.mode = typexpr
		// check for cycle
		// (it's ok to iterate forward because each named type appears at most once in path)
		for i, prev := range path {
			if prev == obj {
				check.errorf(obj.pos, "illegal cycle in declaration of %s", obj.name)
				// print cycle
				for _, obj := range path[i:] {
					check.errorf(obj.Pos(), "\t%s refers to", obj.Name()) // secondary error, \t indented
				}
				check.errorf(obj.Pos(), "\t%s", obj.Name())
				// maintain x.mode == typexpr despite error
				typ = Typ[Invalid]
				break
			}
		}

	case *Var:
		obj.used = true
		check.addDeclDep(obj)
		x.mode = variable

	case *Func:
		obj.used = true
		check.addDeclDep(obj)
		x.mode = value

	case *Builtin:
		obj.used = true // for built-ins defined by package unsafe
		x.id = obj.id
		x.mode = builtin

	case *Nil:
		// no need to "use" the nil object
		x.mode = value

	default:
		unreachable()
	}

	x.typ = typ
}

// typExpr type-checks the type expression e and returns its type, or Typ[Invalid].
// If def != nil, e is the type specification for the named type def, declared
// in a type declaration, and def.underlying will be set to the type of e before
// any components of e are type-checked. Path contains the path of named types
// referring to this type.
//
func (check *checker) typExpr(e ast.Expr, def *Named, path []*TypeName) (T Type) {
	if trace {
		check.trace(e.Pos(), "%s", e)
		check.indent++
		defer func() {
			check.indent--
			check.trace(e.Pos(), "=> %s", T)
		}()
	}

	T = check.typExprInternal(e, def, path)
	assert(isTyped(T))
	check.recordTypeAndValue(e, T, nil)

	return
}

func (check *checker) typ(e ast.Expr) Type {
	return check.typExpr(e, nil, nil)
}

// funcType type-checks a function or method type and returns its signature.
func (check *checker) funcType(sig *Signature, recv *ast.FieldList, ftyp *ast.FuncType) *Signature {
	scope := NewScope(check.scope, "function")
	check.recordScope(ftyp, scope)

	recv_, _ := check.collectParams(scope, recv, false)
	params, variadic := check.collectParams(scope, ftyp.Params, true)
	results, _ := check.collectParams(scope, ftyp.Results, false)

	if len(recv_) > 0 {
		// There must be exactly one receiver.
		if len(recv_) > 1 {
			check.invalidAST(recv_[1].Pos(), "method must have exactly one receiver")
			// ok to continue
		}
		recv := recv_[0]
		// spec: "The receiver type must be of the form T or *T where T is a type name."
		// (ignore invalid types - error was reported before)
		if t, _ := deref(recv.typ); t != Typ[Invalid] {
			var err string
			if T, _ := t.(*Named); T != nil {
				// spec: "The type denoted by T is called the receiver base type; it must not
				// be a pointer or interface type and it must be declared in the same package
				// as the method."
				if T.obj.pkg != check.pkg {
					err = "type not defined in this package"
				} else {
					// TODO(gri) This is not correct if the underlying type is unknown yet.
					switch u := T.underlying.(type) {
					case *Basic:
						// unsafe.Pointer is treated like a regular pointer
						if u.kind == UnsafePointer {
							err = "unsafe.Pointer"
						}
					case *Pointer, *Interface:
						err = "pointer or interface type"
					}
				}
			} else {
				err = "basic or unnamed type"
			}
			if err != "" {
				check.errorf(recv.pos, "invalid receiver %s (%s)", recv.typ, err)
				// ok to continue
			}
		}
		sig.recv = recv
	}

	sig.scope = scope
	sig.params = NewTuple(params...)
	sig.results = NewTuple(results...)
	sig.variadic = variadic

	return sig
}

// typExprInternal drives type checking of types.
// Must only be called by typExpr.
//
func (check *checker) typExprInternal(e ast.Expr, def *Named, path []*TypeName) Type {
	switch e := e.(type) {
	case *ast.BadExpr:
		// ignore - error reported before

	case *ast.Ident:
		var x operand
		check.ident(&x, e, def, path)

		switch x.mode {
		case typexpr:
			typ := x.typ
			def.setUnderlying(typ)
			return typ
		case invalid:
			// ignore - error reported before
		case novalue:
			check.errorf(x.pos(), "%s used as type", &x)
		default:
			check.errorf(x.pos(), "%s is not a type", &x)
		}

	case *ast.SelectorExpr:
		var x operand
		check.selector(&x, e)

		switch x.mode {
		case typexpr:
			typ := x.typ
			def.setUnderlying(typ)
			return typ
		case invalid:
			// ignore - error reported before
		case novalue:
			check.errorf(x.pos(), "%s used as type", &x)
		default:
			check.errorf(x.pos(), "%s is not a type", &x)
		}

	case *ast.ParenExpr:
		return check.typExpr(e.X, def, path)

	case *ast.ArrayType:
		if e.Len != nil {
			typ := new(Array)
			def.setUnderlying(typ)
			typ.len = check.arrayLength(e.Len)
			typ.elem = check.typExpr(e.Elt, nil, path)
			return typ

		} else {
			typ := new(Slice)
			def.setUnderlying(typ)
			typ.elem = check.typ(e.Elt)
			return typ
		}

	case *ast.StructType:
		typ := new(Struct)
		def.setUnderlying(typ)
		check.structType(typ, e, path)
		return typ

	case *ast.StarExpr:
		typ := new(Pointer)
		def.setUnderlying(typ)
		typ.base = check.typ(e.X)
		return typ

	case *ast.FuncType:
		typ := new(Signature)
		def.setUnderlying(typ)
		check.funcType(typ, nil, e)
		return typ

	case *ast.InterfaceType:
		typ := new(Interface)
		def.setUnderlying(typ)
		check.interfaceType(typ, e, def, path)
		return typ

	case *ast.MapType:
		typ := new(Map)
		def.setUnderlying(typ)

		typ.key = check.typ(e.Key)
		typ.elem = check.typ(e.Value)

		// spec: "The comparison operators == and != must be fully defined
		// for operands of the key type; thus the key type must not be a
		// function, map, or slice."
		//
		// Delay this check because it requires fully setup types;
		// it is safe to continue in any case (was issue 6667).
		check.delay(func() {
			if !Comparable(typ.key) {
				check.errorf(e.Key.Pos(), "invalid map key type %s", typ.key)
			}
		})

		return typ

	case *ast.ChanType:
		typ := new(Chan)
		def.setUnderlying(typ)

		dir := SendRecv
		switch e.Dir {
		case ast.SEND | ast.RECV:
			// nothing to do
		case ast.SEND:
			dir = SendOnly
		case ast.RECV:
			dir = RecvOnly
		default:
			check.invalidAST(e.Pos(), "unknown channel direction %d", e.Dir)
			// ok to continue
		}

		typ.dir = dir
		typ.elem = check.typ(e.Value)
		return typ

	default:
		check.errorf(e.Pos(), "%s is not a type", e)
	}

	typ := Typ[Invalid]
	def.setUnderlying(typ)
	return typ
}

// typeOrNil type-checks the type expression (or nil value) e
// and returns the typ of e, or nil.
// If e is neither a type nor nil, typOrNil returns Typ[Invalid].
//
func (check *checker) typOrNil(e ast.Expr) Type {
	var x operand
	check.rawExpr(&x, e, nil)
	switch x.mode {
	case invalid:
		// ignore - error reported before
	case novalue:
		check.errorf(x.pos(), "%s used as type", &x)
	case typexpr:
		return x.typ
	case value:
		if x.isNil() {
			return nil
		}
		fallthrough
	default:
		check.errorf(x.pos(), "%s is not a type", &x)
	}
	return Typ[Invalid]
}

func (check *checker) arrayLength(e ast.Expr) int64 {
	var x operand
	check.expr(&x, e)
	if x.mode != constant {
		if x.mode != invalid {
			check.errorf(x.pos(), "array length %s must be constant", &x)
		}
		return 0
	}
	if !x.isInteger() {
		check.errorf(x.pos(), "array length %s must be integer", &x)
		return 0
	}
	n, ok := exact.Int64Val(x.val)
	if !ok || n < 0 {
		check.errorf(x.pos(), "invalid array length %s", &x)
		return 0
	}
	return n
}

func (check *checker) collectParams(scope *Scope, list *ast.FieldList, variadicOk bool) (params []*Var, variadic bool) {
	if list == nil {
		return
	}

	var named, anonymous bool
	for i, field := range list.List {
		ftype := field.Type
		if t, _ := ftype.(*ast.Ellipsis); t != nil {
			ftype = t.Elt
			if variadicOk && i == len(list.List)-1 {
				variadic = true
			} else {
				check.invalidAST(field.Pos(), "... not permitted")
				// ignore ... and continue
			}
		}
		typ := check.typ(ftype)
		// The parser ensures that f.Tag is nil and we don't
		// care if a constructed AST contains a non-nil tag.
		if len(field.Names) > 0 {
			// named parameter
			for _, name := range field.Names {
				if name.Name == "" {
					check.invalidAST(name.Pos(), "anonymous parameter")
					// ok to continue
				}
				par := NewParam(name.Pos(), check.pkg, name.Name, typ)
				check.declare(scope, name, par)
				params = append(params, par)
			}
			named = true
		} else {
			// anonymous parameter
			par := NewParam(ftype.Pos(), check.pkg, "", typ)
			check.recordImplicit(field, par)
			params = append(params, par)
			anonymous = true
		}
	}

	if named && anonymous {
		check.invalidAST(list.Pos(), "list contains both named and anonymous parameters")
		// ok to continue
	}

	// For a variadic function, change the last parameter's type from T to []T.
	if variadic && len(params) > 0 {
		last := params[len(params)-1]
		last.typ = &Slice{elem: last.typ}
	}

	return
}

func (check *checker) declareInSet(oset *objset, pos token.Pos, obj Object) bool {
	if alt := oset.insert(obj); alt != nil {
		check.errorf(pos, "%s redeclared", obj.Name())
		check.reportAltDecl(alt)
		return false
	}
	return true
}

func (check *checker) interfaceType(iface *Interface, ityp *ast.InterfaceType, def *Named, path []*TypeName) {
	// empty interface: common case
	if ityp.Methods == nil {
		return
	}

	// The parser ensures that field tags are nil and we don't
	// care if a constructed AST contains non-nil tags.

	// use named receiver type if available (for better error messages)
	var recvTyp Type = iface
	if def != nil {
		recvTyp = def
	}

	// Phase 1: Collect explicitly declared methods, the corresponding
	//          signature (AST) expressions, and the list of embedded
	//          type (AST) expressions. Do not resolve signatures or
	//          embedded types yet to avoid cycles referring to this
	//          interface.

	var (
		mset       objset
		signatures []ast.Expr // list of corresponding method signatures
		embedded   []ast.Expr // list of embedded types
	)
	for _, f := range ityp.Methods.List {
		if len(f.Names) > 0 {
			// The parser ensures that there's only one method
			// and we don't care if a constructed AST has more.
			name := f.Names[0]
			pos := name.Pos()
			// spec: "As with all method sets, in an interface type,
			// each method must have a unique non-blank name."
			if name.Name == "_" {
				check.errorf(pos, "invalid method name _")
				continue
			}
			// Don't type-check signature yet - use an
			// empty signature now and update it later.
			// Since we know the receiver, set it up now
			// (required to avoid crash in ptrRecv; see
			// e.g. test case for issue 6638).
			// TODO(gri) Consider marking methods signatures
			// as incomplete, for better error messages. See
			// also the T4 and T5 tests in testdata/cycles2.src.
			sig := new(Signature)
			sig.recv = NewVar(pos, check.pkg, "", recvTyp)
			m := NewFunc(pos, check.pkg, name.Name, sig)
			if check.declareInSet(&mset, pos, m) {
				iface.methods = append(iface.methods, m)
				iface.allMethods = append(iface.allMethods, m)
				signatures = append(signatures, f.Type)
				check.recordDef(name, m)
			}
		} else {
			// embedded type
			embedded = append(embedded, f.Type)
		}
	}

	// Phase 2: Resolve embedded interfaces. Because an interface must not
	//          embed itself (directly or indirectly), each embedded interface
	//          can be fully resolved without depending on any method of this
	//          interface (if there is a cycle or another error, the embedded
	//          type resolves to an invalid type and is ignored).
	//          In particular, the list of methods for each embedded interface
	//          must be complete (it cannot depend on this interface), and so
	//          those methods can be added to the list of all methods of this
	//          interface.

	for _, e := range embedded {
		pos := e.Pos()
		typ := check.typExpr(e, nil, path)
		named, _ := typ.(*Named)
		if named == nil {
			if typ != Typ[Invalid] {
				check.invalidAST(pos, "%s is not named type", typ)
			}
			continue
		}
		// determine underlying (possibly incomplete) type
		// by following its forward chain
		u := underlying(named)
		embed, _ := u.(*Interface)
		if embed == nil {
			if u != Typ[Invalid] {
				check.errorf(pos, "%s is not an interface", named)
			}
			continue
		}
		iface.embeddeds = append(iface.embeddeds, named)
		// collect embedded methods
		for _, m := range embed.allMethods {
			if check.declareInSet(&mset, pos, m) {
				iface.allMethods = append(iface.allMethods, m)
			}
		}
	}

	// Phase 3: At this point all methods have been collected for this interface.
	//          It is now safe to type-check the signatures of all explicitly
	//          declared methods, even if they refer to this interface via a cycle
	//          and embed the methods of this interface in a parameter of interface
	//          type.

	for i, m := range iface.methods {
		expr := signatures[i]
		typ := check.typ(expr)
		sig, _ := typ.(*Signature)
		if sig == nil {
			if typ != Typ[Invalid] {
				check.invalidAST(expr.Pos(), "%s is not a method signature", typ)
			}
			continue // keep method with empty method signature
		}
		// update signature, but keep recv that was set up before
		old := m.typ.(*Signature)
		sig.recv = old.recv
		*old = *sig // update signature (don't replace it!)
	}

	// TODO(gri) The list of explicit methods is only sorted for now to
	// produce the same Interface as NewInterface. We may be able to
	// claim source order in the future. Revisit.
	sort.Sort(byUniqueMethodName(iface.methods))

	// TODO(gri) The list of embedded types is only sorted for now to
	// produce the same Interface as NewInterface. We may be able to
	// claim source order in the future. Revisit.
	sort.Sort(byUniqueTypeName(iface.embeddeds))

	sort.Sort(byUniqueMethodName(iface.allMethods))
}

// byUniqueTypeName named type lists can be sorted by their unique type names.
type byUniqueTypeName []*Named

func (a byUniqueTypeName) Len() int           { return len(a) }
func (a byUniqueTypeName) Less(i, j int) bool { return a[i].obj.Id() < a[j].obj.Id() }
func (a byUniqueTypeName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// byUniqueMethodName method lists can be sorted by their unique method names.
type byUniqueMethodName []*Func

func (a byUniqueMethodName) Len() int           { return len(a) }
func (a byUniqueMethodName) Less(i, j int) bool { return a[i].Id() < a[j].Id() }
func (a byUniqueMethodName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func (check *checker) tag(t *ast.BasicLit) string {
	if t != nil {
		if t.Kind == token.STRING {
			if val, err := strconv.Unquote(t.Value); err == nil {
				return val
			}
		}
		check.invalidAST(t.Pos(), "incorrect tag syntax: %q", t.Value)
	}
	return ""
}

func (check *checker) structType(styp *Struct, e *ast.StructType, path []*TypeName) {
	list := e.Fields
	if list == nil {
		return
	}

	// struct fields and tags
	var fields []*Var
	var tags []string

	// for double-declaration checks
	var fset objset

	// current field typ and tag
	var typ Type
	var tag string
	// anonymous != nil indicates an anonymous field.
	add := func(field *ast.Field, ident *ast.Ident, anonymous *TypeName, pos token.Pos) {
		if tag != "" && tags == nil {
			tags = make([]string, len(fields))
		}
		if tags != nil {
			tags = append(tags, tag)
		}

		name := ident.Name
		fld := NewField(pos, check.pkg, name, typ, anonymous != nil)
		// spec: "Within a struct, non-blank field names must be unique."
		if name == "_" || check.declareInSet(&fset, pos, fld) {
			fields = append(fields, fld)
			check.recordDef(ident, fld)
		}
		if anonymous != nil {
			check.recordUse(ident, anonymous)
		}
	}

	for _, f := range list.List {
		typ = check.typExpr(f.Type, nil, path)
		tag = check.tag(f.Tag)
		if len(f.Names) > 0 {
			// named fields
			for _, name := range f.Names {
				add(f, name, nil, name.Pos())
			}
		} else {
			// anonymous field
			name := anonymousFieldIdent(f.Type)
			pos := f.Type.Pos()
			t, isPtr := deref(typ)
			switch t := t.(type) {
			case *Basic:
				if t == Typ[Invalid] {
					// error was reported before
					continue
				}
				// unsafe.Pointer is treated like a regular pointer
				if t.kind == UnsafePointer {
					check.errorf(pos, "anonymous field type cannot be unsafe.Pointer")
					continue
				}
				add(f, name, Universe.Lookup(t.name).(*TypeName), pos)

			case *Named:
				// spec: "An embedded type must be specified as a type name
				// T or as a pointer to a non-interface type name *T, and T
				// itself may not be a pointer type."
				switch u := t.underlying.(type) {
				case *Basic:
					// unsafe.Pointer is treated like a regular pointer
					if u.kind == UnsafePointer {
						check.errorf(pos, "anonymous field type cannot be unsafe.Pointer")
						continue
					}
				case *Pointer:
					check.errorf(pos, "anonymous field type cannot be a pointer")
					continue
				case *Interface:
					if isPtr {
						check.errorf(pos, "anonymous field type cannot be a pointer to an interface")
						continue
					}
				}
				add(f, name, t.obj, pos)

			default:
				check.invalidAST(pos, "anonymous field type %s must be named", typ)
			}
		}
	}

	styp.fields = fields
	styp.tags = tags
}

func anonymousFieldIdent(e ast.Expr) *ast.Ident {
	switch e := e.(type) {
	case *ast.Ident:
		return e
	case *ast.StarExpr:
		return anonymousFieldIdent(e.X)
	case *ast.SelectorExpr:
		return e.Sel
	}
	return nil // invalid anonymous field
}