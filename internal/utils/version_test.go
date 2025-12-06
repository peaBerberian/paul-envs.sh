package utils

import "testing"

func TestIsCompatibleVersion(t *testing.T) {
	tests := []struct {
		base       string
		other      string
		want       bool
		name       string
		validBase  bool
		validOther bool
	}{
		{
			name:       "same version",
			base:       "3.2.1",
			other:      "3.2.1",
			want:       true,
			validBase:  true,
			validOther: true,
		},
		{
			name:       "base higher minor",
			base:       "3.2.0",
			other:      "3.1.9",
			want:       true,
			validBase:  true,
			validOther: true,
		},
		{
			name:       "base higher patch",
			base:       "3.2.5",
			other:      "3.2.1",
			want:       true,
			validBase:  true,
			validOther: true,
		},
		{
			name:       "base lower minor",
			base:       "3.1.0",
			other:      "3.2.0",
			want:       false,
			validBase:  true,
			validOther: true,
		},
		{
			name:       "base lower patch",
			base:       "3.2.1",
			other:      "3.2.5",
			want:       false,
			validBase:  true,
			validOther: true,
		},
		{
			name:       "different major (base lower)",
			base:       "2.5.0",
			other:      "3.0.0",
			want:       false,
			validBase:  true,
			validOther: true,
		},
		{
			name:       "different major (base higher)",
			base:       "4.0.0",
			other:      "3.9.9",
			want:       false,
			validBase:  true,
			validOther: true,
		},
		{
			name:       "invalid base version",
			base:       "invalid",
			other:      "3.2.1",
			want:       false, // we check error separately
			validBase:  false,
			validOther: true,
		},
		{
			name:       "invalid other version",
			base:       "3.2.1",
			other:      "bad",
			want:       false,
			validBase:  true,
			validOther: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseV, err := ParseVersion(tt.base)
			if !tt.validBase {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				return
			}
			otherV, err := ParseVersion(tt.other)
			if !tt.validOther {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				return
			}

			got := otherV.IsCompatibleWithBase(baseV)
			if got != tt.want {
				t.Fatalf("Compatible(%q, %q) = %v, want %v",
					tt.base, tt.other, got, tt.want)
			}
		})
	}
}
