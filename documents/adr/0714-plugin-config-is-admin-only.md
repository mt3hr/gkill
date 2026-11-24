# ADR-0714: プラグイン設定は管理者だけが開け、入口はアカウント管理画面の利用者の行

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-24 |
| Sources | `.claude/skills/gkill-plugin/SKILL.md`「プラグイン設定の読み書きは管理者だけ。入口はアカウント管理画面の利用者の行」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/gkill_server_api/plugin_target_user.go` / `src/client/pages/dialogs/manage-plugin-list-dialog.vue` / `src/client/pages/views/manage-account-view.vue` |

## Context

プラグインの設定画面を開く道は1本だけだった。**プラグインが取り込んだ記録を長押し（モバイル）
または右クリック（デスクトップ）して「プラグイン設定」を選ぶ。** 実装は
`plugin-html-context-menu.vue` の項目 →`requested_show_plugin_config` →`plugin-html-view.vue` が
直接持つ `plugin-config-dialog.vue`。

この導線には3つの問題があった。

**1. 記録が1件も検索に出ていないと開けない。** 設定画面で変えるものの中心は `source_dirs`、
つまり取り込み元のフォルダの指定である。その指定を間違えていれば取り込みは0件で、記録は1件も出ない。
**直したいときだけ入口が消える。** `source_dirs` が空の初期状態がまさにこれで、
プラグインを置いた直後の利用者は設定画面に辿り着けない。

**2. エラーの案内が実在しない導線を指していた。** `ERROR_HINT_REASON_PLUGIN_BUSY` /
`ERROR_HINT_REASON_PLUGIN_ERROR`（7言語）は「設定 → プラグイン設定 で状態を確認してください」と
書いてあったが、設定画面にプラグインの項目は無かった。

**3. 誰でも設定を書き換えられた。** `/api/get_plugin_config_html` と `/api/post_plugin_config` は
`authSession` だけで、ハンドラの中にも管理者判定が無い。プラグインは利用者ごとの実体
（`GetPluginManager(userID)`、`$GKILL_HOME/plugins/<利用者>/`）なので「自分の分しか触れない」のだが、
`source_dirs` は**gkill のプロセスが読めるフォルダの指定**である。一般利用者が任意のフォルダを指すと、
そのフォルダの中身が自分の記録としてプラグイン経由で読める。自分のデータの外へ手が届く。

## Decision

**入口をアカウント管理画面（サーバ設定 → アカウント管理。`account_is_admin` で出し分けられている）の
利用者の行へ移し、サーバ側にも管理者判定を置く。** 記録のコンテキストメニューからは撤去する。

**1. 置き場所は「アカウント1行ごと」。** `manage-account-view.vue` の行の「保存先割当」と
「パスワードリセット」の間に「プラグイン設定」を置く。押すと `manage-plugin-list-dialog.vue` が開き、
その利用者のプラグインが名前・版・説明・状態（稼働中／停止）の表で並ぶ。行の「設定」で
`plugin-config-dialog.vue` が `show(target_user_id, rep_name)` で開く。
プラグインは利用者ごとの実体なので、入口も利用者ごとにあるのが素直で、
**管理者が利用者に代わって直す**（本番でいちばん多い場面）がそのまま画面になる。

**2. 対象利用者は `target_user_id` で運ぶ。** `reset_password` / `update_account_status` /
`update_user_reps` が既に同名の欄で「管理者が他人のアカウントに対して操作する」形を持っているので、
それに倣う。判定は3ハンドラ共通の `resolvePluginTargetUser` 1か所で、
`AccountDAO.GetAccount` で**対象アカウントの実在を必ず確かめる**（飛ばすと `GetPluginManager` が
存在しない利用者名でマネージャを作って居座る）。

**3. 管理者判定はエンドポイントごとに強さを変える。**

| エンドポイント | 管理者必須 | 理由 |
|---|---|---|
| `/api/get_plugin_config_html` | 常に（`target_user_id` が空でも） | 設定画面には取り込み元のフォルダ構成が出る |
| `/api/post_plugin_config` | 常に | 設定の書き換えそのもの |
| `/api/get_plugin_list` | `target_user_id` が空でないときだけ | 一般利用者も呼ぶ（下記） |
| `/api/get_plugin_content_html` | 変更なし | プラグイン記録の本文。全利用者が必要 |

**4. 設定ダイアログは gkill 標準のフローティングダイアログにした。** 唯一 `v-dialog` のままだった
1枚で、ヘッダ・透過・Escape・履歴スタックの作法が他のダイアログと違っていた。
中身（iframe・postMessage・保存の肩代わり）は触っていない。

## Rejected alternatives

- **`get_plugin_list` も常に管理者必須にする。** 一覧 API だけ別扱いにするのは一貫していないので
  最初に考えたが、`use-plugin-rep-names.ts` がモジュール単位のシングルトンとして全利用者向けに
  これを呼び、プラグイン記録のコンテキストメニューから「編集」「削除」を隠すのに使っている
  （プラグインの記録は読み取り専用なので、出すと押せるのに何も起きない）。塞ぐと一覧が空で返り、
  `is_plugin_rep()` が常に false になって**エラーも警告も出ないまま**一般利用者のメニューが壊れる。
  MCP の `gkill_get_plugin_list` も同じ API を使う
- **アプリケーション設定（設定画面）の「検索条件」と「スキル」の間に置く。** 当初の案。
  ボタン1つで済み、エラーヒント文の「設定 →」もほぼそのまま使えた。採らなかったのは、
  あの画面が一般利用者も開けるからで、画面を隠す仕組みが無い
- **サーバ設定画面のヘッダ（「アカウント管理」の隣）に置く。** 管理者限定は満たせるが、
  **対象の利用者を選べない。** サーバ設定は端末プロファイル別の設定で、
  開いた管理者自身のプラグインしか見えないことになる。他人のプラグインを直したい場面で使えない
- **画面を隠すだけにして、サーバ側は `authSession` のままにする。** 画面を消すのは防御ではない。
  `/api/post_plugin_config` を直接叩けば通る

## Consequences

- プラグインを置いた直後でも、記録が0件のままでも設定画面へ入れる
- 一般利用者は自分のプラグイン設定も開けなくなった。読める記録（本文・一覧）は変わらない
- 管理者は他の利用者のプラグイン設定を代行して直せる。一覧を開くとその利用者のプラグインプロセスが
  起動する（自分の分を開いたときと同じ）
- ダイアログの入れ子が アプリ設定 → サーバ設定 → アカウント管理 → プラグイン一覧 → プラグイン設定 の
  5枚になった。`useFloatingDialog` が z-index を積むので動作は3枚重ねと同じ
- エラーヒント文（7言語）は「管理者はアカウント管理の『プラグイン設定』で状態を確認できます」に直した。
  一般利用者には開けない画面なので、誰が確認できるのかを書いてある
- `plugin-config-dialog.vue` がフローティングダイアログになり、`useFloatingDialog` を使わないダイアログは
  0枚になった

## Evidence

- 旧実装の入口は `plugin-html-context-menu.vue` の1項目だけで、`props.kyou.typed_plugin?.rep_name` が
  無ければ何もせず戻っていた（プラグイン記録の上でしか開かない）
- `GetPluginByRepName` は manifest の `rep_name` でも `get_rep_name` で申告した名前でも引ける
  （`plugin_manager_rep_names_test.go`）ので、一覧の `rep_name` をそのまま設定の引き当てに使える
- `AccountNotHasAdminError` は `http_status.go` で既に 403 に割り当てられており、対応表の変更は不要だった

## Related tests

- `src/server/gkill/api/gkill_server_api/handle_plugin_config_target_user_test.go` —
  非管理者が設定の取得・保存を叩くと 403 + `AccountNotHasAdminError`、`get_plugin_list` は
  `target_user_id` が空なら通して他人を指したときだけ弾く、実在しない `target_user_id` はエラー
- `src/client/__tests__/unit/composables/manage-plugin-list-dialog.test.ts` —
  開いた行の `target_user_id` で一覧を取り、設定画面へ `target_user_id` と rep 名の両方を渡す
- `src/client/__tests__/unit/composables/plugin-config-dialog.test.ts` —
  `show(target_user_id, rep_name)` で開き、読み込みと保存の両方のリクエストに `target_user_id` が載る
- `src/client/__tests__/e2e/plugin-config.spec.ts` — 設定 → サーバ設定 → アカウント管理 →
  行の「プラグイン設定」で一覧が開く
