package template

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Engine evaluates template strings.
type Engine struct {
	vars map[string]string
	seq  int
}

// NewEngine creates a new template engine.
func NewEngine() *Engine {
	return &Engine{vars: make(map[string]string), seq: 0}
}

// Evaluate replaces all placeholders in s with generated values.
func (e *Engine) Evaluate(s string) (string, error) {
	var lastErr error
	var result strings.Builder
	i := 0
	for i < len(s) {
		idx := strings.Index(s[i:], "${")
		if idx < 0 {
			result.WriteString(s[i:])
			break
		}
		result.WriteString(s[i : i+idx])
		start := i + idx
		// find matching }
		end := start + 2
		depth := 1
		for end < len(s) && depth > 0 {
			if s[end] == '{' && end+1 < len(s) && s[end-1] == '$' {
				depth++
			} else if s[end] == '{' {
				// check if preceded by $
				if end > 0 && s[end-1] == '$' {
					depth++
				}
			} else if s[end] == '}' {
				depth--
				if depth == 0 {
					break
				}
			}
			end++
		}
		if depth != 0 {
			lastErr = fmt.Errorf("unmatched placeholder bracket")
			result.WriteString(s[start:])
			break
		}
		inner := s[start+2 : end]
		val, err := e.evalPlaceholder(inner)
		if err != nil {
			lastErr = err
			result.WriteString(s[start : end+1])
		} else {
			result.WriteString(val)
		}
		i = end + 1
	}
	return result.String(), lastErr
}

// NextSeq returns the next sequence number.
func (e *Engine) NextSeq() int {
	e.seq++
	return e.seq
}

func (e *Engine) evalPlaceholder(inner string) (string, error) {
	// Variable assignment: var:name=...
	if strings.HasPrefix(inner, "var:") {
		rest := inner[4:]
		idx := strings.Index(rest, "=")
		if idx >= 0 {
			name := rest[:idx]
			expr := rest[idx+1:]
			val, err := e.Evaluate(expr)
			if err != nil {
				return "", err
			}
			e.vars[name] = val
			return val, nil
		}
		// Variable read
		if v, ok := e.vars[rest]; ok {
			return v, nil
		}
		return "", fmt.Errorf("undefined variable: %s", rest)
	}

	// Number range: 11000-11099 or 001-999
	if idx := strings.Index(inner, "-"); idx > 0 {
		left := inner[:idx]
		right := inner[idx+1:]
		// Not a date/time placeholder
		if !strings.HasPrefix(inner, "date:") && !strings.HasPrefix(inner, "datetime:") {
			return e.evalRange(left, right)
		}
	}

	// Enumeration: A|B|C
	if strings.Contains(inner, "|") {
		return e.evalEnum(inner)
	}

	// Named placeholders
	switch {
	case inner == "uuid":
		return uuid.New().String(), nil
	case inner == "timestamp":
		return strconv.FormatInt(time.Now().Unix(), 10), nil
	case inner == "seq":
		return strconv.Itoa(e.NextSeq()), nil
	case strings.HasPrefix(inner, "number:"):
		return e.evalNumber(inner[7:])
	case strings.HasPrefix(inner, "alpha:"):
		return e.evalAlpha(inner[6:])
	case strings.HasPrefix(inner, "alnum:"):
		return e.evalAlnum(inner[6:])
	case strings.HasPrefix(inner, "date:"):
		return e.evalDate(inner[5:])
	case strings.HasPrefix(inner, "datetime:"):
		return e.evalDateTime(inner[9:])
	}

	return "", fmt.Errorf("unknown placeholder: ${%s}", inner)
}

func (e *Engine) evalRange(left, right string) (string, error) {
	if left == "" || right == "" {
		return "", fmt.Errorf("invalid range: %s-%s", left, right)
	}
	width := 0
	if len(left) > 1 && left[0] == '0' {
		width = len(left)
	}
	lo, err := strconv.ParseInt(left, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid range lower bound: %s", left)
	}
	hi, err := strconv.ParseInt(right, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid range upper bound: %s", right)
	}
	if lo > hi {
		return "", fmt.Errorf("range lower bound greater than upper: %s-%s", left, right)
	}
	diff := hi - lo + 1
	nBig, err := rand.Int(rand.Reader, big.NewInt(diff))
	if err != nil {
		return "", err
	}
	n := lo + nBig.Int64()
	if width > 0 {
		return fmt.Sprintf("%0*d", width, n), nil
	}
	return strconv.FormatInt(n, 10), nil
}

func (e *Engine) evalEnum(inner string) (string, error) {
	parts := strings.Split(inner, "|")
	if len(parts) == 0 {
		return "", fmt.Errorf("empty enumeration")
	}
	for _, p := range parts {
		if strings.TrimSpace(p) == "" {
			return "", fmt.Errorf("enumeration contains empty item")
		}
	}
	idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(parts))))
	if err != nil {
		return "", err
	}
	return parts[idx.Int64()], nil
}

func (e *Engine) evalNumber(spec string) (string, error) {
	n, err := strconv.Atoi(spec)
	if err != nil || n <= 0 {
		return "", fmt.Errorf("invalid number length: %s", spec)
	}
	const digits = "0123456789"
	return randomString(n, digits), nil
}

func (e *Engine) evalAlpha(spec string) (string, error) {
	n, err := strconv.Atoi(spec)
	if err != nil || n <= 0 {
		return "", fmt.Errorf("invalid alpha length: %s", spec)
	}
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	return randomString(n, letters), nil
}

func (e *Engine) evalAlnum(spec string) (string, error) {
	n, err := strconv.Atoi(spec)
	if err != nil || n <= 0 {
		return "", fmt.Errorf("invalid alnum length: %s", spec)
	}
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	return randomString(n, chars), nil
}

func (e *Engine) evalDate(layout string) (string, error) {
	if layout == "" {
		layout = "yyyyMMdd"
	}
	layout = convertLayout(layout)
	return time.Now().Format(layout), nil
}

func (e *Engine) evalDateTime(layout string) (string, error) {
	if layout == "" {
		layout = "yyyyMMddHHmmss"
	}
	layout = convertLayout(layout)
	return time.Now().Format(layout), nil
}

func convertLayout(l string) string {
	l = strings.ReplaceAll(l, "yyyy", "2006")
	l = strings.ReplaceAll(l, "MM", "01")
	l = strings.ReplaceAll(l, "dd", "02")
	l = strings.ReplaceAll(l, "HH", "15")
	l = strings.ReplaceAll(l, "mm", "04")
	l = strings.ReplaceAll(l, "ss", "05")
	return l
}

func randomString(n int, charset string) string {
	b := make([]byte, n)
	max := big.NewInt(int64(len(charset)))
	for i := range b {
		r, err := rand.Int(rand.Reader, max)
		if err != nil {
			b[i] = charset[0]
			continue
		}
		b[i] = charset[r.Int64()]
	}
	return string(b)
}

// Preview generates sample values for a template string.
func Preview(s string, count int) []string {
	results := make([]string, count)
	for i := 0; i < count; i++ {
		eng := NewEngine()
		v, err := eng.Evaluate(s)
		if err != nil {
			results[i] = "[error: " + err.Error() + "]"
			continue
		}
		results[i] = v
	}
	return results
}

// Validate checks if a template string has valid syntax.
func Validate(s string) error {
	eng := NewEngine()
	_, err := eng.Evaluate(s)
	return err
}
