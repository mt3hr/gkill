# ADR-0319: プラグイン SDK が Android で time.Local を端末のゾーンへ直す

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-20 |
| Sources | `.claude/skills/gkill-plugin/SKILL.md` / `src/server/gkill/plugin/README.md` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/plugin/sdk/timezone.go`（`fixAndroidTimezone` / `fixTimezoneFor`） / `src/server/gkill/plugin/sdk/sdk.go`（`Run`） / `src/plugins/gkill_plugin_vrchat/parse.go`（`logLocation` / `logLocationKey`） |

## Context

Android の Go では `time.Local` が常に UTC になる（`time/zoneinfo_android.go` の initLocal は端末のゾーンも TZ 環境変数も読まない）。
gkill 本体は `main/common/fix_timezone.go` の fixTimezone が `getprop persist.sys.timezone` から直している（libc 側は [ADR-0220](0220-sqlite-localtime-follows-libc-zone.md)）。
ところがプラグインは gkill が起動する**別のプロセス**で、この直しが届かない。

ゾーン付きの時刻（RFC 3339 の `Z` / `+09:00`、UNIX 秒）を読むプラグインはどのゾーンで動いても同じ瞬間になるので、表に出なかった。
表に出たのはゾーンの無いローカル時刻を `time.Local` で読むプラグインで、Termux の gkill では時刻が端末のゾーンの分だけずれた（日本なら9時間）:

- `gkill_plugin_vrchat`: 出力ログの各行の時刻（`2006.01.02 15:04:05`）。さらに `var logLocation = time.Local` とパッケージ変数に写していたので、
  あとから time.Local を直しても写した UTC のまま残る作りだった。
- gkill の外の `gkill_plugin_uguisu`（予算）: 予定日（`2006-01-02`）を `time.ParseInLocation(…, time.Local)` で読む。

## Decision

- SDK の `Run` が、ログの初期化の直後・ハンドラを呼ぶ前に `fixAndroidTimezone` を呼ぶ。Android のときだけ `getprop persist.sys.timezone` を読み、
  `time.LoadLocation` できたら `time.Local` を差し替える。読めなければ警告をログに出して UTC のまま続ける（起動は止めない）。gkill 本体の fixTimezone と同じ手順。
- プラグインは time.Local を**呼ぶ時点で**読む。パッケージ変数に写さない（`gkill_plugin_vrchat` の `logLocation()` はテスト用の差し替えが無ければ time.Local を返す関数にした）。
- `gkill_plugin_vrchat` のキャッシュは UNIX 秒で持つので、読んだゾーンを `cache_meta.log_location`（ゾーン名と 2000-01-01 時点のオフセット。
  Windows のゾーン名は常に `Local` なのでオフセットを添える）に残し、違えば作り直す。Termux で UTC のまま作ったキャッシュも、端末のゾーンを変えたときも、これで作り直される。

## Rejected alternatives

- **gkill 本体がプラグインを起動するときに TZ 環境変数を渡す** — Android の Go は TZ も読まない（上の initLocal）。本体が libc 用に入れる `TZ=:<絶対パス>` もそのままでは `time.LoadLocation` できない。
- **各プラグインで直す** — 同じ getprop の手順を全プラグインへ写すことになり、gkill の外のプラグイン（uguisu）も個別に直す必要がある。SDK に置けば SDK を上げるだけで揃う。
- **VRChat のログのゾーンを設定で指定させる** — VRChat を遊ぶ PC とログを読む端末は同じゾーンにある使い方なので、端末のゾーンで読めば足りる。指定を足すと既定値（どのゾーンか）を決める問題が残るだけ。

## Consequences

- Android 以外では何も変わらない（`runtime.GOOS` で分ける）。
- SDK を上げていない gkill の外のプラグインは、Termux では UTC のまま。uguisu は `go get` で SDK を上げれば直る。
- ゾーン付きの時刻を `.Local()` で表示用に変えるだけのプラグイン（google_locationhistory の設定画面の日付など）も、Termux で端末のゾーンの表示になる。

## Evidence

実測なし — コードの読み取りによる判断。PC で作ったキャッシュの時刻はログの壁時計と一致し、Termux だけずれていた。差し替えとキャッシュの作り直しは下のテストで固定した。

## Related tests

- `src/server/gkill/plugin/sdk/timezone_test.go`（`TestFixTimezoneFor` / `TestFixTimezoneForKeepsUTCOnFailure`）
- `src/plugins/gkill_plugin_vrchat/cache_test.go`（`TestLogLocationChangeRebuilds` / `TestLogLocationFollowsTimeLocal`）
