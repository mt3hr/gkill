// gkill 本体を叩くHTTPクライアント。3つのMCPサーバで共有する。
//
// 以前は3本に逐語コピーされていて、同じメソッドが
// callRead / callWrite / callApi と3つの名前で存在していた（中身は同一）。
// 名前は callApi に統一する。
//
// fetchFile は IDF ファイルの実体を取りに行く経路。
// 書き込み専用サーバはファイル系ツールを持たないので呼ばないが、
// 実装を分けるほどの差ではないので同じクラスに置いてある。

import crypto from "node:crypto";

import { Agent, fetch } from "undici";

import { GkillApiError } from "./errors.mjs";

// gkill 側が「セッションが無効」を表すときに返すエラーコード。
// これを見たら1回だけログインし直して再試行する。
export const AUTH_ERROR_CODES = new Set([
  "ERR000002", // AccountNotFoundError
  "ERR000013", // AccountSessionNotFoundError
  "ERR000238", // AccountDisabledError
]);

export class GkillClient {
  constructor() {
    this.baseUrl = process.env.GKILL_BASE_URL || "http://127.0.0.1:9999";
    this.userId = process.env.GKILL_USER || "";
    this.passwordSha256 = process.env.GKILL_PASSWORD_SHA256 || "";
    this.password = process.env.GKILL_PASSWORD || "";
    this.defaultLocale = process.env.GKILL_LOCALE || "ja";
    this.sessionId = process.env.GKILL_SESSION_ID || "";
    const insecure = process.env.GKILL_INSECURE === "true" || process.env.GKILL_INSECURE === "1";
    this.dispatcher = insecure ? new Agent({ connect: { rejectUnauthorized: false } }) : null;
  }

  resolvePasswordSha256() {
    if (this.passwordSha256) {
      return this.passwordSha256;
    }
    if (this.password) {
      return crypto.createHash("sha256").update(this.password).digest("hex");
    }
    return "";
  }

  buildApiUrl(pathname) {
    return new URL(pathname, this.baseUrl).toString();
  }

  hasErrors(responseBody) {
    return Boolean(responseBody && Array.isArray(responseBody.errors) && responseBody.errors.length > 0);
  }

  hasAuthErrors(responseBody) {
    if (!this.hasErrors(responseBody)) {
      return false;
    }
    return responseBody.errors.some((err) => AUTH_ERROR_CODES.has(err.error_code));
  }

  formatErrors(responseBody) {
    if (!this.hasErrors(responseBody)) {
      return "";
    }
    return responseBody.errors
      .map((err) => `${err.error_code ?? "UNKNOWN"}: ${err.error_message ?? "unknown error"}`)
      .join("; ");
  }

  async post(pathname, body) {
    const url = this.buildApiUrl(pathname);
    const timeoutMs = parseInt(process.env.GKILL_FETCH_TIMEOUT_MS || "120000", 10);
    let response;
    try {
      const fetchOptions = {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(body),
        signal: AbortSignal.timeout(timeoutMs),
      };
      if (this.dispatcher) {
        fetchOptions.dispatcher = this.dispatcher;
      }
      response = await fetch(url, fetchOptions);
    } catch (error) {
      throw new GkillApiError(`Network error at ${pathname}.`, {
        url,
        message: error instanceof Error ? error.message : String(error),
        cause:
          error && typeof error === "object" && "cause" in error
            ? String(error.cause && error.cause.message ? error.cause.message : error.cause)
            : null,
      });
    }

    let jsonBody;
    try {
      jsonBody = await response.json();
    } catch (error) {
      throw new GkillApiError(`Failed to parse JSON response from ${pathname}.`, {
        cause: String(error),
      });
    }

    if (!response.ok) {
      throw new GkillApiError(`HTTP ${response.status} from ${pathname}.`, {
        status: response.status,
        body: jsonBody,
      });
    }

    return jsonBody;
  }

  async login() {
    if (this.sessionId) {
      return this.sessionId;
    }

    const passwordSha256 = this.resolvePasswordSha256();
    if (!this.userId || !passwordSha256) {
      throw new GkillApiError(
        "Missing login credentials. Set GKILL_USER and GKILL_PASSWORD_SHA256 (or GKILL_PASSWORD).",
      );
    }

    const response = await this.post("/api/login", {
      user_id: this.userId,
      password_sha256: passwordSha256,
      locale_name: this.defaultLocale,
    });

    if (this.hasErrors(response)) {
      throw new GkillApiError(`Login failed: ${this.formatErrors(response)}`, response);
    }
    if (!response.session_id) {
      throw new GkillApiError("Login succeeded but session_id is missing.", response);
    }

    this.sessionId = response.session_id;
    return this.sessionId;
  }

  async callApi(pathname, requestBody, requiresAuth, sessionIdOverride = null) {
    const localeName = requestBody.locale_name || this.defaultLocale;
    const body = {
      ...requestBody,
      locale_name: localeName,
    };

    if (requiresAuth) {
      body.session_id = sessionIdOverride || body.session_id || (await this.login());
    }

    let response = await this.post(pathname, body);
    if (requiresAuth && this.hasAuthErrors(response)) {
      this.sessionId = "";
      body.session_id = await this.login();
      response = await this.post(pathname, body);
    }

    if (this.hasErrors(response)) {
      throw new GkillApiError(`API error at ${pathname}: ${this.formatErrors(response)}`, response);
    }
    return response;
  }

  async fetchFile(filePath, sessionId) {
    const url = this.buildApiUrl(filePath);
    const timeoutMs = parseInt(process.env.GKILL_FETCH_TIMEOUT_MS || "120000", 10);
    const fetchOptions = {
      method: "GET",
      headers: {
        Cookie: `gkill_session_id=${sessionId}`,
      },
      signal: AbortSignal.timeout(timeoutMs),
    };
    if (this.dispatcher) {
      fetchOptions.dispatcher = this.dispatcher;
    }
    let response;
    try {
      response = await fetch(url, fetchOptions);
    } catch (error) {
      throw new GkillApiError(`Network error fetching file ${filePath}.`, {
        url,
        message: error instanceof Error ? error.message : String(error),
      });
    }
    if (!response.ok) {
      throw new GkillApiError(`HTTP ${response.status} fetching file ${filePath}.`, {
        status: response.status,
      });
    }
    const contentType = response.headers.get("content-type") || "application/octet-stream";
    const buffer = Buffer.from(await response.arrayBuffer());
    return { buffer, contentType };
  }
}
