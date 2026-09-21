# ADR-0222: 静止画のサムネイルは vips があれば CLI で作り、無ければ Go → ffmpeg で作る

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-22 |
| Sources | `.claude/skills/gkill-go-backend/SKILL.md`「静止画のサムネイルは vips があれば」/ [ADR-0214](0214-thumbnail-decodes-by-content-not-extension.md) / [ADR-0206](0206-no-nested-threads-go.md) / [ADR-0212](0212-derived-cache-scan-lists-directories.md) |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/reps/idf_thumb_file_server.go`（`generateThumbJpeg` / `preferNativeThumbDecode` / `runVipsThumb` / `scaleThumbImage` / `ffmpegThumbArgs`） |

## Context

サムネイル生成（`idf_thumb_file_server.go`）に外部から6点の指摘が入った。要旨は「JPEG / PNG 経路の `CatmullRom` が重い」
「ffmpeg のプロセス並列 × 内部スレッド並列で NumCPU² に膨らむ」「EXIF 回転で原寸を3回コピーする」「`.failed` を一括列挙していない」
「本命は libvips」「タイル HEIF の3段経路も libvips へ」。コードを読むと前提はすべて現状と一致していた。

ただし数字は指摘のとおりではなかった。12MP（4000×3000）の JPEG を 400×400 にするとき、Go 経路の内訳は
`image.Decode` の原寸復号が約 250ms、`CatmullRom` の縮小が約 380ms、縦向き（Orientation 6）ならその前の原寸回転が
さらに約 400ms と 48MB×2 の確保。**縮小より回転のほうが重く、復号は pure Go では縮められない**
（`image/jpeg` に DCT 段階の縮小が無い）。一方 libvips は JPEG / HEIC を shrink-on-load で復号し、
EXIF 回転・中央切り抜き・縮小・JPEG 出力まで1プロセスで済む。

利用者の意向は「ffmpeg と同じく、外部ツールが PATH にあれば起動する形」。ffmpeg で実際に踏んだ罠
（本番サービスは LocalSystem 起動でシステムの PATH しか見えず、無いことが「失敗」として印に焼かれた: ADR-0214）を
そのまま引き継ぐ必要がある。

さらに実測すると、**gkill の主な入力である自動取得の画面のスクリーンショット（WebP / PNG、1〜4MP）では vips のほうが遅かった**。
vips はプロセス起動（DLL の読み込み）に百ms 級かかり、PNG / WebP には shrink-on-load が無いので、
Go の復号 + 縮小のほうが 1〜2 割速い。「vips があれば全部 vips」では、写真が速くなる代わりにスクリーンショットが遅くなる。

## Decision

静止画のサムネイルは **vips（libvips の CLI、`vips thumbnail`）が PATH にあればそれで作り、無ければ・失敗したら
Go の `image.Decode` → ffmpeg の順に落ちる**。ただし **Go で読めて小さい画像は vips を飛ばして Go から始める**
（`preferNativeThumbDecode`: `image.DecodeConfig` でヘッダだけ読み、形式別の画素数上限 —— JPEG 3MP / それ以外 6MP —— 以下なら Go）。
vips が無いのは失敗ではなく、最初の生成時に Info を1行残すだけ。3段とも失敗したときだけ印を焼き、vips のエラー文も包む。

外部ツールの内部並列は絞る（vips は `VIPS_CONCURRENCY=1`、静止画の ffmpeg は `-threads 1` + `-filter_threads 1`、
全 ffmpeg に `-nostdin`）。専用のセマフォは足さない。

Go 経路は **縮小カーネルを `CatmullRom` + `Over` から `BiLinear` + `Src` へ**、**EXIF 回転を縮小の後ろへ**移す
（`scaleThumbImage`。軸が入れ替わる向き 5〜8 は切り抜きのアスペクトを源座標で入れ替え、(h,w) に縮小してから回す）。

一括生成は `.failed` の印も同じ列挙で取り、印のある対象を goroutine へ投入しない（`CachedThumbNames` が
generated / failed の2集合を返す）。

キャッシュ名に縮小方式やバックエンドの世代は入れない。

## Rejected alternatives

- **libvips を cgo で組み込む** — このプロジェクトは CGO 無し（pure Go SQLite）で、Android / Termux 向けのビルドも持つ。
  必須の C 依存にすると配布と全プラットフォームのビルドが変わる。CLI なら ffmpeg と同じ「あれば使う」で済む。

- **vips があれば全部 vips に回す** — 実測で WebP のスクリーンショット（3840×1080、4.15MP）は Go 187ms / vips 221ms、
  2048×1536 の PNG は Go 168ms / vips 162ms と、大きな差が無いか Go が速い。vips が桁で速いのは JPEG / HEIC の
  shrink-on-load が効くときだけ。ヘッダを読む（`DecodeConfig`。12MP の JPEG で 14ms）コストで振り分けるほうが総和が小さい。

- **`vipsthumbnail` コマンドを使う** — 版によって EXIF 回転の既定（`--rotate` / `--no-rotate`）が変わった。
  `vips thumbnail` 操作（8.5 以降）は既定で回転し、`--crop centre` の意味も安定している。

- **ApproxBiLinear に変える（指摘のまま）** — 12MP → 400×400 の縮小が 380ms → 5ms になるが、`ApproxBiLinear` は近傍4画素しか
  見ない（`x/image/draw` の `Kernel.Scale` だけが縮小率ぶん支持幅を広げる）ので、7.5倍縮小では縞（エイリアシング）が出る。
  「ApproxBiLinear で 800×800 まで落としてから CatmullRom」の二段も、1段目で同じ欠陥が入る。
  `BiLinear`（tent）カーネルなら面積平均に近い品質のまま `CatmullRom` の約半分（380ms → 177ms）。

- **EXIF 回転を「座標変換する `image.Image` ラッパー」で持つ（指摘のまま）** — `x/image/draw` は `*image.YCbCr` /
  `*image.NRGBA` / `*image.RGBA` の型別 fast path を持ち、独自型は汎用の `At()` 経路に落ちて逆に遅くなる。
  縮小してから 400×400 を回せば回転のコストは無視できる（600ms → 178ms、135MB → 41MB）。

- **ffmpeg 専用のセマフォ（NumCPU/2）を `thumbSem` の内側に置く（指摘のまま）** — 入れ子のセマフォは `thumbSem` の
  スロットを ffmpeg 待ちが占有し、Go で作れる画像まで詰まる（ADR-0206 が `threads.Go` の入れ子を禁じたのと同型）。
  `-threads 1` にすれば NumCPU 個の単一スレッド ffmpeg ≒ NumCPU コアで過剰予約は消える。

- **動画サムネイルの ffmpeg も `-threads 1` にする** — 動画は `-ss` の先読みで数十〜数百フレームを復号するので、
  HTTP 経路（一覧を開いた瞬間）の待ち時間が 4K HEVC で数秒に伸びる。フィルタは1フレームなので `-filter_threads 1` だけ付ける。

- **ffmpeg の `-q:v 2` → `-q:v 4`、`flags=bilinear`、`-pix_fmt yuvj420p`（指摘のまま）** — 400×400 の出力側の
  scale / encode は元画像の読み込みに比べて誤差で、出力が変わるだけで速くならない。

- **キャッシュ名に世代（`thumb-v2`）を入れる（指摘のまま）** — 縮小カーネルやバックエンドの差は 400×400 では見分けがつかない。
  世代を分けると利用者ごとの数十万件を全部作り直す（数時間）。ETag は `W/"<名前>"` のまま。

- **タイル HEIF の3段経路（ffmpeg 原寸 JPEG → Go 再復号 → 縮小）を libvips へ移す（指摘の6番）** — vips 経路が前段に
  入ったことで HEIC は libheif が直接読み、この3段は vips が無い環境と vips が読めなかったファイルの逃げ道として残るだけになった。

## Consequences

- **vips が無い環境（Android 同梱サーバ・Termux・入れていない PC）では挙動は退行しない。** Go 経路が速くなり、
  ffmpeg 経路のスレッド数が減るだけ。
- **vips が見えていないことは、失敗としては現れない。** 速度の差としてしか現れないので、`gkill_info.log` の
  `thumbnail backends detected`（最初の生成時に1行）で確かめること。本番サービスは LocalSystem 起動で
  システムの PATH しか見えない。
- **vips の出力は絶対パス・`.jpg` 終端・`[Q=<品質>,strip]` で渡すこと。** 相対パスだと vips は入力ファイルの隣
  （= rep の中身）へ書く。`.tmp` 終端だとセーバが選べず何も書かない。`strip` が無いと回転済みの画素に
  Orientation タグが残り、ブラウザがもう一度回す。いずれもエラーにならない。
- **vips の失敗は Debug でしか残らない**（`swallowedDebugAllowlist`）。vips が読めない形式は常態で、
  最終的に3段とも失敗すれば印と応答に vips のエラーが包まれて出る。
- **形式別の画素数上限は環境依存の数字。** 速い CPU では Go 側が有利になり境界は上がる。桁が変わらなければ
  どちらに倒れても数十ms の差なので、実測し直す必要があるのは CPU が大きく変わったときだけ。
- **初回の vips 起動は OS の実行ファイル走査で1秒を超えることがある**（Windows Defender）。2回目以降は百数十ms。
- ffmpeg の呼び出しに `-nostdin` が付く。`exec.Command` の stdin は元から nil なので実害は無かったが、
  配線が変わっても ffmpeg が標準入力を待たない。

## Evidence

i7-10510U（8 論理コア）、Windows 11、libvips 8.18.6（`vips-dev-x64-all`）、Go 1.26。`go test -run '^$' -bench Thumb -benchmem`
を単独で走らせた比率。絶対値は環境で変わる。

縮小カーネル（4000×3000 の `*image.YCbCr` を中央切り抜き → 400×400、`BenchmarkThumbScale`）:

| 手順 | ns/op | 備考 |
|---|---|---|
| CatmullRom + Over（旧） | 約 380ms | |
| CatmullRom + Src | 約 340ms | |
| **BiLinear + Src（新）** | **約 177ms** | 旧の 2.1 倍 |
| ApproxBiLinear + Src | 約 5ms | 却下（縞が出る） |

EXIF Orientation 6（`BenchmarkThumbOrientation6`）:

| 手順 | ns/op | B/op |
|---|---|---|
| 原寸で回してから CatmullRom（旧） | 約 595ms | 135MB |
| **BiLinear で縮小してから回す（新）** | **約 178ms** | **41MB** |

端から端まで（`BenchmarkThumbBackends`。プロセス起動と復号を含む）:

| 入力 | vips | Go（新） | 振り分け後 |
|---|---|---|---|
| JPEG 12MP（4000×3000） | **145ms** | 468ms | 159ms（vips + ヘッダ読み） |
| PNG 3.7MP（2560×1440、合成） | 204ms | 219ms | 206ms |

実ファイル（自動取得のスクリーンショットの rep から各 60 件、1件ずつ逐次）:

| 入力 | vips | Go（新） |
|---|---|---|
| WebP 3840×1080（4.15MP、PC のスクリーンショット） | 221ms | **187ms** |
| WebP 1080×2424（2.6MP、スマホのスクリーンショット） | 185ms | **152ms** |
| PNG 2048×1536 級 + JPEG 小（9 件） | 162ms | 168ms |
| JPEG 2.1MP（スマホのアプリが縮小して保存した写真、各 40 件） | 131〜155ms | **108〜120ms** |
| JPEG 0.3MP（小型カメラ、40 件） | 123ms | **52ms** |

一括生成（`generate_thumb_cache`、検証環境の利用者 1 人ぶん 1 万件規模。大半が 4MP 以下のスクリーンショットと縮小済み写真、8 並列）:

| 版 | 所要時間 |
|---|---|
| 変更前（CatmullRom + 原寸回転） | 12分43秒 |
| 変更後・vips 無し | 11分41秒（−8%） |
| 変更後・vips あり | 11分20秒（−11%） |

小さい画像では復号が支配的で、縮小と回転の改善は比例して小さい。vips の効きも大きい写真に限られる。
12MP 級の写真と HEIC が多い rep（カメラの原寸保存）でこそ効く変更で、スクリーンショット主体の rep では 1 割の短縮にとどまる。

vips の JPEG は shrink-on-load でほぼ定数（`--vips-progress` で復号 50ms、残りは起動）。HEIC は手元に素材が無く未計測
（Windows の vips ビルドは HEVC の**エンコーダ**を持たず、ベンチの中で HEIC を合成できない）。

## Related tests

- `src/server/gkill/dao/reps/idf_thumb_vips_test.go`
- `src/server/gkill/dao/reps/idf_thumb_orientation_test.go`
- `src/server/gkill/dao/reps/idf_thumb_external_args_test.go`
- `src/server/gkill/dao/reps/idf_thumb_scale_bench_test.go`
- `src/server/gkill/dao/reps/idf_thumb_batch_test.go`
- `src/server/gkill/dao/reps/idf_thumb_content_decode_test.go`
