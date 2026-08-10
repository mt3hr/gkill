/**
 * InfoBase の AbortController 遅延生成の検証。
 *
 * 数十万件の検索応答を実体化するとき、1件ごとの AbortController 生成が
 * 合計で数百msを占めていたため、初アクセスまで作らない構造にした。
 * ほとんどのインスタンスは一度も abort_controller を使わない。
 */
import { describe, test, expect } from 'vitest'
import { reactive } from 'vue'
import '@/classes/api/gkill-api'
import { Kyou } from '@/classes/datas/kyou'

describe('InfoBase の AbortController 遅延生成', () => {
    test('初アクセスまで生成されない', () => {
        const kyou = new Kyou()
        expect((kyou as unknown as Record<string, unknown>)._abort_controller).toBeNull()
    })

    test('getterの初回アクセスで生成され、2回目以降は同一インスタンスを返す', () => {
        const kyou = new Kyou()
        const first = kyou.abort_controller
        expect(first).toBeInstanceOf(AbortController)
        expect(kyou.abort_controller).toBe(first)
    })

    test('setterで差し替えられる', () => {
        const kyou = new Kyou()
        const replacement = new AbortController()
        kyou.abort_controller = replacement
        expect(kyou.abort_controller).toBe(replacement)
    })

    // KyouはVueのreactive経由で使われる。ES private(#)だとproxyのthisで壊れるため
    // TS privateにしてある。この性質が保たれていることの回帰テスト
    test('Vueのreactive proxy越しでもgetter/setterが動く', () => {
        const kyou = reactive(new Kyou())
        const controller = kyou.abort_controller
        expect(controller).toBeInstanceOf(AbortController)
        expect(controller.signal.aborted).toBe(false)
        expect(kyou.abort_controller).toBe(controller)

        const replacement = new AbortController()
        kyou.abort_controller = replacement
        replacement.abort()
        expect(kyou.abort_controller.signal.aborted).toBe(true)
    })

    // リクエストbodyやDnoteエクスポートのJSONにabort_controllerキーが出ない
    // (以前は "abort_controller":{} が混ざっていた。サーバは未知フィールドを無視する)
    test('JSON.stringifyにabort_controllerキーが現れない', () => {
        const kyou = new Kyou()
        // 生成させてもJSONへは出ないこと
        expect(kyou.abort_controller.signal.aborted).toBe(false)
        const json = JSON.parse(JSON.stringify(kyou)) as Record<string, unknown>
        expect('abort_controller' in json).toBe(false)
        // backing fieldはAbortControllerインスタンス(列挙可能プロパティ無し)なので{}になる
        expect(json._abort_controller).toEqual({})
    })
})
