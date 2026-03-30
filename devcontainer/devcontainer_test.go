package devcontainer

import "testing"

func TestHasNoCdrOption(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{
			name: "with nocdr option",
			args: []string{"down", "--nocdr", "/workspace"},
			want: true,
		},
		{
			name: "without nocdr option",
			args: []string{"down", "/workspace"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasNoCdrOption(tt.args)
			if got != tt.want {
				t.Fatalf("hasNoCdrOption(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
