package console_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/peaberberian/paul-envs/internal/console"
)

func newTestConsole(input string) (*console.Console, *bytes.Buffer, *bytes.Buffer, *io.PipeWriter, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	r, w := io.Pipe()
	if input != "" {
		go func() {
			fmt.Fprint(w, input) // write the input
			w.Close()
		}()
	}
	out := &bytes.Buffer{}
	err := &bytes.Buffer{}
	return console.New(ctx, r, out, err), out, err, w, cancel
}

func TestAskYesNo_DefaultYes(t *testing.T) {
	c, _, _, _, cancel := newTestConsole("\n")
	defer cancel()

	val, err := c.AskYesNo("Continue?", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != true {
		t.Fatalf("expected default YES, got %v", val)
	}
}

func TestAskYesNo_DefaultNo(t *testing.T) {
	c, _, _, _, cancel := newTestConsole("\n")
	defer cancel()

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
		c, _, _, _, cancel := newTestConsole(in)
		defer cancel()

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
		c, _, _, _, cancel := newTestConsole(in)
		defer cancel()

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
	c, _, _, _, cancel := newTestConsole("maybe\n")
	defer cancel()

	_, err := c.AskYesNo("OK?", true)
	if err == nil {
		t.Fatalf("expected error for invalid input")
	}
}

func TestAskYesNo_Cancelled(t *testing.T) {
	c, _, _, pipe, cancel := newTestConsole("")
	// simulate Ctrl+C by cancelling the context
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
		pipe.Close()
	}()

	_, err := c.AskYesNo("Continue?", true)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled error, got %v", err)
	}
}

func TestAskString_ReturnsInput(t *testing.T) {
	c, _, _, _, cancel := newTestConsole("hello\n")
	defer cancel()

	val, err := c.AskString("Name?", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "hello" {
		t.Fatalf("expected 'hello', got %q", val)
	}
}

func TestAskString_DefaultUsed(t *testing.T) {
	c, _, _, _, cancel := newTestConsole("\n")
	defer cancel()

	val, err := c.AskString("Name?", "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "default" {
		t.Fatalf("expected default, got %q", val)
	}
}

func TestAskString_Cancelled(t *testing.T) {
	c, _, _, pipe, cancel := newTestConsole("")
	// simulate Ctrl+C by cancelling the context
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
		pipe.Close()
	}()

	_, err := c.AskString("Your input?", "test")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled error, got %v", err)
	}
}
