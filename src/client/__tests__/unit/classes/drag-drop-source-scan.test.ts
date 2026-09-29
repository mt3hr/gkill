/**
 * 並べ替えの D&D（classes/drag-drop-indicator.ts）が前提にしているテンプレートの形を、ソース走査で固定する。
 *
 * 1. `dnote-item-table-view.vue` の列を並べる 2 つの v-for は `:key="listIndex"`（添字）のまま。
 *    `use-dnote-item-list-view.ts` は `const dnd_list_index = props.dnd_list_index` と setup 時に
 *    一度だけ取り込み（非リアクティブ）、ドロップ時にその値を `requested_move_dnote_item` の
 *    移動先の列として emit する。key が添字なら、列を消しても位置 i のコンポーネントは位置 i の
 *    まま使い回されるので、取り込んだ値と実際の位置が一致し続ける。安定 id を key にすると、
 *    消した列より後ろのコンポーネントが古い添字を抱えたまま 1 つ前へ詰められ、その列へ落とした
 *    項目が隣の列へ入る（例外もエラーも出ない）。
 * 2. `foldable-struct.vue` のドラッグを受ける 2 つの tr（項目・フォルダ）は、見出しの table に
 *    `foldable_struct_header` クラスを持ち、`@dragleave` と `@dragend` を配線している。
 *    `use-foldable-struct.ts` の `resolve_drop_position` は `.foldable_struct_header` の矩形で
 *    入る位置を測る（開いたフォルダの tr は子孫の行まで含んだ高さなので、tr 全体で測ると
 *    3 分割の境界が下へずれる）。クラスが無いと `?? el` で tr 全体に落ちて黙ってずれる。
 *    `@dragleave` が無いと行の外へ出ても線が残る。`@dragend` は、掴んだ時点で
 *    `drag-drop-indicator.ts` の `install_cleanup` が window の capture に張る dragend の受け口と
 *    同じ後始末（`end_drag()`）の二重化で、外しても今は見た目が変わらない。後始末の経路を
 *    window の 1 本だけに頼らないために残すので、ここで一緒に固定する。
 *
 * `@dragend` 以外はどれも型でもビルドでも落ちず、実際に触ってはじめて分かるので機械検査する。
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

const DNOTE_TABLE_VIEW = 'src/client/pages/views/dnote-item-table-view.vue'
const DNOTE_LIST_COMPOSABLE = 'src/client/classes/use-dnote-item-list-view.ts'
const FOLDABLE_STRUCT_VIEW = 'src/client/pages/views/foldable-struct.vue'
const FOLDABLE_STRUCT_COMPOSABLE = 'src/client/classes/use-foldable-struct.ts'

function read_repo_file(repo_path: string): string {
    return readFileSync(join(repo_root, repo_path), 'utf8')
}

type OpeningTag = { text: string, index: number, line: number }

function line_of(source: string, index: number): number {
    return source.slice(0, index).split(/\r?\n/).length
}

/**
 * start から始まる開始タグを、属性値の引用符を尊重して `>` まで切り出す。
 * `@drop="(e) => onCellDrop(e, listIndex)"` のように属性値の中に `>` があるので、`[^>]*` では切れない
 */
function read_opening_tag(source: string, start: number): string {
    let in_quote: string | null = null
    for (let i = start; i < source.length; i++) {
        const ch = source[i]
        if (in_quote !== null) {
            if (ch === in_quote) {
                in_quote = null
            }
            continue
        }
        if (ch === '"' || ch === '\'') {
            in_quote = ch
            continue
        }
        if (ch === '>') {
            return source.slice(start, i + 1)
        }
    }
    throw new Error(`${line_of(source, start)} 行目の開始タグが閉じていない`)
}

/** tag の開始タグ（属性が複数行に渡っても 1 つ）を全部取り出す */
function list_opening_tags(source: string, tag: string): Array<OpeningTag> {
    const found = new Array<OpeningTag>()
    const pattern = new RegExp(`<${tag}(?=[\\s>/])`, 'g')
    for (const m of source.matchAll(pattern)) {
        const index = m.index ?? 0
        found.push({ text: read_opening_tag(source, index), index, line: line_of(source, index) })
    }
    return found
}

function template_of(source: string): string {
    const start = source.indexOf('<template>')
    const end = source.lastIndexOf('</template>')
    if (start < 0 || end < 0) {
        throw new Error('<template> が見つからない')
    }
    return source.slice(start, end)
}

// ── dnote-item-table-view: 列の key ──

const COLUMN_V_FOR_PATTERN = /v-for="\((\w+), (\w+)\) in model_value"/

/** 列を並べる td（`v-for="(..., ...) in model_value"`）を取り出す */
function list_column_tds(source: string): Array<OpeningTag> {
    return list_opening_tags(template_of(source), 'td').filter((td) => COLUMN_V_FOR_PATTERN.test(td.text))
}

/** 列の td の key が v-for の添字変数そのものでなければ違反 */
function find_column_key_violations(source: string, repo_path: string): Array<string> {
    const violations = new Array<string>()
    for (const td of list_column_tds(source)) {
        const index_name = COLUMN_V_FOR_PATTERN.exec(td.text)?.[2]
        const key = /:key="([^"]*)"/.exec(td.text)?.[1]
        if (key === undefined) {
            violations.push(`${repo_path}:${td.line} 列の td に :key が無い`)
            continue
        }
        if (key !== index_name) {
            violations.push(`${repo_path}:${td.line} 列の td の :key が添字ではない（:key="${key}"、添字は ${index_name}）`)
        }
    }
    return violations
}

describe('dnote-item-table-view の列の key', () => {
    const source = read_repo_file(DNOTE_TABLE_VIEW)

    it('列を並べる v-for が 2 つ（削除ボタンの行と本体の行）見つかっている', () => {
        expect(list_column_tds(source).map((td) => td.line)).toHaveLength(2)
    })

    it('列の key は添字（listIndex）のまま', () => {
        expect(
            find_column_key_violations(source, DNOTE_TABLE_VIEW),
            'use-dnote-item-list-view が dnd_list_index を非リアクティブに取り込むので、安定 id の key にすると列を消したあとのドロップ先がずれる',
        ).toEqual([])
        for (const td of list_column_tds(source)) {
            expect(td.text, `${DNOTE_TABLE_VIEW}:${td.line}`).toContain(':key="listIndex"')
        }
    })

    // 上の制約の前提。取り込みをリアクティブにしたなら、key の制約（このファイル）ごと見直すこと
    it('前提: use-dnote-item-list-view は dnd_list_index を setup 時に一度だけ取り込んでいる', () => {
        const composable = read_repo_file(DNOTE_LIST_COMPOSABLE)
        expect(composable).toMatch(/^\s*const dnd_list_index = props\.dnd_list_index\s*$/m)
    })

    // 走査が「何も見つけられないだけ」で緑になっていないことを確かめる
    it('検出ロジックが違反を見つけられる（自己検査）', () => {
        const fixture = [
            '<template>',
            '    <tr>',
            '        <td v-for="(list, listIndex) in model_value" :key="list_id_of(list)" class="x"',
            '            @drop="(e) => onCellDrop(e, listIndex)">',
            '        </td>',
            '        <td v-for="(_list, listIndex) in model_value" class="y"></td>',
            '        <td v-for="(list, listIndex) in model_value" :key="listIndex"></td>',
            '    </tr>',
            '</template>',
        ].join('\n')
        expect(list_column_tds(fixture)).toHaveLength(3)
        const violations = find_column_key_violations(fixture, 'fixture.vue')
        expect(violations).toHaveLength(2)
        expect(violations[0]).toContain('fixture.vue:3')
        expect(violations[0]).toContain('list_id_of(list)')
        expect(violations[1]).toContain('fixture.vue:6')
        expect(violations[1]).toContain(':key が無い')
    })
})

// ── foldable-struct: 見出しの矩形と線の後始末 ──

const REQUIRED_DRAG_ROW_ATTRIBUTES = ['@dragleave="dragleave"', '@dragend="dragend"'] as const

/** resolve_drop_position が矩形を測る見出しのクラス名を、composable の querySelector から読む */
function header_class_of(composable_source: string): string {
    const m = /querySelector<HTMLElement>\('\.([A-Za-z0-9_-]+)'\)\s*\?\?\s*el/.exec(composable_source)
    if (!m) {
        throw new Error(`${FOLDABLE_STRUCT_COMPOSABLE} の resolve_drop_position が見出しを querySelector で取る形ではなくなった。このテストを見直すこと`)
    }
    return m[1]
}

/** ドラッグを受ける行（`@dragover` 付きの tr）を取り出す */
function list_drag_rows(source: string): Array<OpeningTag> {
    return list_opening_tags(template_of(source), 'tr').filter((tr) => tr.text.includes('@dragover='))
}

/**
 * ドラッグを受ける各行について、後始末の配線と見出しの table を確かめる。
 * 見出しは「その行の開始から次のドラッグ行（無ければテンプレートの末尾）まで」に
 * `class="<header_class>"` の table が 1 つ、という形で見る
 */
function find_drag_row_violations(source: string, repo_path: string, header_class: string): Array<string> {
    const violations = new Array<string>()
    const template = template_of(source)
    const rows = list_drag_rows(source)
    if (rows.length === 0) {
        violations.push(`${repo_path} に @dragover 付きの tr が 1 つも無い`)
    }
    rows.forEach((row, i) => {
        for (const attribute of REQUIRED_DRAG_ROW_ATTRIBUTES) {
            if (!row.text.includes(attribute)) {
                violations.push(`${repo_path}:${row.line} ドラッグを受ける tr に ${attribute} が無い`)
            }
        }
        const segment = template.slice(row.index, rows[i + 1]?.index ?? template.length)
        const header_tables = list_opening_tags(segment, 'table').filter((table) => table.text.includes(`class="${header_class}"`))
        if (header_tables.length !== 1) {
            violations.push(`${repo_path}:${row.line} ドラッグを受ける tr の中に class="${header_class}" の table が ${header_tables.length} 個（1 個であること）`)
        }
    })
    return violations
}

describe('foldable-struct のドラッグを受ける行', () => {
    const source = read_repo_file(FOLDABLE_STRUCT_VIEW)
    const header_class = header_class_of(read_repo_file(FOLDABLE_STRUCT_COMPOSABLE))

    it('composable が測る見出しのクラスは foldable_struct_header', () => {
        expect(header_class).toBe('foldable_struct_header')
    })

    it('ドラッグを受ける tr が 2 つ（項目とフォルダ）見つかっている', () => {
        expect(list_drag_rows(source).map((tr) => tr.line)).toHaveLength(2)
    })

    it('各行が見出しの table を持ち、@dragleave / @dragend を配線している', () => {
        expect(
            find_drag_row_violations(source, FOLDABLE_STRUCT_VIEW, header_class),
            'resolve_drop_position は見出しの矩形で測り、dragleave で線を消し、dragend でも（window の capture と二重に）後始末する前提',
        ).toEqual([])
    })

    // 走査が「何も見つけられないだけ」で緑になっていないことを確かめる
    it('検出ロジックが違反を見つけられる（自己検査）', () => {
        const fixture = [
            '<template>',
            '    <tr v-if="is_item()" @dragstart="drag_start" @drop="drop"',
            '        @dragover="dragover" @dragleave="dragleave"',
            '        :class="effective_draggable ? \'a b\' : \'a\'">',
            '        <td>',
            '            <table class="foldable_struct_header"><tbody><tr><td>x</td></tr></tbody></table>',
            '        </td>',
            '    </tr>',
            '    <tr v-if="!is_item()" @dragstart="drag_start" @drop="drop"',
            '        @dragover="dragover" @dragleave="dragleave" @dragend="dragend">',
            '        <td>',
            '            <table class="ml-4"><tbody><tr><td>y</td></tr></tbody></table>',
            '        </td>',
            '    </tr>',
            '</template>',
        ].join('\n')
        expect(list_drag_rows(fixture)).toHaveLength(2)
        const violations = find_drag_row_violations(fixture, 'fixture.vue', 'foldable_struct_header')
        expect(violations).toHaveLength(2)
        expect(violations[0]).toContain('fixture.vue:2')
        expect(violations[0]).toContain('@dragend="dragend"')
        expect(violations[1]).toContain('fixture.vue:9')
        expect(violations[1]).toContain('0 個')
    })
})
