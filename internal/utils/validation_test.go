package utils

import "testing"

func TestValidateProjectName(t *testing.T) {
	tests := []struct {
		name string
		ok   bool
	}{
		{"proj", true},
		{"_proj-123", true},
		{"", false},
		{"-bad", false},
		{"a" + string(make([]byte, 128)), false}, // too long
	}

	for _, tt := range tests {
		err := ValidateProjectName(tt.name)
		if tt.ok && err != nil {
			t.Errorf("expected ok for %q, got %v", tt.name, err)
		}
		if !tt.ok && err == nil {
			t.Errorf("expected error for %q", tt.name)
		}
	}
}

func TestValidateVersionArg(t *testing.T) {
	tests := []struct {
		in string
		ok bool
	}{
		{"", true},
		{"latest", true},
		{"none", true},
		{"1.2.3", true},
		{"1.2", false},
		{"bad", false},
	}

	for _, tt := range tests {
		err := ValidateVersionArg(tt.in)
		if tt.ok && err != nil {
			t.Errorf("expected ok for %q, got %v", tt.in, err)
		}
		if !tt.ok && err == nil {
			t.Errorf("expected error for %q", tt.in)
		}
	}
}

func TestValidateUIDGID(t *testing.T) {
	tests := []struct {
		id     string
		ok     bool
		idType string
	}{
		{"0", true, "uid"},
		{"65535", true, "gid"},
		{"-1", false, "uid"},
		{"70000", false, "gid"},
		{"abc", false, "uid"},
	}

	for _, tt := range tests {
		err := ValidateUIDGID(tt.id)
		if tt.ok && err != nil {
			t.Errorf("expected ok for %q, got %v", tt.id, err)
		}
		if !tt.ok && err == nil {
			t.Errorf("expected error for %q", tt.id)
		}
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		in string
		ok bool
	}{
		{"user", true},
		{"_user1", true},
		{"User", false},
		{"u$er", false},
		{string(make([]byte, 33)), false},
	}

	for _, tt := range tests {
		err := ValidateUsername(tt.in)
		if tt.ok && err != nil {
			t.Errorf("expected ok for %q, got %v", tt.in, err)
		}
		if !tt.ok && err == nil {
			t.Errorf("expected error for %q", tt.in)
		}
	}
}

func TestValidatePort(t *testing.T) {
	tests := []struct {
		port int
		ok   bool
	}{
		{1, true},
		{65535, true},
		{0, false},
		{70000, false},
	}

	for _, tt := range tests {
		err := ValidatePort(tt.port)
		if tt.ok && err != nil {
			t.Errorf("expected ok for %d, got %v", tt.port, err)
		}
		if !tt.ok && err == nil {
			t.Errorf("expected error for %d", tt.port)
		}
	}
}

func TestValidateGitName(t *testing.T) {
	tests := []struct {
		in string
		ok bool
	}{
		{"John Doe", true},
		{"Bad\nName", false},
		{string(make([]byte, 101)), false},
	}

	for _, tt := range tests {
		err := ValidateGitName(tt.in)
		if tt.ok && err != nil {
			t.Errorf("expected ok for %q, got %v", tt.in, err)
		}
		if !tt.ok && err == nil {
			t.Errorf("expected error for %q", tt.in)
		}
	}
}

func TestValidateGitEmail(t *testing.T) {
	tests := []struct {
		in string
		ok bool
	}{
		{"test@example.com", true},
		{"bad@", false},
		{"@bad.com", false},
	}

	for _, tt := range tests {
		err := ValidateGitEmail(tt.in)
		if tt.ok && err != nil {
			t.Errorf("expected ok for %q, got %v", tt.in, err)
		}
		if !tt.ok && err == nil {
			t.Errorf("expected error for %q", tt.in)
		}
	}
}

func TestEscapeEnvValue(t *testing.T) {
	in := "abc\n\"$\\"
	got := EscapeEnvValue(in)
	want := `abc\"\$\\`
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestIsValidUbuntuPackageName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid names
		{"simple", "bash", true},
		{"library", "libssl3", true},
		{"with-dot", "python3.12", true},
		{"with-plus", "g++", true},
		{"with-mixed", "pkg-name+extra", true},
		{"starts-with-digit", "1package", true},
		{"max-length", generateString('a', 255), true},

		// Invalid names
		{"empty", "", false},
		{"one-char", "a", false},
		{"uppercase", "Bash", false},
		{"invalid-char", "name!", false},
		{"invalid-symbol", "foo=bar", false},
		{"bad-start", "-bad", false},
		{"too-long", generateString('a', 256), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidUbuntuPackageName(tt.input)
			if got != tt.expected {
				t.Errorf("IsValidUbuntuPackageName(%q) = %v, expected %v",
					tt.input, got, tt.expected)
			}
		})
	}
}

// generates strings of repeated char c, length n
func generateString(c rune, n int) string {
	r := make([]rune, n)
	for i := range n { // Go 1.22+: range over an integer 0..n-1
		r[i] = c
	}
	return string(r)
}
