# ADR-0901: リリースはテスト済み attestation と CI / Nightly の緑を機械で要求する（人の記憶に頼らない）

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-14 |
| Sources | `.claude/skills/gkill-build-test/SKILL.md`「リリースゲート（テスト済み attestation）」 / `documents/reverse/operations-guide.md`「リリースゲート」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/tools/verify_release_gate.mjs` / `src/tools/run_test_suite.mjs` / `src/tools/attestation.mjs` / `src/tools/put_version_info.mjs` |

## Context

`npm run release` はテストを実行しない（成果物の存在・SHA-256・APK 署名の検証だけ）。
「直近の Nightly が green で、ローカルの `npm test` も green のコミットだけリリースする」という規約は
2026-08-30 監査 F-009 の対応で `documents/reverse/operations-guide.md` に**文章として**書かれたが、
守られているかを確かめる仕組みが無く、リリースする人の記憶に依存していた。

同じ監査の F-006 は、それまで正式配布していた APK 3 本が `assembleDebug` の成果物を配布名へ rename した
もの（debug 鍵署名・minify 無し）だったことを見つけている。リリース工程の検証ギャップは仮定ではなく実在した。

2026-09 の外部フィードバック #10「リリース工程に人間の記憶を要求している。守るべき条件なら機械で強制すべき。
テスト済み commit SHA の attestation が無ければ release 不可、くらいまでやっていい」を受けて設計した。

前提となる実測:

- release は WSL でしか通らない（Android NDK のツールチェーンパスが Linux 固定）が、vitest と E2E は
  `node_modules` を入れた側の OS（Windows）でしか動かない。「release の中でテストを回す」は成立しない
- この開発機は並列プロセス過多で 0xC0000142 の偽失敗が出るため、`npm test` を一発で回さず段階ごとに回す運用
- WSL 側の git は autocrlf 未設定で、Windows が CRLF で checkout した追跡ファイル 184 件が素の `git status` で
  偽 dirty に見える（2026-09-14 実測）。さらに `git status` は index の stat cache と**サイズが違うだけ**で
  内容を読まずに modified と言う（`ie_modified` の `DATA_CHANGED` 短絡）ので、`-c core.autocrlf=true` を
  付けても CRLF でサイズが変わったファイルは偽 dirty のまま。`git diff HEAD` は内容を改行変換のうえで比べる
- E2E は PATH 上の `gkill_server` を起動する。古いバイナリで E2E を通しても現在のツリーを検証したことにならない
- Nightly の `e2e` / `android_wear` ジョブは既知フレークのため `continue-on-error: true` で、
  run 単位の conclusion が success でも E2E が通った保証にはならない（保証されるのは govulncheck / npm audit）

## Decision

`npm run release` の先頭にリリースゲート（`src/tools/verify_release_gate.mjs`）を置き、
①作業ツリーがクリーン ②必須スイート全部が HEAD の tree hash に対して attestation 済み ③E2E は PATH 上の
`gkill_server` も同じ tree から作られている ④GitHub の CI が HEAD で success・直近 Nightly が success かつ
48 時間以内・HEAD の祖先・以後に依存の宣言が変わっていない — を fail-closed で要求する。抜け道フラグは作らない。

attestation は commit SHA ではなく**作業ツリーの tree hash**に束縛し、`npm run test_*` の実体である
`src/tools/run_test_suite.mjs` がスイート成功時にだけ `test_attestation.local.json` へ書く。

## Rejected alternatives

- **`npm run release` の中で `npm test` を回す** — release は WSL 専用、vitest / E2E は Windows 専用で、同じ
  プロセスに同居できない。仮に動いても 30 分の直列実行になり、0xC0000142 の偽失敗 1 つで全部やり直しになる
- **commit SHA に束縛し、記録時もクリーンなツリーを要求する** — 「dirty なツリーでテストを通す → コミット」の
  順序が普通なのに、コミットした瞬間に全スイートが再テストになる。tree hash なら「そのまま全部コミット」で
  `HEAD^{tree}` が一致するので再テスト不要。ゲート側でクリーンを要求するので、結局「テスト済みコミット」と同値
- **`npm test` 一発の成功だけを記録する（スイート単位にしない）** — この開発機では段階ごとに回す運用が
  前提で、一発で通ることのほうが珍しい。スイート単位なら落ちた 1 本だけ回し直せる
- **各 `test_*` スクリプトの末尾に `&& node <記録スクリプト> <suite>` を足す（ランナーを作らない）** — npm は
  `--` 以降の引数をスクリプト文字列の**末尾**に付けるので、`npm run test_client_e2e -- --workers=2` の
  `--workers=2` が playwright ではなく記録スクリプトに渡り、この開発機で必須の並列度指定が使えなくなる
- **`record` サブコマンドを用意して人が打つ** — 「打ったかどうか」がまた人の記憶になる。記録はスイートの
  exit 0 に機械的に連動させる
- **緊急用の強制フラグ（`GKILL_RELEASE_FORCE=1` を成果物に焼き付ける）** — `--no-verify` を「何があっても
  禁止」にしている規約と矛盾する。GitHub 障害なら復旧を待ち、フレークなら直す（ユーザー決定）
- **ローカル attestation だけで GitHub を見ない** — Windows で全緑でも Linux（CI）だけ落ちる事故が
  2026-08-15〜20 に 5 日間続いた実例がある。push 済み・CI 緑を要求することで公開コミットからしか
  リリースできなくなる副次効果もある
- **Nightly をジョブ単位（E2E ジョブの conclusion）で判定する** — E2E ジョブは continue-on-error で
  揺れることが分かっているので、要求するとフレークでリリースが止まる。E2E の実証はローカル attestation が担い、
  Nightly には govulncheck / npm audit の鮮度（48 時間・依存の宣言が変わっていない）だけを求める
- **クリーン判定に `git status --porcelain` を使う** — WSL では CRLF のサイズ差だけで偽 dirty になる
  （上記の実測）。`git diff --name-only HEAD` + `ls-files --others --exclude-standard` にした
- **GitHub Actions でリリース成果物を作る（provenance attestation を使う）** — NDK・署名鍵・7za・WSL 前提を
  丸ごと CI へ移す大改修で、2026-08-30 監査の F-012（SBOM / provenance）でも見送っている
- **リポジトリ外の配置スクリプトにもゲートを入れる** — 別リポジトリなので本 ADR の
  範囲外（ユーザー決定）。資料で「呼ぶ側で `npm run verify_release_gate` を先に実行する」と案内するに留める

## Consequences

- リリースは「バージョンを上げてコミット → push → CI 完了 → その最終ツリーで全スイート → release」の順に
  なる。バージョン bump も tree を変えるので、bump の前に回したテストは無効になる
- ネットワーク不通・GitHub API の障害でもリリースできない（fail-closed）。緊急時の抜け道は無い
- 段階ごとに回すときは `npm run test_*` を経由する。`go test` や `run-e2e.mjs` を直接叩いた結果は記録されない
- `verify_docs` / `test_plugins` / `test_android` / `test_wear_os` は引数が 1 つでもあれば記録されない
  （引数が検査内容そのものを置き換えるため）。**記録されなくてもスイートは exit 0 で終わる**ので、
  「通ったのにゲートで記録なしと言われる」形で気付く（stderr に 1 行出る）
- E2E は `npm run put_version_info_embed` を通したビルドの `gkill_server` を PATH に置いてから回す。
  `tree_hash` 無しの古いバイナリで通した E2E はゲートで弾かれる
- `test_attestation.local.json` が ignore されていないと自己参照で壊れるので、`.gitignore` の明示行を
  消してはいけない（`assertAttestationIgnored` が止める。`*.local` では当たらない）
- ゲートを通ってから 30 分のビルドの間にコミットや編集が入ると、`verify_release_artifacts` が HEAD / tree /
  clean の再検査で止める。**ここまで来た成果物は捨ててやり直す**
- **静かな壊れ方**: `run_test_suite.mjs` の `SUITES` 表と `attestation.mjs` の `REQUIRED_SUITES` がずれると、
  ランナーが起動時に止まる（`test_tools` が 1:1 を検査する）。ゲートの検査を 1 つでも外すと、
  例外もエラーも出ないまま「テスト未実施のコミットが配布される」に戻る

## Evidence

- WSL の素の `git status --porcelain | wc -l` = 184、`git -c core.autocrlf=true` 付き = 0（2026-09-14、この開発機）。
  使い捨てリポジトリで `hello\n` を `hello\r\n` に書き換えると、`-c core.autocrlf=true status` は modified、
  同じ設定の `diff --name-only HEAD` は空（git 2.38.1 Windows。`test_tools` が同じ形を固定している）
- 導入直後のゲート実行（2026-09-14）: HEAD の CI は success だったが、直近 Nightly（`ec4892e`）以後に
  `src/plugins/gkill_plugin_archived_git_commit_log/go.mod` が追加されていたため Nightly の NG で止まった。
  「依存の宣言が変わっているのに Nightly を通していない」を機械が初めて指摘した実例
- 同日、導入前の HEAD `6f7b98cf` は Linux の Go テスト 1 件で CI が赤だった。旧規約のままなら人が気付かない限り
  リリースできていた

## Related tests

- `src/tools/__tests__/attestation.test.mjs`
