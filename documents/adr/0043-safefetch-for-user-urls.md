# ADR-0043: 利用者入力URLと og:image の取得は必ず api/safefetch を通す

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-21 |
| Sources | `bb364253`（監査 H-04） / `.claude/skills/gkill-go-backend/SKILL.md`「利用者入力URL・そのページが指す og:image / #landingImage の取得は必ず api/safefetch を通す」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/safefetch/safefetch.go` |

## Context

URLog（ブックマーク）は、利用者が入れたURLに対して**サーバ側で**本文・favicon・og:image を取りに行く。素の `http.Get` だと3つの問題がある。

- **SSRF** … クラウドのメタデータエンドポイント（`169.254.169.254`）やLAN内のサービスへ到達できる
- **無制限read** … 応答サイズに上限が無く、巨大な応答でメモリを食い潰せる
- **画像爆弾** … 圧縮された巨大画像を復号すると、ピクセル数ぶんのメモリを確保する

## Decision

利用者入力URLと、そのページが指す og:image / `#landingImage` の取得は**必ず `api/safefetch` を通す**。

- `safefetch.GetCapped` … scheme 検査・`Dialer.Control` での**接続先IP検証**・サイズ上限
- `safefetch.CheckImageDimensions` … `image.DecodeConfig` で**復号前に**総ピクセル数を検査

既定は private 拒否（loopback / RFC1918 / link-local（メタデータ）/ multicast / unspecified）。

## Rejected alternatives

- **URLの文字列を検査してから `http.Get` する** — **DNSリバインディングで抜けられる。** 検査時と接続時で名前解決の結果が変わりうるので、**接続の直前に実IPを見る**必要がある。`Dialer.Control` はそのためのフック。

- **リダイレクトは追わないことにする** — 追わないと普通のブックマークが取れない。追ううえで、**リダイレクト先も含めて毎回IPを検査する**。

- **画像は読んでからサイズを見る** — 読んだ時点で確保が終わっている。`image.DecodeConfig` はヘッダだけを読むので、**復号前に**総ピクセル数で弾ける。

- **サイズ上限だけ設けて IP 検査はしない** — SSRF が残る。メタデータエンドポイントの応答は小さい。

## Consequences

**`http.Get` を新しく直に書かないこと。** 利用しているのは `dao/reps/ur_log.go` の `getBody` / `getFavicon` / `getImageOG` / `getAmazonImage` と、`gkill_server_api` の `httpGetBase64Data`（ブックマークレット）。

private 拒否が既定なので、**LAN内のページはブックマークしても本文が取れない**。これは意図した制約。

## Evidence

実測なし — 脅威モデルからの判断（外部監査 H-04 の指摘）。

## Related tests

- `src/server/gkill/api/safefetch/safefetch_test.go`
- `src/server/gkill/api/gkill_server_api/utils_ssrf_test.go`
