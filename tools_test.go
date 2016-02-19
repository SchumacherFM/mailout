package mailout

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidEmail(t *testing.T) {
	t.Parallel()
	tests := []struct {
		have string
		want bool
	}{
		{"gopher@golang.org", true},
		{"gophergolang.org", false},
		{"", false},
		{"gopher.rust@golang.museum", true},
		{"gopher+rust@golang.travel.mil", true},
	}
	for _, test := range tests {
		assert.Exactly(t, isValidEmail(test.have), test.want, test.have)
	}
}
