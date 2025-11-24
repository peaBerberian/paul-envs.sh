package console

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

const (
	// ANSI color codes
	red        = "\033[0;31m"
	green      = "\033[0;32m"
	yellow     = "\033[1;33m"
	blue       = "\033[0;34m"
	colorReset = "\033[0m"
)

type Console struct {
	reader    *bufio.Reader
	writer    io.Writer
	errWriter io.Writer
}

func New(rd io.Reader, w io.Writer, ew io.Writer) *Console {
	return &Console{
		reader:    bufio.NewReader(rd),
		writer:    w,
		errWriter: ew,
	}
}

func (c *Console) Error(format string, args ...any) {
	fmt.Fprintf(c.errWriter, red+format+colorReset+"\n", args...)
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

func (c *Console) AskYesNo(prompt string, defaultVal bool) (bool, error) {
	if defaultVal {
		fmt.Fprintf(c.writer, "%s (Y/n): ", prompt)
	} else {
		fmt.Fprintf(c.writer, "%s (y/N): ", prompt)
	}
	input, err := c.reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read user input: %w", err)
	}
	input = strings.TrimSpace(input)
	switch strings.ToLower(input) {
	case "":
		return defaultVal, nil
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid input %q", input)
	}
}

func (c *Console) AskString(prompt, defaultVal string) (string, error) {
	if defaultVal != "" {
		fmt.Fprintf(c.writer, "%s [%s]: ", prompt, defaultVal)
	} else {
		fmt.Fprintf(c.writer, "%s: ", prompt)
	}
	input, err := c.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read user input: %w", err)
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal, nil
	}
	return input, nil
}
