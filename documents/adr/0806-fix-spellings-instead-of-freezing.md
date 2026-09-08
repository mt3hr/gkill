# ADR-0806: 綴りは直す — plaing を playing へ改名し、互換を残さずデータを一度きりで復旧する

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-08 |
| Sources | [ADR-0802](0802-freeze-plaing-spelling.md) / `AGENTS.md`「Frozen spellings」 / `documents/reverse/glossary.md`「凍結された綴り」 |
| Supersedes | [ADR-0802](0802-freeze-plaing-spelling.md) |
| Superseded-by | なし |
| Anchors | なし — 268ファイルに分散しており、指す先が1つに決まらない。代わりに `src/tools/verify_docs.mjs` の走査検査が全ファイルを見る |

## Context

ADR-0802 は「その綴りが永続データまたは外部契約に乗っているか」で綴り修正の可否を決め、`plaing` を**凍結**に分類した。乗っていた面は5系統（保存済み `FindQuery` JSON／MCP ツールスキーマ／Wear OS の2APK間プロトコル／7言語マニュアルのページ名／`default_page` の保存値）で、うち2つは「相手側の実装だからこちらから移行できない」というのが凍結の主な理由だった。

その凍結は、綴り誤りをコードベースに恒久的に残すことと引き換えに、配布済みデータとバイナリを守るものだった。ADR-0802 自身が「この決定はテストで守られていない。文書だけが防御線で、一括置換で壊せる状態にある」と書いており、防御としても弱かった。

gkill は単一利用者のスタンドアロンアプリで、クライアント（PWA）・サーバ・Wear OS の2APK・MCP サーバのすべてを同じ利用者が同時に更新する。「相手側の実装」は実際には自分の端末にある。凍結が守っていたものの実体は、**利用者自身のDBに残った数行**だった。

## Decision

**綴りは直す。互換は残さない。旧綴りのまま残るデータは、一度きりの変換で復旧する。**

- `plaing` / `Plaing` / `PLAING` は `playing` / `Playing` / `PLAYING` へ全面改名した（2026-09-08、268ファイル・1568箇所）
- 旧綴りを受け続けるシムは**1つも置かない**。旧ルート `/plaing` のリダイレクトも、旧 JSON キーの別名受理も、Wear OS の旧パス受理も置かない
- 旧綴りのまま残っていたデータは、一度きりの変換スクリプトで新綴りへ直した。変換コードは製品に残さない
- 判定基準そのものを差し替える。**「永続に乗っているか」は凍結の理由にしない。** 永続に乗っていても直してよく、その場合は旧綴りのデータを一度きりの変換で直す
- 再発防止に、追跡ファイルへ旧綴りが現れたら落ちる走査検査を `verify_docs`（pre-commit で機械強制）へ入れた

## Rejected alternatives

- **凍結を続ける（ADR-0802 のまま）** — 凍結が守っていた実データは、実測で `user_config.db` 4行・`share_kyou_info.db` 2行・サンプルDB 2行しかなかった。この6行のために、誤綴りを1568箇所・268ファイルに恒久的に残し続ける取引になっていた。`default_page` に至っては全20行が `rykv` で、凍結理由として挙げた5系統のうち1つは実データが**ゼロ件**だった。

- **改名したうえで旧綴りの読み込み互換を残す（`agregate` 方式）** — ADR-0802 が用意していた第3の道。採らなかったのは、綴りが2つ並ぶ状態が恒久化するため。読み込み互換は「いつ消せるか」の判定基準を持てず、`agregate` の互換が実際にそうなったように、**いつの間にか使われなくなっても消されずに残る**（今回の調査で、`agregate` の互換コードは既に削除済みだったのに ADR とスキルの記述だけが残っていた）。旧綴りを受ける口が1つでもあると、走査検査の allowlist が育ち、防御線がまた文書だけに戻る。

- **旧ルート `/plaing` だけリダイレクトで受け続ける** — router には `/regist_first_account` と `/shared_mi` の前例がある。採らなかったのは、あの2つが「配布済みで再発行できない共有URL」と「初回セットアップのトークン付きURL」という**取り返しのつかない外部参照**を受けるためで、`/plaing` はブックマークしか無いから。ブックマークは開き直せる。

- **移行コードを製品の起動時に置く（冪等な UPDATE）** — 変換自体は正しく、`share_kyou_info` DAO に前例もある。採らなかったのは、それが「旧綴りの文字列を製品コードに残す」ことと同義で、しかも消す条件を誰も決められないため。対象が6行と分かっている以上、一度実行して確認して捨てるほうが確実に終わる。

- **新規コードだけ `playing` にする** — 綴りが2つ並ぶ。検索も置換も効かなくなり、どちらが正しいのか誰にも分からなくなる（ADR-0802 の却下理由をそのまま引き継ぐ）。

## Consequences

**旧綴りは追跡ファイルに1つも置けない。** `npm run verify_docs` の走査検査が落ちる。例外は「過去にそう書かれた事実」を記録するものだけで、コードには1つも無い。

1. `documents/adr/0802-freeze-plaing-spelling.md` — 凍結を決めた ADR そのもの。ファイル名の slug も本文も変えない
2. `documents/releasenote/` と `documents/gkill_develop_document.xlsx` — 公開済みの歴史記録
3. この ADR 自身と `src/tools/verify_docs.mjs` — 何を直したのか・何を検出するのかを書くのに旧綴りが要る
4. 用語集の「凍結された綴り」表など、記録することが目的の行。`<!-- retired-spelling-ok -->` を付けて1行ずつ明示的に逃がす

旧形式 `FindQuery` JSON の `use_*` フラグにあった `use_plaing` も `use_playing` へ改名した。これは「我々の綴りではなくデータのキー名だから触らない」と一度は判断したが、**保存データを数えたら3つのDBすべてで `use_*` フラグ自体が0件**で、守る対象が存在しなかった。Go / client / MCP の3実装が同じ16キーを扱うという [ADR-0106](0106-find-query-null-semantics.md) の約束は、3実装を同時に改名したので保たれている。

**互換を残さない選択の代償**は、更新前のPWAキャッシュ・更新前の Watch APK・進行中の MCP セッションが、更新するまで実行中まわりで動かないこと。ブラウザの localStorage に保存されていた列条件の `plaing_time` と、ポートの実行中ウィンドウの位置・サイズは一度だけ既定へ戻る。MCP へ旧引数 `plaing_time` を送ると未知キーとして throw する（黙って無視されるより良い）。

**Wear OS は phone と watch の両APKを同時に入れ直す必要がある。** データレイヤーのパスが変わっており、片方だけ更新すると**例外もエラーも出さずに応答が来ないだけ**になる。

## Evidence

改名の規模（2026-09-08 実測、追跡ファイル）:

| 領域 | ファイル |
|---|---|
| `src/client/` | 145 |
| `src/server/` | 33 |
| `src/wear_os/` | 18 |
| `documents/` | 21 |
| `src/locales/` | 7 |
| `resources/` | 29 |
| `.claude/skills/` | 5 |
| 計 | **268ファイル / 1568箇所 / ファイル名40本** |

`src/android/` と `src/plugins/` は0件。`plaing` が別の語の一部になっている例は無く、置換後に既存識別子と衝突する名前も0件だった。

凍結が守っていた実データ（改名前の実測）:

| 保存先 | 対象 | 件数 |
|---|---|---|
| `user_config.db` | KEY `PLAING_TIMEIS_JSON_DATA`（値は `{"plaing_timeis_find_kyou_query":null}`） | 2行 |
| `user_config.db` | `DASHBOARD_JSON_DATA` の中の `plaing_time` | 2行 |
| `share_kyou_info.db` | `FIND_QUERY_JSON` の `plaing_time` | 2行（全2行） |
| サンプルデータの `user_config.db` | 上と同じ2種 | 2行 |
| `DEFAULT_PAGE` の値 | **全20行が `rykv`** | 0行 |
| 旧形式の `use_*` フラグ（`use_plaing` 含む）を含む保存行 | 本番2DB＋サンプルDB | **0行** |
| `Agregate*`（旧綴り）を含む生存行 | — | **0行**（未VACUUM領域のバイト列のみ） |

ADR-0802 が「凍結しなくてよかった例」として挙げていた `agregate` は、調査時点で**読み込み互換コードが既に存在しなかった**。互換は消されていたのに、ADR と `dnote/README.md` の記述だけが「互換を取っている」と言い続けていた。読み込み互換が資料と実装のドリフト源になる実例として記録しておく。

## Related tests

- `src/tools/verify_docs.mjs` — 追跡ファイルに旧綴りが現れたら落ちる走査検査。`.githooks/pre-commit` が `npm run verify_docs` を機械強制するので、ADR-0802 が「文書だけが防御線」と書いていた状態はここで解消した
- `src/client/__tests__/unit/router.test.ts` — ルート名の集合（`playing` を含む）
- `src/mcp/__tests__/constants.test.mjs` — `KYOUS_QUERY_DATETIME_FIELDS` に `playing_time` があること
