package utils

import (
	"testing"

	"gotest.tools/assert"
)

func Test_removePort(t *testing.T) {
	cases := []struct {
		I string
		O string
	}{
		{"127.0.0.1", "127.0.0.1"},
		{"1.1.1.1:80", "1.1.1.1"},
		{"1.1.1.1:80", "1.1.1.1"},
		{"[2002::]:10", "[2002::]"},
		{"[2002::]", "[2002::]"},
	}

	for _, c := range cases {
		assert.Equal(t, c.O, RemovePort(c.I))
	}
}
