import crypto from "node:crypto";
import {
  verifyCodeChallenge,
  isValidCodeVerifier,
  isSupportedChallengeMethod,
} from "../lib/pkce.mjs";

// ---------------------------------------------------------------------------
// Helper: generate a known code_verifier / code_challenge pair for S256
// ---------------------------------------------------------------------------
function makeS256Pair(verifier) {
  const challenge = crypto.createHash("sha256").update(verifier, "ascii").digest("base64url");
  return { verifier, challenge };
}

// ---------------------------------------------------------------------------
// verifyCodeChallenge — S256
// ---------------------------------------------------------------------------
describe("verifyCodeChallenge (S256)", () => {
  test("accepts a valid verifier/challenge pair", () => {
    const { verifier, challenge } = makeS256Pair("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk");
    expect(verifyCodeChallenge(verifier, challenge, "S256")).toBe(true);
  });

  test("rejects an incorrect verifier", () => {
    const { challenge } = makeS256Pair("correct-verifier-value-that-is-long-enough-here");
    expect(verifyCodeChallenge("wrong-verifier-value-that-is-also-long-enough!", challenge, "S256")).toBe(false);
  });

  test("rejects empty verifier", () => {
    const { challenge } = makeS256Pair("some-verifier-value-that-is-long-enough-12345");
    expect(verifyCodeChallenge("", challenge, "S256")).toBe(false);
  });

  test("rejects empty challenge", () => {
    expect(verifyCodeChallenge("some-verifier", "", "S256")).toBe(false);
  });

  test("rejects null inputs", () => {
    expect(verifyCodeChallenge(null, "challenge", "S256")).toBe(false);
    expect(verifyCodeChallenge("verifier", null, "S256")).toBe(false);
  });

  test("works with RFC 7636 Appendix B test vector", () => {
    // RFC 7636 Appendix B: code_verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
    // code_challenge (S256) = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
    const verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk";
    const challenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM";
    expect(verifyCodeChallenge(verifier, challenge, "S256")).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// verifyCodeChallenge — plain
// ---------------------------------------------------------------------------
describe("verifyCodeChallenge (plain)", () => {
  test("accepts matching verifier and challenge", () => {
    const v = "my-plain-code-verifier-that-is-long-enough-here";
    expect(verifyCodeChallenge(v, v, "plain")).toBe(true);
  });

  test("rejects mismatched verifier", () => {
    expect(verifyCodeChallenge("aaa", "bbb", "plain")).toBe(false);
  });

  test("rejects empty strings", () => {
    expect(verifyCodeChallenge("", "", "plain")).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// verifyCodeChallenge — unsupported method
// ---------------------------------------------------------------------------
describe("verifyCodeChallenge (unsupported method)", () => {
  test("rejects unknown method", () => {
    expect(verifyCodeChallenge("v", "c", "S512")).toBe(false);
  });

  test("rejects undefined method", () => {
    expect(verifyCodeChallenge("v", "c", undefined)).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// isValidCodeVerifier
// ---------------------------------------------------------------------------
describe("isValidCodeVerifier", () => {
  test("accepts 43-character verifier", () => {
    expect(isValidCodeVerifier("a".repeat(43))).toBe(true);
  });

  test("accepts 128-character verifier", () => {
    expect(isValidCodeVerifier("B".repeat(128))).toBe(true);
  });

  test("accepts verifier with all unreserved characters", () => {
    expect(isValidCodeVerifier("ABCDEFghij0123456789-._~" + "x".repeat(19))).toBe(true);
  });

  test("rejects verifier shorter than 43 characters", () => {
    expect(isValidCodeVerifier("a".repeat(42))).toBe(false);
  });

  test("rejects verifier longer than 128 characters", () => {
    expect(isValidCodeVerifier("a".repeat(129))).toBe(false);
  });

  test("rejects verifier with invalid characters", () => {
    expect(isValidCodeVerifier("a".repeat(42) + " ")).toBe(false);
    expect(isValidCodeVerifier("a".repeat(42) + "+")).toBe(false);
    expect(isValidCodeVerifier("a".repeat(42) + "/")).toBe(false);
    expect(isValidCodeVerifier("a".repeat(42) + "=")).toBe(false);
  });

  test("rejects non-string", () => {
    expect(isValidCodeVerifier(123)).toBe(false);
    expect(isValidCodeVerifier(null)).toBe(false);
    expect(isValidCodeVerifier(undefined)).toBe(false);
  });

  test("rejects empty string", () => {
    expect(isValidCodeVerifier("")).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// isSupportedChallengeMethod
// ---------------------------------------------------------------------------
describe("isSupportedChallengeMethod", () => {
  test("accepts S256", () => {
    expect(isSupportedChallengeMethod("S256")).toBe(true);
  });

  test("accepts plain", () => {
    expect(isSupportedChallengeMethod("plain")).toBe(true);
  });

  test("rejects S512", () => {
    expect(isSupportedChallengeMethod("S512")).toBe(false);
  });

  test("rejects empty string", () => {
    expect(isSupportedChallengeMethod("")).toBe(false);
  });

  test("rejects undefined", () => {
    expect(isSupportedChallengeMethod(undefined)).toBe(false);
  });
});
