package textutil

import "testing"

func TestTruncateShortAscii(t *testing.T) {
	got := Truncate("hello", 10)
	if got != "hello" {
		t.Errorf("expected unchanged short ascii, got %q", got)
	}
}

func TestTruncateLongAscii(t *testing.T) {
	got := Truncate("hello world this is long", 10)
	want := "hello w..."
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestTruncateShortChinese(t *testing.T) {
	// Each Chinese character is 3 bytes in UTF-8 but counts as 1 rune.
	got := Truncate("打开网址", 10)
	if got != "打开网址" {
		t.Errorf("expected unchanged short chinese, got %q", got)
	}
}

func TestTruncateLongChinese(t *testing.T) {
	// 11 Chinese characters; max=8 should keep 5 + "..." (no invalid UTF-8).
	got := Truncate("点击含有非常非常长的中文选择器元素", 8)
	if got != "点击含有非..." {
		t.Errorf("expected %q, got %q", "点击含有非...", got)
	}
	if len([]rune(got)) != 8 {
		t.Errorf("result rune count = %d, want 8", len([]rune(got)))
	}
}

func TestTruncateDoesNotSplitMultiByteChars(t *testing.T) {
	// Bug repro: with the old byte-based implementation, truncating
	// Chinese text at a byte boundary produced invalid UTF-8.
	got := Truncate("一二三四五六七八九十", 5)
	runes := []rune(got)
	if len(runes) != 5 {
		t.Errorf("expected 5 runes, got %d in %q", len(runes), got)
	}
	if got != "一二..." {
		t.Errorf("expected %q, got %q", "一二...", got)
	}
}

func TestTruncateExactLength(t *testing.T) {
	got := Truncate("一二三四五", 5)
	if got != "一二三四五" {
		t.Errorf("expected unchanged, got %q", got)
	}
}

func TestTruncateEmpty(t *testing.T) {
	if got := Truncate("", 10); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestTruncateNonPositiveMax(t *testing.T) {
	if got := Truncate("hello", 0); got != "" {
		t.Errorf("expected empty for max=0, got %q", got)
	}
	if got := Truncate("hello", -1); got != "" {
		t.Errorf("expected empty for max<0, got %q", got)
	}
}

func TestTruncateVerySmallMax(t *testing.T) {
	// max <= 3 means no room for the ellipsis.
	if got := Truncate("hello world", 3); got != "hel" {
		t.Errorf("expected %q, got %q", "hel", got)
	}
}

func TestTruncateMixedAsciiAndChinese(t *testing.T) {
	// 3 ASCII + 5 Chinese = 8 runes total. With max=6, truncate.
	// The function keeps the first (max-3)=3 runes and appends "...",
	// so the result is the first 3 runes ("abc") plus "...".
	got := Truncate("abc一二三四五", 6)
	if got != "abc..." {
		t.Errorf("expected %q, got %q", "abc...", got)
	}
	// Same input but max=8 leaves everything unchanged.
	got = Truncate("abc一二三四五", 8)
	if got != "abc一二三四五" {
		t.Errorf("expected unchanged, got %q", got)
	}
	// 10 Chinese + 3 ASCII = 13 runes; max=8 keeps 5 runes + "...".
	got = Truncate("abc一二三四五六七八九十", 8)
	if got != "abc一二..." {
		t.Errorf("expected %q, got %q", "abc一二...", got)
	}
}

func TestTruncateResultIsValidUTF8(t *testing.T) {
	// Property test: any input + any max in [0, 50] should yield a valid
	// UTF-8 string. The old byte-slicing implementation would fail this
	// for many Chinese inputs.
	inputs := []string{
		"hello",
		"打开网址",
		"中文English混合",
		"a" + "一二三四五" + "z",
		"𠮷野家", // surrogate pair BMP char
	}
	for _, in := range inputs {
		for max := 0; max <= 30; max++ {
			got := Truncate(in, max)
			for _, r := range got {
				if r == '\uFFFD' {
					t.Errorf("Truncate(%q, %d) = %q contains invalid UTF-8 replacement char", in, max, got)
				}
			}
		}
	}
}
