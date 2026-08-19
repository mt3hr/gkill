// stdio (標準入出力) の JSON-RPC トランスポート。3つのMCPサーバで共有する。
//
// 以前は3本に逐語コピーされていて、write サーバだけが不正な行をログに残す形へ
// 分岐していた（read / readwrite は黙って捨てていた）。ログを残す側へ揃えてある。
//
// LSP形式 (Content-Length ヘッダ) と NDJSON (1行1メッセージ) の両方を受ける。

// StdioTransport: reads JSON-RPC from stdin (LSP or NDJSON framing), writes NDJSON to stdout.
export class StdioTransport {
  constructor(server) {
    this.server = server;
    this.buffer = Buffer.alloc(0);
    // stdioで話す相手はこのマシン上のプロセスなので、ファイルの絶対パスを渡してよい
    this.server.isLocalTransport = true;
  }

  start() {
    process.stdin.on("data", (chunk) => this.onData(chunk));
    process.stdin.on("error", (e) => this.logError("stdin error", e));
    process.stdin.resume();
  }

  logError(message, error) {
    process.stderr.write(`${message}: ${String(error)}\n`);
  }

  writeMessage(message) {
    const json = JSON.stringify(message);
    process.stdout.write(`${json}\n`);
  }

  async dispatch(message) {
    try {
      const response = await this.server.handlePayload(message);
      if (response) this.writeMessage(response);
    } catch (error) {
      this.logError("unhandled request error", error);
      if (message && !Array.isArray(message) && Object.prototype.hasOwnProperty.call(message, "id")) {
        this.writeMessage({ jsonrpc: "2.0", id: message.id, error: { code: -32603, message: "Internal error" } });
      }
    }
  }

  onData(chunk) {
    this.buffer = Buffer.concat([this.buffer, chunk]);

    while (true) {
      // LSP-style framing: "Content-Length: N\r\n\r\n{...}"
      const headerEnd = this.buffer.indexOf("\r\n\r\n");
      if (headerEnd !== -1) {
        const headerText = this.buffer.subarray(0, headerEnd).toString("utf8");
        const headers = headerText.split("\r\n");
        let contentLength = null;
        for (const line of headers) {
          const idx = line.indexOf(":");
          if (idx === -1) continue;
          const key = line.slice(0, idx).trim().toLowerCase();
          const value = line.slice(idx + 1).trim();
          if (key === "content-length") {
            contentLength = Number.parseInt(value, 10);
          }
        }

        if (!Number.isFinite(contentLength) || contentLength < 0) {
          this.logError("invalid content-length header", headerText);
          this.buffer = Buffer.alloc(0);
          return;
        }

        const totalLength = headerEnd + 4 + contentLength;
        if (this.buffer.length < totalLength) return;

        const bodyBuffer = this.buffer.subarray(headerEnd + 4, totalLength);
        this.buffer = this.buffer.subarray(totalLength);

        let message;
        try {
          message = JSON.parse(bodyBuffer.toString("utf8"));
        } catch (error) {
          this.logError("invalid json body", error);
          continue;
        }

        this.dispatch(message);
        continue;
      }

      // NDJSON-style framing: one JSON-RPC message per line.
      const lf = this.buffer.indexOf("\n");
      if (lf === -1) return;
      const line = this.buffer.subarray(0, lf).toString("utf8").trim();
      this.buffer = this.buffer.subarray(lf + 1);
      if (line.length === 0) continue;

      let message;
      try {
        message = JSON.parse(line);
      } catch (error) {
        // 不正な行を黙って捨てると、クライアント側の枠組みの不具合を追えない
        this.logError("invalid json line", error);
        continue;
      }
      this.dispatch(message);
    }
  }
}
