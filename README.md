# ライフログアプリケーション「gkill」
gkillはライフログアプリケーションです。  

## 資料（準備中）
[マニュアル](.) （準備中 2025年注完了目標）  
[要件書・設計書](https://github.com/mt3hr/gkill/tree/main/documents)  
[仕様書](.) （準備中 2025年中完了目標）  

## ダウンロード
[gkillダウンロード](https://github.com/mt3hr/gkill/releases/latest)  

## 実行
「gkill_server.exe」を実行  
その後「[http://localhost:9999](http://localhost:9999)」にアクセス  

## 初回起動  
初回起動後、ユーザIDとパスワード、管理用パスワードを入力してください。
完了したらログインページに遷移します。
ログインして記録を始められます！
（そのうちマニュアル作ります）  

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

100% 25-03-01 完了目標 06.リリース  
[gkillダウンロード](https://github.com/mt3hr/gkill/releases/latest)  

</details>

<details>
<summary>資料整備しんちょく</summary>
【資料整備フェーズ】  

000% A-1 画面遷移仕様図  

000% A-2 ユースケース仕様書  

000% B-1 ER仕様図  

000% C-1 詳細クラス仕様書  

000% C-2 主要プログラム仕様説明書  

000% D-1 サンプルデータ  

000% D-2 ユーザマニュアル  

000% E-1 README  

</details>