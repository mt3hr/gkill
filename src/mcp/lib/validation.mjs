// Validation functions extracted from gkill-read-server.mjs.

import { isPlainObject, invalidArgument } from "./errors.mjs";

export function assertObject(value, field, { allowUndefined = false } = {}) {
  if (value === undefined && allowUndefined) {
    return undefined;
  }
  if (!isPlainObject(value)) {
    throw invalidArgument(field, "must be an object", value);
  }
  return value;
}

export function assertBoolean(value, field) {
  if (typeof value !== "boolean") {
    throw invalidArgument(field, "must be a boolean", value);
  }
  return value;
}

export function assertNumber(value, field, { minExclusive = null, min = null, max = null } = {}) {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    throw invalidArgument(field, "must be a finite number", value);
  }
  if (minExclusive !== null && value <= minExclusive) {
    throw invalidArgument(field, `must be greater than ${minExclusive}`, value);
  }
  if (min !== null && value < min) {
    throw invalidArgument(field, `must be greater than or equal to ${min}`, value);
  }
  if (max !== null && value > max) {
    throw invalidArgument(field, `must be less than or equal to ${max}`, value);
  }
  return value;
}

export function assertInteger(value, field, { min = null, max = null } = {}) {
  if (typeof value !== "number" || !Number.isInteger(value)) {
    throw invalidArgument(field, "must be an integer", value);
  }
  if (min !== null && value < min) {
    throw invalidArgument(field, `must be greater than or equal to ${min}`, value);
  }
  if (max !== null && value > max) {
    throw invalidArgument(field, `must be less than or equal to ${max}`, value);
  }
  return value;
}

export function assertTrimmedString(value, field) {
  if (typeof value !== "string") {
    throw invalidArgument(field, "must be a string", value);
  }
  const trimmed = value.trim();
  if (!trimmed) {
    throw invalidArgument(field, "must not be empty", value);
  }
  return trimmed;
}

export function assertStringArray(value, field) {
  if (!Array.isArray(value)) {
    throw invalidArgument(field, "must be an array of strings", value);
  }
  return value.map((item, index) => assertTrimmedString(item, `${field}[${index}]`));
}

export function assertIntegerArray(value, field, { min = null, max = null } = {}) {
  if (!Array.isArray(value)) {
    throw invalidArgument(field, "must be an array of integers", value);
  }
  return value.map((item, index) => assertInteger(item, `${field}[${index}]`, { min, max }));
}

export function assertKnownKeys(value, allowedKeys, field) {
  for (const key of Object.keys(value)) {
    if (!allowedKeys.has(key)) {
      throw invalidArgument(`${field}.${key}`, "is not supported", value[key], {
        allowed: Array.from(allowedKeys).sort(),
      });
    }
  }
}
