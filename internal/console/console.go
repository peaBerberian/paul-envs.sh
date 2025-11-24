package console

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

const (
	// ANSI color codes
	// red        = "\033[0;31m"
	green      = "\033[0;32m"
	yellow     = "\033[1;33m"
	blue       = "\033[0;34m"
	colorReset = "\033[0m"
)

type Console struct {
	reader    *bufio.Reader
	writer    io.Writer
	errWriter io.Writer
	ctx       context.Context
}

func New(ctx context.Context, rd io.Reader, w io.Writer, ew io.Writer) *Console {
	return &Console{
		reader:    bufio.NewReader(rd),
		writer:    w,
		errWriter: ew,
		ctx:       ctx,
	}
}

func (c *Console) Error(format string, args ...any) {
	fmt.Fprintf(c.errWriter, // red+
		format+colorReset+"\n", args...)
}

func (c *Console) Success(format string, args ...any) {
	fmt.Fprintf(c.writer, green+format+colorReset+"\n", args...)
}

func (c *Console) Warn(format string, args ...any) {
	fmt.Fprintf(c.writer, yellow+format+colorReset+"\n", args...)
}

func (c *Console) Info(format string, args ...any) {
	fmt.Fprintf(c.writer, blue+format+colorReset+"\n", args...)
}

func (c *Console) WriteLn(format string, args ...any) {
	fmt.Fprintf(c.writer, format+"\n", args...)
}

func (c *Console) AskYesNo(prompt string, defaultVal bool) (bool, error) {
	for {
		if defaultVal {
			fmt.Fprintf(c.writer, "%s (Y/n): ", prompt)
		} else {
			fmt.Fprintf(c.writer, "%s (y/N): ", prompt)
		}

		inputCh := make(chan string, 1)
		errCh := make(chan error, 1)

		go func() {
			input, err := bufio.NewReader(c.reader).ReadString('\n')
			if err != nil {
				errCh <- err
				return
			}
			inputCh <- input
		}()

		// Wait for either context cancellation or user input
		select {
		case <-c.ctx.Done():
			return false, c.ctx.Err()
		case err := <-errCh:
			return false, fmt.Errorf("failed to read input: %w", err)
		case input := <-inputCh:
			input = strings.TrimSpace(input)
			switch strings.ToLower(input) {
			case "":
				return defaultVal, nil
			case "y", "yes":
				return true, nil
			case "n", "no":
				return false, nil
			default:
				fmt.Fprintln(c.writer, "Please enter 'y' or 'n'.")
				// Loop again to re-prompt
			}
		}
	}
}

func (c *Console) AskString(prompt, defaultVal string) (string, error) {
	for {
		if defaultVal != "" {
			fmt.Fprintf(c.writer, "%s [%s]: ", prompt, defaultVal)
		} else {
			fmt.Fprintf(c.writer, "%s: ", prompt)
		}

		inputCh := make(chan string, 1)
		errCh := make(chan error, 1)

		go func() {
			input, err := bufio.NewReader(c.reader).ReadString('\n')
			if err != nil {
				errCh <- err
				return
			}
			inputCh <- input
		}()

		// Wait for either context cancellation or user input
		select {
		case <-c.ctx.Done():
			return "", c.ctx.Err()
		case err := <-errCh:
			return "", fmt.Errorf("failed to read input: %w", err)
		case input := <-inputCh:
			input = strings.TrimSpace(input)
			if input == "" {
				if defaultVal != "" {
					return defaultVal, nil
				}
				fmt.Fprintln(c.writer, "Input cannot be empty.")
				continue // re-prompt
			}
			return input, nil
		}
	}
}
