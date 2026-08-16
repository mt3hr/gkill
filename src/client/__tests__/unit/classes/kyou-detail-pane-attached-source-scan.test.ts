/**
 * Kyou詳細ペインが付随データ（タグ / テキスト / 通知）を表示しているかを走査する。
 *
 * `show_attached_tags` / `show_attached_texts` / `show_attached_notifications` は
 * 「描画するか」ではなく **「読み込むか」** のフラグで、`use-kyou-view.ts` の
 * `load_attached_infos()` が `load_attached_texts()` 等を呼ぶかどうかだけを決める。
 * false にすると配列が空のままになり、`kyou-view.vue` の `v-for` は回るものが無いので
 * 何も出ない。型でもレンダリングでも落ちず「出ないだけ」なので気づけない。
 *
 * 実際に mi-view.vue だけが3つとも false のまま取り残されていて、
 * rykv と共有Mi では出るのに Mi の詳細ペインにだけテキストが出なかった。
 * 同じことが起きないように機械検査する。
 *
 * 一覧の行（kyou-list-view.vue）は高さが決まっているので false のままでよい。
 * 検査対象は `class="kyou_detail_view"` を付けた詳細ペインだけ。
 */
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs'
import { dirname, join, relative, sep } from 'node:path'
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
const pages_root = join(repo_root, 'src', 'client', 'pages')

/** 詳細ペインの目印。この class が付いた KyouView だけを検査する */
const DETAIL_PANE_CLASS = 'class="kyou_detail_view"'

/** 詳細ペインが true を渡していなければならない props */
const REQUIRED_ATTACHED_PROPS = [
    'show_attached_tags',
    'show_attached_texts',
    'show_attached_notifications',
]

interface DetailPaneBlock {
    repo_path: string
    start_line: number
    source: string
}

function list_vue_files(dir: string): Array<string> {
    const found = new Array<string>()
    for (const entry of readdirSync(dir)) {
        const path = join(dir, entry)
        if (statSync(path).isDirectory()) {
            found.push(...list_vue_files(path))
            continue
        }
        if (path.endsWith('.vue')) {
            found.push(path)
        }
    }
    return found
}

function to_repo_path(path: string): string {
    return relative(repo_root, path).split(sep).join('/')
}

/**
 * `<KyouView ... />` の要素を1つのまとまりとして切り出す。
 *
 * このリポジトリは属性を1行1つで書き、コンポーネントは必ず自己閉じなので、
 * 行頭の `<KyouView` から `/>` までを集めれば十分。
 * 属性値の中に `=>`（アロー関数）が入るので、閉じ判定に `>` を使ってはいけない。
 */
function extract_detail_pane_blocks(source: string, repo_path: string): Array<DetailPaneBlock> {
    const blocks = new Array<DetailPaneBlock>()
    const lines = source.split(/\r?\n/)
    let start_line = 0
    let collected: Array<string> | null = null
    for (let i = 0; i < lines.length; i++) {
        const line = lines[i]
        const opening = /^\s*<([A-Za-z][A-Za-z0-9]*)/.exec(line)
        if (opening) {
            // 閉じそこねた要素は次の要素が始まった時点で捨てる
            collected = opening[1] === 'KyouView' ? new Array<string>() : null
            start_line = i + 1
        }
        if (!collected) {
            continue
        }
        collected.push(line)
        if (line.includes('/>')) {
            const element = collected.join('\n')
            if (element.includes(DETAIL_PANE_CLASS)) {
                blocks.push({ repo_path: repo_path, start_line: start_line, source: element })
            }
            collected = null
        }
    }
    return blocks
}

function find_violations(blocks: Array<DetailPaneBlock>): Array<string> {
    const violations = new Array<string>()
    for (const block of blocks) {
        for (const prop of REQUIRED_ATTACHED_PROPS) {
            if (!block.source.includes(`:${prop}="true"`)) {
                violations.push(`${block.repo_path}:${block.start_line} <KyouView> :${prop}`)
            }
        }
    }
    return violations
}

function collect_all_detail_panes(): Array<DetailPaneBlock> {
    const blocks = new Array<DetailPaneBlock>()
    for (const path of list_vue_files(pages_root)) {
        blocks.push(...extract_detail_pane_blocks(readFileSync(path, 'utf8'), to_repo_path(path)))
    }
    return blocks
}

describe('Kyou詳細ペインの付随データ表示の走査', () => {
    const detail_panes = collect_all_detail_panes()

    // 走査が「何も見つけられないだけ」で緑になっていないことを確かめる
    it('詳細ペインを見つけられている', () => {
        expect(detail_panes.map(block => block.repo_path).sort()).toEqual([
            'src/client/pages/views/mi-view.vue',
            'src/client/pages/views/rykv-view.vue',
            'src/client/pages/views/shared-mi-view.vue',
        ])
    })

    it('詳細ペインはタグ・テキスト・通知をすべて表示する', () => {
        expect(
            find_violations(detail_panes),
            'show_attached_* は読み込みゲート。false だと配列が空のままで何も出ない',
        ).toEqual([])
    })

    it('検出ロジックが違反を見つけられる（自己検査）', () => {
        const fixture = [
            '<div class="kyou_detail_view dummy">',
            '    <KyouView v-if="focused_kyou"',
            '        :kyou="focused_kyou" class="kyou_detail_view"',
            '        :show_attached_tags="false" :show_attached_texts="false"',
            '        :show_attached_notifications="false"',
            '        v-on="{ ...crudRelayHandlers }" />',
            '</div>',
        ].join('\n')
        const violations = find_violations(extract_detail_pane_blocks(fixture, 'fixture.vue'))
        expect(violations).toHaveLength(REQUIRED_ATTACHED_PROPS.length)
        expect(violations[0]).toContain('fixture.vue:2')
    })

    // 一覧の行まで拾ってしまうと「行にもテキストを出せ」という誤った検査になる
    it('一覧の行は検査対象に入らない', () => {
        const fixture = [
            '<KyouView :kyou="kyou"',
            '    :show_attached_tags="false" :show_attached_texts="false"',
            '    :show_attached_notifications="false" :height="100" />',
        ].join('\n')
        expect(extract_detail_pane_blocks(fixture, 'fixture.vue')).toEqual([])
    })
})
