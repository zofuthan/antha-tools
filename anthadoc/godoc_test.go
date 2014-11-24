// antha-tools/anthadoc/anthadoc_test.go: Part of the Antha language
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


package anthadoc

import (
	"testing"
)

func TestPkgLinkFunc(t *testing.T) {
	for _, tc := range []struct {
		path string
		want string
	}{
		{"/src/pkg/fmt", "pkg/fmt"},
		{"/fmt", "pkg/fmt"},
	} {
		if got := pkgLinkFunc(tc.path); got != tc.want {
			t.Errorf("pkgLinkFunc(%v) = %v; want %v", tc.path, got, tc.want)
		}
	}
}

func TestSrcPosLinkFunc(t *testing.T) {
	for _, tc := range []struct {
		src  string
		line int
		low  int
		high int
		want string
	}{
		{"/src/pkg/fmt/print.go", 42, 30, 50, "/src/pkg/fmt/print.go?s=30:50#L32"},
		{"/src/pkg/fmt/print.go", 2, 1, 5, "/src/pkg/fmt/print.go?s=1:5#L1"},
		{"/src/pkg/fmt/print.go", 2, 0, 0, "/src/pkg/fmt/print.go#L2"},
		{"/src/pkg/fmt/print.go", 0, 0, 0, "/src/pkg/fmt/print.go"},
		{"/src/pkg/fmt/print.go", 0, 1, 5, "/src/pkg/fmt/print.go?s=1:5#L1"},
		{"fmt/print.go", 0, 0, 0, "/src/pkg/fmt/print.go"},
		{"fmt/print.go", 0, 1, 5, "/src/pkg/fmt/print.go?s=1:5#L1"},
	} {
		if got := srcPosLinkFunc(tc.src, tc.line, tc.low, tc.high); got != tc.want {
			t.Errorf("srcLinkFunc(%v, %v, %v, %v) = %v; want %v", tc.src, tc.line, tc.low, tc.high, got, tc.want)
		}
	}
}

func TestSrcLinkFunc(t *testing.T) {
	for _, tc := range []struct {
		src  string
		want string
	}{
		{"/src/pkg/fmt/print.go", "/src/pkg/fmt/print.go"},
		{"src/pkg/fmt/print.go", "/src/pkg/fmt/print.go"},
		{"/fmt/print.go", "/src/pkg/fmt/print.go"},
		{"fmt/print.go", "/src/pkg/fmt/print.go"},
	} {
		if got := srcLinkFunc(tc.src); got != tc.want {
			t.Errorf("srcLinkFunc(%v) = %v; want %v", tc.src, got, tc.want)
		}
	}
}

func TestQueryLinkFunc(t *testing.T) {
	for _, tc := range []struct {
		src   string
		query string
		line  int
		want  string
	}{
		{"/src/pkg/fmt/print.go", "Sprintf", 33, "/src/pkg/fmt/print.go?h=Sprintf#L33"},
		{"/src/pkg/fmt/print.go", "Sprintf", 0, "/src/pkg/fmt/print.go?h=Sprintf"},
		{"src/pkg/fmt/print.go", "EOF", 33, "/src/pkg/fmt/print.go?h=EOF#L33"},
		{"src/pkg/fmt/print.go", "a%3f+%26b", 1, "/src/pkg/fmt/print.go?h=a%3f+%26b#L1"},
	} {
		if got := queryLinkFunc(tc.src, tc.query, tc.line); got != tc.want {
			t.Errorf("queryLinkFunc(%v, %v, %v) = %v; want %v", tc.src, tc.query, tc.line, got, tc.want)
		}
	}
}

func TestDocLinkFunc(t *testing.T) {
	for _, tc := range []struct {
		src   string
		ident string
		want  string
	}{
		{"/src/pkg/fmt", "Sprintf", "/pkg/fmt/#Sprintf"},
		{"/src/pkg/fmt", "EOF", "/pkg/fmt/#EOF"},
	} {
		if got := docLinkFunc(tc.src, tc.ident); got != tc.want {
			t.Errorf("docLinkFunc(%v, %v) = %v; want %v", tc.src, tc.ident, got, tc.want)
		}
	}
}

func TestSanitizeFunc(t *testing.T) {
	for _, tc := range []struct {
		src  string
		want string
	}{
		{},
		{"foo", "foo"},
		{"func   f()", "func f()"},
		{"func f(a int,)", "func f(a int)"},
		{"func f(a int,\n)", "func f(a int)"},
		{"func f(\n\ta int,\n\tb int,\n\tc int,\n)", "func f(a int, b int, c int)"},
		{"  (   a,   b,  c  )  ", "(a, b, c)"},
		{"(  a,  b, c    int, foo   bar  ,  )", "(a, b, c int, foo bar)"},
		{"{   a,   b}", "{a, b}"},
		{"[   a,   b]", "[a, b]"},
	} {
		if got := sanitizeFunc(tc.src); got != tc.want {
			t.Errorf("sanitizeFunc(%v) = %v; want %v", tc.src, got, tc.want)
		}
	}
}