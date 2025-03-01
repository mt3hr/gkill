# ライフログアプリケーション「gkill」
gkillはライフログアプリケーションです。  

## 資料
[マニュアル](.) （準備中）  
[要件書・設計書](https://github.com/mt3hr/gkill/tree/main/documents)  
[仕様書](.) （準備中）  

## ダウンロード
[gkillダウンロード](https://github.com/mt3hr/gkill/releases/latest)  

## 実行
「gkill.exe」または「gkill_server.exe」をダブルクリック  
（gkill_server.exeの場合は起動後「[http://localhost:9999](http://localhost:9999)」にアクセス）  

<details>
<summary>開発者向け</summary>

### 開発環境

### セットアップ
1. Golang バージョン1.22.4の開発環境を用意する  
2. Cコンパイラを用意する（cgo使用のため）  
3. Node.js バージョン20.15.1の開発環境を用意する  
4. 以下のコマンドを実行する  
```
npm i
```

### ビルド・インストール

アプリケーションインストール  
```
npm run go_mod
npm run install_app
```

サーバインストール  
```
npm run go_mod
npm run install_server
```
</details>

<details>
<summary>かいはつすけじゅーる</summary>
【開発フェーズ】（2025-02-01 リスケ）  
100% 24-07-18 対応完了 01.計画準備  
100% 24-08-15 対応完了 02.全体設計  
100% 25-02-02 対応完了 03.実装  
100% 25-02-16 対応完了 04.全体テスト  
100% 25-02-28 対応完了 05.トライアルテスト フィードバック対応  
100% 25-03-01 完了目標 06.リリース [gkillダウンロード](https://github.com/mt3hr/gkill/releases/latest)  
000% 25-06-01 完了目標 07.資料整備  ←対応中  
</details>
