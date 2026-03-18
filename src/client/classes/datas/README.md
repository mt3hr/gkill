# datas - TypeScript データモデル

## 概要

サーバ側 Go 構造体と1対1で対応する TypeScript データモデル定義。
全 Kyou データ型のクライアント側表現を提供する。
サーバ側の対応: `src/server/gkill/dao/reps/`（エンティティ定義）

## ディレクトリ構造

```
datas/
├── kyou.ts                        # 基底エンティティ
├── kmemo.ts                       # テキストメモ
├── kc.ts                          # 数値記録
├── lantana.ts                     # 気分値（0-10）
├── mi.ts                          # タスク
├── nlog.ts                        # 支出記録
├── ur-log.ts                      # ブックマーク
├── time-is.ts                     # タイムスタンプ
├── tag.ts                         # タグ
├── text.ts                        # テキスト
├── notification.ts                # 通知
├── re-kyou.ts                     # リポスト
├── idf-kyou.ts                    # ファイル
├── git-commit-log.ts              # Git コミットログ
├── gps-log.ts                     # GPS ログ
├── share-kyous-info.ts            # 共有情報
├── info-base.ts                   # 情報ベース型
├── info-identifier.ts             # 情報識別子
├── meta-info-base.ts              # メタ情報ベース型
├── kftl-template-element-data.ts  # KFTL テンプレート要素
├── circle-options.ts              # 地図円オプション
├── lat-lng.ts                     # 緯度経度
└── config/                        # 設定関連モデル
```

## ルートファイル（22ファイル）

### Kyou データ型

| ファイル | Go 対応 | 説明 |
|---------|---------|------|
| `kyou.ts` | `reps/kyou.go` | **基底エンティティ**。全データ型の共通フィールド（ID, CreateTime, UpdateTime 等） |
| `kmemo.ts` | `reps/kmemo.go` | Kmemo — テキストメモ |
| `kc.ts` | `reps/kc.go` | KC — 数値記録（タイトル + 数値） |
| `lantana.ts` | `reps/lantana.go` | Lantana — 気分値（0-10 のスケール） |
| `mi.ts` | `reps/mi.go` | Mi — タスク（チェック状態、期限等） |
| `nlog.ts` | `reps/nlog.go` | Nlog — 支出記録（店名、金額等） |
| `ur-log.ts` | `reps/ur_log.go` | URLog — ブックマーク（URL、タイトル等） |
| `time-is.ts` | `reps/time_is.go` | TimeIs — タイムスタンプ（開始/終了時刻） |
| `tag.ts` | `reps/tag.go` | Tag — タグ（対象 ID に紐づく） |
| `text.ts` | `reps/text.go` | Text — テキスト（対象 ID に紐づく） |
| `notification.ts` | `reps/notification.go` | Notification — 通知（対象 ID に紐づく） |
| `re-kyou.ts` | `reps/re_kyou.go` | ReKyou — リポスト |
| `idf-kyou.ts` | `reps/idf_kyou.go` | IDFKyou — ファイル |
| `git-commit-log.ts` | `reps/git_commit_log.go` | GitCommitLog — Git コミットログ |
| `gps-log.ts` | `reps/gps_log.go` | GPSLog — GPS ログ |

### 共通型

| ファイル | 説明 |
|---------|------|
| `share-kyous-info.ts` | 共有 Kyou 情報 |
| `info-base.ts` | 情報ベース型（Tag/Text/Notification の共通基底） |
| `info-identifier.ts` | 情報識別子 |
| `meta-info-base.ts` | メタ情報ベース型 |
| `kftl-template-element-data.ts` | KFTL テンプレートの要素データ |
| `circle-options.ts` | 地図上の円描画オプション |
| `lat-lng.ts` | 緯度経度座標 |

## `config/` サブディレクトリ（13ファイル）

アプリケーション設定・サーバ設定のモデル定義。
サーバ側の対応: `src/server/gkill/dao/user_config/`, `server_config/`

| ファイル | 説明 |
|---------|------|
| `application-config.ts` | アプリケーション設定（KFTL テンプレート、表示設定等） |
| `server-config.ts` | サーバ設定（ポート、パス等） |
| `repository.ts` | リポジトリ設定（データ保存先定義） |
| `account.ts` | アカウント情報 |
| `device-struct-element-data.ts` | デバイス構造要素 |
| `folder-struct-element-data.ts` | フォルダ構造要素 |
| `kftl-template-struct-element-data.ts` | KFTL テンプレート構造要素 |
| `mi-board-struct-element-data.ts` | Mi ボード構造要素 |
| `mi-board-struct.ts` | Mi ボード構造体 |
| `rep-struct-element-data.ts` | リポジトリ構造要素 |
| `rep-type-struct-element-data.ts` | リポジトリ型構造要素 |
| `rep-type-map.ts` | リポジトリ型マッピング |
| `tag-struct-element-data.ts` | タグ構造要素 |

## 開発ガイドライン

### 新しいデータ型を追加する場合

1. `xxx.ts` でデータモデルクラス/インタフェースを作成
2. サーバ側 `dao/reps/xxx.go` のフィールドと一致させる
3. JSON シリアライズ時のキー名はサーバ側の `json:` タグに合わせる

### 命名規則

- ファイル名: kebab-case（例: `git-commit-log.ts`）
- サーバ側の snake_case と対応（`git_commit_log.go` ↔ `git-commit-log.ts`）
