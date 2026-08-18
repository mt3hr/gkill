/**
 * rykv / mi / dashboard の初期化まわりの不変条件をソース走査で守る。
 *
 * どれも「戻すと静かに壊れる」たぐいで、型でもレンダリングでも落ちない。
 *
 * 1. init() の起動条件は application_config.is_loaded の watch であること。
 *    以前はサイドバーの @inited を起動条件にしていたが、あれは子クエリビューの
 *    「その節が描けた」の集約でしかない。設定の到着を表していたのは
 *    「immediate の付いていない application_config watch から emit する子がいる」
 *    という偶然で、mi では実質 CalendarQuery 1つが律速していた(しかもその節は
 *    application_config のフィールドを1つも読まない)。そのため節を1つ画面から
 *    外すだけで画面ごとスピナーで固まっていた。
 *
 * 2. onSidebarUpdatedQuery に「初期化が終わるまで捨てる」早期returnを置かないこと。
 *    初期検索の飛行中でもユーザの編集は通すのが正しい(同じ query_id を共有するので
 *    abort_controllers が復元を中断し、search_seqs の世代照合が遅れて届いた
 *    復元結果を捨てる)。戻すと初期検索の間の編集が黙って消える。
 *
 * 3. init() で skip_search_this_tick を立てっぱなしにしないこと。
 *    あれは「1tick分の残響を捨てる」短命フラグで、初期化全体の門番に流用すると
 *    機械的なemitが1つ届いただけで onSidebarUpdatedQuery が消費してしまい、
 *    複数列のとき1列目の完了で抑止が途中で解ける。
 *    抑止は run_with_sidebar_search_suppressed だけを使う。
 *
 * 4. ビューのルートが data-gkill-view-ready を出していること。
 *    E2E の準備完了待ち(crud-helpers.ts の waitForColumnViewReady)がこれを見る。
 *    消すと E2E が黙って素通りし、行が出る前に次の操作へ進んでフレークする。
 */
import { existsSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { describe, expect, it } from 'vitest'

// import.meta.url は vitest の変換後は file スキームにならないので使えない。
// package.json を目印に上へ辿ってリポジトリルートを決める
function find_repo_root(): string {
    let dir = process.cwd()
    for (let i = 0; i < 10; i++) {
        if (existsSync(join(dir, 'package.json')) && existsSync(join(dir, 'src', 'client'))) {
            return dir
        }
        const parent = dirname(dir)
        if (parent === dir) {
            break
        }
        dir = parent
    }
    throw new Error(`リポジトリルートが見つからない: cwd=${process.cwd()}`)
}

const repo_root = find_repo_root()

function read(relative_path: string): string {
    return readFileSync(join(repo_root, relative_path), 'utf8')
}

/** 列ビュー(コピー由来の対称実装。修正は必ず両方へ入れる) */
const COLUMN_VIEW_COMPOSABLES = [
    'src/client/classes/use-rykv-view.ts',
    'src/client/classes/use-mi-view.ts',
]

/** ルートに準備完了信号を出すビュー */
const VIEW_READY_TEMPLATES = [
    'src/client/pages/views/rykv-view.vue',
    'src/client/pages/views/mi-view.vue',
    'src/client/pages/views/dashboard-view.vue',
]

/** 設定取得の失敗を画面へ伝えるページ */
const CONFIG_LOAD_PAGES = [
    'src/client/classes/use-rykv-page.ts',
    'src/client/classes/use-mi-page.ts',
    'src/client/classes/use-dashboard-page.ts',
]

describe('列ビューの初期化トリガ', () => {
    for (const path of COLUMN_VIEW_COMPOSABLES) {
        it(`${path} は application_config.is_loaded で init する`, () => {
            const source = read(path)
            expect(
                /watch\(\s*\(\)\s*=>\s*props\.application_config\.is_loaded/.test(source),
                'init() のトリガが application_config.is_loaded の watch ではない',
            ).toBe(true)
            expect(
                source.includes('onSidebarInited'),
                'onSidebarInited が復活している(サイドバーへの偶然の依存が戻る)',
            ).toBe(false)
        })
    }

    for (const path of COLUMN_VIEW_COMPOSABLES) {
        it(`${path} の onSidebarUpdatedQuery は初期化中でも編集を捨てない`, () => {
            const source = read(path)
            const start = source.indexOf('function onSidebarUpdatedQuery')
            expect(start, 'onSidebarUpdatedQuery が見つからない').toBeGreaterThan(-1)
            const body = source.slice(start, start + 1200)
            expect(
                /if\s*\(\s*!inited\.value\s*\)/.test(body),
                '初期化中の編集を捨てる早期returnが復活している',
            ).toBe(false)
        })
    }

    for (const path of COLUMN_VIEW_COMPOSABLES) {
        it(`${path} の init は skip_search_this_tick を立てっぱなしにしない`, () => {
            const source = read(path)
            const start = source.indexOf('async function init()')
            expect(start, 'init() が見つからない').toBeGreaterThan(-1)
            const end = source.indexOf('async function search(', start)
            expect(end, 'init() の終わりが見つからない').toBeGreaterThan(start)
            const body = source.slice(start, end)
            expect(
                body.includes('skip_search_this_tick.value = true'),
                'init() が抑止フラグを直接立てている(run_with_sidebar_search_suppressed を使うこと)',
            ).toBe(false)
            expect(
                body.includes('run_with_sidebar_search_suppressed'),
                'focused_query の差し替えが抑止で包まれていない',
            ).toBe(true)
        })
    }

    for (const path of COLUMN_VIEW_COMPOSABLES) {
        it(`${path} の init は preserve_scroll で復元する`, () => {
            const source = read(path)
            const start = source.indexOf('async function init()')
            const end = source.indexOf('async function search(', start)
            const body = source.slice(start, end)
            expect(
                /search\(i,\s*saved_querys\[i\],\s*true,\s*false,\s*true\)/.test(body),
                'preserve_scroll=true で呼んでいない(起動のたびにリストの先頭へ飛ぶ)',
            ).toBe(true)
        })
    }
})

describe('E2Eの準備完了信号', () => {
    for (const path of VIEW_READY_TEMPLATES) {
        it(`${path} は data-gkill-view-ready を出す`, () => {
            const source = read(path)
            expect(
                source.includes('data-gkill-view-ready'),
                'E2E の待ち(waitForColumnViewReady)が黙って素通りする',
            ).toBe(true)
            // 真偽値をそのまま bind すると Vue は false のとき属性ごと消すので、
            // 「属性の有無」で判定するセレクタが壊れる
            expect(
                /:data-gkill-view-ready="is_view_ready \? 'true' : 'false'"/.test(source),
                "文字列の三項で書いていない(false のとき属性ごと消える)",
            ).toBe(true)
        })
    }
})

describe('ApplicationConfig 取得の失敗経路', () => {
    for (const path of CONFIG_LOAD_PAGES) {
        it(`${path} は失敗を画面へ伝える`, () => {
            const source = read(path)
            expect(
                source.includes('application_config_load_failed'),
                '設定取得の失敗を画面へ伝えていない(永久スピナーに戻る)',
            ).toBe(true)
            const start = source.indexOf('function load_application_config')
            expect(start, 'load_application_config が見つからない').toBeGreaterThan(-1)
            const body = source.slice(start, start + 2600)
            expect(
                body.includes('.catch('),
                '通信例外を握れていない(unhandled rejection になり画面が固まる)',
            ).toBe(true)
        })
    }
})
