'use strict'

/**
 * 「HTML で出力」したファイル用の `@/router` の差し替え（vite.export.config.ts の alias）。
 *
 * **本体の `@/router` を出力ビルドへ持ち込んではいけない。** 理由が2つある。
 *
 * 1つめ（これが本題）。`@/router` は **module 評価時に**
 * `createRouter({ history: createWebHistory(import.meta.env.BASE_URL) })` を走らせる。
 * `createWebHistory` はその場で現在地を正規化するため、`history.replaceState` を
 * **URL 付きで**呼ぶ。`file://` で開いた出力 HTML で実測したところ、渡された URL は
 * `/./C:/Users/.../gkill_export.html` で、結果ドキュメントの URL が
 * `file:///C:/C:/Users/.../gkill_export.html` に化けた
 * （file URL の解決では、先頭のパス要素がドライブ文字でないとき
 * ベースのドライブ文字が前に付く）。そこへ遷移が走って、ブラウザの
 * 「ファイルにアクセスできませんでした」になる。
 * **画面が出る前に起きるので、出力 HTML が丸ごと開けない。**
 *
 * 2つめ。`@/router` は全ページのコンポーネントを import している。
 * 差し替えないと、1枚しか画面が無い出力 HTML のガワに画面が全部載る。
 *
 * 出力 HTML にはルートが1つも無いので、何もしない版を渡す。
 * 呼び出し側（`use-rykv-view.ts` の `navigate_to_page` など）は出力モードでは
 * 導線が消えていて到達しないが、**到達しても例外にならない**ようにしておく。
 */
const export_router_stub = {
    replace(_to: string): Promise<void> {
        return Promise.resolve()
    },
    push(_to: string): Promise<void> {
        return Promise.resolve()
    },
    go(_delta: number): void {
        // 何もしない
    },
    back(): void {
        // 何もしない
    },
    forward(): void {
        // 何もしない
    },
}

export default export_router_stub
