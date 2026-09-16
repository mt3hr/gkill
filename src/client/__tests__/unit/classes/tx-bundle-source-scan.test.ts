/**
 * 複数書き込みの保存経路が tx（run_in_tx / tx_id）で束ねられていることをソース走査で固定する。
 *
 * 追加/編集画面は「本体 + タグ（+ 通知）」を1つの tx_id で temp rep に積み、commit_tx で
 * 確定する（ADR-0410、gkill-tx.ts）。サーバの commit_tx は1つの SQLite トランザクションなので
 * 「全部書くか、何も書かないか」になる。ここを外して add_* / update_* を直接呼ぶ形に戻すと、
 * 型検査も既存テストも通ったまま「本体だけ書けてタグが無い記録」が静かに残る
 * （タグで絞った列に現れない = 2026-08 に実際に起きた形）。tx を通す view は 18 本あり、
 * 直接テストがあるのは数本なので、残りを機械検査で守る。
 *
 * 規則:
 *   1. src/client/classes の use-add-*-view.ts / use-edit-*-view.ts / use-confirm-re-kyou-view.ts /
 *      cascade-delete-kyou.ts で Kyou 10 種の add_* / update_* を呼ぶものは run_in_tx を import し、
 *      その呼び出し回数以上の `tx_id = tx_id` 代入を持つ
 *   2. それ以外のファイルが Kyou 10 種の add_* / update_* を呼ぶ場合は、単発の書き込み
 *      （チェックボックスの切り替え・打刻の終了など、1 レコードの更新で完結するもの）として
 *      名指しの許可リストに載っていること。載っていなければ「tx が要るか」を判断してから足す
 */
import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const classes_dir = join(process.cwd(), 'src', 'client', 'classes')

// Kyou 10 種（tag / text / notification は付随データで、単独の画面は1レコードしか書かない）
const KYOU_TYPES = ['kmemo', 'kc', 'lantana', 'mi', 'nlog', 'timeis', 'urlog', 'idf_kyou', 'rekyou', 'mirekyou']
const kyou_write_call = new RegExp(`[.](?:add|update)_(?:${KYOU_TYPES.join('|')})[(]`, 'g')
const tx_id_assign = /[.]tx_id = tx_id\b/g

// tx を通さず Kyou を直接書いてよいファイル（1 レコードの更新で完結し、束ねる相手が無い）
const SINGLE_WRITE_ALLOWLIST: Record<string, string> = {
    'use-mi-view.ts': 'Mi / リポストタスクのチェック切り替え（1 レコードの update）',
    'use-mi-kyou-view.ts': 'Mi のチェック切り替え（1 レコードの update）',
    'use-mi-re-kyou-view.ts': 'リポストタスクのチェック切り替え（1 レコードの update）',
    'use-end-time-is-playing-view.ts': '実行中の打刻の終了（1 レコードの update）',
}

function is_bundled_save_file(name: string): boolean {
    return /^use-(?:add|edit)-.*-view[.]ts$/.test(name)
        || name === 'use-confirm-re-kyou-view.ts'
        || name === 'cascade-delete-kyou.ts'
}

function count(source: string, pattern: RegExp): number {
    return (source.match(pattern) ?? []).length
}

describe('tx 束ねのソース走査', () => {
    const files = readdirSync(classes_dir).filter(name => name.endsWith('.ts') && !name.endsWith('.d.ts'))
    const writers = files
        .map(name => ({ name, source: readFileSync(join(classes_dir, name), 'utf8') }))
        .filter(file => count(file.source, kyou_write_call) > 0)

    it('Kyou を書くファイルが見つかる（走査対象が空になっていない）', () => {
        expect(writers.length).toBeGreaterThanOrEqual(20)
    })

    it('複数書き込みの保存経路は run_in_tx を import し、書き込みごとに tx_id を積む', () => {
        const violations = new Array<string>()
        for (const { name, source } of writers) {
            if (!is_bundled_save_file(name)) {
                continue
            }
            const writes = count(source, kyou_write_call)
            const assigns = count(source, tx_id_assign)
            if (!source.includes("from '@/classes/gkill-tx'") || !source.includes('run_in_tx(')) {
                violations.push(`${name}: run_in_tx を通していない（add_*/update_* ${writes} 箇所）`)
                continue
            }
            if (assigns < writes) {
                violations.push(`${name}: add_*/update_* が ${writes} 箇所あるのに tx_id の代入が ${assigns} 箇所`)
            }
        }
        expect(violations, violations.join(' / ')).toEqual([])
    })

    it('tx を通さずに Kyou を書くファイルは単発書き込みの許可リストに限る', () => {
        const unexpected = writers
            .filter(({ name }) => !is_bundled_save_file(name) && !(name in SINGLE_WRITE_ALLOWLIST))
            .map(({ name }) => name)
        expect(unexpected, `tx を通さずに Kyou を書いている: ${unexpected.join(', ')}（束ねる相手が無い単発の update なら SINGLE_WRITE_ALLOWLIST へ理由付きで足す）`).toEqual([])
    })

    it('許可リストの各ファイルは今も Kyou を書き、run_in_tx を持たない', () => {
        for (const name of Object.keys(SINGLE_WRITE_ALLOWLIST)) {
            const file = writers.find(writer => writer.name === name)
            expect(file, `${name} が Kyou を書かなくなっている。許可リストから外すこと`).toBeDefined()
            expect(file!.source.includes('run_in_tx('), `${name} は tx を通すようになった。許可リストから外すこと`).toBe(false)
        }
    })
})
