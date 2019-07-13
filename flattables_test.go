package flattables

import (
	"go/token"
	"testing"
)

func TestIsGoKeyword(t *testing.T) {
	var tests = []struct {
		maybeKeyword string
		isKeyword    bool
	}{
		{"break", true},
		{"case", true},
		{"chan", true},
		{"const", true},
		{"continue", true},
		{"default", true},
		{"defer", true},
		{"else", true},
		{"fallthrough", true},
		{"for", true},
		{"func", true},
		{"go", true},
		{"goto", true},
		{"if", true},
		{"import", true},
		{"interface", true},
		{"map", true},
		{"package", true},
		{"range", true},
		{"return", true},
		{"select", true},
		{"struct", true},
		{"switch", true},
		{"type", true},
		{"var", true},
		{"int", false},
	}

	for i, test := range tests {
		var result bool = token.Lookup(test.maybeKeyword).IsKeyword()

		if result != test.isKeyword && test.isKeyword == true {
			t.Errorf("test[%d] expected %s to be a Go keyword, but it's not", i, test.maybeKeyword)
		}

		if result != test.isKeyword && test.isKeyword == false {
			t.Errorf("test[%d] expected %s to NOT be a Go keyword, but it is", i, test.maybeKeyword)
		}
	}
}
