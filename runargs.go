package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const runargWhitespace = " \t\r\n"

// parseRunargs treats the file as Docker arguments. Expansion writes into the
// current argument so paths and environment values are never split or re-parsed.
func parseRunargs(input string) ([]string, error) {
	if strings.ContainsRune(input, 0) {
		return nil, errors.New("NUL byte in runargs")
	}
	var args []string
	var word strings.Builder
	var quote byte
	inWord := false
	for i := 0; i < len(input); i++ {
		c := input[i]
		switch {
		case quote != 0 && c == quote:
			quote = 0
		case quote == '\'':
			word.WriteByte(c)
		case quote == 0 && (c == '\'' || c == '"'):
			quote = c
		case quote == 0 && strings.ContainsRune(runargWhitespace, rune(c)):
			if inWord {
				args = append(args, word.String())
				word.Reset()
				inWord = false
			}
			continue
		case quote == 0 && !inWord && c == '#':
			end := strings.IndexByte(input[i:], '\n')
			if end < 0 {
				return args, nil
			}
			i += end
			continue
		case c == '\\':
			if i+1 == len(input) {
				return nil, errors.New("unfinished escape in runargs")
			}
			next := input[i+1]
			if quote == '"' && !strings.ContainsRune("$`\"\\\n", rune(next)) {
				word.WriteByte(c)
				break
			}
			i++
			if next == '\n' {
				continue
			}
			word.WriteByte(next)
		case c == '$':
			value, consumed, err := expandRunarg(input[i+1:])
			if err != nil {
				return nil, err
			}
			word.WriteString(value)
			i += consumed
		case quote == 0 && !inWord && c == '~' &&
			(i+1 == len(input) || strings.ContainsRune("/"+runargWhitespace, rune(input[i+1]))):
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("expand ~ in runargs: %w", err)
			}
			word.WriteString(home)
		case c == '`':
			return nil, errors.New("backtick command substitution is unsupported in runargs")
		case quote == 0 && strings.ContainsRune(";&|<>()", rune(c)):
			return nil, fmt.Errorf("unquoted shell operator %q is unsupported in runargs", c)
		default:
			word.WriteByte(c)
		}
		inWord = true
	}
	if quote != 0 {
		return nil, errors.New("unclosed quote in runargs")
	}
	if inWord {
		args = append(args, word.String())
	}
	return args, nil
}

// expandRunarg receives the text after '$' and reports how many bytes it used.
func expandRunarg(rest string) (string, int, error) {
	switch {
	case strings.HasPrefix(rest, "(pwd)"):
		cwd, err := os.Getwd()
		if err != nil {
			return "", 0, fmt.Errorf("expand $(pwd) in runargs: %w", err)
		}
		return cwd, len("(pwd)"), nil
	case strings.HasPrefix(rest, "("):
		return "", 0, errors.New("only $(pwd) is supported in runargs")
	case strings.HasPrefix(rest, "{"):
		name, _, closed := strings.Cut(rest[1:], "}")
		if !closed || name == "" || runargEnvNameLen(name) != len(name) {
			return "", 0, errors.New("invalid environment variable expansion in runargs")
		}
		return os.Getenv(name), len(name) + 2, nil
	default:
		length := runargEnvNameLen(rest)
		if length == 0 {
			return "$", 0, nil
		}
		return os.Getenv(rest[:length]), length, nil
	}
}

func runargEnvNameLen(input string) int {
	for i := 0; i < len(input); i++ {
		c := input[i]
		if c == '_' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || i > 0 && c >= '0' && c <= '9' {
			continue
		}
		return i
	}
	return len(input)
}
