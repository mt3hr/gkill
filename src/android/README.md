# android - Android APK ラッパー

## 概要

gkill_server バイナリを Android に同梱し、WebView でアクセスする APK ラッパー。
内蔵の gkill_server を起動し、そのローカルサーバを WebView で表示することで、
Android デバイス上でスタンドアロンのライフログアプリとして動作する。

## ディレクトリ構造

```
android/
├── build.gradle.kts           # ルート Gradle 設定
├── settings.gradle.kts        # Gradle 設定
├── gradle.properties          # Gradle プロパティ
├── local.properties           # ローカル SDK パス（Git 管理外推奨）
├── gradlew / gradlew.bat      # Gradle ラッパー
├── gradle/
│   └── wrapper/               # Gradle Wrapper JAR
└── app/
    ├── build.gradle.kts       # アプリモジュール Gradle 設定
    ├── proguard-rules.pro     # ProGuard ルール
    └── src/
        ├── main/
        │   ├── AndroidManifest.xml
        │   ├── jniLibs/arm64-v8a/
        │   │   └── libgkill_server.so     # gkill_server バイナリ（~58MB）
        │   ├── ic_launcher-playstore.png  # Play Store 用アイコン
        │   ├── java/.../MainActivity.kt   # メインアクティビティ
        │   └── res/                       # Android リソース
        ├── androidTest/                   # インストゥルメンテッドテスト
        └── test/                          # ユニットテスト
```

## ソースコード

### `MainActivity.kt`

パッケージ: `com.gkill_android.mobile_app.src.gkill.mt3hr.gkill`

唯一のソースファイル。起動から画面表示までの流れは次のとおり（守るべき約束と経緯は
[gkill-mobile スキル](../../.claude/skills/gkill-mobile/SKILL.md) と
[ADR-1104](../../documents/adr/1104-android-server-follows-server-config.md)）:
1. 共有ストレージへの権限を確認する（Android 11 以上は「すべてのファイルへのアクセス」、10 以下は
   `WRITE_EXTERNAL_STORAGE`）。無ければ起動待ち画面を説明文と「許可する」ボタンに切り替える。
   権限の要求画面を自動で出すのは Activity ごとに 1 回だけで、以後は画面のボタンから出す
   （onResume のたびに出すと設定画面から抜けられない）。**許可されるまでサーバを起動しない**
   （データ置き場が共有ストレージなので、権限なしで起動すると置き場を作れずに落ちる）
2. アプリ専用領域にデータを置いていた版から更新した端末では、`copyAppPrivateHomeIfNeeded` が
   `/sdcard/gkill` が無いか空のときだけ専用領域の中身を複製する（一時ディレクトリへ複製してから改名。
   `/sdcard/gkill` に中身があればそちらを正とし、複製元はどの場合も消さない）。
   **複製に失敗したらサーバを起動しない**（起動すると空の置き場が作られ、次回から「中身あり」と
   見なされて専用領域のデータへ戻れなくなる）
3. `nativeLibraryDir/libgkill_server.so` を `--gkill_home_dir /sdcard/gkill --log debug` だけで起動する
   （targetSdk 29 以降、アプリのデータディレクトリ配下は W^X 制約で実行できないため、jniLibs 経由で
   配置している）。`--address` / `--disable_tls` は渡さず、待受アドレスと TLS はサーバ設定
   （ServerConfig）に従う。前回拾った URL のポートが既に応答していれば、その既存サーバを使い回して
   起動処理を省く
4. サーバが標準出力へ出す `Access your record space at : ` 行から WebView で開く URL を取る
   （Kotlin 側にポートもスキームも持たない）。ポートが応答するまで待ってから表示する
5. サーバ設定を保存するとサーバが内部で作り直され、同じ行がもう一度出る。オリジン
   （スキーム・ホスト・ポート）が変わったときだけ開き直す
6. TLS 有効時の自己署名証明書は `onReceivedSslError` がループバック（localhost / 127.0.0.1 / ::1）に
   限って通す。それ以外のホストと、ホストを取り出せない URL は止める
7. ステータスバーは gkill のテーマ色（`colors.xml` の `gkill_indigo`。Web の primary と同じ値）で塗る。
   API 34 以下はテーマの `android:statusBarColor`、API 35 以上（edge-to-edge が強制され
   `statusBarColor` が無視される）は `activity_main.xml` のステータスバーの裏の帯を insets の高さに合わせる

守るテストは `test/` の `MainActivityUnitTest.kt`（起動引数・URL 行の解析・オリジン比較・SSL エラーの判定・
権限ゲート・専用領域からの複製・ステータスバーの色・マニフェストの権限宣言・レイアウトの id）。

## リソース構造

### レイアウト

| ファイル | 説明 |
|---------|------|
| `layout/activity_main.xml` | メインレイアウト（WebView・起動待ち画面・ステータスバーの裏に敷くテーマ色の帯） |
| `layout-sw600dp/activity_main.xml` | タブレット用レイアウト |

### アイコン

| ディレクトリ | 説明 |
|-------------|------|
| `drawable/` | ベクタードロワブル（ランチャーアイコン前景/背景） |
| `mipmap-*/` | 各解像度のランチャーアイコン（hdpi, mdpi, xhdpi, xxhdpi, xxxhdpi） |
| `mipmap-anydpi-v26/` | Adaptive Icon 定義 |

### 値リソース

| ファイル | 説明 |
|---------|------|
| `values/colors.xml` | カラー定義 |
| `values/strings.xml` | 文字列リソース |
| `values/themes.xml` | テーマ定義（ライトモード） |
| `values-night/themes.xml` | テーマ定義（ダークモード） |

### XML 設定

| ファイル | 説明 |
|---------|------|
| `xml/backup_rules.xml` | バックアップルール |
| `xml/data_extraction_rules.xml` | データ抽出ルール |
| `xml/network_security_config.xml` | 平文（HTTP）通信の許可範囲。既定は全面禁止で、`localhost` / `127.0.0.1` に限って許可する（ポートは問わない）。同梱サーバの既定は TLS 無効なので、これが無いと WebView が localhost の平文を拒む。TLS 有効時の自己署名証明書は `MainActivity.kt` の `onReceivedSslError` が通す |

### `AndroidManifest.xml`

| 項目 | 説明 |
|------|------|
| `MANAGE_EXTERNAL_STORAGE` | 全ファイルアクセス（Android 11 以上）。Go サーバは SAF の `content://` を扱えず実パスでしか読み書きできないので、データ置き場 `/sdcard/gkill` と共有ストレージ上のファイルリポジトリのために要る。許可されるまでサーバは起動しない |
| `READ_EXTERNAL_STORAGE`（`maxSdkVersion="32"`）/ `WRITE_EXTERNAL_STORAGE`（`maxSdkVersion="29"`） | 旧権限。33 以上は粒度別メディア権限、30 以上は Scoped Storage に置き換わるので上限付き。Android 10 以下では `WRITE_EXTERNAL_STORAGE` の許可でサーバを起動する |
| `requestLegacyExternalStorage="true"` | Android 10 の端末でだけ効く（11 以上では無視される）。10 は Scoped Storage のため、これが無いと `WRITE_EXTERNAL_STORAGE` を許可されても `/sdcard/gkill` へ実パスで書けない |
| `networkSecurityConfig` | 上の `xml/network_security_config.xml` を指す |
| `configChanges` | 回転などで Activity を再生成させない（WebView の履歴を保ち、SQLite 書き込み中のサーバを `kill -9` で巻き込まない） |

## ビルド方法

```bash
cd src/android

# デバッグビルド
./gradlew assembleDebug

# リリースビルド
./gradlew assembleRelease
```

**前提条件:**
- Android SDK
- gkill_server バイナリを `app/src/main/jniLibs/arm64-v8a/libgkill_server.so` に配置

## 開発ガイドライン

### gkill_server バイナリの更新

1. Go バックエンドをクロスコンパイル（`npm run build_android_arm64`。
   `CGO_ENABLED=1 GOOS=android GOARCH=arm64` + NDK clang を使うため、環境変数 `NDK` が必要）
2. 生成されたバイナリを `app/src/main/jniLibs/arm64-v8a/libgkill_server.so` に配置
   （`npm run copy_android_release` が自動で行う。`lib*.so` という名前でないと
   nativeLibraryDir に展開されないので、リネームは必須）
3. APK をリビルド

### パッケージ名

`com.gkill_android.mobile_app.src.gkill.mt3hr.gkill`

Wear OS プロジェクトと同一のパッケージ名を使用（Wearable Data Layer 通信のため）。
