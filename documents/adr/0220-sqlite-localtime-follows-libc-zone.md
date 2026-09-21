# ADR-0220: SQLite の 'localtime' は libc のゾーンで決まるので、Android では libc にも端末のゾーンを教える

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-16 |
| Sources | `.claude/skills/gkill-go-backend/SKILL.md`「SQLite の 'localtime' は Go の time.Local と独立に決まる」 / `.claude/skills/gkill-mobile/SKILL.md`「同梱サーバのタイムゾーンは Go と libc の両方へ教える」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/main/common/fix_timezone.go` / `src/server/gkill/dao/sqlite3impl/localtime_check.go` |

## Context

時間帯フィルタ（開始/終了/曜日）は2段で判定する。SQL 段は `GenerateFindSQLCommon` が
`strftime('%H:%M:%S', datetime(列, 'unixepoch', 'localtime'))` を `"11:45:00"` のような文字列と比較して行を落とし、
Go 段は `find_filter.go` の `newKyouTimeFilter` が `RelatedTime.In(time.Local)` で再判定する。

`'localtime'` 修飾子は SQLite が libc の `localtime_r` に聞く値で、gkill の SQLite（modernc、musl を Go に転写したもの）は
`TZ` 環境変数 → `/etc/localtime` → どちらも無ければ UTC で決める。一方 Go の `time.Local` は Android では常に UTC
（`time/zoneinfo_android.go` の `initLocal`）なので、`main/common` の `fixTimezone` が `getprop persist.sys.timezone` で
直していた。ただし直していたのは Go 側だけだった。Android には `/etc/localtime` が無く `TZ` も未設定なので、
**SQL 段は UTC・Go 段は JST** という状態で動いていた。

12:09 JST の記録は SQL 段では `03:09:04` として窓の外に落ち、代わりに SQL 段を通る「UTC で 11:45〜12:45」
（JST 20:45〜21:45）の記録は Go 段で落ちる。両段を同時に満たす記録は存在しないので、9時間より狭い窓の検索は
**エラーも警告も出ないまま常に0件**だった。曜日だけの指定でも JST 0〜9 時の記録は SQL 段で前日扱いになって消える。
PC（Windows / tzdata のある Linux）では libc も同じゾーンを見るので再現せず、2026-09-16 にスマホで「11:45〜12:45
水曜で検索してもヒットしない」として見つかった。

## Decision

Android では、Go の `time.Local` を直すのと同じ `fixTimezone` の経路で libc にも同じゾーンを教える。端末の packed tzdata
（AOSP の索引形式）から TZif を切り出して `$GKILL_HOME/tz/localtime` に置き、`TZ=:<そのパス>` を最初の SQLite 接続より前に
環境変数へ入れる。tzdata が読めない端末では POSIX の固定オフセット文字列（例 `JST-9`）で妥協する。

加えて、起動時に SQLite の `'localtime'` と Go の `time.Local` を同じ瞬間で突き合わせ（`CheckLocaltimeAgreesWithGo`）、
食い違っていれば `gkill_error.log` に両方の値を1行残す。Android に限らない（tzdata を入れ忘れた Docker 上の Linux でも同じ穴に落ちる）。

## Rejected alternatives

- **SQL から時間帯・曜日の条件を外し、Go 段だけで判定する（`'localtime'` に依存しなくなる）** — SQL 段は行を Go へ渡す前に
  落とす前置フィルタで、全履歴走査のときに効く。外すと 74 万行規模のアカウントでは全行を Go の構造体へ実体化してから
  捨てることになり、いまでも全 rep で1分以上かかる検索が数分〜メモリ枯渇へ向かう。
- **`'localtime'` をやめ、Go で計算したオフセット秒を `datetime(列 + ?, 'unixepoch')` にバインドする** — オフセットは
  行ごとに違う（夏時間のある地域では半年ごとに変わる）。クエリ時点のオフセット1つで代用すると、DST の反対側にある行は
  SQL 段で1時間ずれて判定され、窓の端の1時間ぶんが黙って落ちる。gkill は7言語で配布しており日本だけの前提を置けない。
- **POSIX の固定オフセット文字列だけで済ませる（TZif の切り出しをしない）** — 同じ理由で DST 地域では半年ずれる。
  fallback としてだけ残した。
- **Android の tzdata を読まず、Go に埋め込んである `time/tzdata` の TZif を書き出す** — 埋め込みデータは `time` パッケージの
  非公開領域にあり取り出せない。自前で `zoneinfo.zip` をもう一度埋め込むと 450KB がバイナリと更新作業に二重に載る。
- **Android アプリ（`MainActivity` の `ProcessBuilder`）側で `TZ` を渡す** — Termux 等で同じバイナリを手で起動する経路には効かない。
  サーバ自身が直せば起動経路によらず揃う。
- **食い違いを検出したら起動を止める** — 時間帯フィルタ以外は正しく動くので、止めるより原因を1行残して動かすほうが役に立つ。
  黙るのは駄目だが、止めるのもやり過ぎ。

## Consequences

- SQLite の `'localtime'` を使う SQL は、libc のゾーンが Go と揃っている前提でしか正しくない。**揃っていなくても
  SQL はエラーにならず、結果が静かに変わるだけ**。新しい `'localtime'` の用法を足すときは、この前提が要ることを知っておくこと
  （TimeIs の playing 判定は両辺に同じ修飾子が掛かるので相殺されて影響しない）。
- `TZ` を環境変数へ入れるのは最初の SQLite 接続より前でなければ効かない。modernc の libc は最初の接続を開くときに
  `os.Environ()` を一度だけ写し取り、以後 `os.Setenv` しても `getenv` には映らない。`InitGkillOptions` の末尾で入れているのは
  そのため（`InitGkillServerAPI` が最初の接続を開く）。
- **`TZ=:<パス>` のパスは環境変数を展開した絶対パスでなければならない。** musl の `do_tzset` は `:` の後ろが `/` でも `.` でも
  始まらない名前を `/usr/share/zoneinfo/` `/share/zoneinfo/` `/etc/zoneinfo/` の相対名として探し、無ければエラーも警告も出さず
  UTC にする。既定の `--gkill_home_dir` は `$HOME/gkill` の**未展開文字列**（`gkill_options/option.go`。他の利用箇所は使う側で
  `os.ExpandEnv` する流儀）で、この ADR の初版はそれをそのまま `applyLibcTimezone` に渡していたため、`TZ=:$HOME/gkill/tz/localtime`
  （リテラル）になり、CWD 直下に文字どおり `$HOME` という名のディレクトリを掘って TZif を置いたうえで libc は UTC のままだった。
  APK 同梱サーバは `MainActivity` が絶対パスで `--gkill_home_dir` を渡すので踏まず、`--gkill_home_dir` を渡さない Termux だけが
  同日（2026-09-16）に「修正後も0件」として見つかった。いまは `InitGkillOptions` が展開済みの絶対パスを渡し、
  `applyLibcTimezoneFor` 側でも `os.ExpandEnv` + `filepath.Abs` して絶対パスにできなければ TZif 経路を使わず POSIX 文字列へ落ちる。
- `$GKILL_HOME/tz/localtime` は起動のたびに中身を比べ、同じなら書き直さない。端末のゾーンを変えれば次の起動で置き換わる。
- 自己検査は Android では起動ログで確認できる（一致なら Debug、不一致なら Error）。CI（Linux + tzdata）では
  `TZ=:/nonexistent` の子プロセスで Android と同じ食い違いを再現し、検出できることを固定している。

## Evidence

- 本番 PC（Windows サービス）: Web と同じ epoch 形式で `/api/get_kyous` を叩いても、rykv の実 UI（時間帯 11:45–12:45・水）でも
  対象の記録がヒット（75件の先頭）。PC では libc も JST。
- WSL（linux/amd64。android/arm64 と同じ musl 転写コード）で modernc sqlite を使う探針: ゾーンファイルが見つからない条件
  （`TZ=:/nonexistent`）では `datetime(…,'localtime')` が `03:09:04`（UTC）、`TZ=JST-9` では `12:09:04`、`TZ=:/usr/share/zoneinfo/Asia/Tokyo`
  でも `12:09:04`。Go の `time.Local` は `TZ=JST-9` を解釈できず UTC のまま（POSIX 文字列は libc 専用）。
- Go 本体 `time/zoneinfo_android.go`: `initLocal() { localLoc = *UTC }`（`// TODO: getprop persist.sys.timezone`）。
- 全 rep で日付範囲なしの時間帯検索は1分以上（数千件）。SQL 段を外す案を採らない根拠。
- musl 転写（modernc libc v1.74.3 の `_do_tzset`）: `TZ=:$HOME/gkill/tz/localtime` は `posix_form` 0 → 先頭が `/` でも `.` でもない →
  `search` の3ディレクトリで `__map_file` → 全て失敗 → `s = __utc`。WSL の子プロセスで、CWD に文字どおり `$HOME/gkill/tz/localtime`
  として本物の TZif を置いても `datetime(…,'localtime')` は `03:09:04`（UTC）、同じファイルを絶対パスで渡すと `12:09:04`
  （`TestCheckLocaltimeAgreesWithGo_RelativeTZPathFallsBackToUTC`）。
- 未展開のまま渡す旧実装に `$GKILL_TEST_HOME/gkill` を与えると `TZ=:$GKILL_TEST_HOME\gkill\tz\localtime` になる
  （`TestApplyLibcTimezoneForExpandsEnvAndUsesAbsolutePath` を旧実装で走らせた変異確認）。

## Related tests

- `src/server/gkill/dao/sqlite3impl/localtime_check_test.go`（`TestCheckLocaltimeAgreesWithGo_RelativeTZPathFallsBackToUTC` を含む）
- `src/server/gkill/main/common/fix_timezone_test.go`（`TestApplyLibcTimezoneForExpandsEnvAndUsesAbsolutePath` を含む）
