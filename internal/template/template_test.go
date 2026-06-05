package template

import (
	"strings"
	"testing"
)

func TestRange(t *testing.T) {
	eng := NewEngine()
	v, err := eng.Evaluate("SP${11000-11099}")
	if err != nil {
		t.Fatalf("evaluate error: %v", err)
	}
	if !strings.HasPrefix(v, "SP") {
		t.Fatalf("expected prefix SP, got %s", v)
	}
	num := strings.TrimPrefix(v, "SP")
	if len(num) != 5 {
		t.Fatalf("expected 5 digit number, got %s", num)
	}
}

func TestRangeLeadingZero(t *testing.T) {
	eng := NewEngine()
	v, err := eng.Evaluate("SHOP-${0001-9999}")
	if err != nil {
		t.Fatalf("evaluate error: %v", err)
	}
	if !strings.HasPrefix(v, "SHOP-") {
		t.Fatalf("expected prefix SHOP-, got %s", v)
	}
	num := strings.TrimPrefix(v, "SHOP-")
	if len(num) != 4 {
		t.Fatalf("expected 4 digits, got %s", num)
	}
}

func TestEnum(t *testing.T) {
	eng := NewEngine()
	v, err := eng.Evaluate("${dev|test|stage}")
	if err != nil {
		t.Fatalf("evaluate error: %v", err)
	}
	if v != "dev" && v != "test" && v != "stage" {
		t.Fatalf("unexpected enum value: %s", v)
	}
}

func TestNumber(t *testing.T) {
	eng := NewEngine()
	v, err := eng.Evaluate("${number:6}")
	if err != nil {
		t.Fatalf("evaluate error: %v", err)
	}
	if len(v) != 6 {
		t.Fatalf("expected 6 digits, got %s (%d)", v, len(v))
	}
}

func TestAlpha(t *testing.T) {
	eng := NewEngine()
	v, err := eng.Evaluate("${alpha:8}")
	if err != nil {
		t.Fatalf("evaluate error: %v", err)
	}
	if len(v) != 8 {
		t.Fatalf("expected 8 chars, got %s", v)
	}
}

func TestAlnum(t *testing.T) {
	eng := NewEngine()
	v, err := eng.Evaluate("${alnum:10}")
	if err != nil {
		t.Fatalf("evaluate error: %v", err)
	}
	if len(v) != 10 {
		t.Fatalf("expected 10 chars, got %s", v)
	}
}

func TestUUID(t *testing.T) {
	eng := NewEngine()
	v, err := eng.Evaluate("${uuid}")
	if err != nil {
		t.Fatalf("evaluate error: %v", err)
	}
	if len(v) != 36 {
		t.Fatalf("expected uuid length 36, got %d", len(v))
	}
}

func TestTimestamp(t *testing.T) {
	eng := NewEngine()
	v, err := eng.Evaluate("${timestamp}")
	if err != nil {
		t.Fatalf("evaluate error: %v", err)
	}
	if v == "" {
		t.Fatal("expected non-empty timestamp")
	}
}

func TestDate(t *testing.T) {
	eng := NewEngine()
	v, err := eng.Evaluate("${date:yyyyMMdd}")
	if err != nil {
		t.Fatalf("evaluate error: %v", err)
	}
	if len(v) != 8 {
		t.Fatalf("expected 8 chars date, got %s", v)
	}
}

func TestSeq(t *testing.T) {
	eng := NewEngine()
	v1, _ := eng.Evaluate("${seq}")
	v2, _ := eng.Evaluate("${seq}")
	if v1 != "1" {
		t.Fatalf("expected seq 1, got %s", v1)
	}
	if v2 != "2" {
		t.Fatalf("expected seq 2, got %s", v2)
	}
}

func TestVariableReuse(t *testing.T) {
	eng := NewEngine()
	v1, err := eng.Evaluate("${var:user=SP${11000-11099}}")
	if err != nil {
		t.Fatalf("assign error: %v", err)
	}
	v2, err := eng.Evaluate("${var:user}")
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if v1 != v2 {
		t.Fatalf("variable reuse failed: %s != %s", v1, v2)
	}
}

func TestInvalidRange(t *testing.T) {
	eng := NewEngine()
	_, err := eng.Evaluate("${100-50}")
	if err == nil {
		t.Fatal("expected error for invalid range")
	}
}

func TestUndefinedVar(t *testing.T) {
	eng := NewEngine()
	_, err := eng.Evaluate("${var:undefined}")
	if err == nil {
		t.Fatal("expected error for undefined variable")
	}
}

func TestPreview(t *testing.T) {
	results := Preview("${number:4}", 5)
	if len(results) != 5 {
		t.Fatalf("expected 5 previews, got %d", len(results))
	}
	for _, r := range results {
		if len(r) != 4 {
			t.Fatalf("expected 4 digit preview, got %s", r)
		}
	}
}

func TestValidate(t *testing.T) {
	if err := Validate("${number:4}"); err != nil {
		t.Fatalf("unexpected validate error: %v", err)
	}
	if err := Validate("${bad}"); err == nil {
		t.Fatal("expected validate error")
	}
}
