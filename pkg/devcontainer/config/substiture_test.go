package config

import (
	"fmt"
	"testing"
)

func TestLookupValue(t *testing.T) {
	tests := []struct {
		args  []string
		match string
		want  string
	}{
		{args: []string{}, match: "", want: ""},
		{args: []string{"foz"}, match: "${env.foz}", want: ""},
		{args: []string{"foo", "biz"}, match: "${env.foo:biz}", want: "bar"},
		{args: []string{"baz", "bar"}, match: "${env.baz:bar}", want: "bar"},
		{args: []string{"baz", "biz", "buz"}, match: "${env.baz:biz:buz}", want: "biz:buz"},
	}

	localVar := map[string]string{
		"foo": "bar",
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("#%d", i), func(t *testing.T) {
			if got := lookupValue(false, localVar, tt.args, tt.match); got != tt.want {
				t.Errorf("lookupValue(%v, %v, %v, %v) = %v, want %v", false, localVar, tt.args, tt.match, got, tt.want)
			}
		})
	}
}
