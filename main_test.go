package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrimTabs(t *testing.T) {
	for _, tc := range []struct {
		in     string
		column int
		want   string
	}{
		{
			in:     "Foo",
			column: 0,
			want:   "Foo",
		},
		{
			in:     "Foo",
			column: 1,
			want:   "Foo",
		},
		{
			in:     "\tFoo",
			column: 0,
			want:   "\tFoo",
		},
		{
			in:     "\tFoo",
			column: 1,
			want:   "Foo",
		},
		{
			in:     "\t\tFoo",
			column: 1,
			want:   "\tFoo",
		},
	} {
		t.Run(tc.in, func(t *testing.T) {
			got := trimTabs(tc.in, tc.column)

			assert.Equal(t, tc.want, got)
		})
	}
}
