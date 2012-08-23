// Copyright 2011 Google Inc. All Rights Reserved.
// This file is available under the Apache license.

package main

import (
	"reflect"
	"strings"
	"testing"
)

type in_out struct {
	input string // no newlines
	ok    bool
}

type testProgram struct {
	name   string
	source string
	prog   []instr // expected bytecode 

	io []in_out // multiple test data and expected outupt per program
}

var programs = []testProgram{
	{"simple line counter",
		"counter line_count\n/$/ { line_count++ }",
		[]instr{
			instr{match, 0},
			instr{jnm, 5},
			instr{mload, 0},
			instr{inc, 0},
			instr{ret, 0}},
		[]in_out{in_out{"", true}}},
	{"count a",
		"counter a_count\n/a$/ { a_count++ }",
		[]instr{
			instr{match, 0},
			instr{jnm, 5},
			instr{mload, 0},
			instr{inc, 0},
			instr{ret, 0}},
		[]in_out{
			in_out{"", false},
			in_out{"a", true}}},
	{"strptime and capref",
		"counter foo\n" +
			"/(.*)/ { strptime($1, \"2006-01-02T15:04:05\")" +
			"foo++ }",
		[]instr{
			instr{match, 0},
			instr{jnm, 9},
			instr{push, 0},
			instr{capref, 1},
			instr{str, 0},
			instr{strptime, 2},
			instr{mload, 0},
			instr{inc, 0},
			instr{ret, 0}},
		[]in_out{
			in_out{"2006-01-02T15:04:05", true}}},
	{"strptime and named capref",
		"counter foo\n" +
			"/(?P<date>.*)/ { strptime($date, \"2006-01-02T15:04:05\")" +
			"foo++ }",
		[]instr{
			instr{match, 0},
			instr{jnm, 9},
			instr{push, 0},
			instr{capref, 1},
			instr{str, 0},
			instr{strptime, 2},
			instr{mload, 0},
			instr{inc, 0},
			instr{ret, 0}},
		[]in_out{
			in_out{"2006-01-02T15:04:05", true}}},
	{"inc by and set",
		"counter foo\ncounter bar\n" +
			"/(.*)/ {\n" +
			"foo += $1\n" +
			"bar = $1\n" +
			"}",
		[]instr{
			instr{match, 0},
			instr{jnm, 11},
			instr{mload, 0},
			instr{push, 0},
			instr{capref, 1},
			instr{inc, 1},
			instr{mload, 1},
			instr{push, 0},
			instr{capref, 1},
			instr{set, 0},
			instr{ret, 0}},
		[]in_out{
			in_out{"37", true}}},
	{"cond expr gt",
		"counter foo\n" +
			"1 > 0 {\n" +
			"  foo++\n" +
			"}\n",
		[]instr{
			instr{push, 1},
			instr{push, 0},
			instr{cmp, 1},
			instr{jnm, 7},
			instr{mload, 0},
			instr{inc, 0},
			instr{ret, 0}},
		[]in_out{
			in_out{"", true}}},
	{"cond expr lt",
		"counter foo\n" +
			"1 < 0 {\n" +
			"  foo++\n" +
			"}\n",
		[]instr{
			instr{push, 1},
			instr{push, 0},
			instr{cmp, -1},
			instr{jnm, 7},
			instr{mload, 0},
			instr{inc, 0},
			instr{ret, 0}},
		[]in_out{
			in_out{"", false}}},
	{"cond expr eq",
		"counter foo\n" +
			"1 == 0 {\n" +
			"  foo++\n" +
			"}\n",
		[]instr{
			instr{push, 1},
			instr{push, 0},
			instr{cmp, 0},
			instr{jnm, 7},
			instr{mload, 0},
			instr{inc, 0},
			instr{ret, 0}},
		[]in_out{
			in_out{"", false}}},
	{"cond expr le",
		"counter foo\n" +
			"1 <= 0 {\n" +
			"  foo++\n" +
			"}\n",
		[]instr{
			instr{push, 1},
			instr{push, 0},
			instr{cmp, 1},
			instr{jm, 7},
			instr{mload, 0},
			instr{inc, 0},
			instr{ret, 0}},
		[]in_out{
			in_out{"", false}}},
	{"cond expr ge",
		"counter foo\n" +
			"1 >= 0 {\n" +
			"  foo++\n" +
			"}\n",
		[]instr{
			instr{push, 1},
			instr{push, 0},
			instr{cmp, -1},
			instr{jm, 7},
			instr{mload, 0},
			instr{inc, 0},
			instr{ret, 0}},
		[]in_out{
			in_out{"", true}}},
	{"cond expr ne",
		"counter foo\n" +
			"1 != 0 {\n" +
			"  foo++\n" +
			"}\n",
		[]instr{
			instr{push, 1},
			instr{push, 0},
			instr{cmp, 0},
			instr{jm, 7},
			instr{mload, 0},
			instr{inc, 0},
			instr{ret, 0}},
		[]in_out{
			in_out{"", true}}},
	{"nested cond",
		"counter foo\n" +
			"/(.*)/ {\n" +
			"  $1 <= 1 {\n" +
			"    foo++\n" +
			"  }\n" +
			"}",
		[]instr{
			instr{match, 0},
			instr{jnm, 10},
			instr{push, 0},
			instr{capref, 1},
			instr{push, 1},
			instr{cmp, 1},
			instr{jm, 10},
			instr{mload, 0},
			instr{inc, 0},
			instr{ret, 0},
		},
		[]in_out{
			in_out{"0", true},
			in_out{"1", true},
			in_out{"2", false}}},
}

func TestCompileAndRun(t *testing.T) {
	for _, tc := range programs {
		v, err := Compile(tc.name, strings.NewReader(tc.source))
		if err != nil {
			t.Errorf("Compile errors: %q", err)
			continue
		}
		if !reflect.DeepEqual(tc.prog, v.prog) {
			t.Errorf("%s: VM prog doesn't match.\n\texpected: %v\n\treceived: %v",
				tc.name, tc.prog, v.prog)
		}
		for _, i := range tc.io {
			r := v.Run(i.input)
			if r != i.ok {
				t.Errorf("%s: Unexpected result after running on test input %q\n\tprog: %v\n\texpected: %v\n\treceived: %v",
					tc.name, i.input, v.prog, i.ok, r)
			}
		}
	}
}
