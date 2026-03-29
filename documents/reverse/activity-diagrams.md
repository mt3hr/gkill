# gkill アクティビティ図

コードの実装から抽出した主要処理フローのアクティビティ図。

## 1. KFTL テキストパース処理フロー

```mermaid
flowchart TD
    Start([KFTLテキスト受信]) --> Init[kftlFactory初期化<br>トランザクションID生成<br>baseTime = 現在時刻]
    Init --> Split[テキストを改行で行分割]
    Split --> LoopStart{次の行がある?}

    LoopStart -->|Yes| CheckSave{保存文字「！」?<br>かつ最初の行でない?}
    CheckSave -->|Yes| LoopEnd[行ループ終了]
    CheckSave -->|No| CheckPrefix{行頭プレフィックス判定}

    LoopStart -->|No| LoopEnd

    CheckPrefix -->|「。」| TagLine[Tag行として処理]
    CheckPrefix -->|「ーー」| TextLine[Text行として処理]
    CheckPrefix -->|「？」| RelTimeLine[関連時刻行として処理]
    CheckPrefix -->|「、」| SplitLine[区切り行<br>次ステートメントへ]
    CheckPrefix -->|「、、」| SplitNextSec[区切り行<br>AddSecond++]
    CheckPrefix -->|「ーか」| KCLine[KC開始行]
    CheckPrefix -->|「ーみ」| MiLine[Mi開始行]
    CheckPrefix -->|「ーら」| LantanaLine[Lantana開始行]
    CheckPrefix -->|「ーん」| NlogLine[Nlog開始行]
    CheckPrefix -->|「ーた」| TimeIsStartLine[TimeIs開始行]
    CheckPrefix -->|「ーえ」| TimeIsEndLine[TimeIs終了行]
    CheckPrefix -->|「ーいえ」| TimeIsEndIfExistLine[TimeIs終了行<br>存在時のみ]
    CheckPrefix -->|「ーたえ」| TimeIsEndByTagLine[TimeIs終了行<br>タグ指定]
    CheckPrefix -->|「ーいたえ」| TimeIsEndByTagIfExistLine[TimeIs終了行<br>タグ指定・存在時のみ]
    CheckPrefix -->|「ーち」| TimeIsLine[TimeIs行]
    CheckPrefix -->|「ーう」| URLogLine[URLog開始行]
    CheckPrefix -->|その他| KmemoLine[Kmemo行<br>デフォルト]

    TagLine --> ApplyToMap[line.ApplyThisLineToRequestMap<br>リクエストマップに蓄積]
    TextLine --> ApplyToMap
    RelTimeLine --> ApplyToMap
    SplitLine --> ApplyToMap
    SplitNextSec --> ApplyToMap
    KCLine --> ApplyToMap
    MiLine --> ApplyToMap
    LantanaLine --> ApplyToMap
    NlogLine --> ApplyToMap
    TimeIsStartLine --> ApplyToMap
    TimeIsEndLine --> ApplyToMap
    TimeIsEndIfExistLine --> ApplyToMap
    TimeIsEndByTagLine --> ApplyToMap
    TimeIsEndByTagIfExistLine --> ApplyToMap
    TimeIsLine --> ApplyToMap
    URLogLine --> ApplyToMap
    KmemoLine --> ApplyToMap

    ApplyToMap --> LoopStart

    LoopEnd --> ExecStart{次の未実行<br>リクエストがある?}
    ExecStart -->|Yes| ExecReq[リクエストの DoRequest 実行<br>Repository へ保存]
    ExecReq --> ExecCheck{エラー発生?}
    ExecCheck -->|Yes| Error([エラー返却])
    ExecCheck -->|No| ExecStart
    ExecStart -->|No| Success([成功返却])
```

## 2. Kyou 検索フィルタリングフロー

```mermaid
flowchart TD
    Start([FindKyous呼び出し]) --> GetRepos[全リポジトリ取得<br>KmemoReps, KCReps, LantanaReps,<br>MiReps, NlogReps, URLogReps,<br>TimeIsReps, IDFKyouReps,<br>ReKyouReps, GitCommitLogReps]

    GetRepos --> FetchAll[各リポジトリから<br>Kyou候補を取得]
    FetchAll --> GetCache[LatestDataRepositoryAddress<br>キャッシュ取得]
    GetCache --> FilterLatest[OnlyLatestDataフィルタ<br>同一IDで最新のみ残す]

    FilterLatest --> FilterKeyword{キーワード<br>フィルタ?}
    FilterKeyword -->|Yes| ApplyKeyword[タイトル・内容に<br>キーワード含むかチェック]
    FilterKeyword -->|No| FilterTime

    ApplyKeyword --> FilterTime{日時範囲<br>フィルタ?}
    FilterTime -->|Yes| ApplyTime[RELATED_TIMEが<br>指定範囲内かチェック]
    FilterTime -->|No| FilterTag

    ApplyTime --> FilterTag{タグ<br>フィルタ?}
    FilterTag -->|Yes| ApplyTag[指定タグが<br>付与されているかチェック]
    FilterTag -->|No| FilterDataType

    ApplyTag --> FilterDataType{データ型<br>フィルタ?}
    FilterDataType -->|Yes| ApplyDataType[指定データ型<br>のみ残す]
    FilterDataType -->|No| FilterDevice

    ApplyDataType --> FilterDevice{デバイス<br>フィルタ?}
    FilterDevice -->|Yes| ApplyDevice[指定デバイスの<br>レコードのみ残す]
    FilterDevice -->|No| FilterDeleted

    ApplyDevice --> FilterDeleted[IS_DELETED=TRUE<br>のレコードを除外]
    FilterDeleted --> Sort[RELATED_TIME<br>で降順ソート]
    Sort --> Return([フィルタ結果返却])
```

## 3. Repository 4層のデータ取得フロー

```mermaid
flowchart TD
    Start([データ取得要求]) --> CachedCheck{キャッシュ層<br>にデータあり?}

    CachedCheck -->|Hit| ReturnCached[キャッシュから返却]
    CachedCheck -->|Miss| SQLite3[SQLite3実装層に問い合わせ]

    SQLite3 --> Query[SELECT * FROM テーブル<br>WHERE ID = ?<br>ORDER BY UPDATE_TIME DESC]
    Query --> LatestCheck{OnlyLatestData?}

    LatestCheck -->|Yes| Latest[最新1件のみ取得<br>LIMIT 1]
    LatestCheck -->|No| AllVersions[全バージョン取得]

    Latest --> DeleteCheck{IS_DELETED = TRUE?}
    AllVersions --> ReturnAll([全バージョン返却])

    DeleteCheck -->|Yes| ReturnNil([nil返却<br>論理削除済み])
    DeleteCheck -->|No| UpdateCache[キャッシュ更新]
    UpdateCache --> ReturnData([データ返却])

    ReturnCached --> End([完了])
    ReturnNil --> End
    ReturnData --> End
    ReturnAll --> End
```

## 4. ファイルアップロード処理フロー

```mermaid
flowchart TD
    Start([ファイルアップロード要求]) --> Auth[セッション認証]
    Auth --> GetRepos[リポジトリ取得]
    GetRepos --> CheckQuota{月間アップロード<br>容量チェック}

    CheckQuota -->|超過| QuotaError([容量超過エラー])
    CheckQuota -->|OK| LoopFiles{次のファイルがある?}

    LoopFiles -->|Yes| CheckConflict{同名ファイル<br>が存在する?}
    CheckConflict -->|No| SaveFile[ファイルをディスクに保存]
    CheckConflict -->|Yes| ConflictBehavior{競合動作}

    ConflictBehavior -->|上書き| SaveFile
    ConflictBehavior -->|スキップ| LoopFiles
    ConflictBehavior -->|リネーム| Rename[ファイル名をリネーム]
    Rename --> SaveFile

    SaveFile --> AddIDF[IDFKyouリポジトリに<br>ファイル情報をINSERT]
    AddIDF --> AddHistory[FileUploadHistoryに<br>履歴を記録]
    AddHistory --> LoopFiles

    LoopFiles -->|No| Success([アップロード完了])
```

## 5. DNote 集計処理フロー

```mermaid
flowchart TD
    Start([DNote集計要求]) --> LoadKyous[Kyouデータロード<br>KyouLoader]

    LoadKyous --> ApplyPredicate{Predicate<br>フィルタ適用}
    ApplyPredicate --> CheckPredType{述語タイプ}

    CheckPredType -->|AND| ApplyAND[全条件をAND結合]
    CheckPredType -->|OR| ApplyOR[いずれかの条件をOR結合]
    CheckPredType -->|NOT| ApplyNOT[条件を反転]
    CheckPredType -->|単一| ApplySingle[個別条件チェック<br>キーワード/数値範囲/日時/タグ等]

    ApplyAND --> Filtered[フィルタ済みデータ]
    ApplyOR --> Filtered
    ApplyNOT --> Filtered
    ApplySingle --> Filtered

    Filtered --> GroupBy[KeyGetterでグルーピング<br>日付/曜日/週/月/タグ/データ型等]

    GroupBy --> Aggregate[AggregateTargetで集計<br>平均/合計/最大/最小/カウント]

    Aggregate --> TopBottomFilter{Top/Bottom<br>フィルタ?}
    TopBottomFilter -->|Yes| ApplyFilter[上位/下位N件に絞り込み]
    TopBottomFilter -->|No| Return
    ApplyFilter --> Return([集計結果返却])
```

## 6. ZIP内容閲覧処理フロー

```mermaid
flowchart TD
    Start([ZIP内容閲覧リクエスト]) --> Auth[セッション認証]
    Auth --> GetIDFKyou[IDFKyou取得<br>ファイルパス特定]
    GetIDFKyou --> CalcHash[ZIPファイルのSHA1ハッシュ計算]
    CalcHash --> CacheCheck{zip_cache/{rep_name}/{sha1}/<br>が存在する?}

    CacheCheck -->|Yes| BuildEntries[ZipEntryリスト生成<br>キャッシュから]
    CacheCheck -->|No| ExtractToTemp[一時ディレクトリに展開開始]

    ExtractToTemp --> LoopEntries{次のZIPエントリがある?}
    LoopEntries -->|Yes| TraversalCheck{パストラバーサル<br>チェック}

    TraversalCheck -->|../含む| SkipEntry[エントリをスキップ]
    SkipEntry --> LoopEntries
    TraversalCheck -->|OK| SymlinkCheck{シンボリックリンク?}

    SymlinkCheck -->|Yes| SkipEntry
    SymlinkCheck -->|No| DecodeFilename[Shift_JISファイル名<br>デコード（必要な場合）]

    DecodeFilename --> WriteFile[ファイルを一時ディレクトリに書き込み]
    WriteFile --> LoopEntries

    LoopEntries -->|No| AtomicRename[一時ディレクトリ→<br>zip_cache/{rep_name}/{sha1}/ にリネーム<br>（アトミック展開）]
    AtomicRename --> BuildEntries

    BuildEntries --> ReturnEntries([ZipEntryリスト返却<br>MSG000080])
```

## 7. ログイン認証フロー

```mermaid
flowchart TD
    Start([ログインリクエスト受信]) --> DecodeReq[JSONリクエストデコード]
    DecodeReq --> RateLimit{レート制限チェック<br>IP単位15分間10回}
    RateLimit -->|超過| ErrorRate([ERR000374<br>レート制限エラー])
    RateLimit -->|OK| GetAccount[AccountDAO.GetAccount<br>user_idで検索]

    GetAccount --> AccountExists{アカウント存在?}
    AccountExists -->|No| Error401([認証エラー])
    AccountExists -->|Yes| IsEnabled{アカウント有効?<br>IsEnable}

    IsEnabled -->|No| Error403([アカウント無効エラー])
    IsEnabled -->|Yes| ResetPending{パスワード<br>リセット中?}

    ResetPending -->|Yes| Error403b([リセット中エラー])
    ResetPending -->|No| PasswordMatch{パスワード<br>SHA256一致?}

    PasswordMatch -->|No| Error401b([認証エラー])
    PasswordMatch -->|Yes| CheckLocal{ローカルアクセス?<br>localhost/127.0.0.1/::1}

    CheckLocal --> CreateSession[LoginSession作成<br>UUID生成<br>有効期限30日]
    CreateSession --> SaveSession[LoginSessionDAO<br>.AddLoginSession]
    SaveSession --> CheckBookmarklet{URLogブックマークレット<br>セッション存在?}

    CheckBookmarklet -->|No| CreateBookmarklet[ブックマークレット用<br>セッション作成]
    CheckBookmarklet -->|Yes| ReturnSession
    CreateBookmarklet --> ReturnSession([session_id返却])
```

## 8. データ更新（Append-Only）フロー

```mermaid
flowchart TD
    Start([データ更新リクエスト]) --> Auth[セッション認証]
    Auth --> GetRepos[リポジトリ取得]
    GetRepos --> ExistCheck{既存データ<br>存在チェック}

    ExistCheck -->|存在しない| Error404([データ未存在エラー])
    ExistCheck -->|存在する| CheckTxID{トランザクションID<br>あり?}

    CheckTxID -->|あり| UseTempRep[一時リポジトリに保存]
    CheckTxID -->|なし| UseMainRep[メインリポジトリに保存]

    UseTempRep --> InsertTemp[TempRepository<br>.AddXxxInfo<br>一時テーブルにINSERT]
    UseMainRep --> InsertMain[WriteXxxRep<br>.AddXxxInfo<br>メインテーブルにINSERT]

    InsertTemp --> ResponseCheck{レスポンス要求?<br>want_response_kyou}
    InsertMain --> UpdateCache{キャッシュ更新<br>必要?}

    UpdateCache -->|Yes| DoCache[キャッシュ更新<br>LatestDataRepositoryAddress更新]
    UpdateCache -->|No| ResponseCheck

    DoCache --> ResponseCheck
    ResponseCheck -->|Yes| FetchResult[更新後データ取得<br>GetXxx + GetKyou]
    ResponseCheck -->|No| Return([成功レスポンス])
    FetchResult --> Return
```
