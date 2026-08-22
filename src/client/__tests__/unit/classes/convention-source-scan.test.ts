/**
 * 規約スキル（.claude/skills/gkill-client-foundation/SKILL.md）に明文化してある規約のうち、
 * 「外れても型チェックも lint も通り、実行時にエラーも出ない」ものをソース走査で見張る。
 *
 * ここで見ているのはどれも過去に実際に取り残しが出たもの:
 *   - `autofocus` を view 側に書く（1本だけ剥がして18本を取り残した）
 *   - `:draggable` を `is_pc` でゲートし忘れる（タッチパネル付きPCで D&D が死ぬ / 逆も）
 *   - Kyou の引き直しを手書きする（`load_all(force_attached)` を落として添付タグを引き直さない）
 *   - 中継束を `@evt="xxxHandlers['evt']"` と展開して並べる（畳み忘れが増殖する）
 *   - 表示文字列に HTML タグのリテラルを埋める（`format_duration` の `<br>` が
 *     剥がしていない画面でタグのまま見えていた）
 */
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs'
import { dirname, join, relative, sep } from 'node:path'
import { describe, expect, it } from 'vitest'

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

/** `//` 行コメントと `<!-- -->` を落とす。規約違反かどうかは「実際に書いた属性/呼び出し」で見るため */
function strip_comments(source: string): string {
    return source
        .replace(/<!--[\s\S]*?-->/g, '')
        .split(/\r?\n/)
        .map(line => line.replace(/^\s*\/\/.*$/, ''))
        .join('\n')
}

const vue_files = list_files_recursive(join(client_root, 'pages'), path => path.endsWith('.vue'))
const ts_files = list_files_recursive(join(client_root, 'classes'), path => path.endsWith('.ts'))

describe('クライアントの規約のソース走査', () => {
    it('走査対象のファイルが見つかる（パスがずれたら気づけるように）', () => {
        expect(vue_files.length).toBeGreaterThan(100)
        expect(ts_files.length).toBeGreaterThan(100)
    })

    // ダイアログの自動フォーカスは useFloatingDialog の autofocus オプション1箇所に閉じている。
    // view 側に書くと、その view をサイドバーやページ直下で使ったときに
    // ページ読込でフォーカスを奪う（ダイアログの中とは限らないため）。
    it('view に autofocus を書いていない', () => {
        const violations = new Array<string>()
        for (const path of vue_files) {
            if (!path.includes(`${sep}views${sep}`)) {
                continue
            }
            const source = strip_comments(readFileSync(path, 'utf8'))
            if (/\bautofocus\b/.test(source)) {
                violations.push(`${to_repo_path(path)}: view に autofocus がある（useFloatingDialog の autofocus に任せること）`)
            }
        }
        expect(violations).toEqual([])
    })

    // D&Dの可否は「タッチできるか」ではなく「PCか」で決める（タッチパネル付きWindowsノートでも使えるように）。
    // 生の props をそのまま `:draggable` に流すのは、親から降ってきた値を子が中継するときだけ許す。
    //
    // 見張るのは **DOM要素**（小文字タグ）に付いた `:draggable` だけ。
    // 子コンポーネント（大文字タグ）へ渡すのは「この一覧では D&D を使う」という機能スイッチで、
    // 実際のゲートは受け取った側の `effective_draggable` が掛ける
    // （例: `mi-view.vue` の `<KyouListView :draggable="true">` → `use-mi-kyou-view.ts`）。
    const allowed_draggable_expressions = new Set(['effective_draggable', 'draggable', 'props.draggable'])
    /** テンプレートを要素ごとに切る。属性は1行1つで書くので行頭の `<Tag` を境目にすれば足りる */
    function split_elements(source: string): Array<{ tag: string, text: string }> {
        const chunks = new Array<{ tag: string, lines: Array<string> }>()
        let current: { tag: string, lines: Array<string> } | null = null
        for (const line of source.split(/\r?\n/)) {
            const opened = /^\s*<([A-Za-z][\w.-]*)/.exec(line)
            if (opened) {
                current = { tag: opened[1], lines: [] }
                chunks.push(current)
            }
            if (current) {
                current.lines.push(line)
            }
        }
        return chunks.map(chunk => ({ tag: chunk.tag, text: chunk.lines.join('\n') }))
    }
    it(':draggable は is_pc でゲートした値を渡している', () => {
        const violations = new Array<string>()
        for (const path of vue_files) {
            const source = strip_comments(readFileSync(path, 'utf8'))
            for (const element of split_elements(source)) {
                // 大文字始まりは子コンポーネント。渡した値のゲートは受け取り側の責任
                if (/^[A-Z]/.test(element.tag)) {
                    continue
                }
                for (const match of element.text.matchAll(/:draggable=["']?([^"'\s>]+)["']?/g)) {
                    const expression = match[1].trim()
                    if (allowed_draggable_expressions.has(expression)) {
                        continue
                    }
                    violations.push(`${to_repo_path(path)}: <${element.tag}> の :draggable="${expression}"（is_pc でゲートした effective_draggable を使うこと）`)
                }
            }
        }
        expect(violations).toEqual([])
    })

    it('effective_draggable の定義はすべて is_pc を見ている', () => {
        const definitions = new Array<string>()
        const violations = new Array<string>()
        for (const path of ts_files) {
            const source = readFileSync(path, 'utf8')
            for (const match of source.matchAll(/const\s+effective_draggable\s*=\s*computed\(\(\)\s*=>([^\n]*)/g)) {
                definitions.push(to_repo_path(path))
                if (!match[1].includes('is_pc')) {
                    violations.push(`${to_repo_path(path)}: effective_draggable が is_pc を見ていない`)
                }
            }
        }
        // 定義が1つも見つからないと「違反なし」で緑になるので、拾えていることを確かめる
        expect(definitions.length).toBeGreaterThan(3)
        expect(violations).toEqual([])
    })

    // 引き直しの手順は4つあり（SWキャッシュ削除 → reload(true) → is_typed_data_loaded=false →
    // load_all(query, true)）、1つでも欠けると引き直しに失敗する。
    // とくに load_all の force_attached を落とすと添付タグを一度も引き直さない。
    const hand_written_reload_allowlist = new Set([
        // 引き直しの実装そのもの
        'src/client/classes/kyou-reload.ts',
        // reload / reload_with_typed_datas の定義側
        'src/client/classes/datas/kyou.ts',
    ])
    it('Kyou の引き直しを手書きしていない', () => {
        const violations = new Array<string>()
        for (const path of [...ts_files, ...vue_files]) {
            const repo_path = to_repo_path(path)
            if (hand_written_reload_allowlist.has(repo_path)) {
                continue
            }
            const source = strip_comments(readFileSync(path, 'utf8'))
            if (/\.reload\(\s*true/.test(source)) {
                violations.push(`${repo_path}: 引き直しを手書きしている（classes/kyou-reload.ts の refresh_kyou / refresh_kyou_in_list を使うこと）`)
            }
        }
        expect(violations).toEqual([])
    })

    // 束を作ったのにテンプレートで1イベントずつ添字で取り出すと、
    // v-on 1行で済むものが属性18行に戻り、足したイベントの書き忘れが起きる。
    it('中継束を @evt="xxxHandlers[\'evt\']" と展開していない', () => {
        const violations = new Array<string>()
        for (const path of vue_files) {
            const source = strip_comments(readFileSync(path, 'utf8'))
            for (const match of source.matchAll(/@([\w.]+)="(\w*Handlers)\['/g)) {
                violations.push(`${to_repo_path(path)}: @${match[1]}="${match[2]}['...']"（v-on="${match[2]}" 1行に畳むこと）`)
            }
        }
        expect(violations).toEqual([])
    })

    // ダイアログのロジックは classes/use-*.ts に置く。
    // .vue の <script setup> に残してよいのは import と define* とコンポーザブルの分割代入だけ。
    // 116本中89本が .vue にロジックを抱えていて、同じ処理が孤児コンポーザブルと二重管理になっていた。
    it('ダイアログの <script setup> にロジックを残していない', () => {
        const violations = new Array<string>()
        const dialog_files = vue_files.filter(path => path.includes(`${sep}dialogs${sep}`))
        expect(dialog_files.length).toBeGreaterThan(100)
        for (const path of dialog_files) {
            const source = readFileSync(path, 'utf8')
            const script = /<script[^>]*\bsetup\b[^>]*>([\s\S]*?)<\/script>/.exec(source)
            if (!script) {
                violations.push(`${to_repo_path(path)}: <script setup> が無い`)
                continue
            }
            for (const line of strip_comments(script[1]).split(/\r?\n/)) {
                const trimmed = line.trim()
                // 継続行（字下げのある行）は先頭行で判定済み
                if (trimmed === '' || line.startsWith(' ') || line.startsWith('\t')) {
                    continue
                }
                if (/^(import|export)\b/.test(trimmed)) {
                    continue
                }
                if (/\bdefine(Props|Emits|Expose|Options|Model|Slots)\b/.test(trimmed)) {
                    continue
                }
                // 子コンポーネントへのテンプレート ref は .vue 側に置くのが規約。
                // コンポーザブルは options で受け取る（use-time-is-view.ts など）
                if (/^const \w+ = ref<(InstanceType<typeof \w+>|ComponentRef)[^>]*>?[^)]*\)/.test(trimmed)) {
                    continue
                }
                // コンポーザブルの分割代入
                if (/^const \{[^}]*\}\s*=\s*use[A-Z]\w*\(/.test(trimmed) || trimmed === 'const {') {
                    continue
                }
                // 複数行にわたる分割代入の閉じ側
                if (/^\}\s*=\s*use[A-Z]\w*\(/.test(trimmed) || trimmed === '})' || trimmed === '}') {
                    continue
                }
                violations.push(`${to_repo_path(path)}: ${trimmed}（classes/use-*.ts へ移すこと）`)
            }
        }
        expect(violations).toEqual([])
    })

    // 中断（AbortController）の判定は classes/abort-error.ts に1つだけ。
    // 20箇所へ手書きで複製されていて、片方のブラウザの文言しか見ていない写しも混ざっていた。
    it('中断の判定を手書きしていない', () => {
        const violations = new Array<string>()
        for (const path of [...ts_files, ...vue_files]) {
            const repo_path = to_repo_path(path)
            if (repo_path === 'src/client/classes/abort-error.ts') {
                continue
            }
            const source = readFileSync(path, 'utf8')
            if (/signal is aborted without reason|user aborted a request/.test(source)) {
                violations.push(`${repo_path}: 中断の判定を手書きしている（classes/abort-error.ts の is_abort_error / log_unless_aborted を使うこと）`)
            }
        }
        expect(violations).toEqual([])
    })

    // 表示用の文字列に HTML タグのリテラルを埋めない。
    // 表示側は {{ }} 補間なので Vue がエスケープし、剥がし忘れた画面では
    // タグが文字として見える（format_duration の "<br>" が Dnote の集計リストと
    // 相関グラフで実際に出ていた）。区切りが要るなら本物の改行を使い、
    // 改行として見せたい場所だけが white-space: pre-line で opt-in する。
    const html_tag_literal = /(["'`])<\s*\/?\s*(br|p|div|span|b|i|hr)\b[^>]*>/i
    it('表示文字列に HTML タグのリテラルを埋めていない', () => {
        const violations = new Array<string>()
        for (const path of ts_files) {
            const source = strip_comments(readFileSync(path, 'utf8'))
            const match = html_tag_literal.exec(source)
            if (match) {
                violations.push(`${to_repo_path(path)}: ${match[0]}（改行が要るなら "\\n" を使い、見せる側が white-space: pre-line で opt-in すること）`)
            }
        }
        expect(violations).toEqual([])
    })

    // 上の規約の受け皿。ここを外すと Dnote の集計リストが
    // "23時間 6分\n（23.1時間）" を1行に畳んで表示してしまう
    it('集計リストの値が pre-line で改行を見せている', () => {
        const source = readFileSync(join(client_root, 'pages', 'views', 'aggregated-list-item.vue'), 'utf8')
        expect(source).toContain('aggregated_list_item_value')
        expect(source).toContain('white-space: pre-line')
    })

    // 走査が「何も見つけられないだけ」で緑になっていないことを確かめる
    it('検出ロジックが違反を見つけられる（自己検査）', () => {
        expect(html_tag_literal.test('diff_str += "<br>（"')).toBe(true)
        expect(html_tag_literal.test('const s = "改行なし"')).toBe(false)
        // 比較演算子や型引数を誤検出しない
        expect(html_tag_literal.test('if (a < b) { return "x" }')).toBe(false)
        expect(html_tag_literal.test('const x = ref<Array<Kyou>>([])')).toBe(false)

        expect(/\bautofocus\b/.test(strip_comments('<v-text-field autofocus />'))).toBe(true)
        expect(/\bautofocus\b/.test(strip_comments('// autofocus は書かない\n<v-text-field />'))).toBe(false)

        const draggable_elements = split_elements(`<h2 :draggable="editable">\n<KyouListView :draggable="true">`)
        expect(draggable_elements.map(e => e.tag)).toEqual(['h2', 'KyouListView'])
        const dom_element = draggable_elements[0]
        expect([...dom_element.text.matchAll(/:draggable=["']?([^"'\s>]+)["']?/g)].map(m => m[1])).toEqual(['editable'])
        expect(allowed_draggable_expressions.has('editable')).toBe(false)

        expect(/\.reload\(\s*true/.test('await kyou.reload(true)')).toBe(true)
        expect([...'@received_errors="fooHandlers[\'received_errors\']"'.matchAll(/@([\w.]+)="(\w*Handlers)\['/g)].length).toBe(1)
    })
})
