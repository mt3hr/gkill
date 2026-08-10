/**
 * KyouView に「行と誤判定される高さ」を渡していないかを走査する。
 *
 * `classes/kyou-row-height.ts` の `is_row_height()` は高さを `Number.parseFloat` して
 * 120 未満なら一覧の行とみなす。パーセント文字列は `parseFloat('80%') === 80` で
 * 数値になってしまうため、詳細ペインやダイアログで `'80%'` / `'100%'` を渡すと
 * 行扱いになり、MiReKyou の参照先 (`mi-re-kyou-view.vue` の `v-if="!is_compact"`)
 * が丸ごと描画されなくなる。Kyou ダイアログで実際にこれが起きていた。
 *
 * 型でもレンダリングでも落ちず、「参照先が出ない」という形でしか気づけないので機械検査する。
 * 行ではない場所は `'unset'` か `'auto'` を渡すこと。
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

/** 高さで行かどうかを判定するビュー。ここに渡す高さだけが検査対象 */
const HEIGHT_SENSITIVE_TAGS = ['KyouView', 'ReKyouView', 'MiReKyouView']

/**
 * パーセントを渡してよい唯一の場所。
 * 画像一覧のセルは 200x200 固定で、参照先を埋め込むビューは詰めた表示にする必要がある。
 */
const ALLOWED_PERCENT_HEIGHT_FILES = [
    'src/client/pages/views/kyou-list-view.vue',
]

const PERCENT_HEIGHT_PATTERN = /:height="'[0-9.]+%'"/

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
 * テンプレートを要素ごとに切る。
 * このリポジトリは属性を1行1つで書くので、行頭の `<Tag` を境目にすれば十分。
 */
function find_percent_height_violations(source: string, repo_path: string): Array<string> {
    const violations = new Array<string>()
    let current_tag: string | null = null
    const lines = source.split(/\r?\n/)
    for (let i = 0; i < lines.length; i++) {
        const line = lines[i]
        const opening = /^\s*<([A-Za-z][A-Za-z0-9]*)/.exec(line)
        if (opening) {
            current_tag = opening[1]
        }
        if (!current_tag || !HEIGHT_SENSITIVE_TAGS.includes(current_tag)) {
            continue
        }
        if (PERCENT_HEIGHT_PATTERN.test(line)) {
            violations.push(`${repo_path}:${i + 1} <${current_tag}> ${line.trim()}`)
        }
    }
    return violations
}

describe('KyouView に渡す高さの走査', () => {
    const vue_files = list_vue_files(pages_root)

    it('走査対象の .vue が見つかっている', () => {
        expect(vue_files.length).toBeGreaterThan(100)
    })

    it('行ではない場所にパーセントの高さを渡していない', () => {
        const violations = new Array<string>()
        for (const path of vue_files) {
            const repo_path = to_repo_path(path)
            if (ALLOWED_PERCENT_HEIGHT_FILES.includes(repo_path)) {
                continue
            }
            violations.push(...find_percent_height_violations(readFileSync(path, 'utf8'), repo_path))
        }
        expect(violations, "パーセントは行と誤判定される。'unset' か 'auto' を渡すこと").toEqual([])
    })

    it('例外に挙げたファイルは実際にパーセントを渡している（許可の空振り防止）', () => {
        for (const repo_path of ALLOWED_PERCENT_HEIGHT_FILES) {
            const source = readFileSync(join(repo_root, repo_path), 'utf8')
            expect(
                find_percent_height_violations(source, repo_path).length,
                `${repo_path} はもうパーセントを渡していない。許可リストから外すこと`,
            ).toBeGreaterThan(0)
        }
    })

    // 走査が「何も見つけられないだけ」で緑になっていないことを確かめる
    it('検出ロジックが違反を見つけられる（自己検査）', () => {
        const fixture = [
            '<v-card :height="\'80%\'">',
            '    <KyouView :kyou="kyou"',
            '        :height="\'80%\'" :width="\'100%\'" />',
            '</v-card>',
        ].join('\n')
        const violations = find_percent_height_violations(fixture, 'fixture.vue')
        expect(violations).toHaveLength(1)
        expect(violations[0]).toContain('fixture.vue:3')
    })
})
