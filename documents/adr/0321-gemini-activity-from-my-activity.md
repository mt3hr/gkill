# ADR-0321: Gemini の会話は Takeout の「マイ アクティビティ」から JSON だけを読み、エントリはロケール非依存に選ぶ

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-24 |
| Sources | `.claude/skills/gkill-plugin/SKILL.md`「プラグイン一覧と実測仕様」 / `documents/reverse/plugin-system.md`「取り込み元の ZIP — sdk.OpenSources」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/plugins/gkill_plugin_gemini/source.go`（`isGeminiActivityEntry`） / `src/plugins/gkill_plugin_gemini/loader.go`（`promptOf` / `convIDOf` / `classifyEntries`） / `src/plugins/gkill_plugin_gemini/uuid.go`（`kyouIDOf`） |

## Context

ChatGPT・Claude.ai・Claude Code・Codex・Antigravity の会話はプラグインで gkill のタイムラインに載っているが、Gemini だけ載っていなかった。

Gemini の会話は Google Takeout から取り出せるが、**取り出し方に3つの罠がある**。

- **Takeout のトップレベルの「Gemini」を選んでも会話は入らない。** 入るのは Gem と予定アクションの設定だけで、実物の書き出しでは中身が空の HTML が2本できるだけだった。会話は「マイ アクティビティ」を選び、その中で「Gemini アプリ」に絞って初めて入る
- **Takeout はフォルダ名もファイル名もアカウントの言語で訳す。** 日本語アカウントでは `Takeout/マイ アクティビティ/Gemini アプリ/マイアクティビティ.json`、英語アカウントでは `Takeout/My Activity/Gemini Apps/MyActivity.json` になる。`MyActivity.json` という名前でエントリを選ぶと、日本語環境では**例外もエラーも出さずに0件**になる
- **書き出しの既定の形式は HTML** で、JSON にするには書き出し時に「複数の形式」から明示的に変える必要がある

形式そのものにも、既存の会話系プラグインと違う点がある。

- 1要素 = 1往復（人間のプロンプトと Gemini の返答がひとまとまり）で、会話の入れ物が無い。会話の名前も無い
- **`titleUrl` が無い。** 会話 URL は `details[].url`（`name` も同値）にしか入らない
- **メッセージ ID が無い。** 時刻は RFC 3339 の UTC（ミリ秒つき）で入る
- プロンプトは `title` に `<接頭辞>: <本文>` の形で入り、接頭辞はロケールで変わる。接頭辞を持たない記録（操作の記録）はプロンプトも返答も持たない
- 返答は `safeHtmlItem[].html` に HTML で入る。1レコードに2つ以上入ることがある
- 添付（音声・画像・ZIP）は同じ ZIP フォルダに実ファイルとして並び、**その添付が ZIP なので「ZIP の中の ZIP」になる**

## Decision

- **取り込み元は Google Takeout の ZIP。** `sdk.OpenSources` に任せ、展開済みのフォルダは読まない（ADR-0310 と同じ）。既定の `source_dirs` は fitbit / 位置情報と同じ `~/Kyou/GoogleTakeout_*` にし、1つの ZIP を3本で共用する
- **エントリは名前ではなく位置で選び、中身で確かめる。** 受理するのは「**親フォルダ名が `Gemini` で始まる** `.json` / `.html`」。Takeout が訳さないのは製品名の `Gemini` だけなので、これならどのロケールでも当たる。JSON が Gemini のものかは先頭の1レコードの `header` / `products` / `activityControls` / 会話 URL に `Gemini` が部分一致するかで決め、当たらなければ**黙って読み飛ばす**（マイアクティビティには製品ごとに同じ名前のファイルが並ぶので、別製品は「問題」ではない）
- **JSON だけを読む。HTML 形式は読まない。** ただし Gemini のアクティビティらしい HTML（先頭に会話 URL を含む）を見つけたら、`source_problems` に「JSON で書き出し直してください」を出す。黙って0件にしない
- **1往復から Kyou は最大2件**（人間のプロンプトと Gemini の返答）。ChatGPT / Claude.ai と同じ粒度にする。プロンプトも返答も無い記録からは1件も作らない
- **Kyou の ID は `(会話 ID, 時刻, 役割, 連番)` から UUIDv5 で導出する。** 書き出しにメッセージ ID が無いため。連番は**書き出し（`ExportID`）ごとに**数えるので、書き出しを足しても既存の ID は動かない
- **会話の見出しは、その会話で最も古いプロンプトの先頭**（書き出しに会話の名前が無いため）
- **返答の HTML は取り込み時に markdown 風の平文へ写す。** chatview は本文を `RenderText` に通すので、HTML を渡すとタグが文字として並ぶ。写し方を変えたら `cacheSchemaVersion` を上げる
- **差分は `Path:CRC32:Size` の署名1本で、変わったら全部作り直す**（ADR-0303）。同じ往復が複数の書き出しにあれば、新しい書き出し（`SourceSet.Exports()` の先頭に近いほう）を採る
- **`source_problems` から `nested_zip` は落とす。** Gemini の添付が ZIP で、実物では1フォルダに多数並ぶ。そのまま出すと設定画面が「ZIPの中のZIPは読みません」で埋まって他の警告が読めなくなる。**`mixed_exports` は残す**（Takeout では世代が並ぶのが普通で、どの書き出しを見ているかは利用者が知りたい情報。chatgpt / claudeai が落としているのは、あちらに世代の概念が無いから）
- `provides` は書かない（型別データを返さない）

## Rejected alternatives

- **`MyActivity.json` というファイル名でエントリを選ぶ** — 日本語アカウントではファイル名が `マイアクティビティ.json` なので、**例外もエラーも出さずに0件**になる。取り込み元が空なのかプラグインが壊れているのか、設定画面を見ても分からない。この罠は実物の書き出しで踏んだ
- **`Takeout/Gemini/` を読む** — Gem と予定アクションの設定しか入らない。会話は1件も無い
- **`title` の接頭辞（`送信したメッセージ: `）を既知の文字列として持つ** — ロケールのぶんだけ辞書が要り、Google が文言を変えるたびに黙って0件になる。「最初の `: ` までを接頭辞とみなす（長すぎるものは本文の一部とみなす）」＋「英語の `Prompted ` を落とす」＋「どちらでもない title はプロンプトではない」の3段にした
- **title をそのままプロンプトにする** — 操作の記録（「アシスタント機能を使用」）が人間の発言として一覧に並ぶ
- **HTML 形式も推定で読む** — 実物の HTML が手元に無く、推定で書いたパーサは壊れていても気づけないまま配られる。実物が出てきたら足す。それまでは設定画面に「JSON で書き出し直してください」と出す
- **1レコード = 1 Kyou にする** — 人間の発言と返答が1行に混ざり、他の会話系プラグインと形が揃わない。一覧でどちらの発言か分からなくなる
- **Kyou の ID を中身のハッシュから作る** — Google が返答の切り出し方を変えると ID が総入れ替えになり、利用者が付けたタグが全部外れる
- **連番を書き出し全体で数える** — 2つ目の書き出しを足しただけで同じ往復に別の番号が付き、ID が総入れ替えになる
- **fitbit 流に `export` 表で世代の順位を持つ** — 値を合計しないので順位付けが要らない。会話は ID で畳めば済む
- **返答の HTML を chatview へそのまま渡す** — `RenderText` が全部エスケープするので、タグが文字として並ぶ。chatview に生 HTML の欄を足す案は、4本の既存プラグインとの契約（ADR-0315）を崩すので採らない
- **`nested_zip` も含めて走査の問題を無加工で出す**（fitbit 流） — 添付の ZIP で設定画面が埋まる
- **添付ファイルの中身も取り込む** — 音声・画像・ZIP を読むのはこのプラグインの仕事ではない。名前と注記を本文の折りたたみに並べるだけにした

## Consequences

- Takeout の書き出しで「マイ アクティビティ」→「Gemini アプリ」を選び、形式を JSON にする必要がある。手順は設定画面とプラグインの README に出す
- トップレベルの「Gemini」だけを選んだ書き出しでは0件になるが、設定画面に取り込み元と件数が出るので「0件」と分かる
- HTML 形式で書き出していると取り込みは進まないが、設定画面に理由が出る
- 既定の `source_dirs` が fitbit / 位置情報と同じなので、1つの Takeout の ZIP を3本が別々に走査する。キャッシュ DB・`config.json`・ログはプラグイン名ごとに分かれているので干渉しない
- 会話の見出しは最も古いプロンプトなので、Gemini の画面に出る会話名とは一致しない（書き出しに会話名が入らないため）
- `kyouIDOf` の導出を変えると全 Kyou の ID が変わり、利用者が付けたタグやテキストの紐付けが切れる。変えないこと

## Evidence

- 実物の書き出し（JSON 形式）で、`Takeout/Gemini/` に入っていたのは Gem と予定アクションの HTML 2本だけで、どちらも中身は空の `<div></div>` だった
- 同じ書き出しの `Takeout/マイ アクティビティ/` には製品ごとのフォルダが並び、どのフォルダにも**同じ名前の**マイアクティビティ JSON が入っていた（ファイル名で選ぶと別製品まで開く）
- `Gemini アプリ` のフォルダには、アクティビティ JSON のほかに音声・画像・ZIP の添付が並んでいた。ZIP の添付は「ZIP の中の ZIP」として `nested_zip` が立つ
- `details[].name` と `details[].url` は実物の全レコードで同値だった。`titleUrl` は1レコードにも無かった
- 実物の書き出しで `(会話 ID, 時刻)` の組は重複が無かった（連番は安全網で、ふつうは全部 0）
- 実物の書き出しを取り込んだ結果が、同じ規則を別実装で計算した期待値（プロンプト数・返答数・会話数）と一致し、本文が空の行も ID の重複も無かった
- 返答の HTML に出てきたタグは `p` `strong` `li` `code` `ul` `ol` `h2`-`h4` `table` `thead` `tbody` `tr` `th` `td` `pre` `a` `em` `br` `hr` `blockquote` だけで、属性は `href` と `align` だけだった

## Related tests

- `src/plugins/gkill_plugin_gemini/source_test.go`（`TestIsGeminiActivityEntry` / `TestRelevantProblemsKeepsMixedExportsAndDropsNestedZip` / `TestKyouIDOfIsStableAndDistinct`）
- `src/plugins/gkill_plugin_gemini/cache_test.go`（`TestGeminiReadsLocalizedAndEnglishLayouts` / `TestGeminiIgnoresOtherProductsAndGemsPlaceholder` / `TestGeminiHTMLFormatIsReported` / `TestGeminiSameExchangeAcrossExportsIsSingle`）
- `src/plugins/gkill_plugin_gemini/loader_test.go`（`TestPromptOf` / `TestConversationIDFromURL` / `TestIsGeminiRecord`）
- `src/plugins/gkill_plugin_gemini/htmltext_test.go`（`TestHTMLToText`）
