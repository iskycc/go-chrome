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

func TestDateTimeDefaultLayout(t *testing.T) {
	eng := NewEngine()
	v, err := eng.Evaluate("${datetime:}")
	if err != nil {
		t.Fatalf("evaluate datetime: %v", err)
	}
	if len(v) != 14 {
		t.Fatalf("expected yyyyMMddHHmmss length, got %q", v)
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

func TestInvalidPlaceholderSyntaxPreservesText(t *testing.T) {
	eng := NewEngine()
	res, err := eng.EvaluateDetailed("prefix ${number:4")
	if err == nil {
		t.Fatal("expected unmatched placeholder error")
	}
	if res.Value != "prefix ${number:4" || res.MaskedValue != res.Value {
		t.Fatalf("unexpected preserved value: %+v", res)
	}
}

func TestInvalidTemplateBranches(t *testing.T) {
	tests := []string{
		"${-1}",
		"${abc-2}",
		"${2-abc}",
		"${dev||stage}",
		"${number:0}",
		"${alpha:bad}",
		"${alnum:-1}",
		"${env:}",
	}
	for _, tc := range tests {
		if _, err := NewEngineWithEnv(&testEnvProvider{}).Evaluate(tc); err == nil {
			t.Fatalf("expected error for %s", tc)
		}
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

func TestPreviewReturnsErrors(t *testing.T) {
	results := Preview("${bad}", 2)
	if len(results) != 2 {
		t.Fatalf("expected 2 previews, got %d", len(results))
	}
	for _, r := range results {
		if !strings.HasPrefix(r, "[error: ") {
			t.Fatalf("expected preview error, got %s", r)
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

type testEnvProvider struct {
	data    map[string]string
	secrets map[string]bool
}

func (p *testEnvProvider) GetEnvValue(key string) (string, bool, bool) {
	v, ok := p.data[key]
	return v, ok, p.secrets[key]
}

func TestEnvVariable(t *testing.T) {
	eng := NewEngineWithEnv(&testEnvProvider{data: map[string]string{"BASE_URL": "https://test.com"}})
	v, err := eng.Evaluate("${env:BASE_URL}/login")
	if err != nil {
		t.Fatalf("evaluate error: %v", err)
	}
	if v != "https://test.com/login" {
		t.Errorf("expected https://test.com/login, got %s", v)
	}
}

func TestEnvVariableMissing(t *testing.T) {
	eng := NewEngineWithEnv(&testEnvProvider{data: map[string]string{}})
	_, err := eng.Evaluate("${env:MISSING}")
	if err == nil {
		t.Fatal("expected error for missing env var")
	}
}

func TestEnvVariableWithoutProvider(t *testing.T) {
	if _, err := NewEngine().Evaluate("${env:BASE_URL}"); err == nil {
		t.Fatal("expected no environment selected error")
	}
}

func TestEnvVariableSecretMetadata(t *testing.T) {
	eng := NewEngineWithEnv(&testEnvProvider{
		data:    map[string]string{"PASSWORD": "super-secret-value"},
		secrets: map[string]bool{"PASSWORD": true},
	})
	res, err := eng.EvaluateDetailed("pw=${env:PASSWORD}")
	if err != nil {
		t.Fatalf("evaluate error: %v", err)
	}
	if res.Value != "pw=super-secret-value" {
		t.Fatalf("unexpected value: %s", res.Value)
	}
	if !res.HasSecret {
		t.Fatal("expected secret metadata")
	}
	if strings.Contains(res.MaskedValue, "super-secret-value") {
		t.Fatalf("masked value leaked secret: %s", res.MaskedValue)
	}
}

func TestEnvVariableSecretMetadataSurvivesVarReuse(t *testing.T) {
	eng := NewEngineWithEnv(&testEnvProvider{
		data:    map[string]string{"TOKEN": "abcdef123456"},
		secrets: map[string]bool{"TOKEN": true},
	})
	if _, err := eng.EvaluateDetailed("${var:token=${env:TOKEN}}"); err != nil {
		t.Fatalf("assign secret var: %v", err)
	}
	res, err := eng.EvaluateDetailed("token=${var:token}")
	if err != nil {
		t.Fatalf("read secret var: %v", err)
	}
	if !res.HasSecret {
		t.Fatal("expected reused variable to retain secret metadata")
	}
}

func TestInvalidEnvVariableName(t *testing.T) {
	eng := NewEngineWithEnv(&testEnvProvider{data: map[string]string{"BAD-NAME": "value"}})
	if _, err := eng.Evaluate("${env:BAD-NAME}"); err == nil {
		t.Fatal("expected invalid env name error")
	}
}

func TestScanEnvVars(t *testing.T) {
	keys := ScanEnvVars("${env:BASE_URL}/login ${env:USER} plain ${env:BASE_URL}")
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d: %v", len(keys), keys)
	}
	if keys[0] != "BASE_URL" || keys[1] != "USER" {
		t.Errorf("unexpected keys: %v", keys)
	}
}

func TestScanEnvVarsIgnoresIncompleteAndDuplicates(t *testing.T) {
	keys := ScanEnvVars("${env:A} ${env:A} ${env:B")
	if len(keys) != 1 || keys[0] != "A" {
		t.Fatalf("unexpected keys: %v", keys)
	}
}
