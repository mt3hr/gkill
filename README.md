# ライフログアプリケーション「gkill」
gkillはライフログアプリケーションです。  

## 開発中・・・ （ [要件書・設計書](https://github.com/mt3hr/gkill/tree/main/documents) ）
【開発フェーズ】  
100% 240718対応完了 01.計画準備  
065% 240809完了目標 02.全体設計  ←対応中
000% 240816完了目標 03.実装
000% 240823完了目標 04.テスト
000% 240826完了目標 05.テストFB
000% 240829完了目標 06.整備
000% 240830完了目標 07.運用準備
000% 240831完了目標 08.リリース

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