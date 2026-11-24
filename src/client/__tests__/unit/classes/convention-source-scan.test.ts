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
 *   - ダイアログのヘッダのタイトル欄に中身を入れる（全ダイアログ空にそろえる約束。1本だけ名前を出していた）
 *   - 一覧の行のクリックを文字（v-list-item-title）にだけ付ける（画面切替メニュー8画面ぶんが全部この形で、
 *     行の余白を押すとメニューが閉じるだけで遷移しなかった）
 *   - 参照先を埋め込むビュー（ReKyou / MiReKyou）が参照先へ「一覧ではない」を固定で渡す
 *     （一覧の行の中なのに、参照先の画像を原寸で読み、動画のメタデータも読みに行っていた）
 */
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs'
import { basename, dirname, join, relative, sep } from 'node:path'
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
    // （例: `mi-columns-view.vue` の `<KyouListView :draggable="…">` → `use-mi-kyou-view.ts`）。
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

    // フローティングダイアログのヘッダのタイトル欄は全ダイアログで空にそろえる（利用者の指定）。
    // 新しいダイアログは既存のものを書き写して作るので、1本に中身を入れると以後の写しに広がる
    // （スキル閲覧ダイアログだけがスキル名を出していた）。見せたい名前は本文の先頭に置くこと。
    const floating_dialog_title = /class="gkill-floating-dialog__title"[^>]*>([\s\S]*?)<\/div>/g
    it('ダイアログのヘッダのタイトル欄を空にしている', () => {
        const titles = new Array<string>()
        const violations = new Array<string>()
        for (const path of vue_files.filter(path => path.includes(`${sep}dialogs${sep}`))) {
            const source = strip_comments(readFileSync(path, 'utf8'))
            for (const match of source.matchAll(floating_dialog_title)) {
                titles.push(to_repo_path(path))
                if (match[1].trim() !== '') {
                    violations.push(`${to_repo_path(path)}: タイトル欄に「${match[1].trim()}」がある（空にして、名前は本文に出すこと）`)
                }
            }
        }
        // 1本も拾えないと「違反なし」で緑になるので、拾えていることを確かめる
        expect(titles.length).toBeGreaterThan(100)
        expect(violations).toEqual([])
    })

    // 一覧の行のクリックは行（v-list-item）に付け、文字（v-list-item-title）に付けない。
    // v-list-item-title は行の中の文字1行ぶん（高さ48pxの行に24px。面積で4割）しか無く、
    // 上下12pxと左右16pxの余白を押すとリップルが出てメニューは閉じるのに何も起きない。
    // 指で押すスマホで外しやすく、画面切替メニューが「押しても遷移しない」ように見えていた。
    const list_item_title_with_click = /<v-list-item-title\b[^>]*@click/
    it('v-list-item-title に @click を付けていない', () => {
        const violations = new Array<string>()
        for (const path of vue_files) {
            const source = strip_comments(readFileSync(path, 'utf8'))
            if (list_item_title_with_click.test(source)) {
                violations.push(`${to_repo_path(path)}: v-list-item-title に @click がある（行の v-list-item に付けること）`)
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

    // 参照先を埋め込むビューは、自分が一覧の行かどうかを参照先の KyouView へそのまま引き継ぐ。
    // false を固定で渡すと、一覧の行の中でも参照先の画像は原寸で読まれ、
    // 動画は preload="metadata" になって行の数だけリクエストと互換変換が走る。
    it('参照先を埋め込むビューが is_image_request_to_thumb_size を引き継いでいる', () => {
        for (const file_name of ['re-kyou-view.vue', 'mi-re-kyou-view.vue']) {
            const source = strip_comments(readFileSync(join(client_root, 'pages', 'views', file_name), 'utf8'))
            expect(source, file_name).toContain(':is_image_request_to_thumb_size="is_image_request_to_thumb_size"')
            expect(source, file_name).not.toMatch(/:is_image_request_to_thumb_size="(true|false)"/)
        }
    })

    // 日付・時刻入力を空に戻す手段は、欄の中の ×（Vuetify の clearable）1つに統一してある（ADR-0419）。
    // × を付けてはいけないのは必須の日時欄（related_* / notification_* / timeis_start_*）。
    // 裏の ref が Ref<Date>（null 不可）で表示 computed に null ガードが無く、
    // 空にすると保存時に *_time_is_blank になる。
    // 表で持つのは、同じ ref 名が「× を付ける画面」と「付けない画面」の両方にありうるため
    // （timeis_start_* は必須、timeis_end_* は任意で、どちらも同じ3ダイアログに並ぶ）
    const clearable_date_time_fields = [
        'add-mi-re-kyou-view.vue mi_estimate_start_date_string',
        'add-mi-re-kyou-view.vue mi_estimate_start_time_string',
        'add-mi-re-kyou-view.vue mi_estimate_end_date_string',
        'add-mi-re-kyou-view.vue mi_estimate_end_time_string',
        'add-mi-re-kyou-view.vue mi_limit_date_string',
        'add-mi-re-kyou-view.vue mi_limit_time_string',
        'add-mi-view.vue mi_estimate_start_date_string',
        'add-mi-view.vue mi_estimate_start_time_string',
        'add-mi-view.vue mi_estimate_end_date_string',
        'add-mi-view.vue mi_estimate_end_time_string',
        'add-mi-view.vue mi_limit_date_string',
        'add-mi-view.vue mi_limit_time_string',
        'add-time-is-view.vue timeis_end_date_string',
        'add-time-is-view.vue timeis_end_time_string',
        'calendar-query.vue start_time_string',
        'calendar-query.vue end_time_string',
        'edit-mi-re-kyou-view.vue mi_estimate_start_date_string',
        'edit-mi-re-kyou-view.vue mi_estimate_start_time_string',
        'edit-mi-re-kyou-view.vue mi_estimate_end_date_string',
        'edit-mi-re-kyou-view.vue mi_estimate_end_time_string',
        'edit-mi-re-kyou-view.vue mi_limit_date_string',
        'edit-mi-re-kyou-view.vue mi_limit_time_string',
        'edit-mi-view.vue mi_estimate_start_date_string',
        'edit-mi-view.vue mi_estimate_start_time_string',
        'edit-mi-view.vue mi_estimate_end_date_string',
        'edit-mi-view.vue mi_estimate_end_time_string',
        'edit-mi-view.vue mi_limit_date_string',
        'edit-mi-view.vue mi_limit_time_string',
        'edit-time-is-view.vue timeis_end_date_string',
        'edit-time-is-view.vue timeis_end_time_string',
        'end-time-is-playing-view.vue timeis_end_date_string',
        'end-time-is-playing-view.vue timeis_end_time_string',
        'period-of-time-query.vue period_of_time_start_time_string',
        'period-of-time-query.vue period_of_time_end_time_string',
    ]

    // calendar-query.vue だけは v-model のままでよい。時刻 ref が Ref<string | null> で、
    // Vuetify が書く null をそのまま「未指定」として読む（use-calendar-query.ts の
    // parse_time_string が null を受ける）。ほかは Ref<string> のまま保存側が === "" を
    // 見ているので、Vuetify に null を書かせず @click:clear で "" を入れる
    const clearable_with_v_model = new Set([
        'calendar-query.vue start_time_string',
        'calendar-query.vue end_time_string',
    ])

    const date_time_field_element = /<v-text-field\b[\s\S]*?>/g
    const date_time_bound_ref = /(?::model-value|v-model)="(\w*_(?:date|time)_string)"/
    const has_clearable = /(?::clearable=|\sclearable[\s>/])/

    /** テンプレートの <v-text-field> のうち、日付・時刻の文字列を結んでいるものを集める */
    function collect_date_time_fields(): Array<{ key: string, element: string }> {
        const found = new Array<{ key: string, element: string }>()
        for (const path of vue_files) {
            const source = strip_comments(readFileSync(path, 'utf8'))
            for (const match of source.matchAll(date_time_field_element)) {
                const element = match[0]
                const bound = date_time_bound_ref.exec(element)
                if (!bound) {
                    continue
                }
                found.push({ key: `${basename(path)} ${bound[1]}`, element })
            }
        }
        return found
    }

    it('日付・時刻欄の × は表のとおりに付いている', () => {
        const fields = collect_date_time_fields()
        // 必須の日時欄のほうが多いので、表の2倍近くある（パスがずれたら気づけるように）
        expect(fields.length).toBeGreaterThan(clearable_date_time_fields.length)
        const actual = fields.filter(field => has_clearable.test(field.element)).map(field => field.key).sort()
        expect(actual).toEqual([...clearable_date_time_fields].sort())
    })

    it('× を付けた日付・時刻欄は :model-value と @click:clear で結んでいる', () => {
        const violations = new Array<string>()
        for (const field of collect_date_time_fields()) {
            if (!has_clearable.test(field.element) || clearable_with_v_model.has(field.key)) {
                continue
            }
            if (/\sv-model=/.test(field.element)) {
                violations.push(`${field.key}: v-model で結んでいる（日付欄は読み取り専用 computed なので × が黙って効かず、時刻欄には null が入って保存側の === "" 判定をすり抜ける）`)
            }
            const clear = /@click:clear="([^"]*)"/.exec(field.element)
            if (!clear) {
                violations.push(`${field.key}: @click:clear が無い（× を押しても本物の状態が落ちない）`)
                continue
            }
            const expected = field.key.endsWith('_time_string') ? "= ''" : '= null'
            if (!clear[1].includes(expected)) {
                violations.push(`${field.key}: @click:clear が ${expected} を入れていない（${clear[1]}）`)
            }
        }
        expect(violations).toEqual([])
    })

    // 日付欄は × の有無によらず min-width="160"、追加/編集ダイアログの時刻欄は 140。
    // 120 のままだと末尾が切れる（× を付けた欄は × のぶん、付けない欄も既定の左右余白のぶん
    // 内側が足りない）。とくに「現在日時」が入れるのは HH:mm:ss で、ピッカーの HH:mm より長い。
    // サイドバーの検索条件（*-query.vue）の時刻欄は HH:mm しか入らず、横幅も無いので 120 のまま
    it('日付欄は min-width="160"、ダイアログの時刻欄は 140 で揃っている', () => {
        const violations = new Array<string>()
        for (const field of collect_date_time_fields()) {
            const is_date = field.key.endsWith('_date_string')
            const is_query_sidebar = /-query\.vue /.test(field.key)
            const expected = is_date ? '160' : (is_query_sidebar ? '120' : '140')
            if (!field.element.includes(`min-width="${expected}"`)) {
                violations.push(`${field.key}: min-width が ${expected} でない（値の末尾が切れる）`)
            }
        }
        expect(violations).toEqual([])
    })

    // 入力欄の横のボタン群は「リセット」「現在日時」だけ。空に戻すのは欄の中の × へ移した
    it('日付・時刻欄の横に「クリア」ボタンを残していない', () => {
        const violations = new Array<string>()
        for (const path of vue_files) {
            const source = strip_comments(readFileSync(path, 'utf8'))
            if (!source.includes('gkill-field-side-buttons')) {
                continue
            }
            if (source.includes('CLEAR_TITLE')) {
                violations.push(`${to_repo_path(path)}: 入力欄の横のボタン群に CLEAR_TITLE が残っている（空に戻すのは欄の中の ×）`)
            }
        }
        expect(violations).toEqual([])
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

        expect([...'<div class="gkill-floating-dialog__title">{{ name }}</div>'.matchAll(floating_dialog_title)].map(m => m[1])).toEqual(['{{ name }}'])
        expect([...'<div class="gkill-floating-dialog__title"></div>'.matchAll(floating_dialog_title)].map(m => m[1].trim())).toEqual([''])

        expect(list_item_title_with_click.test('<v-list-item-title @click="go()">')).toBe(true)
        expect(list_item_title_with_click.test('<v-list-item-title\n    @click="go()">')).toBe(true)
        expect(list_item_title_with_click.test('<v-list-item @click="go()">\n    <v-list-item-title>')).toBe(false)

        expect(/\.reload\(\s*true/.test('await kyou.reload(true)')).toBe(true)
        expect([...'@received_errors="fooHandlers[\'received_errors\']"'.matchAll(/@([\w.]+)="(\w*Handlers)\['/g)].length).toBe(1)

        // 日付・時刻欄の × の走査
        expect(has_clearable.test('<v-text-field :clearable="!is_busy" />')).toBe(true)
        expect(has_clearable.test('<v-text-field clearable />')).toBe(true)
        expect(has_clearable.test('<v-text-field hide-details />')).toBe(false)
        // clearable を含む別の属性名を拾わない
        expect(has_clearable.test('<v-text-field :not-clearable="true" />')).toBe(false)
        expect(date_time_bound_ref.exec('<v-text-field :model-value="mi_limit_time_string"')?.[1]).toBe('mi_limit_time_string')
        expect(date_time_bound_ref.exec('<v-text-field v-model="related_date_string"')?.[1]).toBe('related_date_string')
        expect(date_time_bound_ref.exec('<v-text-field v-model="mi_title"')).toBe(null)
        // 時刻へ null を入れる形（保存側の === "" をすり抜ける）を見つけられる
        expect(/@click:clear="([^"]*)"/.exec('@click:clear="mi_limit_time_string = null"')?.[1]?.includes("= ''")).toBe(false)
        expect(/@click:clear="([^"]*)"/.exec('@click:clear="mi_limit_time_string = \'\'"')?.[1]?.includes("= ''")).toBe(true)
    })
})
