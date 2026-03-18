package exploration

import "testing"

func TestParseWorkspaceID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    uint
		wantErr bool
	}{
		{name: "valid", input: "42", want: 42},
		{name: "trims spaces", input: " 7 ", want: 7},
		{name: "rejects zero", input: "0", wantErr: true},
		{name: "rejects text", input: "ws-1", wantErr: true},
		{name: "rejects empty", input: " ", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseWorkspaceID(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for input %q", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, got)
			}
		})
	}
}
