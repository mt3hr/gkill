# gkill 画面遷移図（ステートマシン）

コードの `src/client/router/index.ts` と画面設計シートに基づく画面遷移。

## 1. 全体画面遷移

```mermaid
stateDiagram-v2
    [*] --> LoginPage: アプリケーション起動

    LoginPage --> KFTLPage: ログイン成功 → /kftl
    LoginPage --> RykvPage: ログイン成功 → /rykv
    LoginPage --> MiPage: ログイン成功 → /mi
    LoginPage --> KyouPage: ログイン成功 → /kyou
    LoginPage --> MkflPage: ログイン成功 → /mkfl
    LoginPage --> PlaingPage: ログイン成功 → /plaing
    LoginPage --> SaihatePage: ログイン成功 → /saihate
    LoginPage --> SetNewPasswordPage: パスワードリセットリンク → /set_new_password
    LoginPage --> RegistFirstAccountPage: 初回起動 → /regist_first_account

    KFTLPage --> RykvPage: ナビゲーション
    RykvPage --> KFTLPage: ナビゲーション
    RykvPage --> MiPage: ナビゲーション
    MiPage --> RykvPage: ナビゲーション
    KyouPage --> RykvPage: ナビゲーション
    MkflPage --> RykvPage: ナビゲーション
    PlaingPage --> RykvPage: ナビゲーション

    KFTLPage --> LoginPage: ログアウト
    RykvPage --> LoginPage: ログアウト
    MiPage --> LoginPage: ログアウト
    KyouPage --> LoginPage: ログアウト
    MkflPage --> LoginPage: ログアウト
    PlaingPage --> LoginPage: ログアウト
    SaihatePage --> LoginPage: ログアウト

    [*] --> SharedPage: 共有リンクアクセス
    [*] --> SharedMiPage: 共有タスクリンクアクセス
```

**メイン画面群（認証必要）:** KFTLPage, RykvPage, MiPage, KyouPage, MkflPage, PlaingPage, SaihatePage

**共有ページ（認証不要）:** SharedPage (`/shared_page`), SharedMiPage (`/shared_mi`)

## 2. 各画面の役割と遷移条件

### ルートページ一覧（12ルート）

| パス | ページ | 認証要否 | 役割 |
|-----|-------|---------|------|
| `/` | LoginPage | 不要 | ログイン画面 |
| `/kftl` | KFTLPage | 要 | KFTL テキストベース記録 |
| `/mi` | MiPage | 要 | タスク管理（ボード形式） |
| `/rykv` | RykvPage | 要 | ライフログ閲覧・検索・編集 |
| `/kyou` | KyouPage | 要 | Kyou 記録一覧 |
| `/mkfl` | MkflPage | 要 | 打刻メモ帳（KFTL入力+TimeIs表示） |
| `/plaing` | PlaingPage | 要 | 稼働中 TimeIs 一覧 |
| `/saihate` | SaihatePage | 要 | 記録特化画面（他画面への遷移なし） |
| `/set_new_password` | SetNewPasswordPage | 不要 | 新パスワード設定 |
| `/regist_first_account` | RegistFirstAccountPage | 不要 | 初回アカウント登録 |
| `/shared_page` | SharedPage | 不要 | 共有 Kyou 閲覧 |
| `/shared_mi` | SharedMiPage | 不要 | 共有タスク閲覧 |

## 3. Rykv 画面のダイアログ遷移

Rykv 画面は最も多くのダイアログを呼び出す中心的な画面。

```mermaid
stateDiagram-v2
    KyouListView --> KyouCtx: 長押し/右クリック
    KyouListView --> TagCtx: タグ長押し
    KyouListView --> TextCtx: テキスト長押し

    KyouCtx --> EditKmemo: 編集選択(Kmemo)
    KyouCtx --> EditKC: 編集選択(KC)
    KyouCtx --> EditURLog: 編集選択(URLog)
    KyouCtx --> EditMi: 編集選択(Mi)
    KyouCtx --> EditNlog: 編集選択(Nlog)
    KyouCtx --> EditTimeIs: 編集選択(TimeIs)
    KyouCtx --> EditLantana: 編集選択(Lantana)
    KyouCtx --> EditIDFKyou: 編集選択(IDFKyou)
    KyouCtx --> EditReKyou: 編集選択(ReKyou)
    KyouCtx --> DeleteKyou: 削除選択
    KyouCtx --> ConfirmReKyou: リポスト選択
    KyouCtx --> KyouHistory: 履歴選択
    KyouCtx --> AddTag: タグ追加選択
    KyouCtx --> AddText: テキスト追加選択

    TagCtx --> EditTag: 編集選択
    TagCtx --> DeleteTag: 削除選択
    TagCtx --> TagHistory: 履歴選択

    TextCtx --> EditText: 編集選択
    TextCtx --> DeleteText: 削除選択
    TextCtx --> TextHistory: 履歴選択
```

**コンテキストメニュー:** KyouCtx（Kyou用）、TagCtx（タグ用）、TextCtx（テキスト用）

**編集ダイアログ:** データ型ごとに EditKmemo, EditKC, EditURLog, EditMi, EditNlog, EditTimeIs, EditLantana, EditIDFKyou, EditReKyou

**メタデータダイアログ:** AddTag, EditTag, DeleteTag, AddText, EditText, DeleteText

**履歴ダイアログ:** KyouHistory, TagHistory, TextHistory

## 4. Mi 画面のダイアログ遷移

```mermaid
stateDiagram-v2
    BoardView --> AddMiDialog: 「+」ボタン
    BoardView --> MiContextMenu: Mi長押し/右クリック
    BoardView --> NewBoardDialog: 新規ボード作成

    MiContextMenu --> EditMiDialog: 編集選択
    MiContextMenu --> AddTagDialog: タグ追加選択
    MiContextMenu --> AddTextDialog: テキスト追加選択

    BoardView --> ShareTaskListDialog: 共有ボタン
    ShareTaskListDialog --> ShareTaskListLinkDialog: リンク表示
    ShareTaskListDialog --> DeleteShareTaskList: 削除選択
```

**Mi操作ダイアログ:** AddMiDialog, EditMiDialog, NewBoardDialog

**共有機能:** ShareTaskListDialog → ShareTaskListLinkDialog, DeleteShareTaskList

## 5. 設定画面のダイアログ遷移

```mermaid
stateDiagram-v2
    AppConfig --> AddTagStructElement: タグ構造追加
    AppConfig --> EditTagStructElement: タグ構造編集
    AppConfig --> AddRepStructElement: Rep構造追加
    AppConfig --> EditRepStructElement: Rep構造編集
    AppConfig --> AddKFTLTemplateElement: テンプレート追加
    AppConfig --> EditKFTLTemplateElement: テンプレート編集
    AppConfig --> AddDeviceElement: Device追加
    AppConfig --> EditDeviceElement: Device編集
    AppConfig --> AddRepTypeElement: RepType追加
    AppConfig --> EditRepTypeElement: RepType編集
    AppConfig --> AddNewFolder: フォルダ追加
    AppConfig --> EditFolder: フォルダ編集
```

**アプリケーション設定（AppConfig）:** TagStruct, RepStruct, KFTLTemplate, DeviceStruct, RepTypeStruct の各構造を編集

```mermaid
stateDiagram-v2
    ServerConfig --> CreateAccount: アカウント作成
    ServerConfig --> ConfirmResetPassword: パスワードリセット確認
    ServerConfig --> ShowPasswordResetLink: リセットリンク表示
    ServerConfig --> ConfirmGenerateTLS: TLSファイル生成確認
    ServerConfig --> AddRep: Rep追加
    ServerConfig --> DeleteRep: Rep削除確認
    ServerConfig --> AllocateRep: Rep割当管理
```

**サーバ設定（ServerConfig）:** アカウント管理、パスワードリセット、TLS生成、Rep管理

## 6. ファイルアップロードのダイアログ遷移

```mermaid
stateDiagram-v2
    UploadFileDialog --> SelectTargetRep: アップロード先指定
    UploadFileDialog --> SelectGPSTargetRep: GPSログアップロード先指定
    UploadFileDialog --> DecideRelatedTime: 関連日時設定
    UploadFileDialog --> EditIDFKyou: ファイル情報編集
```
