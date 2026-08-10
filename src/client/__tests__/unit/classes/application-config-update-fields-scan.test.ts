/**
 * 設定画面の保存（update_application_config）の詰め替え漏れを検出する。
 *
 * use-application-config-view.ts の update_application_config は
 * `new ApplicationConfig()` から知っているフィールドだけを詰め直して送信する。
 * ここに永続化フィールドを書き忘れると、フィールドが JSON から欠落し、
 * サーバ側でゼロ値に巻き戻って保存される（「設定画面で保存すると別の設定が消える」）。
 * show_tutorial_on_startup が実際にこの穴に落ちていた。
 * 型では検出できない（詰め替えは任意なので省略しても正しいコード）ため機械検査する。
 *
 * フィールド一覧の出どころはサーバの永続化キー
 * （src/server/gkill/dao/user_config/application_config_dao_sqlite3_impl.go の
 * applicationConfigDefaultValue。USER_ID / DEVICE はサーバがセッション値で
 * 上書きするので対象外）。サーバに永続化キーを足したらこの一覧にも足すこと。
 */
import { existsSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
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
const composable_path = join(repo_root, 'src', 'client', 'classes', 'use-application-config-view.ts')
const server_dao_path = join(repo_root, 'src', 'server', 'gkill', 'dao', 'user_config', 'application_config_dao_sqlite3_impl.go')

/** サーバがセッション値で上書きするので、詰め替えの対象外にするキー */
const SERVER_OWNED_KEYS = ['USER_ID', 'DEVICE']

/**
 * サーバの永続化キー一覧（applicationConfigDefaultValue のキー）を抜き出す。
 * 下の PERSISTED_FIELDS は可読性のため手書きで持つが、この抽出結果と
 * 突き合わせることでサーバ側にキーが増えたときの取りこぼしを止める。
 */
function extract_server_persisted_fields(source: string): Array<string> {
    const start = source.indexOf('var applicationConfigDefaultValue = map[string]any{')
    expect(start, 'applicationConfigDefaultValue が見つからない（サーバ側をリネームしたらこのテストも直す）').toBeGreaterThanOrEqual(0)
    const end = source.indexOf('\n}', start)
    expect(end, 'applicationConfigDefaultValue の終端が見つからない').toBeGreaterThan(start)
    const block = source.slice(start, end)
    const keys = Array.from(block.matchAll(/^\s*"([A-Z0-9_]+)":/gm)).map(m => m[1])
    return keys
        .filter(key => !SERVER_OWNED_KEYS.includes(key))
        .map(key => key.toLowerCase())
}

/** サーバが永続化する ApplicationConfig のフィールド（TS側プロパティ名） */
const PERSISTED_FIELDS = [
    'use_dark_theme',
    'google_map_api_key',
    'rykv_image_list_column_number',
    'rykv_hot_reload',
    'mi_default_board',
    'rykv_default_period',
    'mi_default_period',
    'is_show_share_footer',
    'default_page',
    'show_tags_in_list',
    'show_tutorial_on_startup',
    'ryuu_json_data',
    'tag_struct',
    'rep_struct',
    'rep_type_struct',
    'device_struct',
    'mi_board_struct',
    'kftl_template_struct',
    'dnote_json_data',
    'dashboard_json_data',
    'plaing_timeis_json_data',
    'saved_find_query_json_data',
] as const

function extract_update_function_body(source: string): string {
    const start = source.indexOf('async function update_application_config')
    expect(start, 'update_application_config が見つからない（リネームしたらこのテストも直す）').toBeGreaterThanOrEqual(0)
    const end = source.indexOf('props.gkill_api.update_application_config(req)', start)
    expect(end, 'update_application_config の送信箇所が見つからない').toBeGreaterThan(start)
    return source.slice(start, end)
}

describe('update_application_config の詰め替え網羅の走査', () => {
    const source = readFileSync(composable_path, 'utf8')
    const body = extract_update_function_body(source)

    it('永続化フィールドすべてに application_config への代入がある', () => {
        const missing = PERSISTED_FIELDS.filter(field =>
            !new RegExp(`application_config\\.${field}\\s*=`).test(body))
        expect(missing, '詰め替えが漏れているフィールド（保存のたびにゼロ値へ巻き戻る）').toEqual([])
    })

    // 手書き一覧そのものがサーバから漂流するのを止める。
    // サーバに永続化キーを足したらここが落ちるので、PERSISTED_FIELDS にも足すこと
    it('永続化フィールド一覧がサーバの永続化キーと一致する', () => {
        const server_fields = extract_server_persisted_fields(readFileSync(server_dao_path, 'utf8'))
        expect(server_fields.length, 'サーバ側のキー抽出に失敗している').toBeGreaterThan(10)
        expect([...server_fields].sort(), 'サーバの永続化キーと TS 側の一覧がずれている').toEqual([...PERSISTED_FIELDS].sort())
    })

    // 走査が「何も見つけられないだけ」で緑になっていないことを確かめる
    it('検出ロジックが漏れを見つけられる（自己検査）', () => {
        const fixture_body = body.replace(/application_config\.show_tutorial_on_startup\s*=/, 'removed =')
        const missing = PERSISTED_FIELDS.filter(field =>
            !new RegExp(`application_config\\.${field}\\s*=`).test(fixture_body))
        expect(missing).toEqual(['show_tutorial_on_startup'])
    })
})
