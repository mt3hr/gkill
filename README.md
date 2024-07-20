# ライフログアプリケーション「gkill」
gkillはライフログアプリケーションです。  

## 開発中・・・ （ [要件書・設計書](https://github.com/mt3hr/gkill/tree/main/documents) ）
【開発フェーズ】  
100% 01.計画準備  
060% 02.全体設計  
000% 03.基盤部分実装 データ移送  
000% 04.FB・レトロスペクティブ  
000% 05.rykv画面・サーバ実装  
000% 06.dnote画面・サーバ実装  
000% 07.FB・レトロスペクティブ  
000% 08.mkfl画面・サーバ実装  
000% 09.FB・レトロスペクティブ  
000% 10.mi画面・サーバ実装  
000% 11.timeis画面・サーバ実装  
000% 12.nlog画面・サーバ実装  
000% 13.FB・レトロスペクティブ  
000% 14.lantanaダイアログ実装 urlogサーバ実装  

## ダウンロード
[gkillダウンロード](https://github.com/mt3hr/gkill/releases/latest)（準備中・・・）  

## 実行
「gkill.exe」または「gkill_server.exe」をダブルクリック  
（gkill_server.exeの場合は起動後「[http://localhost:1111](http://localhost:1111)」にアクセス）  

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