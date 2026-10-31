# ADR-0902: Nightly の npm audit は、修正版の無い開発用依存の advisory だけを名指しの例外リストで条件付きに通す

| | |
|---|---|
| Status | Accepted |
| Date | 2026-10-16 |
| Sources | `.claude/skills/gkill-build-test/SKILL.md`「Nightly の npm audit と例外リスト」 / `src/tools/README.md`「npm_audit_gate.mjs」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/tools/npm_audit_gate.mjs` / `.github/workflows/nightly.yml` |

## Context

Nightly の `npm_audit` ジョブは `npm audit --audit-level=high` を fail-closed で回している（指摘 F-009）。
外部 advisory が増えて赤くなるのは「新しい既知脆弱性が届いた」という本物の信号で、握りつぶさないための設定である。
さらに ADR-0901 のリリースゲートは直近 Nightly の run 単位の success を要求するので、
Nightly が赤い間はリリースもできない。

ここへ braces の advisory（GHSA-vfj7-8cjw-p6xm、high。深く入れ子にしたブレース展開でスタックを使い切る DoS）が届いた。

- 修正版が公開されていない。対象範囲は `<=3.0.3` で、3.0.3 が公開中の最新版
- 経路は `@vue/eslint-config-typescript` と `shx`（→ `shelljs`）から `fast-glob` → `micromatch` → `braces`。
  上流の各パッケージの最新版も同じ経路を引くので、依存を上げても消えない
- どちらも開発用依存で、配布物（埋め込みのフロントエンドと Go のバイナリ）には入らない。
  ブレース展開に渡るパターンは `@vue/eslint-config-typescript` の `**/*.vue` と ESLint 設定の ignore、
  `package.json` のスクリプトの固定文字列だけで、外から与えられる経路が無い
- Dependabot はこの advisory を開発用依存として自動で却下したので、アラート一覧には出ず、Nightly だけが赤いまま続いた

「修正版が出るまで待つ」は、待っている間ずっとリリースできないことを意味する。一方で閾値や対象を緩めると、
次に届く本物の信号まで消える。

## Decision

Nightly の npm audit の判定を `src/tools/npm_audit_gate.mjs` に移し、`src/tools/npm_audit_allowlist.json` に
GHSA ID で名指しした advisory だけを、①本番依存（`--omit=dev`）の監査に出てこない ②例外に書いたパッケージ名と
対象範囲が advisory の今の内容と一致する ③見直し期限を過ぎていない — の間に限って通す。
それ以外の high 以上は今までどおり落とす。例外リストはリリースゲートの `DEPENDENCY_MANIFESTS` に含める。

## Rejected alternatives

- **Nightly を `npm audit --omit=dev` だけにする** — 1行で済むが、開発用依存の脆弱性を今後まったく見なくなる。
  開発用依存はビルドとテストの工程で実行されるコードで、shell-quote（critical）のように `npm-run-all2` 経由で
  ビルドスクリプトのコマンド行を組み立てる経路もある。サプライチェーンの入口を丸ごと監視から外すことになる
- **閾値を `--audit-level=critical` に上げる** — 本番依存の high まで見逃す。braces 1件のために閾値全体を動かす理由が無い
- **`npm_audit` ジョブを `continue-on-error: true` にする** — リリースゲートは run 単位の conclusion を見るので、
  ジョブが赤くても run は success になり、新しく届いた脆弱性もゲートを素通りする。E2E を continue-on-error にしているのは
  フレークのためで、npm audit にフレーク要因は無い
- **例外を ID だけで持つ（条件も期限も付けない）** — 修正版が出ても例外が残り続けて誰も上げないし、
  advisory のパッケージが本番依存に入っても通ってしまう。「修正版が無い・開発用依存だけ」という例外の前提が
  崩れたら自動で赤くなるよう、対象範囲の字面・`--omit=dev` との突き合わせ・期限を条件にした
- **既製のツール（audit-ci・better-npm-audit など）を入れる** — 開発用依存を1つ増やす（それ自体が監査対象になる）。
  ID 単位の除外や期限は表せても、「本番依存では例外を効かせない」「advisory の対象範囲が変わったら落とす」には
  結局 `--omit=dev` の結果との突き合わせが要る。判定は Node 標準だけで書ける小さなスクリプトで済む
- **braces を経路ごと外す（shx を置き換え、`@vue/eslint-config-typescript` をやめる）** — shx は置き換えられるが、
  `@vue/eslint-config-typescript` は最新版も `fast-glob` を引き、代わりの設定を自前で組むのは ESLint 設定の大改修になる。
  片方だけ外しても braces は残る
- **`overrides` で `fast-glob` や `micromatch` を別のパッケージへ差し替える** — API 互換の無いものを差し込むことになり、
  ESLint が拾う `.vue` の列挙が例外もエラーも出ずに変わりうる。修正版が出ていない以上、版の下限を張る形の overrides も使えない
- **修正版が出るまで待つ（何もしない）** — 待つ間ずっとリリースゲートが通らず、Nightly が赤いことが平常になる。
  赤が平常になると、同じジョブに新しい脆弱性が届いても気付けない

## Consequences

- 例外は見直し期限が来ると Nightly を赤くする（意図どおり）。修正版の有無を確かめ、例外を消すか期限を延ばす。
  例外リストは `DEPENDENCY_MANIFESTS` に入っているので、触ったら Nightly を回し直すまでリリースゲートは通らない
- advisory の対象範囲が変わると（修正版が出たときに変わる）、条件が一致しなくなって赤くなる。上流が上げてくれれば
  例外を消し、上がらなければ overrides で下限を張る
- 例外にした advisory のパッケージが本番依存に入ると、例外リストに関係なく赤くなる
- `npm audit` の出力が JSON として読めない・エラーを返した・形が違うときは落ちる。via の形が変わって advisory を
  拾えないのに high 以上のエントリがある場合も落ちる（黙って 0 件にしない）
- **静かな壊れ方**: 判定スクリプトの条件を1つでも外すと（`--omit=dev` との突き合わせ・範囲の一致・期限）、
  例外もエラーも出ないまま本番依存の脆弱性や修正版の出た advisory が Nightly を素通りし、リリースゲートまで緑になる。
  条件ごとに落ちることを `test_tools` が固定している。どの advisory にも当たらない例外は警告だけで落とさないので、
  不要になった例外は人が消す

## Evidence

- `npm audit` はこの advisory 1件を、`micromatch` → `fast-glob` → `@vue/eslint-config-typescript` / `shelljs` → `shx` と
  連鎖したエントリ6件の high として数える。`--omit=dev` では0件（この変更時点の `package-lock.json` に対する出力）
- braces は 3.0.3 が公開中の最新版で、advisory の対象範囲は `<=3.0.3`・修正版なし。`@vue/eslint-config-typescript` 14.9.0・
  `shx` 0.4.0・`shelljs` 0.10.0・`fast-glob` 3.3.3・`micromatch` 4.0.8 の各最新版も同じ経路を引く（npm レジストリの照会）
- 同じ期間の Nightly は govulncheck も Go 標準ライブラリの脆弱性で赤かった（`go` 指令を 1.26.9 へ上げて解消）。
  2つの失敗が重なり、どちらが原因で赤いのかが run の見た目からは分からなかった

## 追記（2026-10-31）

braces の advisory（GHSA-vfj7-8cjw-p6xm）は上流で取り下げられ（Withdrawn Advisory）、`npm audit` に出なくなった。
`npm_audit_gate.mjs` が「どの advisory にも当たらない例外」の警告を出したので、例外リストから消して `exceptions` は空になった。
判定の仕組みと条件はこの ADR のまま残す。次に修正版の無い開発用依存の advisory が届いたら、同じ条件で名指しして足す。

## Related tests

- `src/tools/__tests__/npm_audit_gate.test.mjs`
