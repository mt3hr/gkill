/**
 * 一覧の行でタグが折り返さないようになっているかを走査する。
 *
 * KyouListViewの行は高さが固定でoverflow:hiddenなので、タグが2行になるとその分だけ
 * 本文(Miのチェックボックス・タイトル・板名など)が下へ押し出されて切り落とされ、
 * 内容が読めなくなる。抑止は
 *   1. kyou-view.vue がタグ群を .kyou_attached_tags で包む (折り返しの抑止はコンテナに掛かる)
 *   2. kyou-list-view.vue が :deep() でそこに nowrap + ellipsis を当てる (行だけに効かせる)
 * の2つが揃って初めて効く。
 *
 * どちらが欠けても型でもレンダリングでも落ちず、「タグが多い記録の内容が見えない」という
 * 形でしか気づけないので機械検査する。詳細ペインとKyouダイアログは従来どおり折り返すのが
 * 正しいので、CSSは kyou-list-view.vue の scoped の中から動かさないこと。
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

/** タグ群を包む要素のクラス。kyou-view.vue が付け、kyou-list-view.vue が :deep() で掴む */
const TAGS_CONTAINER_CLASS = 'kyou_attached_tags'

const NOWRAP_DECLARATIONS = [
    'white-space: nowrap',
    'overflow: hidden',
    'text-overflow: ellipsis',
]

function read_source(repo_path: string): string {
    return readFileSync(join(repo_root, repo_path), 'utf8')
}

/**
 * `selector` のルールの中身を取り出す。
 * このリポジトリのCSSは1宣言1行なので、`{` から `}` までを素朴に切れば足りる。
 */
function extract_rule_body(source: string, selector: string): string | null {
    const start = source.indexOf(selector)
    if (start === -1) {
        return null
    }
    const open = source.indexOf('{', start)
    const close = source.indexOf('}', open)
    if (open === -1 || close === -1) {
        return null
    }
    return source.slice(open + 1, close)
}

describe('一覧の行でのタグの折り返し抑止', () => {
    it('kyou-view.vue がタグ群を容れ物で包んでいる', () => {
        const source = read_source('src/client/pages/views/kyou-view.vue')
        const container = `<div class="${TAGS_CONTAINER_CLASS}">`
        const container_at = source.indexOf(container)
        expect(container_at, `${container} が無い。折り返しの抑止はコンテナに掛ける必要がある`)
            .toBeGreaterThan(-1)

        // 包んでいることを確かめる。容れ物の直後の閉じ</div>までにAttachedTagのv-forがあること
        const close_at = source.indexOf('</div>', container_at)
        const wrapped = source.slice(container_at, close_at)
        expect(wrapped, 'AttachedTag の v-for が容れ物の外に出ている')
            .toContain('<AttachedTag v-for=')
    })

    it('kyou-list-view.vue が行のタグに nowrap + ellipsis を当てている', () => {
        const source = read_source('src/client/pages/views/kyou-list-view.vue')
        const body = extract_rule_body(source, `:deep(.${TAGS_CONTAINER_CLASS})`)
        expect(body, `:deep(.${TAGS_CONTAINER_CLASS}) のルールが無い`).not.toBeNull()
        for (const declaration of NOWRAP_DECLARATIONS) {
            expect(body, `${declaration} が無いと1行ellipsisにならない`).toContain(declaration)
        }
    })

    it('容れ物のスタイルは一覧側にしかない（詳細ペイン・ダイアログでは折り返す）', () => {
        const source = read_source('src/client/pages/views/kyou-view.vue')
        const style_at = source.indexOf('<style')
        expect(style_at, 'kyou-view.vue に style ブロックが無い').toBeGreaterThan(-1)
        expect(
            source.slice(style_at),
            `kyou-view.vue で .${TAGS_CONTAINER_CLASS} を飾ると詳細ペインとKyouダイアログにも効いてしまう`,
        ).not.toContain(TAGS_CONTAINER_CLASS)
    })
})
