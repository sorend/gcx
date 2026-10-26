package terminal //nolint:testpackage // Exercises the unexported DA1 reply parser.

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDAReplyHasSixel(t *testing.T) {
	tests := []struct {
		name  string
		reply string
		want  bool
	}{
		{name: "sixel listed", reply: "\x1b[?62;4;6c", want: true},
		{name: "sixel only", reply: "\x1b[?4c", want: true},
		{name: "sixel listed last", reply: "\x1b[?64;1;22;4c", want: true},
		{name: "sixel absent", reply: "\x1b[?62;6c", want: false},
		{name: "similar capability is not sixel", reply: "\x1b[?62;14c", want: false},
		{name: "no reply", reply: "", want: false},
		{name: "unrelated bytes", reply: "garbage", want: false},
		{name: "unterminated reply", reply: "\x1b[?62;4", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, daReplyHasSixel([]byte(tt.reply)))
		})
	}
}
