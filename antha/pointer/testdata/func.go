// antha-tools/antha/pointer/testdata/func.go: Part of the Antha language
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

// +build ignore

package main

var a, b, c int

var unknown bool // defeat dead-code elimination

func func1() {
	var h int // @line f1h
	f := func(x *int) *int {
		if unknown {
			return &b
		}
		return x
	}

	// FV(g) = {f, h}
	g := func(x *int) *int {
		if unknown {
			return &h
		}
		return f(x)
	}

	print(g(&a)) // @pointsto main.a | main.b | h@f1h:6
	print(f(&a)) // @pointsto main.a | main.b
	print(&a)    // @pointsto main.a
}

// @calls main.func1 -> func1$2
// @calls main.func1 -> func1$1
// @calls func1$2 ->  func1$1

func func2() {
	var x, y *int
	defer func() {
		x = &a
	}()
	go func() {
		y = &b
	}()
	print(x) // @pointsto main.a
	print(y) // @pointsto main.b
}

func func3() {
	x, y := func() (x, y *int) {
		x = &a
		y = &b
		if unknown {
			return nil, &c
		}
		return
	}()
	print(x) // @pointsto main.a
	print(y) // @pointsto main.b | main.c
}

func swap(x, y *int) (*int, *int) { // @line swap
	print(&x) // @pointsto x@swap:11
	print(x)  // @pointsto makeslice[*]@func4make:11
	print(&y) // @pointsto y@swap:14
	print(y)  // @pointsto j@f4j:5
	return y, x
}

func func4() {
	a := make([]int, 10) // @line func4make
	i, j := 123, 456     // @line f4j
	_ = i
	p, q := swap(&a[3], &j)
	print(p) // @pointsto j@f4j:5
	print(q) // @pointsto makeslice[*]@func4make:11

	f := &b
	print(f) // @pointsto main.b
}

type T int

func (t *T) f(x *int) *int {
	print(t) // @pointsto main.a
	print(x) // @pointsto main.c
	return &b
}

func (t *T) g(x *int) *int {
	print(t) // @pointsto main.a
	print(x) // @pointsto main.b
	return &c
}

func (t *T) h(x *int) *int {
	print(t) // @pointsto main.a
	print(x) // @pointsto main.b
	return &c
}

var h func(*T, *int) *int

func func5() {
	// Static call of method.
	t := (*T)(&a)
	print(t.f(&c)) // @pointsto main.b

	// Static call of method as function
	print((*T).g(t, &b)) // @pointsto main.c

	// Dynamic call (not invoke) of method.
	h = (*T).h
	print(h(t, &b)) // @pointsto main.c
}

// @calls main.func5 -> (*main.T).f
// @calls main.func5 -> (*main.T).g
// @calls main.func5 -> (*main.T).h

func func6() {
	A := &a
	f := func() *int {
		return A // (free variable)
	}
	print(f()) // @pointsto main.a
}

// @calls main.func6 -> func6$1

type I interface {
	f()
}

type D struct{}

func (D) f() {}

func func7() {
	var i I = D{}
	imethodClosure := i.f
	imethodClosure()
	// @calls main.func7 -> bound$(main.I).f
	// @calls bound$(main.I).f -> (main.D).f

	var d D
	cmethodClosure := d.f
	cmethodClosure()
	// @calls main.func7 -> bound$(main.D).f
	// @calls bound$(main.D).f ->(main.D).f

	methodExpr := D.f
	methodExpr(d)
	// @calls main.func7 -> (main.D).f
}

func main() {
	func1()
	func2()
	func3()
	func4()
	func5()
	func6()
	func7()
}

// @calls <root> -> main.main
// @calls <root> -> main.init