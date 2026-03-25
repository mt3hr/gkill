// PKCE (Proof Key for Code Exchange) utilities for OAuth 2.1.
// Supports S256 (SHA-256) and plain challenge methods per RFC 7636.

import crypto from "node:crypto";

/**
 * Verify a PKCE code_verifier against a stored code_challenge.
 * @param {string} codeVerifier - The verifier sent by the client in the token request.
 * @param {string} codeChallenge - The challenge stored during the authorization request.
 * @param {string} method - "S256" or "plain".
 * @returns {boolean} True if the verifier matches the challenge.
 */
export function verifyCodeChallenge(codeVerifier, codeChallenge, method) {
  if (!codeVerifier || !codeChallenge) return false;

  if (method === "S256") {
    const hash = crypto.createHash("sha256").update(codeVerifier, "ascii").digest();
    const computed = hash.toString("base64url");
    return computed === codeChallenge;
  }

  if (method === "plain") {
    return codeVerifier === codeChallenge;
  }

  return false;
}

/**
 * Validate that a code_verifier conforms to RFC 7636 Section 4.1:
 * 43-128 characters, unreserved characters only [A-Z a-z 0-9 -._~].
 * @param {string} value
 * @returns {boolean}
 */
export function isValidCodeVerifier(value) {
  if (typeof value !== "string") return false;
  if (value.length < 43 || value.length > 128) return false;
  return /^[A-Za-z0-9\-._~]+$/.test(value);
}

/**
 * Validate that a code_challenge_method is supported.
 * @param {string} method
 * @returns {boolean}
 */
export function isSupportedChallengeMethod(method) {
  return method === "S256" || method === "plain";
}
