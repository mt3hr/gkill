# tools テスト仕様

## 概要

`src/tools/` のうち、リリースゲート（テスト済み attestation）を構成する 4 ファイルのテスト。
壊れると「テスト未実施のコミットが配布される」（2026-08-30 監査 F-006 の再来）に戻るので、
ローカル `npm test` と CI（`ci.yml` の `frontend` ジョブ）の両方で回す。

## テストフレームワーク

Vitest（`vitest.config.tools.ts`、`environment: node`）。使い捨ての git リポジトリを `os.tmpdir()` に作って
git を実際に呼ぶので、`git` が PATH に要る。リポジトリ本体の index・作業ツリーには触らない。

## テストファイル

| ファイル | テスト内容 |
|---------|-----------|
| `src/tools/__tests__/attestation.test.mjs` | `attestation.mjs` / `run_test_suite.mjs` / `verify_release_gate.mjs` の単体・結合テスト（41テスト（1ファイル）。静的計数、`test.each` は 1 と数える） |

## テスト内容

- **`parseGithubRepo`**: `https://` / `git@host:o/r.git` / `ssh://` の 3 形式と、解釈できない文字列
- **`evaluateSuites`**: 全スイート記録済み／記録なし／tree 不一致（記録側の先頭 7 桁と時刻を出す）／E2E の `server_tree` 未記録・不一致
- **`evaluateCiRuns`**: success あり／未完了だけ／run 無し（push されていない）／failure だけ
- **`evaluateNightly`**: success・48 時間以内・祖先・依存未変更で ok。run 無し／failure／古い／祖先でない／依存変更のそれぞれで NG
- **`firstDisallowedArg`（ランナーの引数許可リスト）**: `=` 区切りと空白区切りの等価性、値の欠落・形の違い、値を取らないフラグへの値、絞り込み引数（`-run` / `--grep` / ファイル名）、許可リストが空のスイート（`verify_docs` / `test_plugins` / `test_android` / `test_wear_os`）
- **`SUITES` 表**: `REQUIRED_SUITES` と 1:1、引数が検査内容を置き換えるスイートは許可リストが空、`server_tree` を記録するのは E2E だけ
- **git（使い捨てリポジトリ）**: クリーンなら `workingTree()` = `HEAD^{tree}`、編集・untracked で変わる、実 index に stage されない、そのままコミットすれば一致する、**CRLF で書き換えても tree は変わらない**（WSL の偽 dirty を吸収する根拠）、attestation ファイルが ignore されていなければ止まる、`isAncestor` / `pathsChangedBetween`（exit 1 と 128 の区別）
- **記録ファイル**: 無ければ空、追記で別スイートが残る、temp ファイルが残らない、壊れた JSON・違うスキーマは「記録なし」扱い
- **GitHub API**: URL の形、非 2xx は投げる（fail-closed）、トークンがあれば `Authorization`、fetch が無い環境は投げる
- **`runGate` end-to-end**（fetch スタブ）: 全部そろえば `RELEASE_ATTESTATION_<version>.json` を書く／dirty／attestation 不足は 1 スイート 1 件／GitHub 不通は CI と Nightly の 2 件／CI failure + Nightly が古い／Nightly 以後の依存変更

## 実行方法

```bash
npm run test_tools
```

`run_test_suite.mjs` 経由で走るので、通れば `test_attestation.local.json` に `test_tools` が記録される。
