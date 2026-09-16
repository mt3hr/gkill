# tools

gkill プロジェクト用のユーティリティスクリプト。

## ファイル一覧

| ファイル | 内容 |
|---|---|
| `dev.mjs` | `npm run dev` の実体。`--api=<url>` で接続先gkill_serverを指定してViteを起動する |
| `license_getter.mjs` | `npm run license_getter` の実体。依存ライブラリのライセンス一覧を `LICENSES_DEPENDENCE` に出力する |
| `verify_docs.mjs` | `npm run verify_docs` の実体。資料の件数・リンク・Mermaid・マニュアル鮮度等の機械検査 |
| `build_manuals.mjs` | `resources/manual_src/` から `resources/manual/` を生成する |
| `manual_build.mjs` | マニュアル生成の共通ロジック（`build_manuals.mjs` / `verify_docs.mjs` から利用） |
| `manual_a11y.mjs` | マニュアルのアクセシビリティ検査ヘルパー |
| `manual_ascii_fix.mjs` | マニュアル内の文字化け・ASCII 修正ユーティリティ |
| `extract_manual_src.mjs` | 既存 `resources/manual/` から `manual_src/` を抽出する移行ツール |
| `verify_release_artifacts.mjs` | `npm run release` 成果物の検証（存在・SHA-256・APK 署名・リリースゲート記録との一致） |
| `verify_release_gate.mjs` | `npm run verify_release_gate` の実体。`npm run release` の先頭で、テスト済み attestation・GitHub の CI / Nightly を検査して通らなければ止める |
| `run_test_suite.mjs` | `npm run test_*` / `npm run verify_docs` の実体。スイートを 1 本走らせ、成功時に `test_attestation.local.json` へ作業ツリーの tree hash を記録する |
| `attestation.mjs` | 上 2 つと `put_version_info.mjs` / `verify_release_artifacts.mjs` の共有ライブラリ（git ヘルパ・記録の読み書き・評価関数・GitHub API） |
| `put_version_info.mjs` | `npm run put_version_info_embed` の実体。`embed/version.json`（commit / build_time / version / tree_hash）を書く |
| `mcp_schema_budget.mjs` | `npm run mcp:schema-budget` の実体。MCP の tools/list のバイト量を予算ファイル `src/mcp/tool-schema-budget.json` と突き合わせる（`--update` で実測を予算へ書く）。判定は `src/mcp/tool-schema-budget.mjs` が正本 |
| `__tests__/attestation.test.mjs` | 上記のうち attestation / run_test_suite（引数許可リスト）/ verify_release_gate のテスト（`npm run test_tools`、`vitest.config.tools.ts`） |
| `__tests__/release_scripts.test.mjs` | run_test_suite の記録条件・put_version_info・verify_release_artifacts の判定関数のテスト |
| `test_plugins.mjs` | `npm run test_plugins` / `npm run vet_plugins` の実体。`src/plugins/` 配下の各 Go モジュールに `go test` / `go vet` を回す |
| `gradle_test.mjs` | `npm run test_android` / `npm run test_wear_os` の実体。Windows では `cmd /c <絶対パス>gradlew.bat`、それ以外は `./gradlew`（絶対パス）を使う |
| `codeql.mjs` | `npm run codeql` の実体。CI と同じ設定で CodeQL をローカル実行し、ベースラインに無い指摘が出たら失敗する。CodeQL CLI が無ければスキップして正常終了する |

## リリースゲート（run_test_suite.mjs / attestation.mjs / verify_release_gate.mjs）

「Nightly green + ローカル `npm test` green のコミットだけリリースする」を人の記憶ではなく機械で強制する
（2026-09-14、フィードバック #10。2026-08-30 監査 F-006 はリリース工程に検証ギャップがあった実例）。
規約の正本は `.claude/skills/gkill-build-test/SKILL.md`「リリースゲート（テスト済み attestation）」、経緯と却下案は
`documents/adr/0901-release-requires-tested-attestation.md`。

```bash
npm run test_server                       # 通れば test_attestation.local.json に test_server を記録
npm run test_client_e2e -- --workers=2    # 余剰引数は素通し。並列度は記録される
npm run test_client_unit -- xxx.test.ts   # 絞り込みなので記録されない（警告だけ出る）
npm run verify_release_gate               # release の先頭で自動実行。単体でも使える
```

### なぜ commit SHA ではなく tree hash に束縛するか

attestation は「作業ツリーの tree hash」に束縛する。dirty なツリーでテストを通し、そのまま全部コミットすれば
`HEAD^{tree}` が一致するので再テスト無しでリリースできる。ゲート側は「作業ツリーがクリーン」を別に要求するので、
結局「`HEAD^{tree}` == attested tree」＝テスト済みコミット、と同値になる。commit SHA に束縛すると
「テスト → コミット」の順序が全部再テストになる。

### なぜ git を `-c core.autocrlf=true` で呼び、クリーン判定に `git status` を使わないか

release は WSL で走るが、WSL 側の git は autocrlf 未設定で、Windows が CRLF で checkout した追跡ファイル
184 件が偽 dirty に見える（2026-09-14 実測）。さらに `git status` は index の stat cache と**サイズが違うだけ**で
内容を読まずに modified と言う（`ie_modified` の `DATA_CHANGED` 短絡）ので、`-c core.autocrlf=true` を付けても
CRLF でサイズが変わったファイルは偽 dirty のまま。`git diff --name-only HEAD` は内容を改行変換のうえで比べる
（`diffcore_skip_stat_unmatch`）ので、クリーン判定はこれと `git ls-files --others --exclude-standard` で行う。

### 記録しない引数

余剰引数はスイートへ素通しする。並列度・レポータ形式（go `-p` / `-count` / `-v` / `-timeout`、vitest `--pool` /
`--maxWorkers` / `--no-file-parallelism` / `--reporter`、playwright `--workers` / `--retries` / `--reporter`）は
網羅性を変えないので記録する。それ以外（`-run`、`--grep`、ファイル名）は「全件通った」と言えないので記録しない。
`verify_docs` / `test_plugins` / `test_android` / `test_wear_os` は引数が検査内容そのものを置き換える
（`--list` は検査せず exit 0、`vet` はテストでない、Gradle は引数が `test` タスクを置き換える）ので、
引数が 1 つでもあれば記録しない。記録しない場合も exit は 0（スイート自体は通っている）。

### E2E は PATH 上の gkill_server の tree も記録する

E2E は PATH の `gkill_server` を起動するので、古いバイナリで通しても現在のツリーを検証したことにならない。
`put_version_info.mjs` が `version.json` に `tree_hash` を焼き込み、`gkill_server version` が `tree:` 行で出す。
ランナーはそれを `server_tree` として記録し、ゲートは attested tree との一致を要求する。
`version` サブコマンドも `PersistentPreRun` でログを初期化するので、run-e2e.mjs と同じ
`--gkill_home_dir <home>/gkill_test --log none` を付けて呼ぶ（本番の `~/gkill/logs` を触らない）。

### 記録ファイルは .gitignore の明示行で無視する

`test_attestation.local.json` は `*.local`（`.local` で終わる名前にしか当たらない）では無視されない。
無視されていないと自分自身が作業ツリーを汚し、tree hash も clean 判定も壊れるので、
`assertAttestationIgnored` が `git check-ignore` で確かめてから記録・検査する。

### gradle_test.mjs — ラッパーは絶対パスで呼ぶ

Windows で `gradlew.bat` と**裸の名前**を cmd へ渡してはいけない。
`NoDefaultCurrentDirectoryInExePath=1` が設定された環境ではカレントディレクトリを探索しないので、
`cwd` を渡していても「gradlew.bat は、内部コマンドまたは外部コマンド… として認識されていません」で落ちる。
探索順の問題であって作業ディレクトリの問題ではないため、`cwd` を直しても解決しない。

`shell: true` も使わない。引数がエスケープされずに連結されるだけで（Node が DEP0190 で警告する）、
`.\gradlew.bat` のようなパスはバックスラッシュが落ちて `.gradlew.bat` に化ける。
`cmd.exe /d /s /c <絶対パス> <タスク>` を `shell: false` で起動するのが一番素直で、エスケープの余地も無い。

## dev.mjs

Vite開発サーバーを起動するラッパー。Viteは未知のCLIオプションをエラーにするため、`--api` / `--api-target` をここで取り除き、`GKILL_API_PROXY_TARGET` 環境変数に変換してViteへ渡す（`vite.config.ts` の `server.proxy` がこれを読む）。`--api` 以外の引数はそのままViteへ素通しする。

```bash
npm run dev                                  # http://localhost:9999 (既定)
npm run dev -- --api=http://127.0.0.1:19999
npm run dev -- --api=19999                   # ポート番号だけなら 127.0.0.1 を補完
npm run dev -- --api=19999 --port 5180
```

優先順位は `--api` > 環境変数 `GKILL_API_PROXY_TARGET` > 既定値。

## license_getter.mjs

依存パッケージのライセンス一覧をリポジトリルートの `LICENSES_DEPENDENCE` に生成する。4セクション構成:

| セクション | ソース | 内容 |
|---|---|---|
| Go Modules | `src/server` + `src/plugins` 配下の全 go.mod（9モジュール自動発見） | 推移依存含む本文全文。`path@version` でモジュール横断の重複排除。リポジトリ内 replace（自プロジェクト）は除外 |
| Node.js Modules | ルート `package-lock.json` (v3) | ルート package.json の dependencies から到達可能な本番依存のみ（devDependencies はビルドツールチェーンであり配布物に含まれないため対象外）。本文全文 |
| Android (Gradle) | `src/android` の `releaseRuntimeClasspath` | 推移依存のライセンス名+URL（POM から抽出、無ければ親 POM を追跡）。テスト専用依存は配布物に入らないため対象外 |
| Wear OS (Gradle) | `src/wear_os` の phone_companion + watch_app（同上） | 同上（2プロジェクトを GAV で重複排除して統合） |

```bash
npm run license_getter                 # 全セクション収集
npm run license_getter -- --skip-gradle  # Gradle 環境が無い環境用（該当セクションは未収集の注記のみ）
```

**前提:**
- `go` が PATH にあること（`go mod tidy` は実行しない。go.mod / go.sum への副作用なし）
- `npm ci`（または `npm i`）済み
- Gradle 部のみ JDK + Android SDK（無ければ `--skip-gradle`）

出力は全セクション成功後に一括書き出し（途中失敗で既存ファイルを壊さない）、エントリはソート済みで2回実行しても diff が出ない。

## codeql.mjs

GitHub の Code scanning アラートを push する前に手元で再現する。

```bash
npm run codeql                       # go + javascript-typescript
npm run codeql -- go                 # 言語を指定
npm run codeql -- --all              # java-kotlin も含める（Gradle ビルドを伴う）
npm run codeql -- --update-baseline  # 既知の指摘一覧を今回の結果で作り直す
npm run codeql -- --reuse-db         # データベースを作り直さず、クエリだけ回し直す
npm run codeql -- --required         # CodeQL CLI が無いとき失敗にする
```

**CodeQL CLI が無ければスキップして正常終了する。** 環境が揃っている開発機だけで効き、
揃っていない環境（CI の他ジョブや他の開発機）では素通りするので、`npm test` に混ぜても壊れない。
無いことを失敗として扱いたいときだけ `--required` を付ける。

### 設定ファイルは新しく作らない

解析対象パス（`paths-ignore`）は `.github/codeql/codeql-config.yml` を **CI と共用**する。
ローカル専用の設定を別に置くと CI とドリフトして、
「ローカルでは消えたのに CI では出る」という一番タチの悪い状態になる。

クエリスイートだけは CLI 側で明示的に渡している（`<lang>-security-and-quality.qls`）。
config の `queries:` を解決するのは codeql-action であって CLI ではないため。

### 成果物はリポジトリの外に置く

データベースと SARIF は `GKILL_CODEQL_WORK`（既定は OS のテンポラリ配下）へ書く。
言語ごとに数GBになるので、空きのあるドライブを指定すること。
リポジトリ配下に置くと `.gitignore` の管理対象が増えるうえ、数GBが居座る。

CodeQL CLI 自体の場所は `GKILL_CODEQL`（実行ファイルでもバンドルの展開先でもよい）か PATH。
`GKILL_CODEQL` を指定していて外れているときは PATH へ**落とさない** —— 指定したつもりの版と
違うものを黙って使うのを防ぐため。

### 合否はベースライン差分で決める

いま open なアラートには「修正しない」と判断したものが含まれるので、
「指摘が1件でもあれば失敗」では永久に赤いままになり、テストとして機能しない。
`.github/codeql/local-baseline.json` に既知の指摘を記録し、**そこに無い指摘が出たときだけ失敗**する。

指標には CodeQL が出す `partialFingerprints.primaryLocationLineHash` を使う。
行番号だと無関係な編集で全件が「新しい指摘」に化ける。
ベースラインは指標順に並べて書き出すので、同じ結果なら2回実行しても diff が出ない。

ベースラインが存在しない間は全件を表示して合否を判定しない（正常終了する）。
最初の1回だけ `--update-baseline` で作る。

**`--update-baseline` は「今回走らせた言語ぶん」だけを差し替える。** 各エントリは `lang` を持つ。
`npm run codeql -- go --update-baseline` で全体を書き直すと、走らせていない
javascript-typescript / java-kotlin の記録が黙って消え、次に全言語で回したとき
**それらが一斉に「新しい指摘」として赤くなる**（しかも消えたことに気付く手がかりが無い）。
同じ理由で「ベースラインにあって今回出なかった指摘」の集計も走らせた言語だけを数える。
全言語を回したときは全面的に作り直す。

### ローカルは「汚れた作業ツリー」を見る

CI はクリーンチェックアウトなので `dist/` も `build/` も存在しないが、ローカルの作業ツリーには全部ある。
CodeQL の JS/TS 解析はビルドを伴わず**そこにあるファイルを読む**ので、
外さないと生成物由来の指摘が本物を埋めてしまう。

実測（2026-08-22、除外を入れる前）:

| 出所 | 件数 | 正体 |
|---|---|---|
| `src/wear_os/**/build/`, `src/android/build/` | 2812 | Gradle の configuration-cache レポート HTML（インライン JS） |
| `playwright-report/index.html` | 81 | Playwright のレポート |
| `dist/assets/*.js`, `embed/html/assets/*.js` | 22 | Vite の minified バンドル（mermaid 等） |
| **本物** | **5** | `src/client/classes/api/gkill-api.ts` / `src/mcp/lib/*.mjs` |

2919件中2914件がノイズで、しかも `js/use-before-declaration` のような minified コードなら必ず出る規則が
上位を占めるため、本物が埋もれて見えなくなる。

除外は `.github/codeql/codeql-config.yml` の `paths-ignore` に入れてある。
**ローカル専用の設定を作らずに共有 config へ入れるのが正しい** —— これらは CI にも存在しないので
CI 側の意味は変わらず（実質 no-op）、設定を2箇所に分けずに済む。
`git ls-files` で確認したとおり `dist/` `build/` 配下に追跡ファイルは1つも無い。

なお `paths-ignore` が効くのは js-ts のようにビルドせず解析する言語だけで、
autobuild の go / manual build の java-kotlin では無視される（config のコメントにも書いてある）。

### java-kotlin は追跡ファイルを一時的に書き換える

CodeQL の Kotlin extractor は 2.4.10 未満にしか対応しておらず、CI にも
「Pin Kotlin version for CodeQL」ステップがある。ローカルでも同じ固定が要るが、
対象の `libs.versions.toml` 2つは**追跡ファイル**なので、CI のように使い捨てにはできない。

- 実行前に `git status --porcelain` で確認し、未コミットの変更があれば**何もせず失敗**する
- 元の内容を控えて `finally` と SIGINT / SIGTERM で必ず書き戻す
- 戻すのに `git checkout` は使わない（HEAD に戻ってしまい、関係のない未コミット変更まで消える）
- 置換は行末に依存しない正規表現で行い、**当たった件数を数えて0なら失敗**する
  （作業ツリーは CRLF なので、行末を当てにする置換は黙って0件になる）

`go` と `javascript-typescript` はこの固定が要らないため、既定の対象はこの2つだけ。
