# ADR-0708: 待受の既定はループバック限定 — LAN 公開は設定画面での明示操作にし、既存の設定は移行も拒否もしない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-14 |
| Sources | `.claude/skills/gkill-go-backend/SKILL.md`「待受の既定はループバック限定。既定値は `server_config` の定数1組から引く」節 / `.claude/skills/gkill-cli-ops/SKILL.md`「Two Deployment Modes」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | なし — 既定値は `src/server/gkill/dao/server_config/server_config.go` の定数コメントが自分で理由を持つ |

## Context

外部レビュー（2026-09-13、「セキュリティデフォルトはライフログ製品として攻めすぎ」）の指摘。
初回起動で自動生成されるサーバ設定が次の値だった。

| 項目 | 旧既定 | 意味 |
|---|---|---|
| `Address` | `":9999"` | ホスト部が空＝**全インターフェース**で待つ |
| `IsLocalOnlyAccess` | `false` | ループバック以外からのリクエストも通す |
| `EnableTLS` | `false` | 平文 |

つまり `gkill_server` を起動しただけで、同じ LAN の第三者がログイン画面まで届く。
アカウントには初期パスワードが無く（`admin` はリセットトークン待ち）、初回登録のリダイレクトは
ループバックにしか返さない（`utils.go` の `ifRedirectResetAdminAccountIsNotFound`）ので
**即座に記録が読める穴ではない**が、人生記録を保存するローカルファーストのアプリの初期値として
「何も知らない利用者が起動しただけで LAN に露出する」のは逆向きである。

加えて既定値が **3系統に分かれて食い違っていた**。

| 場所 | `Address` | `IsLocalOnlyAccess` | 効く場面 |
|---|---|---|---|
| 初回起動ブロック `gkill_server_api.go` | `":9999"` | `false` | **新規インストールの実効既定** |
| DAO の既定マップ `serverConfigDefaultValue` | `":9999"` | `true` | 行が欠けている端末の SQL フォールバックと `GetDefaultServerConfig` |
| クライアント `server-config.ts` のコンストラクタ | `":9999"` | `true` | 空の `ServerConfig` を作るとき |

資料（`operations-guide.md`）は初回起動ブロックの値を書いていたので「資料どおり」ではあったが、
DAO 側だけ読んだ人は「既定はローカル限定」と誤解する状態だった。

なお `--address` は設定 DB を書き換えない実行時上書きで、Android 同梱サーバ（`--address 127.0.0.1:9999`）と
E2E（`--address 127.0.0.1:<空きポート>`）は既定に依存していない。デスクトップ版・CLI（`ResolveLocalServerEndpoint`）・
起動メッセージ（Android がパースする行）はいずれも `localhost` + ポート接尾辞（`ServerAddressPortSuffix`）で
URL を組むので、既定アドレスにホスト部が付いても壊れない。

## Decision

**既定を `127.0.0.1:9999` + `IsLocalOnlyAccess=true` にし、この2値は `server_config` パッケージの定数
（`DefaultListenAddress` / `DefaultIsLocalOnlyAccess`）1組から引く。** 初回起動ブロックと DAO の既定マップは
両方その定数を参照し、クライアントのコンストラクタも同じ値に揃える。

**既存の `server_config.db` は移行も拒否もしない。** 旧既定のまま LAN で使っている環境はそのまま動き、
TLS 無し・非ループバック待受のときの起動時警告（`printInsecureBindWarning`）だけが従来どおり出る。

**LAN の他端末から使うのはサーバ設定画面での明示操作**（「アドレス」を `:9999` 等にし、
「ローカルアクセスのみ許可」をオフにする）。TLS は推奨として案内するに留め、強制しない。

## Rejected alternatives

- **既存の `server_config.db` を新既定へ自動移行する** — LAN で使っている利用者を締め出す。
  サーバ設定を直す CLI は無く（`update_cache` / `add_tag` は HTTP クライアント、`reset_password` は
  アカウントの話）、設定画面自体がループバックからしか開けなくなるので、別の端末から運用していた
  人は SQLite を手で書き換えるしか復旧手段が無い。既定は「初めて起動する人」のためのもので、
  既に決めた人の設定を上書きする権限は無い。

- **`Address` は `":9999"` のままにし、`IsLocalOnlyAccess=true` だけにする** — bind 自体は
  全インターフェースで開いたまま、アプリ層の `filterLocalOnly` だけで守ることになる。
  `filterLocalOnly` は全ルートを通るが、それは「今そうなっている」だけで、ルートを1本足して
  ラッパを通し忘れた瞬間に LAN へ出る。bind をループバックに閉じておけば、ルート側の漏れは
  同じホスト内にしか届かない。2枚の防御線を両方閉じるのが既定。

- **初回起動ブロックのリテラルだけ直し、DAO の既定マップは触らない** — 既定が2箇所に
  書かれている構造そのものが、今回の食い違いの原因。片方を直しても次の変更でまたずれる。
  定数1組にして、両方がそれを引く形にする。

- **LAN 公開時に TLS 無しの設定を保存で拒否する（`update_server_configs` で新エラー）** —
  レビューの指摘には「LAN 公開時は TLS 設定を要求」も含まれていたが、**採らない**（2026-09-14 の判断）。
  TLS を使うかは利用者の判断（同じホストのリバースプロキシで終端する構成など）で、既定を閉じておけば
  LAN 公開は設定画面での明示操作になり、そこで TLS を推奨として案内すれば足りる。
  保存拒否・起動拒否・UI の警告・新しいエラーコードは足さない。

- **起動時に旧既定（`:9999` + 許可なし + TLS 無し）を検出して起動を拒否する** —
  上の「自動移行」と同じ理由で復旧手段が無い。警告に留める。

## Consequences

**既定値を `server_config.go` の定数以外に書かないこと。** 初回起動ブロック・DAO 既定マップ・
クライアントのコンストラクタのどれかにリテラルを戻すと、また2系統に分かれて静かにずれる。
守るのは `gkill_server_api_test.go` の `TestNewGkillServerAPI_FirstRunDefaultsAreLocalOnly`
（初回起動ブロックが定数を引いているか）と `server_config_dao_sqlite3_impl_test.go` の
`TestGetDefaultServerConfigUsesListenDefaults`（DAO 側）。クライアント側はテストで守っていない
（空の `ServerConfig` を画面に出す経路が無く、実害が無いため）。

**新規インストールでは、同じ端末のブラウザ（`http://localhost:9999`）からしか開けない。**
スマートフォンや別の PC から使いたい利用者は、まず同じ端末で初回登録を済ませてから
サーバ設定画面でアドレスと「ローカルアクセスのみ許可」を開く必要がある。マニュアルの
サーバ設定ページに手順を書いた。

**既存環境の挙動は変わらない。** 旧既定のまま LAN 公開している環境を安全側へ倒す仕組みは
起動時警告だけで、それは stdout にしか出ない（サービス起動では誰も見ない）。
この決定はその状態を受け入れている。

## Evidence

実測なし — 脅威モデルからの判断。

壊れないことの根拠は構造で確認した:
`main/gkill/main.go`（デスクトップ）・`main/common/common.go` の `ResolveLocalServerEndpoint`（CLI）・
`close.go` の `PrintStartedMessage`（起動メッセージ。Android の `MainActivity` がこの行から URL を取る）は
いずれもホストを `localhost` に固定し、`gkill_options.ServerAddressPortSuffix` でポートだけを設定値から取る。
`127.0.0.1:9999` を渡しても `:9999` が返るので、組み立てる URL は従来と同じ `http://localhost:9999`。

## Related tests

- `src/server/gkill/api/gkill_server_api/close_bind_address_test.go`
- `src/server/gkill/api/gkill_server_api/gkill_server_api_test.go`
- `src/server/gkill/dao/server_config/server_config_dao_sqlite3_impl_test.go`
- `src/server/gkill/api/gkill_server_api/filter_local_only_test.go`
- `src/server/gkill/main/common/gkill_options/option_test.go`
