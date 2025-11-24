package console_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/peaberberian/paul-envs/internal/console"
)

func newTestConsole(input string) (*console.Console, *bytes.Buffer, *bytes.Buffer) {
	in := strings.NewReader(input)
	out := &bytes.Buffer{}
	err := &bytes.Buffer{}
	return console.New(in, out, err), out, err
}

func TestAskYesNo_DefaultYes(t *testing.T) {
	c, _, _ := newTestConsole("\n")

	val, err := c.AskYesNo("Continue?", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != true {
		t.Fatalf("expected default YES, got %v", val)
	}
}

func TestAskYesNo_DefaultNo(t *testing.T) {
	c, _, _ := newTestConsole("\n")

	val, err := c.AskYesNo("Continue?", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != false {
		t.Fatalf("expected default NO, got %v", val)
	}
}

func TestAskYesNo_YesValues(t *testing.T) {
	inputs := []string{"y\n", "Y\n", "yes\n", "YES\n"}
	for _, in := range inputs {
		c, _, _ := newTestConsole(in)

		val, err := c.AskYesNo("OK?", false)
		if err != nil {
			t.Fatalf("unexpected error for input %q: %v", in, err)
		}
		if !val {
			t.Fatalf("expected YES for %q", in)
		}
	}
}

func TestAskYesNo_NoValues(t *testing.T) {
	inputs := []string{"n\n", "N\n", "no\n", "NO\n"}
	for _, in := range inputs {
		c, _, _ := newTestConsole(in)

		val, err := c.AskYesNo("OK?", true)
		if err != nil {
			t.Fatalf("unexpected error for input %q: %v", in, err)
		}
		if val {
			t.Fatalf("expected NO for %q", in)
		}
	}
}

func TestAskYesNo_InvalidInput(t *testing.T) {
	c, _, _ := newTestConsole("maybe\n")

	_, err := c.AskYesNo("OK?", true)
	if err == nil {
		t.Fatalf("expected error for invalid input")
	}
}

func TestAskString_ReturnsInput(t *testing.T) {
	c, _, _ := newTestConsole("hello\n")

	val, err := c.AskString("Name?", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "hello" {
		t.Fatalf("expected 'hello', got %q", val)
	}
}

func TestAskString_DefaultUsed(t *testing.T) {
	c, _, _ := newTestConsole("\n")

	val, err := c.AskString("Name?", "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "default" {
		t.Fatalf("expected default, got %q", val)
	}
}
