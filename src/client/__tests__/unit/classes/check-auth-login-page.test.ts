/**
 * ログイン画面での「セッション無効 → ログイン画面へ飛ばす」の抑止。
 *
 * `check_auth` は API 応答のエラーコードを見て、セッションが無効なら
 * ブラウザ側の状態を掃除して `location.replace("/")` する。
 * ところが**ログインの失敗そのものも同じコードを返す**
 * （存在しないユーザIDは `handle_login.go` が ERR000002 = AccountNotFoundError）。
 *
 * ログイン画面で飛ばすと、行き先が同じ `/` なのにページを作り直すことになり、
 * `login-page.vue` がいま出したばかりのエラー表示が消える。
 * 利用者から見ると「画面が一瞬光って、理由も出ないまま元のまま」になる。
 *
 * E2E の `login.spec.ts` の「login with invalid credentials shows error」は
 * この作り直しとの競争で、負荷が高いときだけ落ちていた（2026-08-19 修正）。
 */
import { describe, test, expect } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { is_on_login_page } from '@/classes/api/gkill-api'

describe('is_on_login_page', () => {
    test('ルートはログイン画面', () => {
        // router の login ルートは path: '/'
        expect(is_on_login_page('/')).toBe(true)
    })

    test('pathname が空でもログイン画面とみなす', () => {
        // 実ブラウザでは空にならないが、落とすと「飛ばさない」側へ倒れず
        // 判定の穴になるので明示しておく
        expect(is_on_login_page('')).toBe(true)
    })

    test('他の画面はログイン画面ではない（セッション切れでは飛ばす必要がある）', () => {
        for (const pathname of [
            '/kftl', '/rykv', '/mi', '/saihate', '/dashboard', '/plaing',
            '/mkfl', '/rudbeckia', '/kyou', '/shared_page',
            '/register_first_account', '/set_new_password',
        ]) {
            expect(is_on_login_page(pathname), `${pathname} をログイン画面と誤判定している`).toBe(false)
        }
    })
})

describe('check_auth への配線（ソース走査）', () => {
    // 純関数が正しくても、check_auth から呼んでいなければ何も直らない。
    // 実ブラウザの location を差し替える試験は jsdom では脆いので、
    // 「早期returnが replace より前にあること」をソースで固定する
    const source = readFileSync(resolve(__dirname, '../../../classes/api/gkill-api.ts'), 'utf-8')
    const check_auth_body = source.slice(source.indexOf('check_auth(res: GkillAPIResponse)'))

    test('check_auth が is_on_login_page で早期returnする', () => {
        expect(check_auth_body).toContain('is_on_login_page(window.location.pathname)')
    })

    test('その判定は location.replace("/") より前にある', () => {
        const guard_at = check_auth_body.indexOf('is_on_login_page(window.location.pathname)')
        const replace_at = check_auth_body.indexOf('window.location.replace("/")')
        expect(guard_at, 'ログイン画面の判定が check_auth から消えている').toBeGreaterThan(-1)
        expect(replace_at, 'ログイン画面へ飛ばす処理が見つからない').toBeGreaterThan(-1)
        expect(guard_at, '判定が replace より後ろにある（先に飛んでしまう）').toBeLessThan(replace_at)
    })
})
