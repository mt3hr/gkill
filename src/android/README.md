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
        │   ├── assets/
        │   │   └── gkill_server          # gkill_server バイナリ（~45MB）
        │   ├── ic_launcher-playstore.png  # Play Store 用アイコン
        │   ├── java/.../MainActivity.kt   # メインアクティビティ
        │   └── res/                       # Android リソース
        ├── androidTest/                   # インストゥルメンテッドテスト
        └── test/                          # ユニットテスト
```

## ソースコード

### `MainActivity.kt`

パッケージ: `com.gkill_android.mobile_app.src.gkill.mt3hr.gkill`

唯一のソースファイル。以下の処理を行う:
1. assets から gkill_server バイナリを内部ストレージにコピー
2. gkill_server プロセスを起動
3. WebView で `http://localhost:9999` を表示

## リソース構造

### レイアウト

| ファイル | 説明 |
|---------|------|
| `layout/activity_main.xml` | メインレイアウト（WebView） |
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
- gkill_server バイナリを `app/src/main/assets/gkill_server` に配置

## 開発ガイドライン

### gkill_server バイナリの更新

1. Go バックエンドをクロスコンパイル（`GOOS=linux GOARCH=arm64`）
2. 生成されたバイナリを `app/src/main/assets/gkill_server` に配置
3. APK をリビルド

### パッケージ名

`com.gkill_android.mobile_app.src.gkill.mt3hr.gkill`

Wear OS プロジェクトと同一のパッケージ名を使用（Wearable Data Layer 通信のため）。
