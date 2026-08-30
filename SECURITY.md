# セキュリティポリシー / Security Policy

## 対象バージョン / Supported Versions

セキュリティ修正は最新リリースに対してのみ提供します。
[リリースページ](https://github.com/mt3hr/gkill/releases/latest) の最新版をご利用ください。

Security fixes are provided for the latest release only.

## 脆弱性の報告 / Reporting a Vulnerability

**公開の Issue に脆弱性の詳細を書かないでください。**
gkill はライフログを扱うため、脆弱性情報の公開は利用者の記録の露出に直結します。

報告は GitHub の **Private Vulnerability Reporting** からお願いします:
リポジトリの [Security タブ → Report a vulnerability](https://github.com/mt3hr/gkill/security/advisories/new)

Please do NOT open a public issue for security vulnerabilities.
Use GitHub's Private Vulnerability Reporting (Security tab → Report a vulnerability).

報告に含めてほしいもの:

- 影響を受けるバージョン・コンポーネント（Go サーバ / Web クライアント / MCP / Android / Wear OS）
- 再現手順（可能なら最小構成で）
- 想定される影響（情報漏えい・改ざん・サービス停止など）

## 対応の目安 / Response Targets

個人開発のプロジェクトのため、以下は目標であり保証ではありません。

- 受領確認: 7日以内
- 初期評価（影響と深刻度の見立て）: 14日以内
- 修正の公開: 深刻度に応じて順次。修正リリースまで詳細の公開は控えてください
  （coordinated disclosure）

## 範囲外 / Out of Scope

- 利用者が自分の環境で TLS を無効化した構成（`disable_tls` はローカル利用向けの明示オプション）
- 物理アクセスや端末ロック解除を前提とする攻撃
- 依存パッケージ自体の脆弱性のうち、gkill から到達不能なもの（報告自体は歓迎します）
