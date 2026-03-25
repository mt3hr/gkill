// OAuth 2.1 Authorization Server for gkill MCP.
// Implements the MCP spec 2025-03-26 combined authorization server + resource server model.
// Delegates user credential verification to the gkill backend via /api/login.

import crypto from "node:crypto";
import { URL } from "node:url";

import { OAuthStore, TTL, generateToken } from "./oauth-store.mjs";
import { verifyCodeChallenge, isValidCodeVerifier, isSupportedChallengeMethod } from "./pkce.mjs";
import { renderLoginPage, renderSuccessPage } from "./oauth-html.mjs";

/**
 * OAuth 2.1 server for MCP HTTP transport.
 *
 * @param {object} options
 * @param {string} options.issuer - Issuer URL (e.g., "http://localhost:8808").
 * @param {function} options.authenticateUser - async (userId, passwordSha256) => { sessionId } | null.
 *   Called to verify credentials against the gkill backend.
 */
export class OAuthServer {
  /**
   * @param {object} options
   * @param {string} options.issuer - Issuer URL (e.g., "http://localhost:8808").
   * @param {function} options.authenticateUser - async (userId, passwordSha256) => { sessionId } | null.
   * @param {string} [options.persistPath] - Path to JSON file for persisting refresh tokens and DCR clients.
   */
  constructor({ issuer, authenticateUser, persistPath }) {
    if (!issuer) throw new Error("OAuthServer requires an issuer URL");
    if (typeof authenticateUser !== "function") {
      throw new Error("OAuthServer requires an authenticateUser function");
    }
    this.issuer = issuer.replace(/\/+$/, "");
    this.authenticateUser = authenticateUser;
    this.store = new OAuthStore(persistPath);
    this.store.load();
    this.store.startCleanup();
  }

  /** Stop background cleanup (for tests or shutdown). */
  close() {
    this.store.stopCleanup();
  }

  // ---------------------------------------------------------------------------
  // Metadata endpoint: GET /.well-known/oauth-authorization-server
  // ---------------------------------------------------------------------------

  getMetadata() {
    return {
      issuer: this.issuer,
      authorization_endpoint: `${this.issuer}/oauth/authorize`,
      token_endpoint: `${this.issuer}/oauth/token`,
      registration_endpoint: `${this.issuer}/oauth/register`,
      response_types_supported: ["code"],
      grant_types_supported: ["authorization_code", "refresh_token"],
      token_endpoint_auth_methods_supported: ["none"],
      code_challenge_methods_supported: ["S256", "plain"],
      scopes_supported: ["gkill:read"],
    };
  }

  // ---------------------------------------------------------------------------
  // Authorization endpoint: GET /oauth/authorize — show login form
  // ---------------------------------------------------------------------------

  handleAuthorizeGet(query) {
    const { client_id, redirect_uri, state, code_challenge, code_challenge_method, scope, response_type, resource } = query;

    const error = this._validateAuthorizeParams(query);
    if (error) {
      return { status: 400, contentType: "text/html", body: this._errorHtml(error) };
    }

    const html = renderLoginPage({
      clientId: client_id,
      redirectUri: redirect_uri,
      state: state || "",
      codeChallenge: code_challenge,
      codeChallengeMethod: code_challenge_method || "S256",
      scope: scope || "gkill:read",
      resource: resource || "",
      error: null,
    });
    return { status: 200, contentType: "text/html", body: html };
  }

  // ---------------------------------------------------------------------------
  // Authorization endpoint: POST /oauth/authorize — authenticate and issue code
  // ---------------------------------------------------------------------------

  async handleAuthorizePost(formData) {
    const {
      client_id, redirect_uri, state, code_challenge,
      code_challenge_method, scope, response_type,
      user_id, password_sha256, resource,
    } = formData;

    const error = this._validateAuthorizeParams(formData);
    if (error) {
      return { status: 400, contentType: "text/html", body: this._errorHtml(error) };
    }

    // Authenticate user via gkill backend
    let authResult;
    try {
      authResult = await this.authenticateUser(user_id, password_sha256 || "");
    } catch {
      authResult = null;
    }

    if (!authResult || !authResult.sessionId) {
      // Re-render login form with error
      const html = renderLoginPage({
        clientId: client_id,
        redirectUri: redirect_uri,
        state: state || "",
        codeChallenge: code_challenge,
        codeChallengeMethod: code_challenge_method || "S256",
        scope: scope || "gkill:read",
        resource: resource || "",
        error: "ログインに失敗しました。ユーザーIDまたはパスワードを確認してください。",
      });
      return { status: 200, contentType: "text/html", body: html };
    }

    // Generate authorization code
    const code = generateToken();
    this.store.putCode(code, {
      clientId: client_id,
      redirectUri: redirect_uri,
      codeChallenge: code_challenge,
      codeChallengeMethod: code_challenge_method || "S256",
      scope: scope || "gkill:read",
      resource: resource || "",
      gkillSessionId: authResult.sessionId,
      userId: user_id,
    });

    // Build redirect URL with code
    const redirectUrl = new URL(redirect_uri);
    redirectUrl.searchParams.set("code", code);
    if (state) redirectUrl.searchParams.set("state", state);

    // Show success page with auto-redirect (instead of immediate 302)
    const html = renderSuccessPage({ redirectUrl: redirectUrl.toString() });
    return { status: 200, contentType: "text/html", body: html };
  }

  // ---------------------------------------------------------------------------
  // Token endpoint: POST /oauth/token
  // ---------------------------------------------------------------------------

  handleTokenRequest(body) {
    const { grant_type } = body;

    if (grant_type === "authorization_code") {
      return this._handleAuthorizationCodeGrant(body);
    }

    if (grant_type === "refresh_token") {
      return this._handleRefreshTokenGrant(body);
    }

    return this._tokenError("unsupported_grant_type", "Supported: authorization_code, refresh_token");
  }

  _handleAuthorizationCodeGrant(body) {
    const { code, code_verifier, client_id, redirect_uri, resource } = body;

    if (!code) return this._tokenError("invalid_request", "Missing code");
    if (!code_verifier) return this._tokenError("invalid_request", "Missing code_verifier");

    // Retrieve and consume the authorization code (one-time use)
    const codeData = this.store.getAndDeleteCode(code);
    if (!codeData) {
      return this._tokenError("invalid_grant", "Invalid or expired authorization code");
    }

    // Validate client_id matches
    if (client_id && client_id !== codeData.clientId) {
      return this._tokenError("invalid_grant", "client_id mismatch");
    }

    // Validate redirect_uri matches
    if (redirect_uri && redirect_uri !== codeData.redirectUri) {
      return this._tokenError("invalid_grant", "redirect_uri mismatch");
    }

    // Validate resource matches (RFC 8707)
    if (resource && codeData.resource && resource !== codeData.resource) {
      return this._tokenError("invalid_grant", "resource mismatch");
    }

    // Validate PKCE
    if (!verifyCodeChallenge(code_verifier, codeData.codeChallenge, codeData.codeChallengeMethod)) {
      return this._tokenError("invalid_grant", "PKCE verification failed");
    }

    // Issue tokens
    const accessToken = generateToken();
    const refreshToken = generateToken();
    const tokenData = {
      clientId: codeData.clientId,
      scope: codeData.scope,
      gkillSessionId: codeData.gkillSessionId,
      userId: codeData.userId,
    };

    this.store.putAccessToken(accessToken, tokenData);
    this.store.putRefreshToken(refreshToken, tokenData);

    return {
      status: 200,
      body: {
        access_token: accessToken,
        token_type: "Bearer",
        expires_in: Math.floor(TTL.ACCESS_TOKEN / 1000),
        refresh_token: refreshToken,
        scope: codeData.scope,
      },
    };
  }

  _handleRefreshTokenGrant(body) {
    const { refresh_token, client_id } = body;

    if (!refresh_token) return this._tokenError("invalid_request", "Missing refresh_token");

    const tokenData = this.store.getRefreshToken(refresh_token);
    if (!tokenData) {
      return this._tokenError("invalid_grant", "Invalid or expired refresh token");
    }

    // Validate client_id if provided
    if (client_id && client_id !== tokenData.clientId) {
      return this._tokenError("invalid_grant", "client_id mismatch");
    }

    // Rotate refresh token (delete old, issue new)
    this.store.deleteRefreshToken(refresh_token);
    const newAccessToken = generateToken();
    const newRefreshToken = generateToken();

    const newTokenData = { ...tokenData };
    this.store.putAccessToken(newAccessToken, newTokenData);
    this.store.putRefreshToken(newRefreshToken, newTokenData);

    return {
      status: 200,
      body: {
        access_token: newAccessToken,
        token_type: "Bearer",
        expires_in: Math.floor(TTL.ACCESS_TOKEN / 1000),
        refresh_token: newRefreshToken,
        scope: tokenData.scope,
      },
    };
  }

  // ---------------------------------------------------------------------------
  // Dynamic Client Registration: POST /oauth/register
  // ---------------------------------------------------------------------------

  handleRegister(body) {
    const { redirect_uris, client_name } = body;

    if (!redirect_uris || !Array.isArray(redirect_uris) || redirect_uris.length === 0) {
      return {
        status: 400,
        body: { error: "invalid_client_metadata", error_description: "redirect_uris is required" },
      };
    }

    // Validate each redirect_uri is a valid URL
    for (const uri of redirect_uris) {
      try {
        new URL(uri);
      } catch {
        return {
          status: 400,
          body: { error: "invalid_client_metadata", error_description: `Invalid redirect_uri: ${uri}` },
        };
      }
    }

    const clientId = generateToken();
    const metadata = {
      client_id: clientId,
      client_id_issued_at: Math.floor(Date.now() / 1000),
      client_name: client_name || "",
      redirect_uris,
      grant_types: ["authorization_code", "refresh_token"],
      response_types: ["code"],
      token_endpoint_auth_method: "none",
    };

    this.store.putClient(clientId, metadata);

    return { status: 201, body: metadata };
  }

  // ---------------------------------------------------------------------------
  // Token validation (for resource server — validating incoming Bearer tokens)
  // ---------------------------------------------------------------------------

  /**
   * Validate a Bearer access token.
   * @param {string} token
   * @returns {object|null} Token data ({ clientId, scope, gkillSessionId, userId }) or null.
   */
  validateAccessToken(token) {
    if (!token) return null;
    return this.store.getAccessToken(token);
  }

  /**
   * Extract Bearer token from an Authorization header value.
   * @param {string} authHeader
   * @returns {string} Token string or empty string.
   */
  static extractBearerToken(authHeader) {
    if (!authHeader || !authHeader.startsWith("Bearer ")) return "";
    return authHeader.slice(7);
  }

  // ---------------------------------------------------------------------------
  // Internal helpers
  // ---------------------------------------------------------------------------

  _validateAuthorizeParams(params) {
    if (!params.response_type || params.response_type !== "code") {
      return "response_type must be 'code'";
    }
    if (!params.client_id) {
      return "client_id is required";
    }
    if (!params.redirect_uri) {
      return "redirect_uri is required";
    }
    try {
      new URL(params.redirect_uri);
    } catch {
      return "redirect_uri must be a valid URL";
    }
    if (!params.code_challenge) {
      return "code_challenge is required (PKCE)";
    }
    if (params.code_challenge_method && !isSupportedChallengeMethod(params.code_challenge_method)) {
      return `Unsupported code_challenge_method: ${params.code_challenge_method}. Use S256 or plain.`;
    }
    // Validate redirect_uri against DCR-registered client
    const client = this.store.getClient(params.client_id);
    if (client && client.redirect_uris) {
      if (!client.redirect_uris.includes(params.redirect_uri)) {
        return "redirect_uri does not match registered client";
      }
    }
    return null;
  }

  _tokenError(error, description) {
    return {
      status: 400,
      body: { error, error_description: description },
    };
  }

  _errorHtml(message) {
    const esc = (s) => String(s).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
    return `<!DOCTYPE html><html><head><meta charset="utf-8"><title>Error</title></head><body><h1>Error</h1><p>${esc(message)}</p></body></html>`;
  }
}
