# ADR-1103: ウォッチアプリのチップはタイルと同じ見た目にし、確認画面は ScalingLazyColumn で収める

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-13 |
| Sources | `.claude/skills/gkill-mobile/SKILL.md`「ウォッチアプリのチップはタイルと同じ見た目にし、確認画面は ScalingLazyColumn で収める」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/wear_os/watch_app/src/main/java/com/gkill_android/mobile_app/src/gkill/mt3hr/gkill/wear/watch/presentation/components/MenuChip.kt` の KDoc / `MainActivity.kt` の `HomeMenuScreen` と `DuplicateConfirmScreen` の KDoc |

## Context

ウォッチ側には同じ3つの導線（記録する / 実行中 / 気分記録）を出す場所が2つある。ウォッチフェイスから
横スワイプで出るタイル（`GkillTileService`、protolayout-material）と、アプリのトップメニュー
（`MainActivity` の `HomeMenuScreen`、Compose for Wear OS）である。両者は別のUIフレームワークで別々に
書かれていて、2026-09-10 の気分記録追加（[ADR-1101](1101-wear-mood-goes-through-kftl-text.md)）で
トップメニューが3項目になったとき「gkill タイトル込みでは丸画面に収まらない」として `ScalingLazyColumn` 化された。
その結果、タイルは「140dp 幅の primary 色チップ3つ・文字中央揃え・画面中央」なのに、トップメニューは
「タイトル＋全幅チップ（1つ目だけ primary、残りはグレー）・文字左寄せ・端の項目が縮小/減光」になり、
利用者から「ウォッチフェイスとアプリ画面のデザインが違う。タイル側に揃えてほしい」と指摘された。
一覧画面（テンプレート一覧・実行中一覧）のチップも全幅・左寄せで、同じ指摘の対象だった。

もう1つ、同じ内容を直前に保存済みのときの重複確認画面（`DuplicateConfirmScreen`）が画面に収まっていなかった。
固定の `Column(fillMaxSize, padding 16dp, Center)` に3行の本文（約68dp）と全幅チップ2つ（52dp＋余白 8dp ×2）を
積むと約190dp必要だが、Pixel Watch 2 の 192dp 角の丸画面から余白を引くと 160dp しか無い。
下の「キャンセル」がエラーも警告も出さずに画面外へ落ち、利用者には「収まっていない」としか見えない。

## Decision

1. **見た目の基準はタイル**（protolayout-material `Chip` の既定値）とし、アプリ側のチップは共通部品
   `MenuChip`（140dp 固定幅・primary 色・文字中央揃え・末尾省略）で揃える。幅の定数 `MENU_CHIP_WIDTH_DP` は
   タイルと `MenuChip` で共有する。トップメニューはタイルの `buildLayout` をそのまま写した
   「`Box` 中央の `Column` に隙間なく3つ」で、タイトルを持たず `ScalingLazyColumn` にも入れない。
   一覧画面の項目チップも `MenuChip` にする（利用者の選択）。
2. **重複確認画面は文言を変えず `ScalingLazyColumn` でスクロールできるようにする**（利用者の選択）。
   最初は本文（item 0）を中央に置き、「それでも送信」は本文の下に見え、「キャンセル」は一段スクロールで届く。

## Rejected alternatives

- **トップメニューを `ScalingLazyColumn` のまま色と幅だけ揃える** ―― `ScalingLazyColumn` は端の項目を縮小・減光する
  （それが部品の目的）ので、3項目が全部見えていても上下の項目がタイルより小さく薄く見える。3チップ 156dp は
  タイルが同じ配置で丸画面に収まっていることが実証しているので、スクロールは要らない。
- **タイル側に隙間やタイトルを足してアプリ側へ寄せる** ―― 利用者が気に入っているのはタイル側で、依頼は「タイルに揃える」。
  片側だけ足すと再びずれるので、足すなら両方同時（定数の共有と同じ考え方）。
- **一覧画面のチップは全幅のまま残す** ―― 長いテンプレート名の切り詰めが遅くなる利点はあるが、タイル→一覧と
  遷移した直後に見た目が変わるのは指摘そのもの。利用者が「一覧のチップも揃える」を選んだ。
  切り詰めはタイルと同じ規則（副ラベル無しなら2行・有りなら1行・末尾省略）にして情報量を確保する。
- **重複確認を他の確認画面と同じ「本文＋✕ / ✓ の丸ボタン2つ」に置き換える** ―― 高さは 124dp に収まり、
  テンプレート送信確認・気分記録確認・打刻終了確認と同型になる。しかし「それでも送信」「キャンセル」の文字ラベルを
  失う。利用者が文言を残す方を選んだ。
- **重複確認の2つのチップに `CompactChip`（32dp）を使って固定 `Column` に収める** ―― 140dp に収まるが、
  他の画面で「🔄 更新」にだけ使っている小型部品を主操作に使うことになり、文字が大きい端末設定では再び溢れる。
  スクロールできる形なら本文や訳の長さに依存しない。
- **`ScalingLazyColumn` の既定（item 1 を中央）のままにする** ―― 3行本文（68dp）が中央のチップの上に置かれると
  本文の1行目が丸画面の上端に掛かって欠ける。問いを読んでから操作する画面なので本文を最初に中央へ置く。

## Consequences

- タイルとアプリのチップ幅は `MENU_CHIP_WIDTH_DP` の1箇所。片方だけ変えられない構造だが、色・高さ・パディング・
  ピル形状は両フレームワークの「既定値が同じ」ことに依っている。protolayout-material か Wear Compose Material の
  メジャー更新で既定が変わると静かにずれるので、依存を上げたら実機でタイルとトップメニューを並べて見る。
- トップメニューは固定配置なので、4つ目の導線を足すと丸画面に収まらない。足すならタイル側の収まり方を先に決める
  （タイルは `Column` に隙間なく積む以外の配置を持っていない）。
- 一覧のチップは 140dp 固定になり、長いテンプレート名は2行で末尾省略される。de / fr / es の訳を短く保つ制約
  （[ADR-1102](1102-wear-ui-strings-in-android-resources-with-ja-default.md)）は一覧の項目名にも及ぶ。
- 重複確認は `ScalingLazyColumn` なので「キャンセル」までは一段スクロールが要る。Back（スワイプ）でもキャンセルになる
  （`BackHandler` は既存）。他の確認画面（テンプレート送信確認など）は固定 `Column` のままで、長いテンプレート名では
  同型の溢れが起こり得る。今回は依頼の範囲外として触っていない。

## Evidence

- protolayout-material 1.4.1 の既定値（`ChipDefaults` / `Chip$Builder` / `Colors` のバイトコードを javap で確認）:
  `DEFAULT_HEIGHT` 52dp、`HORIZONTAL_PADDING` 14dp、`PRIMARY_COLORS` = `Colors.DEFAULT`（primary `#AECBFA` /
  onPrimary `#303133`）、`getCorrectHorizontalAlignment` は primary label のみなら `HORIZONTAL_ALIGN_CENTER`、
  `getCorrectMaxLines` は副ラベル無しなら 2・有りなら 1、`TYPOGRAPHY_BUTTON` は 15sp / weight 700。
  materialcore `Chip` は外側に margin を持たず、タイルの `Column` に Spacer も無いのでチップは隙間なく積まれる。
  Wear Compose Material の `Colors()` 既定・`ChipDefaults.Height` 52dp・`ContentPadding` 水平 14dp・
  `typography.button` 15sp Bold・`shapes.small` 50% はこれと一致する。
- 重複確認の高さ: 本文3行（body1 16sp / 行高 20sp ≈ 60dp ＋ 余白 8dp）＋ チップ 52dp ×2 ＋ 各 8dp の余白 ＝ 約190dp。
  Pixel Watch 2 は 384px / 320dpi ＝ 192dp 角で、`padding(16dp)` を引くと 160dp。
- タイルの3チップ（156dp）が丸画面に収まっていることは利用者の実機で確認済み（今回の依頼の前提）。
- 実機の見た目の比較はウォッチが adb 未接続のため未実施（`assembleDebug` と 226 テストは通過）。

## Related tests

なし — Compose UI と protolayout の見た目は JVM 単体テストで検査できない。幅は定数の共有で守り、
色・高さ・形状は両ライブラリの既定値に依る（上の Consequences）。既存の `MainActivityTest` の画面状態数（13）は変わらない。
