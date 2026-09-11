package lib

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedactURI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "amqp with credentials",
			input: "amqps://user:secret@broker:5672",
			want:  "amqps://redacted:redacted@broker:5672",
		},
		{
			name:  "http with credentials and path",
			input: "http://admin:pass@loki:3100/loki/api/v1/push",
			want:  "http://redacted:redacted@loki:3100/loki/api/v1/push",
		},
		{
			name:  "username only",
			input: "amqps://user@broker:5672",
			want:  "amqps://redacted@broker:5672",
		},
		{
			name:  "no credentials",
			input: "http://localhost:3100",
			want:  "http://localhost:3100",
		},
		{
			name:  "invalid uri",
			input: "://bad",
			want:  "redacted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, RedactURI(tt.input))
		})
	}
}
