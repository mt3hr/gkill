package mcp

// PKCE (Proof Key for Code Exchange) utilities for OAuth 2.1.
// S256 (SHA-256) のみをサポートする。OAuth 2.1 / MCP は S256 を必須とし、
// plain は verifier==challenge のため中間者が challenge を書き換えるだけで無効化できる。

import (
	"crypto/sha256"
	"encoding/base64"
	"regexp"
)

var codeVerifierRegex = regexp.MustCompile(`^[A-Za-z0-9\-._~]+$`)

// VerifyCodeChallenge は code_verifier が保存済みの code_challenge と一致するか（S256 のみ）。
func VerifyCodeChallenge(codeVerifier, codeChallenge, method string) bool {
	if codeVerifier == "" || codeChallenge == "" {
		return false
	}
	if method == "S256" {
		sum := sha256.Sum256([]byte(codeVerifier))
		computed := base64.RawURLEncoding.EncodeToString(sum[:])
		return computed == codeChallenge
	}
	return false
}

// IsValidCodeVerifier は RFC 7636 4.1 の code_verifier か（43〜128 文字、unreserved のみ）。
func IsValidCodeVerifier(value any) bool {
	s, ok := value.(string)
	if !ok {
		return false
	}
	if len(s) < 43 || len(s) > 128 {
		return false
	}
	return codeVerifierRegex.MatchString(s)
}

// IsSupportedChallengeMethod は code_challenge_method が受理できる値か（S256 のみ）。
func IsSupportedChallengeMethod(method string) bool {
	return method == "S256"
}
