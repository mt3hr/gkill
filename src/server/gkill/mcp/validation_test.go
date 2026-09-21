package mcp

// 旧 src/mcp/__tests__/validation.test.mjs の移植（describe → Test 関数、test → t.Run）。

import (
	"math"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

func TestAssertObject(t *testing.T) {
	t.Run("returns a plain object unchanged", func(t *testing.T) {
		o := obj("a", 1)
		got, err := AssertObject(o, "field")
		expectNoError(t, err)
		expectTrue(t, got == o, "must return the same object")
	})
	t.Run("returns empty object", func(t *testing.T) {
		o := obj()
		got, err := AssertObject(o, "field")
		expectNoError(t, err)
		expectTrue(t, got == o, "must return the same object")
	})
	t.Run("throws for null", func(t *testing.T) {
		_, err := AssertObject(nil, "field")
		expectGkillApiError(t, err)
	})
	t.Run("throws for array", func(t *testing.T) {
		_, err := AssertObject(arr(1, 2), "field")
		expectGkillApiError(t, err)
	})
	t.Run("throws for string", func(t *testing.T) {
		_, err := AssertObject("hello", "field")
		expectGkillApiError(t, err)
	})
	t.Run("throws for number", func(t *testing.T) {
		_, err := AssertObject(42, "field")
		expectGkillApiError(t, err)
	})
	t.Run("throws for undefined by default", func(t *testing.T) {
		_, err := AssertObject(jsonobj.Undefined, "field")
		expectGkillApiError(t, err)
	})
	t.Run("returns undefined when allowUndefined is true", func(t *testing.T) {
		got, err := AssertObjectAllowUndefined(jsonobj.Undefined, "field")
		expectNoError(t, err)
		expectTrue(t, got == nil, "must be nil for undefined")
	})
	t.Run("still throws for null even when allowUndefined is true", func(t *testing.T) {
		_, err := AssertObjectAllowUndefined(nil, "field")
		expectGkillApiError(t, err)
	})
}

func TestAssertBoolean(t *testing.T) {
	t.Run("returns true", func(t *testing.T) {
		got, err := AssertBoolean(true, "field")
		expectNoError(t, err)
		expectTrue(t, got, "true")
	})
	t.Run("returns false", func(t *testing.T) {
		got, err := AssertBoolean(false, "field")
		expectNoError(t, err)
		expectTrue(t, !got, "false")
	})
	t.Run("throws for number 0", func(t *testing.T) {
		_, err := AssertBoolean(0, "field")
		expectGkillApiError(t, err)
	})
	t.Run("throws for number 1", func(t *testing.T) {
		_, err := AssertBoolean(1, "field")
		expectGkillApiError(t, err)
	})
	t.Run("throws for string", func(t *testing.T) {
		_, err := AssertBoolean("true", "field")
		expectGkillApiError(t, err)
	})
	t.Run("throws for null", func(t *testing.T) {
		_, err := AssertBoolean(nil, "field")
		expectGkillApiError(t, err)
	})
	t.Run("throws for undefined", func(t *testing.T) {
		_, err := AssertBoolean(jsonobj.Undefined, "field")
		expectGkillApiError(t, err)
	})
}

func TestAssertNumber(t *testing.T) {
	t.Run("returns valid number", func(t *testing.T) {
		got, err := AssertNumber(3.14, "field", NumberRange{})
		expectNoError(t, err)
		expectTrue(t, got == 3.14, "got %v", got)
	})
	t.Run("returns zero", func(t *testing.T) {
		got, err := AssertNumber(0, "field", NumberRange{})
		expectNoError(t, err)
		expectTrue(t, got == 0, "got %v", got)
	})
	t.Run("returns negative number", func(t *testing.T) {
		got, err := AssertNumber(-5, "field", NumberRange{})
		expectNoError(t, err)
		expectTrue(t, got == -5, "got %v", got)
	})
	t.Run("throws for NaN", func(t *testing.T) {
		_, err := AssertNumber(math.NaN(), "field", NumberRange{})
		expectGkillApiError(t, err)
	})
	t.Run("throws for Infinity", func(t *testing.T) {
		_, err := AssertNumber(math.Inf(1), "field", NumberRange{})
		expectGkillApiError(t, err)
	})
	t.Run("throws for -Infinity", func(t *testing.T) {
		_, err := AssertNumber(math.Inf(-1), "field", NumberRange{})
		expectGkillApiError(t, err)
	})
	t.Run("throws for string", func(t *testing.T) {
		_, err := AssertNumber("3.14", "field", NumberRange{})
		expectGkillApiError(t, err)
	})
	t.Run("respects min constraint", func(t *testing.T) {
		got, err := AssertNumber(5, "field", NumberRange{Min: F(5)})
		expectNoError(t, err)
		expectTrue(t, got == 5, "got %v", got)
		_, err = AssertNumber(4, "field", NumberRange{Min: F(5)})
		expectErrorMatches(t, err, `greater than or equal to 5`)
	})
	t.Run("respects max constraint", func(t *testing.T) {
		got, err := AssertNumber(10, "field", NumberRange{Max: F(10)})
		expectNoError(t, err)
		expectTrue(t, got == 10, "got %v", got)
		_, err = AssertNumber(11, "field", NumberRange{Max: F(10)})
		expectErrorMatches(t, err, `less than or equal to 10`)
	})
	t.Run("respects minExclusive constraint", func(t *testing.T) {
		got, err := AssertNumber(0.001, "field", NumberRange{MinExclusive: F(0)})
		expectNoError(t, err)
		expectTrue(t, got == 0.001, "got %v", got)
		_, err = AssertNumber(0, "field", NumberRange{MinExclusive: F(0)})
		expectErrorMatches(t, err, `greater than 0`)
	})
}

func TestAssertInteger(t *testing.T) {
	t.Run("returns valid integer", func(t *testing.T) {
		got, err := AssertInteger(42, "field", IntRange{})
		expectNoError(t, err)
		expectTrue(t, got == 42, "got %v", got)
	})
	t.Run("returns zero", func(t *testing.T) {
		got, err := AssertInteger(0, "field", IntRange{})
		expectNoError(t, err)
		expectTrue(t, got == 0, "got %v", got)
	})
	t.Run("returns negative integer", func(t *testing.T) {
		got, err := AssertInteger(-10, "field", IntRange{})
		expectNoError(t, err)
		expectTrue(t, got == -10, "got %v", got)
	})
	t.Run("throws for float", func(t *testing.T) {
		_, err := AssertInteger(3.14, "field", IntRange{})
		expectGkillApiError(t, err)
	})
	t.Run("throws for string", func(t *testing.T) {
		_, err := AssertInteger("42", "field", IntRange{})
		expectGkillApiError(t, err)
	})
	t.Run("throws for NaN", func(t *testing.T) {
		_, err := AssertInteger(math.NaN(), "field", IntRange{})
		expectGkillApiError(t, err)
	})
	t.Run("respects min constraint", func(t *testing.T) {
		got, err := AssertInteger(0, "field", IntRange{Min: I(0)})
		expectNoError(t, err)
		expectTrue(t, got == 0, "got %v", got)
		_, err = AssertInteger(-1, "field", IntRange{Min: I(0)})
		expectErrorMatches(t, err, `greater than or equal to 0`)
	})
	t.Run("respects max constraint", func(t *testing.T) {
		got, err := AssertInteger(86399, "field", IntRange{Max: I(86399)})
		expectNoError(t, err)
		expectTrue(t, got == 86399, "got %v", got)
		_, err = AssertInteger(86400, "field", IntRange{Max: I(86399)})
		expectErrorMatches(t, err, `less than or equal to 86399`)
	})
}

func TestAssertTrimmedString(t *testing.T) {
	t.Run("returns trimmed string", func(t *testing.T) {
		got, err := AssertTrimmedString("hello", "field")
		expectNoError(t, err)
		expectTrue(t, got == "hello", "got %q", got)
	})
	t.Run("trims whitespace", func(t *testing.T) {
		got, err := AssertTrimmedString("  hello  ", "field")
		expectNoError(t, err)
		expectTrue(t, got == "hello", "got %q", got)
	})
	t.Run("throws for empty string", func(t *testing.T) {
		_, err := AssertTrimmedString("", "field")
		expectGkillApiError(t, err)
	})
	t.Run("throws for whitespace-only string", func(t *testing.T) {
		_, err := AssertTrimmedString("   ", "field")
		expectGkillApiError(t, err)
	})
	t.Run("throws for number", func(t *testing.T) {
		_, err := AssertTrimmedString(42, "field")
		expectGkillApiError(t, err)
	})
	t.Run("throws for null", func(t *testing.T) {
		_, err := AssertTrimmedString(nil, "field")
		expectGkillApiError(t, err)
	})
	t.Run("throws for boolean", func(t *testing.T) {
		_, err := AssertTrimmedString(true, "field")
		expectGkillApiError(t, err)
	})
}

func TestAssertStringArray(t *testing.T) {
	t.Run("returns array of trimmed strings", func(t *testing.T) {
		got, err := AssertStringArray(strs("a", " b ", "c"), "field")
		expectNoError(t, err)
		expectEqual(t, got, strs("a", "b", "c"))
	})
	t.Run("returns empty array", func(t *testing.T) {
		got, err := AssertStringArray(arr(), "field")
		expectNoError(t, err)
		expectEqual(t, got, arr())
	})
	t.Run("throws for non-array", func(t *testing.T) {
		_, err := AssertStringArray("not-array", "field")
		expectGkillApiError(t, err)
	})
	t.Run("throws for null", func(t *testing.T) {
		_, err := AssertStringArray(nil, "field")
		expectGkillApiError(t, err)
	})
	t.Run("throws if array contains non-string", func(t *testing.T) {
		_, err := AssertStringArray(arr("ok", 42), "field")
		expectGkillApiError(t, err)
	})
	t.Run("throws if array contains empty string", func(t *testing.T) {
		_, err := AssertStringArray(arr("ok", ""), "field")
		expectGkillApiError(t, err)
	})
	t.Run("error message includes index", func(t *testing.T) {
		_, err := AssertStringArray(arr("ok", 42), "tags")
		expectErrorMatches(t, err, `tags\[1\]`)
	})
}

func TestAssertIntegerArray(t *testing.T) {
	t.Run("returns array of integers", func(t *testing.T) {
		got, err := AssertIntegerArray(arr(0, 3, 6), "field", IntRange{})
		expectNoError(t, err)
		expectEqual(t, got, arr(0, 3, 6))
	})
	t.Run("returns empty array", func(t *testing.T) {
		got, err := AssertIntegerArray(arr(), "field", IntRange{})
		expectNoError(t, err)
		expectEqual(t, got, arr())
	})
	t.Run("throws for non-array", func(t *testing.T) {
		_, err := AssertIntegerArray("not-array", "field", IntRange{})
		expectGkillApiError(t, err)
	})
	t.Run("throws if element is float", func(t *testing.T) {
		_, err := AssertIntegerArray(arr(1, 2.5), "field", IntRange{})
		expectGkillApiError(t, err)
	})
	t.Run("respects min/max constraints on elements", func(t *testing.T) {
		got, err := AssertIntegerArray(arr(0, 6), "field", IntRange{Min: I(0), Max: I(6)})
		expectNoError(t, err)
		expectEqual(t, got, arr(0, 6))
		_, err = AssertIntegerArray(arr(0, 7), "field", IntRange{Min: I(0), Max: I(6)})
		expectErrorMatches(t, err, `less than or equal to 6`)
		_, err = AssertIntegerArray(arr(-1, 3), "field", IntRange{Min: I(0), Max: I(6)})
		expectErrorMatches(t, err, `greater than or equal to 0`)
	})
	t.Run("error message includes index", func(t *testing.T) {
		_, err := AssertIntegerArray(arr(0, "x"), "days", IntRange{})
		expectErrorMatches(t, err, `days\[1\]`)
	})
}

func TestAssertKnownKeys(t *testing.T) {
	t.Run("does not throw for known keys", func(t *testing.T) {
		allowed := NewStringSet("a", "b", "c")
		expectNoError(t, AssertKnownKeys(obj("a", 1, "b", 2), allowed, "args", nil))
	})
	t.Run("does not throw for empty object", func(t *testing.T) {
		allowed := NewStringSet("a", "b")
		expectNoError(t, AssertKnownKeys(obj(), allowed, "args", nil))
	})
	t.Run("throws for unknown key", func(t *testing.T) {
		allowed := NewStringSet("a", "b")
		expectGkillApiError(t, AssertKnownKeys(obj("a", 1, "z", 2), allowed, "args", nil))
	})
	t.Run("error includes unknown key name", func(t *testing.T) {
		allowed := NewStringSet("a")
		expectErrorMatches(t, AssertKnownKeys(obj("unknown_key", 1), allowed, "args", nil), `args\.unknown_key`)
	})
	t.Run("error detail includes allowed keys", func(t *testing.T) {
		allowed := NewStringSet("b", "a")
		err := AssertKnownKeys(obj("z", 1), allowed, "args", nil)
		apiErr := expectGkillApiError(t, err)
		expectEqual(t, apiErr.Detail.Value("allowed"), strs("a", "b"))
	})
}

func TestAssertKnownKeysDefaultFieldName(t *testing.T) {
	t.Run("unknown keys are reported under arguments when no field is given", func(t *testing.T) {
		// 第3引数を渡さない呼び出しが write 側に20箇所あり、
		// エラーが `Invalid argument 'undefined.contnet'` になっていた
		err := AssertKnownKeys(obj("content", "x", "contnet", "y"), NewStringSet("content"), "", nil)
		apiErr := expectGkillApiError(t, err)
		expectEqual(t, apiErr.Detail.Value("field"), "arguments.contnet")
		mustNotContain(t, apiErr.Message, "undefined")
	})
}

func TestAssertIntegerSafeIntegerRange(t *testing.T) {
	// assertInteger — 安全整数の範囲 (2巡目の指摘 P-37)
	t.Run("rejects integers beyond the safe range instead of storing a value that cannot round-trip", func(t *testing.T) {
		// 2^53。JSON を往復するだけで別の値になるので、保存できたように見えて読み戻すと違う
		_, err := AssertInteger(float64(9007199254740992), "amount", IntRange{})
		expectErrorMatches(t, err, `safe integer range`)
		_, err = AssertInteger(float64(-9007199254740992), "amount", IntRange{})
		expectErrorMatches(t, err, `safe integer range`)
	})
	t.Run("still accepts the largest value that does round-trip", func(t *testing.T) {
		got, err := AssertInteger(jsonobj.MaxSafeInteger, "amount", IntRange{})
		expectNoError(t, err)
		expectTrue(t, float64(got) == jsonobj.MaxSafeInteger, "got %v", got)
		got, err = AssertInteger(-jsonobj.MaxSafeInteger, "amount", IntRange{})
		expectNoError(t, err)
		expectTrue(t, float64(got) == -jsonobj.MaxSafeInteger, "got %v", got)
	})
	t.Run("the safe-range error is a GkillApiError like every other validation error", func(t *testing.T) {
		_, err := AssertInteger(math.Pow(2, 53), "amount", IntRange{})
		expectGkillApiError(t, err)
	})
}
