// Shared error class and helper utilities for gkill MCP validation/normalization.

export class GkillApiError extends Error {
  constructor(message, detail = null) {
    super(message);
    this.name = "GkillApiError";
    this.detail = detail;
  }
}

export function isPlainObject(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

export function describeValueType(value) {
  if (value === null) return "null";
  if (Array.isArray(value)) return "array";
  return typeof value;
}

export function previewValue(value) {
  if (typeof value === "string") {
    return value.length > 120 ? `${value.slice(0, 117)}...` : value;
  }
  if (typeof value === "number" || typeof value === "boolean" || value === null) {
    return value;
  }
  if (Array.isArray(value)) {
    return `array(${value.length})`;
  }
  if (isPlainObject(value)) {
    return "object";
  }
  return describeValueType(value);
}

export function invalidArgument(field, message, value, extra = {}) {
  return new GkillApiError(`Invalid argument '${field}': ${message}.`, {
    field,
    actualType: describeValueType(value),
    actualValue: previewValue(value),
    ...extra,
  });
}
