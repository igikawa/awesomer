package tui

import (
	"strings"
	"testing"
)

func TestParseCPUCores(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []int
		wantErr string
	}{
		{
			name:  "normal values are parsed and sorted",
			input: "3,1,2",
			want:  []int{1, 2, 3},
		},
		{
			name:  "duplicates are removed",
			input: "2,2,1",
			want:  []int{1, 2},
		},
		{
			name:    "negative values are rejected",
			input:   "0,-1",
			wantErr: "must be >=",
		},
		{
			name:    "non numeric values are rejected",
			input:   "0,a",
			wantErr: "invalid CPU core",
		},
		{
			name:    "empty input is rejected",
			input:   " , ",
			wantErr: "no CPU cores",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCPUCores(tt.input)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("parseCPUCores() error = nil, want %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("parseCPUCores() error = %q, want substring %q", err.Error(), tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseCPUCores() error = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("parseCPUCores() len = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("parseCPUCores()[%d] = %d, want %d", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestIntsToCSV(t *testing.T) {
	got := intsToCSV([]int{0, 2, 7})
	if got != "0,2,7" {
		t.Fatalf("intsToCSV() = %q, want %q", got, "0,2,7")
	}
}
