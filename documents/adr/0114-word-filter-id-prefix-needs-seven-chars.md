# ADR-0114: ワード検索の ID 前方一致は7文字以上の語だけにする

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-23 |
| Sources | `.claude/skills/gkill-find-query/SKILL.md`「ワード検索の照合規則は SQL（`GenerateFindSQLCommon`）と Go（`find_word.MatchLoweredWords`。プラグイン SDK も同じ関数）で揃える」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/find_word/match_words.go`（`MinIDPrefixMatchLength` / `IsIDPrefixMatchWord`）/ `src/server/gkill/dao/sqlite3impl/sqlite3impl_util.go`（ワード検索の SQL 追記の直前） |

## Context

[ADR-0113](0113-word-filter-columns-and-id-prefix.md) で ID の照合を部分一致から前方一致へ狭めた。
部分一致なら1文字の hex 語が UUID の約 86% に当たっていたのが、前方一致では 1/16 になった。
それでも短い語は当たる。利用者から「気分の `8` を検索すると git のコミットが出てくる」と報告があった。
git のコミットハッシュも ID として前方一致に掛かるので、`8` で始まるハッシュのコミットが本文と無関係に混ざる。
UUID の記録も同じで、`8` なら 1/16、`8a` なら 1/256 が本文に関係なく当たる。

ID の前方一致を残しているのは、UUID を丸ごと貼る検索と、git の短縮ハッシュで引く検索のためだけである。
どちらも短い語は使わない。

## Decision

肯定語が ID の前方一致を見るのは、語が **7文字（rune 数、前後の空白を除く）以上**のときだけにする。
しきい値は `find_word.MinIDPrefixMatchLength` の1か所に置き、SQL（`GenerateFindSQLCommon`）と Go（`MatchLoweredWords`。
プラグイン SDK も同じ関数）の両方が `find_word.IsIDPrefixMatchWord` で判定する。
完全一致（`findWordUseLike=false` の `ID = ?`）は偶然には当たらないので、長さを問わない。

## Rejected alternatives

- **ID の照合をやめる** — UUID を貼っての検索と git の短縮ハッシュでの検索ができなくなる。
  どちらも Web のキーワード欄から実際に使われている。
- **ID の完全一致だけにする** — UUID 丸ごとは引けるが、git の短縮ハッシュ（既定7文字）が引けなくなる。
- **しきい値を 8（UUID の最初の区切りまで）にする** — git の短縮ハッシュの既定長は7文字（`git log --oneline` の表示）で、
  それをそのまま貼ると外れる。
- **しきい値を 4〜6 にする** — 4文字の hex 語でも 1/65,536 の確率で当たり、数十万件の記録では数件が本文と無関係に混ざる。
  7文字なら 1/2.7億で、実用上は当たらない。7文字未満の ID 断片で引きたい用途も無い。
- **hex だけの語（`[0-9a-f]+`）のときだけ切る** — `8` や `cafe` のような語はまさに hex だけで、切りたいのはそれそのもの。
  ID は hex 以外の文字を含むことがある（プラグインの ID など）ので、字種で分けると説明が増えるだけで効果は変わらない。
- **Lantana など本文列の無い型だけ ID を見ない** — 報告の実害は git のコミット（本文列を持つ型）の側で起きている。
  型で分けると同じ語で型ごとに結果が変わる。
- **しきい値を SQL と Go に別々に書く** — 片方だけを変えると、rep 種別によって結果が食い違い、エラーにならない（ADR-0113 の Consequences と同じ理由）。

## Consequences

- 7文字未満の語は、ID にはもう当たらない。ID の断片で引いていた人には見え方が変わる（UUID は8文字目で区切りがあるので、
  最初の区切りまで貼れば引ける）。
- 対象列の無い呼び出しで短い語を渡すと `0 = 1`（何にも一致しない）になる。プレースホルダとバインド値の数は必ず揃える。
  揃わないと、エラーにならず常に0件になる。
- プラグインは SDK の `find_word` を使うので規則は自動で揃う。ただし、各プラグインのバイナリを作り直して配り直すまでは
  旧規則（長さを問わない前方一致）のまま動く。
- テストの記録に素の UUID を使っていても、1文字の語で「16回に1回落ちる」ことはもう起きない（7文字の語では起きうる）。

## Evidence

- 利用者報告（2026-09-23）: 気分値 `8` の検索で git のコミットが混ざる。
- 確率: 語の長さを n とすると、hex の ID の先頭に偶然一致する確率は 16^-n。n=1 で 6.25%、n=4 で 0.0015%、n=7 で約 3.7×10^-9。

## Related tests

- `src/server/gkill/api/find_word/match_words_test.go`（`TestMatchLoweredWords` の「id_6文字の語は前方一致しない」ほか、`TestIsIDPrefixMatchWord`）
- `src/server/gkill/dao/sqlite3impl/sqlite3impl_util_test.go`（`TestGenerateFindSQLCommon_ShortWordDoesNotMatchID`、`TestGenerateFindSQLCommon_IDMatchesByPrefixOnly`）
