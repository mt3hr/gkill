package mcp

// 旧 src/mcp/__tests__/pkce.test.mjs の移植。

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
)

// makeS256Pair は既知の code_verifier / code_challenge の対を作る。
func makeS256Pair(verifier string) (string, string) {
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:])
}

func TestVerifyCodeChallengeS256(t *testing.T) {
	t.Run("accepts a valid verifier/challenge pair", func(t *testing.T) {
		verifier, challenge := makeS256Pair("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk")
		expectTrue(t, VerifyCodeChallenge(verifier, challenge, "S256"), "must verify")
	})
	t.Run("rejects an incorrect verifier", func(t *testing.T) {
		_, challenge := makeS256Pair("correct-verifier-value-that-is-long-enough-here")
		expectTrue(t, !VerifyCodeChallenge("wrong-verifier-value-that-is-also-long-enough!", challenge, "S256"), "must reject")
	})
	t.Run("rejects empty verifier", func(t *testing.T) {
		_, challenge := makeS256Pair("some-verifier-value-that-is-long-enough-12345")
		expectTrue(t, !VerifyCodeChallenge("", challenge, "S256"), "must reject")
	})
	t.Run("rejects empty challenge", func(t *testing.T) {
		expectTrue(t, !VerifyCodeChallenge("some-verifier", "", "S256"), "must reject")
	})
	t.Run("rejects null inputs", func(t *testing.T) {
		expectTrue(t, !VerifyCodeChallenge("", "challenge", "S256"), "null verifier")
		expectTrue(t, !VerifyCodeChallenge("verifier", "", "S256"), "null challenge")
	})
	t.Run("works with RFC 7636 Appendix B test vector", func(t *testing.T) {
		// RFC 7636 Appendix B: code_verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		// code_challenge (S256) = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
		verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		challenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
		expectTrue(t, VerifyCodeChallenge(verifier, challenge, "S256"), "RFC vector must verify")
	})
}

// verifyCodeChallenge — plain は廃止 (S256 必須)。plain は verifier==challenge のため
// 中間者が challenge を書き換えるだけで無効化でき、OAuth 2.1 / MCP でも禁止されている。
func TestVerifyCodeChallengePlainRejected(t *testing.T) {
	t.Run("rejects plain even when verifier equals challenge", func(t *testing.T) {
		v := "my-plain-code-verifier-that-is-long-enough-here"
		expectTrue(t, !VerifyCodeChallenge(v, v, "plain"), "plain must be rejected")
	})
}

func TestVerifyCodeChallengeUnsupportedMethod(t *testing.T) {
	t.Run("rejects unknown method", func(t *testing.T) {
		expectTrue(t, !VerifyCodeChallenge("v", "c", "S512"), "S512")
	})
	t.Run("rejects undefined method", func(t *testing.T) {
		expectTrue(t, !VerifyCodeChallenge("v", "c", ""), "undefined")
	})
}

func TestIsValidCodeVerifier(t *testing.T) {
	t.Run("accepts 43-character verifier", func(t *testing.T) {
		expectTrue(t, IsValidCodeVerifier(strings.Repeat("a", 43)), "43")
	})
	t.Run("accepts 128-character verifier", func(t *testing.T) {
		expectTrue(t, IsValidCodeVerifier(strings.Repeat("B", 128)), "128")
	})
	t.Run("accepts verifier with all unreserved characters", func(t *testing.T) {
		expectTrue(t, IsValidCodeVerifier("ABCDEFghij0123456789-._~"+strings.Repeat("x", 19)), "unreserved")
	})
	t.Run("rejects verifier shorter than 43 characters", func(t *testing.T) {
		expectTrue(t, !IsValidCodeVerifier(strings.Repeat("a", 42)), "42")
	})
	t.Run("rejects verifier longer than 128 characters", func(t *testing.T) {
		expectTrue(t, !IsValidCodeVerifier(strings.Repeat("a", 129)), "129")
	})
	t.Run("rejects verifier with invalid characters", func(t *testing.T) {
		expectTrue(t, !IsValidCodeVerifier(strings.Repeat("a", 42)+" "), "space")
		expectTrue(t, !IsValidCodeVerifier(strings.Repeat("a", 42)+"+"), "plus")
		expectTrue(t, !IsValidCodeVerifier(strings.Repeat("a", 42)+"/"), "slash")
		expectTrue(t, !IsValidCodeVerifier(strings.Repeat("a", 42)+"="), "equals")
	})
	t.Run("rejects non-string", func(t *testing.T) {
		expectTrue(t, !IsValidCodeVerifier(123), "number")
		expectTrue(t, !IsValidCodeVerifier(nil), "null")
	})
	t.Run("rejects empty string", func(t *testing.T) {
		expectTrue(t, !IsValidCodeVerifier(""), "empty")
	})
}

func TestIsSupportedChallengeMethod(t *testing.T) {
	t.Run("accepts S256", func(t *testing.T) {
		expectTrue(t, IsSupportedChallengeMethod("S256"), "S256")
	})
	t.Run("rejects plain (S256 only)", func(t *testing.T) {
		expectTrue(t, !IsSupportedChallengeMethod("plain"), "plain")
	})
	t.Run("rejects S512", func(t *testing.T) {
		expectTrue(t, !IsSupportedChallengeMethod("S512"), "S512")
	})
	t.Run("rejects empty string", func(t *testing.T) {
		expectTrue(t, !IsSupportedChallengeMethod(""), "empty")
	})
	t.Run("rejects undefined", func(t *testing.T) {
		expectTrue(t, !IsSupportedChallengeMethod(""), "undefined")
	})
}
