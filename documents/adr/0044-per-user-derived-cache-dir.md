# ADR-0044: 派生キャッシュは利用者IDでディレクトリを分ける

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-03 |
| Sources | `0575753e` / `documents/reverse/design-philosophy.md`「派生キャッシュを利用者ごとに分ける判断」 / `CLAUDE.md` の `clear_cache` の節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/reps/local_rep_cache_path.go` |

## Context

派生キャッシュ（サムネイル / 動画 / ZIP展開）は rep 名でディレクトリを切っていた。

ところが **rep 名は利用者間で一意ではない**。`filepath.Base(contentDir)` で決まり、UNIQUE 制約も無い。別の利用者が同じ名前の rep を持つのは普通に起きる。

## Decision

派生キャッシュは**利用者IDでディレクトリを分ける**。

```
caches/zip_cache/{userID}/{repName}/{sha1(zipPath)}/
caches/{thumb,video}_cache/{userID}/{repName}/
```

配信の起点は**セッション由来のパス**に固定する。

## Rejected alternatives

- **rep 名の照合で他人のキャッシュを弾く** — **原理的に守れない。** 同名の rep が別の利用者に存在するので、名前だけでは所有者を決められない。

- **キャッシュのファイル名にハッシュを入れて推測不能にする** — 推測できないだけで、知られたら取れる。**利用者IDをパスに挟めば、他人のディレクトリを指名すること自体ができなくなる**（配信の起点がセッション由来なので）。

- **キャッシュを利用者ごとに分けず、配信時に毎回所有権を検証する** — 検証を1箇所書き忘れれば穴が開く。パスの構造で保証するほうが、書き忘れが起きる場所自体が無くなる（→ ADR-0041 と同じ形の判断）。

## Consequences

`Clear{Thumb,Video,Zip}Cache(userID)` が userID を取ることになった。**rep 名だけで消す API を足さないこと**（他人のキャッシュを巻き込むか、逆に消し残す）。

`clear_cache <thumb|video|zip|plugin|all> <all|user_id...>` は対象を必須にしてある。`all` を渡せば `$HOME/gkill/caches/` 配下を丸ごと消す（利用者の文脈が要らない）。user_id を渡せばその利用者の rep だけを読み込んで消す。

サムネイル / 動画には専用ルートが無く、`/files/{repName}/...?thumb=` からのみ到達できる。

## Evidence

実測なし — 構造からの判断（rep 名が利用者間で一意でないことをスキーマで確認した）。

## Related tests

- `src/server/gkill/dao/reps/derived_cache_path_test.go`
- `src/server/gkill/api/gkill_server_api/handle_zip_cache_file_serve_test.go`
