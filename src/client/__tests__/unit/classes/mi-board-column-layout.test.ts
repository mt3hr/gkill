/**
 * Mi板の列の見出し高さが、定数とCSSで食い違っていないかを走査する。
 *
 * 列は「板名の見出し + KyouListView」で app_content_height を分け合う。
 * KyouListView に渡す list_height は `app_content_height - MI_BOARD_TITLE_HEIGHT` なので、
 * 見出しの実高さがこの定数とずれた分だけ列の下に空白が残る（または列がはみ出す）。
 * 以前は Vuetify の v-card-title の実高さ44pxに対して48を引いており、4pxの空白が出ていた。
 *
 * 見た目の隙間としてしか現れず型でも検出できないので、
 * 「定数」と「見出しに当てているCSSの高さ」が一致していることを機械検査する。
 */
import { existsSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { describe, expect, it } from 'vitest'

import { MI_BOARD_TITLE_HEIGHT } from '@/classes/mi-board-column-layout'

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

/** 板名の見出しを持つ画面 */
const COLUMN_VIEWS = [
    'src/client/pages/views/mi-view.vue',
    'src/client/pages/views/shared-mi-view.vue',
]

function read_view(repo_path: string): string {
    return readFileSync(join(repo_root, repo_path), 'utf8')
}

/** `.mi_board_column_title { height: Npx }` の N を取り出す */
function extract_title_height_px(source: string): number | null {
    const matched = /\.mi_board_column_title\s*\{[^}]*height:\s*([0-9.]+)px/.exec(source)
    return matched ? Number(matched[1]) : null
}

describe('Mi板の列の見出し高さ', () => {
    it.each(COLUMN_VIEWS)('%s のCSSの高さが MI_BOARD_TITLE_HEIGHT と一致する', (repo_path) => {
        const height = extract_title_height_px(read_view(repo_path))
        expect(height, `${repo_path} に .mi_board_column_title の height が無い`).not.toBeNull()
        expect(height).toBe(MI_BOARD_TITLE_HEIGHT)
    })

    it.each(COLUMN_VIEWS)('%s が list_height から定数を引いている', (repo_path) => {
        const source = read_view(repo_path)
        expect(source, `${repo_path} が list_height を定数で引いていない`)
            .toContain('kyou_list_view_height.valueOf() - MI_BOARD_TITLE_HEIGHT')
        // 数値の直書きに戻っていないこと
        expect(/kyou_list_view_height\.valueOf\(\)\s*-\s*[0-9]/.test(source), `${repo_path} で引き算が数値の直書きに戻っている`)
            .toBe(false)
    })

    it.each(COLUMN_VIEWS)('%s の見出しにクラスが付いている', (repo_path) => {
        const source = read_view(repo_path)
        const title_count = (source.match(/<v-card-title[^>]*class="[^"]*mi_board_column_title/g) ?? []).length
        const all_title_count = (source.match(/<v-card-title/g) ?? []).length
        expect(title_count, `${repo_path} の v-card-title にクラスが付いていないものがある`).toBe(all_title_count)
    })

    // 走査が「何も見つけられないだけ」で緑になっていないことを確かめる
    it('検出ロジックが不一致を見つけられる（自己検査）', () => {
        const fixture = '.mi_board_column_title {\n    height: 48px;\n}'
        expect(extract_title_height_px(fixture)).toBe(48)
        expect(extract_title_height_px('.other { height: 44px; }')).toBeNull()
    })
})
