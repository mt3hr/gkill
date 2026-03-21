import { GkillApiError } from "../lib/errors.mjs";
import {
  assertObject,
  assertBoolean,
  assertNumber,
  assertInteger,
  assertTrimmedString,
  assertStringArray,
  assertIntegerArray,
  assertKnownKeys,
} from "../lib/validation.mjs";

// ---------------------------------------------------------------------------
// assertObject
// ---------------------------------------------------------------------------
describe("assertObject", () => {
  test("returns a plain object unchanged", () => {
    const obj = { a: 1 };
    expect(assertObject(obj, "field")).toBe(obj);
  });

  test("returns empty object", () => {
    const obj = {};
    expect(assertObject(obj, "field")).toBe(obj);
  });

  test("throws for null", () => {
    expect(() => assertObject(null, "field")).toThrow(GkillApiError);
  });

  test("throws for array", () => {
    expect(() => assertObject([1, 2], "field")).toThrow(GkillApiError);
  });

  test("throws for string", () => {
    expect(() => assertObject("hello", "field")).toThrow(GkillApiError);
  });

  test("throws for number", () => {
    expect(() => assertObject(42, "field")).toThrow(GkillApiError);
  });

  test("throws for undefined by default", () => {
    expect(() => assertObject(undefined, "field")).toThrow(GkillApiError);
  });

  test("returns undefined when allowUndefined is true", () => {
    expect(assertObject(undefined, "field", { allowUndefined: true })).toBeUndefined();
  });

  test("still throws for null even when allowUndefined is true", () => {
    expect(() => assertObject(null, "field", { allowUndefined: true })).toThrow(GkillApiError);
  });
});

// ---------------------------------------------------------------------------
// assertBoolean
// ---------------------------------------------------------------------------
describe("assertBoolean", () => {
  test("returns true", () => {
    expect(assertBoolean(true, "field")).toBe(true);
  });

  test("returns false", () => {
    expect(assertBoolean(false, "field")).toBe(false);
  });

  test("throws for number 0", () => {
    expect(() => assertBoolean(0, "field")).toThrow(GkillApiError);
  });

  test("throws for number 1", () => {
    expect(() => assertBoolean(1, "field")).toThrow(GkillApiError);
  });

  test("throws for string", () => {
    expect(() => assertBoolean("true", "field")).toThrow(GkillApiError);
  });

  test("throws for null", () => {
    expect(() => assertBoolean(null, "field")).toThrow(GkillApiError);
  });

  test("throws for undefined", () => {
    expect(() => assertBoolean(undefined, "field")).toThrow(GkillApiError);
  });
});

// ---------------------------------------------------------------------------
// assertNumber
// ---------------------------------------------------------------------------
describe("assertNumber", () => {
  test("returns valid number", () => {
    expect(assertNumber(3.14, "field")).toBe(3.14);
  });

  test("returns zero", () => {
    expect(assertNumber(0, "field")).toBe(0);
  });

  test("returns negative number", () => {
    expect(assertNumber(-5, "field")).toBe(-5);
  });

  test("throws for NaN", () => {
    expect(() => assertNumber(NaN, "field")).toThrow(GkillApiError);
  });

  test("throws for Infinity", () => {
    expect(() => assertNumber(Infinity, "field")).toThrow(GkillApiError);
  });

  test("throws for -Infinity", () => {
    expect(() => assertNumber(-Infinity, "field")).toThrow(GkillApiError);
  });

  test("throws for string", () => {
    expect(() => assertNumber("3.14", "field")).toThrow(GkillApiError);
  });

  test("respects min constraint", () => {
    expect(assertNumber(5, "field", { min: 5 })).toBe(5);
    expect(() => assertNumber(4, "field", { min: 5 })).toThrow(/greater than or equal to 5/);
  });

  test("respects max constraint", () => {
    expect(assertNumber(10, "field", { max: 10 })).toBe(10);
    expect(() => assertNumber(11, "field", { max: 10 })).toThrow(/less than or equal to 10/);
  });

  test("respects minExclusive constraint", () => {
    expect(assertNumber(0.001, "field", { minExclusive: 0 })).toBe(0.001);
    expect(() => assertNumber(0, "field", { minExclusive: 0 })).toThrow(/greater than 0/);
  });
});

// ---------------------------------------------------------------------------
// assertInteger
// ---------------------------------------------------------------------------
describe("assertInteger", () => {
  test("returns valid integer", () => {
    expect(assertInteger(42, "field")).toBe(42);
  });

  test("returns zero", () => {
    expect(assertInteger(0, "field")).toBe(0);
  });

  test("returns negative integer", () => {
    expect(assertInteger(-10, "field")).toBe(-10);
  });

  test("throws for float", () => {
    expect(() => assertInteger(3.14, "field")).toThrow(GkillApiError);
  });

  test("throws for string", () => {
    expect(() => assertInteger("42", "field")).toThrow(GkillApiError);
  });

  test("throws for NaN", () => {
    expect(() => assertInteger(NaN, "field")).toThrow(GkillApiError);
  });

  test("respects min constraint", () => {
    expect(assertInteger(0, "field", { min: 0 })).toBe(0);
    expect(() => assertInteger(-1, "field", { min: 0 })).toThrow(/greater than or equal to 0/);
  });

  test("respects max constraint", () => {
    expect(assertInteger(86399, "field", { max: 86399 })).toBe(86399);
    expect(() => assertInteger(86400, "field", { max: 86399 })).toThrow(/less than or equal to 86399/);
  });
});

// ---------------------------------------------------------------------------
// assertTrimmedString
// ---------------------------------------------------------------------------
describe("assertTrimmedString", () => {
  test("returns trimmed string", () => {
    expect(assertTrimmedString("hello", "field")).toBe("hello");
  });

  test("trims whitespace", () => {
    expect(assertTrimmedString("  hello  ", "field")).toBe("hello");
  });

  test("throws for empty string", () => {
    expect(() => assertTrimmedString("", "field")).toThrow(GkillApiError);
  });

  test("throws for whitespace-only string", () => {
    expect(() => assertTrimmedString("   ", "field")).toThrow(GkillApiError);
  });

  test("throws for number", () => {
    expect(() => assertTrimmedString(42, "field")).toThrow(GkillApiError);
  });

  test("throws for null", () => {
    expect(() => assertTrimmedString(null, "field")).toThrow(GkillApiError);
  });

  test("throws for boolean", () => {
    expect(() => assertTrimmedString(true, "field")).toThrow(GkillApiError);
  });
});

// ---------------------------------------------------------------------------
// assertStringArray
// ---------------------------------------------------------------------------
describe("assertStringArray", () => {
  test("returns array of trimmed strings", () => {
    expect(assertStringArray(["a", " b ", "c"], "field")).toEqual(["a", "b", "c"]);
  });

  test("returns empty array", () => {
    expect(assertStringArray([], "field")).toEqual([]);
  });

  test("throws for non-array", () => {
    expect(() => assertStringArray("not-array", "field")).toThrow(GkillApiError);
  });

  test("throws for null", () => {
    expect(() => assertStringArray(null, "field")).toThrow(GkillApiError);
  });

  test("throws if array contains non-string", () => {
    expect(() => assertStringArray(["ok", 42], "field")).toThrow(GkillApiError);
  });

  test("throws if array contains empty string", () => {
    expect(() => assertStringArray(["ok", ""], "field")).toThrow(GkillApiError);
  });

  test("error message includes index", () => {
    expect(() => assertStringArray(["ok", 42], "tags")).toThrow(/tags\[1\]/);
  });
});

// ---------------------------------------------------------------------------
// assertIntegerArray
// ---------------------------------------------------------------------------
describe("assertIntegerArray", () => {
  test("returns array of integers", () => {
    expect(assertIntegerArray([0, 3, 6], "field")).toEqual([0, 3, 6]);
  });

  test("returns empty array", () => {
    expect(assertIntegerArray([], "field")).toEqual([]);
  });

  test("throws for non-array", () => {
    expect(() => assertIntegerArray("not-array", "field")).toThrow(GkillApiError);
  });

  test("throws if element is float", () => {
    expect(() => assertIntegerArray([1, 2.5], "field")).toThrow(GkillApiError);
  });

  test("respects min/max constraints on elements", () => {
    expect(assertIntegerArray([0, 6], "field", { min: 0, max: 6 })).toEqual([0, 6]);
    expect(() => assertIntegerArray([0, 7], "field", { min: 0, max: 6 })).toThrow(/less than or equal to 6/);
    expect(() => assertIntegerArray([-1, 3], "field", { min: 0, max: 6 })).toThrow(/greater than or equal to 0/);
  });

  test("error message includes index", () => {
    expect(() => assertIntegerArray([0, "x"], "days")).toThrow(/days\[1\]/);
  });
});

// ---------------------------------------------------------------------------
// assertKnownKeys
// ---------------------------------------------------------------------------
describe("assertKnownKeys", () => {
  test("does not throw for known keys", () => {
    const allowed = new Set(["a", "b", "c"]);
    expect(() => assertKnownKeys({ a: 1, b: 2 }, allowed, "args")).not.toThrow();
  });

  test("does not throw for empty object", () => {
    const allowed = new Set(["a", "b"]);
    expect(() => assertKnownKeys({}, allowed, "args")).not.toThrow();
  });

  test("throws for unknown key", () => {
    const allowed = new Set(["a", "b"]);
    expect(() => assertKnownKeys({ a: 1, z: 2 }, allowed, "args")).toThrow(GkillApiError);
  });

  test("error includes unknown key name", () => {
    const allowed = new Set(["a"]);
    expect(() => assertKnownKeys({ unknown_key: 1 }, allowed, "args")).toThrow(/args\.unknown_key/);
  });

  test("error detail includes allowed keys", () => {
    const allowed = new Set(["b", "a"]);
    try {
      assertKnownKeys({ z: 1 }, allowed, "args");
      expect.unreachable("should have thrown");
    } catch (err) {
      expect(err.detail.allowed).toEqual(["a", "b"]);
    }
  });
});
