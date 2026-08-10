/**
 * `v-on` で渡した中継束と、同じ要素に手書きした `@イベント` の二重配線を検出する。
 *
 * Vue は同名イベントに複数のハンドラが登録されると**全部呼ぶ**ので、
 * 束に含まれるイベントを `@` でも書くと二重に発火する。
 * 手書きの `@` を17個並べていた箇所を `v-on="crudRelayHandlers"` に畳む改修で
 * 消し忘れが起きやすく、しかも型では検出できない（どちらも正しい記法なので通る）。
 * 実行時にフォーカスが飛ぶ・同じ更新が2回走る、という形でしか気づけないため機械検査する。
 *
 * 「手書きの束が全イベントを網羅しているか」を機械検査することも試したが、
 * このコードベースではページが `crudRelayHandlers` / `allColumnsRequestHandlers` /
 * `rykvDialogHandlers` のように束を**意図的に分割してスプレッドする**のが正規の書き方で、
 * 束単位で網羅を求めると正しいコードが大量に違反になる。
 * 束の分割は要素単位で合流するので、束だけを見て判定できない。
 * 網羅性のほうは `kyou-view-relay.ts` の `Exclude` 網羅チェック（型）と
 * `build_kyou_dialog_host_handlers` の必須フィールド（型）、
 * および Dnote チェーンのマウントテストで担保する。
 */
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs'
import { dirname, join, relative, sep } from 'node:path'
import { describe, expect, it } from 'vitest'

import { kyou_dialog_relay_event_names, kyou_view_relay_event_names } from '@/classes/kyou-view-relay'

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
const client_root = join(repo_root, 'src', 'client')

/** 束の生成関数と、それが張るイベント */
const BUILDER_COVERAGE: Record<string, ReadonlyArray<string>> = {
    'build_kyou_view_relay': kyou_view_relay_event_names,
    'build_kyou_dialog_relay': kyou_dialog_relay_event_names,
    'build_kyou_dialog_host_handlers': [...kyou_dialog_relay_event_names, 'closed'],
}

function list_files_recursive(dir: string, matcher: (path: string) => boolean): Array<string> {
    const found = new Array<string>()
    for (const entry of readdirSync(dir)) {
        const path = join(dir, entry)
        if (statSync(path).isDirectory()) {
            if (entry === '__tests__' || entry === 'node_modules') {
                continue
            }
            found.push(...list_files_recursive(path, matcher))
            continue
        }
        if (matcher(path)) {
            found.push(path)
        }
    }
    return found
}

function to_repo_path(path: string): string {
    return relative(repo_root, path).split(sep).join('/')
}

// 同じ composable を何百回も読むので覚えておく
const source_cache = new Map<string, string>()
function read_source(path: string): string {
    const cached = source_cache.get(path)
    if (cached !== undefined) {
        return cached
    }
    const source = readFileSync(path, 'utf8')
    source_cache.set(path, source)
    return source
}

/**
 * `.vue` のテンプレートを要素ごとに切る。
 * このリポジトリは属性を1行1つで書くので、行頭の `<Tag` を境目にすれば十分。
 */
function split_elements(source: string): Array<Array<string>> {
    const chunks = new Array<Array<string>>()
    let current: Array<string> | null = null
    for (const line of source.split(/\r?\n/)) {
        if (/^\s*<[A-Za-z]/.test(line)) {
            current = []
            chunks.push(current)
        }
        if (current) {
            current.push(line)
        }
    }
    return chunks
}

/**
 * その `.vue` から見える「生成関数で作られた束」の名前 → 張るイベント。
 * 自分の `<script>` と、import している `use-*.ts` の両方を見る。
 * 束名は同名でもファイルごとに中身が違うので、必ずこの2箇所に閉じて解決する。
 */
function collect_builder_bundles(vue_path: string, vue_source: string): Map<string, ReadonlyArray<string>> {
    const bundles = new Map<string, ReadonlyArray<string>>()

    const sources = [vue_source]
    const import_pattern = /from\s+['"]@\/classes\/(use-[\w-]+)['"]/g
    let import_match: RegExpExecArray | null
    while ((import_match = import_pattern.exec(vue_source)) !== null) {
        const composable_path = join(client_root, 'classes', `${import_match[1]}.ts`)
        if (existsSync(composable_path)) {
            sources.push(read_source(composable_path))
        }
    }
    void vue_path

    for (const source of sources) {
        for (const [builder, covered] of Object.entries(BUILDER_COVERAGE)) {
            const pattern = new RegExp(`(?:const\\s+(\\w+)\\s*=\\s*|\\.\\.\\.)${builder}\\(`, 'g')
            let match: RegExpExecArray | null
            while ((match = pattern.exec(source)) !== null) {
                if (match[1]) {
                    bundles.set(match[1], covered)
                }
            }
            // `const x = { ...build_kyou_dialog_relay(...), 'closed': ... }` の形
            const spread_pattern = new RegExp(`const\\s+(\\w+)\\s*=\\s*\\{[^}]*\\.\\.\\.${builder}\\(`, 'g')
            let spread_match: RegExpExecArray | null
            while ((spread_match = spread_pattern.exec(source)) !== null) {
                bundles.set(spread_match[1], covered)
            }
        }
    }
    return bundles
}

/** `v-on="foo"` / `v-on="{ ...foo, ...bar }"` から束の名前を取り出す */
function bundle_names_in_v_on(element_text: string): Array<string> {
    const names = new Array<string>()
    const pattern = /v-on="([^"]*)"/g
    let match: RegExpExecArray | null
    while ((match = pattern.exec(element_text)) !== null) {
        for (const identifier of match[1].match(/[A-Za-z_$][\w$]*/g) ?? []) {
            names.push(identifier)
        }
    }
    return names
}

describe('中継束の二重配線の走査', () => {
    const vue_files = list_files_recursive(join(client_root, 'pages'), path => path.endsWith('.vue'))

    it('走査対象のファイルが見つかる（パスがずれたら気づけるように）', () => {
        expect(vue_files.length).toBeGreaterThan(100)
    })

    // 300近い .vue と composable を読むので、遅いファイルシステムでは既定の5秒に収まらない
    it('生成関数で作った束を v-on で渡した要素に、同じイベントの @ を書いていない', () => {
        const violations = new Array<string>()
        for (const path of vue_files) {
            const source = read_source(path)
            if (!source.includes('v-on="')) {
                continue
            }
            const bundles = collect_builder_bundles(path, source)
            if (bundles.size === 0) {
                continue
            }
            for (const chunk of split_elements(source)) {
                const text = chunk.join('\n')
                for (const bundle_name of bundle_names_in_v_on(text)) {
                    const covered = bundles.get(bundle_name)
                    if (!covered) {
                        continue
                    }
                    const duplicated = covered.filter(event_name => text.includes(`@${event_name}=`))
                    if (duplicated.length !== 0) {
                        violations.push(`${to_repo_path(path)}: ${chunk[0].trim()} が v-on="${bundle_name}" と @${duplicated.join(' / @')} を併記している（両方発火する）`)
                    }
                }
            }
        }
        expect(violations).toEqual([])
    }, 30000)

    // 走査が「何も見つけられないだけ」で緑になっていないことを確かめる
    it('検出ロジックが二重配線を見つけられる（自己検査）', () => {
        const fixture = [
            '<template>',
            '    <Child v-on="crudRelayHandlers"',
            '        @updated_kyou="(kyou) => noop(kyou)" />',
            '</template>',
            '<script setup lang="ts">',
            'const crudRelayHandlers = build_kyou_view_relay(emits)',
            '</script>',
        ].join('\n')

        const bundles = collect_builder_bundles('fixture.vue', fixture)
        expect(bundles.get('crudRelayHandlers')).toContain('updated_kyou')

        const element = split_elements(fixture).find(chunk => chunk.join('\n').includes('v-on='))
        expect(element).toBeDefined()
        const text = (element ?? []).join('\n')
        expect(bundle_names_in_v_on(text)).toContain('crudRelayHandlers')
        expect(text.includes('@updated_kyou=')).toBe(true)
    })
})
