# tools テスト仕様

## 概要

`src/tools/` のうち、リリースゲート（テスト済み attestation）を構成する 5 ファイル（`attestation.mjs` / `run_test_suite.mjs` / `verify_release_gate.mjs` / `verify_release_artifacts.mjs` / `put_version_info.mjs`）の判定関数と、`verify_docs.mjs` の解析部（正規表現と composeTools の読み方）のテスト。MCP の tools/list 予算は Go 側（`gkill_server mcp schema-budget`、`src/server/gkill/mcp/tool_schema_budget_test.go`）が持つ。
壊れると「テスト未実施のコミットが配布される」（指摘 F-006 の再来）に戻るので、
ローカル `npm test` と CI（`ci.yml` の `frontend` ジョブ）の両方で回す。

## テストフレームワーク

Vitest（`vitest.config.tools.ts`、`environment: node`）。使い捨ての git リポジトリを `os.tmpdir()` に作って
git を実際に呼ぶので、`git` が PATH に要る。リポジトリ本体の index・作業ツリーには触らない。

## テストファイル

60テスト（3ファイル）。静的計数、`test.each` は 1 と数える。

| ファイル | テスト内容 |
|---------|-----------|
| `src/tools/__tests__/attestation.test.mjs` | `attestation.mjs` / `run_test_suite.mjs`（引数許可リスト・`SUITES` 表）/ `verify_release_gate.mjs` の単体・結合テスト（読む側） |
| `src/tools/__tests__/verify_docs.test.mjs` | `verify_docs.mjs` の解析部。MCP のテスト件数を数える `MCP_TEST_RE`（`t.Run(` の行だけ。コメント・別名は数えない）と、`return composeTools(...)` の1行と `newNameSet(...)` の絞り込みからサーバごとのツール数を出す `countComposedToolNames`（連結・重複・知らないモジュール名・定義行を読まないこと）。ここが空振りしても verify_docs は「件数 0 が資料と合わない」でしか落ちず、資料側を 0 に直されると気付けない |
| `src/tools/__tests__/release_scripts.test.mjs` | 書く側。`run_test_suite.mjs` の記録条件（`decideRecording`: 失敗・シグナル中断・CI・絞り込み引数では記録しない）と `gkill_server version` からの tree 読み取り、`put_version_info.mjs` の `version.json`（tree_hash・git が無いときは環境変数の SHA か unknown で止めない）、`verify_release_artifacts.mjs` の成果物一覧（12件）・7za 一覧の読み方・必須エントリ・debug 署名判定・apksigner の探索 |

## テスト内容

- **`parseGithubRepo`**: `https://` / `git@host:o/r.git` / `ssh://` の 3 形式と、解釈できない文字列
- **`evaluateSuites`**: 全スイート記録済み／記録なし／tree 不一致（記録側の先頭 7 桁と時刻を出す）／E2E の `server_tree` 未記録・不一致
- **`evaluateCiRuns`**: success あり／未完了だけ／run 無し（push されていない）／failure だけ
- **`evaluateNightly`**: success・48 時間以内・祖先・依存未変更で ok。run 無し／failure／古い／祖先でない／依存変更のそれぞれで NG
- **`firstDisallowedArg`（ランナーの引数許可リスト）**: `=` 区切りと空白区切りの等価性、値の欠落・形の違い、値を取らないフラグへの値、絞り込み引数（`-run` / `--grep` / ファイル名）、許可リストが空のスイート（`verify_docs` / `test_plugins` / `test_android` / `test_wear_os`）
- **`SUITES` 表**: `REQUIRED_SUITES` と 1:1、引数が検査内容を置き換えるスイートは許可リストが空、`server_tree` を記録するのは E2E だけ、`test_mcp` は `src/server` で Go の MCP パッケージを `go test` する（旧 Node 実装の vitest 設定を指したままだと 0 件実行で緑になる）
- **`verify_docs.mjs` の解析部**: `MCP_TEST_RE` と `countComposedToolNames`（`verify_docs.test.mjs`）
- **git（使い捨てリポジトリ）**: クリーンなら `workingTree()` = `HEAD^{tree}`、編集・untracked で変わる、実 index に stage されない、そのままコミットすれば一致する、**CRLF で書き換えても tree は変わらない**（WSL の偽 dirty を吸収する根拠）、attestation ファイルが ignore されていなければ止まる、`isAncestor` / `pathsChangedBetween`（exit 1 と 128 の区別）
- **記録ファイル**: 無ければ空、追記で別スイートが残る、temp ファイルが残らない、壊れた JSON・違うスキーマは「記録なし」扱い
- **GitHub API**: URL の形、非 2xx は投げる（fail-closed）、トークンがあれば `Authorization`、fetch が無い環境は投げる
- **`runGate` end-to-end**（fetch スタブ）: 全部そろえば `RELEASE_ATTESTATION_<version>.json` を書く／dirty／attestation 不足は 1 スイート 1 件／GitHub 不通は CI と Nightly の 2 件／CI failure + Nightly が古い／Nightly 以後の依存変更
- **記録の条件（`decideRecording`）**: 成功・非 CI・許可された引数だけなら記録し、exit 非 0・シグナル中断・CI・絞り込み引数・引数が検査内容を置き換えるスイートの引数ありでは記録しない。「失敗したのに記録する」変更は attestation の前提を壊す
- **`version.json`**: `build_time` はローカル時刻＋タイムゾーン、`commit_hash` / `tree_hash` は git から、git が無ければ `GITHUB_SHA` / `COMMIT_SHA` か unknown でビルドを止めない
- **成果物の検証**: 成果物はゲートの記録 + Go 7種 + APK 3本 + サンプルデータの 12 件、`7za l -slt` の `Path = ` 行だけを拾い区切りを `/` に揃える、必須エントリの欠けの列挙、`CN=Android Debug` の検出、`build-tools` の最新版から `apksigner` を探す

## 実行方法

```bash
npm run test_tools
```

`run_test_suite.mjs` 経由で走るので、通れば `test_attestation.local.json` に `test_tools` が記録される。
