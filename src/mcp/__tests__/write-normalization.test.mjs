/**
 * Tests for write normalization functions from lib/write-normalization.mjs.
 */

import { describe, test, expect } from "vitest";
import {
  normalizeKmemoArgs,
  normalizeUrlogArgs,
  normalizeNlogArgs,
  normalizeLantanaArgs,
  normalizeTimeIsArgs,
  normalizeMiArgs,
  normalizeKcArgs,
  normalizeTagArgs,
  normalizeTextArgs,
  normalizeKftlArgs,
  normalizeDeleteArgs,
  DELETE_DATA_TYPES,
} from "../lib/write-normalization.mjs";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function expectThrowsField(fn, field) {
  expect(fn).toThrow();
  try { fn(); } catch (e) { expect(e.detail?.field).toBe(field); }
}

// ---------------------------------------------------------------------------
// normalizeKmemoArgs
// ---------------------------------------------------------------------------
describe("normalizeKmemoArgs", () => {
  test("accepts valid content", () => {
    const result = normalizeKmemoArgs({ content: "hello" });
    expect(result.content).toBe("hello");
    expect(result.related_time).toBeUndefined();
  });

  test("normalizes related_time date-only", () => {
    const result = normalizeKmemoArgs({ content: "test", related_time: "2026-03-15" });
    expect(result.related_time).toMatch(/^2026-03-15T/);
  });

  test("passes through locale_name", () => {
    const result = normalizeKmemoArgs({ content: "test", locale_name: "en" });
    expect(result.locale_name).toBe("en");
  });

  test("rejects missing content", () => {
    expectThrowsField(() => normalizeKmemoArgs({}), "content");
  });

  test("rejects non-string content", () => {
    expectThrowsField(() => normalizeKmemoArgs({ content: 123 }), "content");
  });

  test("rejects unknown keys", () => {
    expect(() => normalizeKmemoArgs({ content: "x", foo: "bar" })).toThrow();
  });

  test("rejects non-object args", () => {
    expect(() => normalizeKmemoArgs("string")).toThrow();
    expect(() => normalizeKmemoArgs(null)).toThrow();
  });
});

// ---------------------------------------------------------------------------
// normalizeUrlogArgs
// ---------------------------------------------------------------------------
describe("normalizeUrlogArgs", () => {
  test("accepts url only", () => {
    const result = normalizeUrlogArgs({ url: "https://example.com" });
    expect(result.url).toBe("https://example.com");
    expect(result.title).toBeUndefined();
  });

  test("accepts url with title", () => {
    const result = normalizeUrlogArgs({ url: "https://example.com", title: "Example" });
    expect(result.title).toBe("Example");
  });

  test("rejects missing url", () => {
    expectThrowsField(() => normalizeUrlogArgs({}), "url");
  });
});

// ---------------------------------------------------------------------------
// normalizeNlogArgs
// ---------------------------------------------------------------------------
describe("normalizeNlogArgs", () => {
  test("accepts title and amount", () => {
    const result = normalizeNlogArgs({ title: "lunch", amount: 1500 });
    expect(result.title).toBe("lunch");
    expect(result.amount).toBe(1500);
  });

  test("accepts optional shop", () => {
    const result = normalizeNlogArgs({ title: "lunch", amount: 1500, shop: "cafe" });
    expect(result.shop).toBe("cafe");
  });

  test("accepts negative amounts", () => {
    const result = normalizeNlogArgs({ title: "refund", amount: -500 });
    expect(result.amount).toBe(-500);
  });

  test("rejects missing amount", () => {
    expectThrowsField(() => normalizeNlogArgs({ title: "lunch" }), "amount");
  });

  test("rejects non-number amount", () => {
    expectThrowsField(() => normalizeNlogArgs({ title: "lunch", amount: "abc" }), "amount");
  });

  test("rejects missing title", () => {
    expectThrowsField(() => normalizeNlogArgs({ amount: 100 }), "title");
  });
});

// ---------------------------------------------------------------------------
// normalizeLantanaArgs
// ---------------------------------------------------------------------------
describe("normalizeLantanaArgs", () => {
  test("accepts mood 0", () => {
    expect(normalizeLantanaArgs({ mood: 0 }).mood).toBe(0);
  });

  test("accepts mood 10", () => {
    expect(normalizeLantanaArgs({ mood: 10 }).mood).toBe(10);
  });

  test("accepts mood 5", () => {
    expect(normalizeLantanaArgs({ mood: 5 }).mood).toBe(5);
  });

  test("rejects mood < 0", () => {
    expectThrowsField(() => normalizeLantanaArgs({ mood: -1 }), "mood");
  });

  test("rejects mood > 10", () => {
    expectThrowsField(() => normalizeLantanaArgs({ mood: 11 }), "mood");
  });

  test("rejects non-integer mood", () => {
    expectThrowsField(() => normalizeLantanaArgs({ mood: 5.5 }), "mood");
  });

  test("rejects missing mood", () => {
    expectThrowsField(() => normalizeLantanaArgs({}), "mood");
  });
});

// ---------------------------------------------------------------------------
// normalizeTimeIsArgs
// ---------------------------------------------------------------------------
describe("normalizeTimeIsArgs", () => {
  test("accepts title only", () => {
    const result = normalizeTimeIsArgs({ title: "coding" });
    expect(result.title).toBe("coding");
    expect(result.start_time).toBeUndefined();
    expect(result.end_time).toBeUndefined();
  });

  test("accepts start_time and end_time", () => {
    const result = normalizeTimeIsArgs({
      title: "meeting",
      start_time: "2026-03-15T09:00:00+09:00",
      end_time: "2026-03-15T10:00:00+09:00",
    });
    expect(result.start_time).toBe("2026-03-15T09:00:00+09:00");
    expect(result.end_time).toBe("2026-03-15T10:00:00+09:00");
  });

  test("rejects missing title", () => {
    expectThrowsField(() => normalizeTimeIsArgs({}), "title");
  });
});

// ---------------------------------------------------------------------------
// normalizeMiArgs
// ---------------------------------------------------------------------------
describe("normalizeMiArgs", () => {
  test("accepts required fields", () => {
    const result = normalizeMiArgs({ title: "fix bug", board_name: "dev" });
    expect(result.title).toBe("fix bug");
    expect(result.board_name).toBe("dev");
    expect(result.is_checked).toBe(false);
  });

  test("accepts is_checked true", () => {
    const result = normalizeMiArgs({ title: "done", board_name: "dev", is_checked: true });
    expect(result.is_checked).toBe(true);
  });

  test("accepts optional time fields", () => {
    const result = normalizeMiArgs({
      title: "task",
      board_name: "dev",
      limit_time: "2026-04-01",
      estimate_start_time: "2026-03-20T09:00:00+09:00",
    });
    expect(result.limit_time).toMatch(/^2026-04-01T/);
    expect(result.estimate_start_time).toBe("2026-03-20T09:00:00+09:00");
  });

  test("rejects missing title", () => {
    expectThrowsField(() => normalizeMiArgs({ board_name: "dev" }), "title");
  });

  test("rejects missing board_name", () => {
    expectThrowsField(() => normalizeMiArgs({ title: "task" }), "board_name");
  });

  test("rejects non-boolean is_checked", () => {
    expectThrowsField(() => normalizeMiArgs({ title: "t", board_name: "b", is_checked: "yes" }), "is_checked");
  });
});

// ---------------------------------------------------------------------------
// normalizeKcArgs
// ---------------------------------------------------------------------------
describe("normalizeKcArgs", () => {
  test("accepts title and num_value", () => {
    const result = normalizeKcArgs({ title: "steps", num_value: 10000 });
    expect(result.title).toBe("steps");
    expect(result.num_value).toBe(10000);
  });

  test("accepts float num_value", () => {
    const result = normalizeKcArgs({ title: "temp", num_value: 36.5 });
    expect(result.num_value).toBe(36.5);
  });

  test("rejects missing num_value", () => {
    expectThrowsField(() => normalizeKcArgs({ title: "x" }), "num_value");
  });
});

// ---------------------------------------------------------------------------
// normalizeTagArgs
// ---------------------------------------------------------------------------
describe("normalizeTagArgs", () => {
  test("accepts tag and target_id", () => {
    const result = normalizeTagArgs({ tag: "important", target_id: "uuid-123" });
    expect(result.tag).toBe("important");
    expect(result.target_id).toBe("uuid-123");
  });

  test("rejects missing tag", () => {
    expectThrowsField(() => normalizeTagArgs({ target_id: "id" }), "tag");
  });

  test("rejects missing target_id", () => {
    expectThrowsField(() => normalizeTagArgs({ tag: "t" }), "target_id");
  });
});

// ---------------------------------------------------------------------------
// normalizeTextArgs
// ---------------------------------------------------------------------------
describe("normalizeTextArgs", () => {
  test("accepts text and target_id", () => {
    const result = normalizeTextArgs({ text: "annotation", target_id: "uuid-456" });
    expect(result.text).toBe("annotation");
    expect(result.target_id).toBe("uuid-456");
  });

  test("rejects missing text", () => {
    expectThrowsField(() => normalizeTextArgs({ target_id: "id" }), "text");
  });

  test("rejects missing target_id", () => {
    expectThrowsField(() => normalizeTextArgs({ text: "t" }), "target_id");
  });
});

// ---------------------------------------------------------------------------
// normalizeKftlArgs
// ---------------------------------------------------------------------------
describe("normalizeKftlArgs", () => {
  test("accepts kftl_text", () => {
    const result = normalizeKftlArgs({ kftl_text: "/mi Buy milk" });
    expect(result.kftl_text).toBe("/mi Buy milk");
  });

  test("rejects missing kftl_text", () => {
    expectThrowsField(() => normalizeKftlArgs({}), "kftl_text");
  });

  test("rejects empty kftl_text", () => {
    expectThrowsField(() => normalizeKftlArgs({ kftl_text: "  " }), "kftl_text");
  });
});

// ---------------------------------------------------------------------------
// normalizeDeleteArgs
// ---------------------------------------------------------------------------
describe("normalizeDeleteArgs", () => {
  test("accepts valid id and data_type", () => {
    const result = normalizeDeleteArgs({ id: "uuid-789", data_type: "kmemo" });
    expect(result.id).toBe("uuid-789");
    expect(result.data_type).toBe("kmemo");
  });

  test("accepts all valid data_types", () => {
    for (const dt of DELETE_DATA_TYPES) {
      const result = normalizeDeleteArgs({ id: "id", data_type: dt });
      expect(result.data_type).toBe(dt);
    }
  });

  test("rejects invalid data_type", () => {
    expectThrowsField(() => normalizeDeleteArgs({ id: "id", data_type: "invalid" }), "data_type");
  });

  test("rejects missing id", () => {
    expectThrowsField(() => normalizeDeleteArgs({ data_type: "kmemo" }), "id");
  });

  test("rejects missing data_type", () => {
    expectThrowsField(() => normalizeDeleteArgs({ id: "id" }), "data_type");
  });
});

// ---------------------------------------------------------------------------
// DELETE_DATA_TYPES constant
// ---------------------------------------------------------------------------
describe("DELETE_DATA_TYPES", () => {
  test("contains 9 data types", () => {
    expect(DELETE_DATA_TYPES.size).toBe(9);
  });

  test("contains expected types", () => {
    const expected = ["kmemo", "urlog", "nlog", "lantana", "timeis", "mi", "kc", "tag", "text"];
    for (const dt of expected) {
      expect(DELETE_DATA_TYPES.has(dt)).toBe(true);
    }
  });
});
