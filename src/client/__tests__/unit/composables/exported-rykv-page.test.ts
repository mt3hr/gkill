/**
 * useExportedRykvPage の焼き込み読み取り。
 *
 * 「HTML で出力」したファイルは通信をしないので、データは HTML 自身へ焼き込まれた
 * <script type="application/json"> から読む。読めなければ「記録が含まれていません」を
 * 出して終わる（サーバへ取りに行くフォールバックを足してはいけない）。
 *
 * ガワだけの template.html を直接開いた場合もここに来る（差し込み前は中身が {}）。
 */
import { describe, it, expect, afterEach } from 'vitest'
import '@/classes/api/gkill-api'
import { useExportedRykvPage } from '@/classes/use-exported-rykv-page'

function put_json_element(element_id: string, text: string): void {
    const element = document.createElement('script')
    element.type = 'application/json'
    element.id = element_id
    element.textContent = text
    document.body.appendChild(element)
}

afterEach(() => {
    document.body.innerHTML = ''
})

describe('useExportedRykvPage の焼き込み読み取り', () => {
    it('差し込みが無ければ is_ready は false', () => {
        const { is_ready, share_title } = useExportedRykvPage()
        expect(is_ready.value).toBe(false)
        expect(share_title.value).toBe('')
    })

    it('ガワだけの {} でも is_ready は false（template.html を直接開いた場合）', () => {
        put_json_element('gkill-export-data', '{}')
        const { is_ready } = useExportedRykvPage()
        expect(is_ready.value, 'schema_version が無い差し込みを受け入れている').toBe(false)
    })

    it('壊れた JSON でも例外を投げず is_ready は false', () => {
        put_json_element('gkill-export-data', '{ こわれている')
        const { is_ready } = useExportedRykvPage()
        expect(is_ready.value).toBe(false)
    })

    it('schema_version のある差し込みを読み、タイトルを取り出す', () => {
        put_json_element('gkill-export-data', JSON.stringify({
            schema_version: 1,
            title: '出力したタイトル',
            locale_name: 'ja',
        }))
        const { is_ready, share_title } = useExportedRykvPage()
        expect(is_ready.value).toBe(true)
        expect(share_title.value).toBe('出力したタイトル')
    })

    it('タイトルが欠けていても読める（空文字になる）', () => {
        put_json_element('gkill-export-data', JSON.stringify({ schema_version: 1 }))
        const { is_ready, share_title } = useExportedRykvPage()
        expect(is_ready.value).toBe(true)
        expect(share_title.value).toBe('')
    })

    // ja 以外で出力したときは、サーバがメッセージも別の script タグへ差し込む。
    // 差し込みが無い（ja で出力した）ときに落ちないこと
    it('ロケールの差し込みが無くても落ちない', () => {
        put_json_element('gkill-export-data', JSON.stringify({
            schema_version: 1,
            title: 't',
            locale_name: 'en',
        }))
        expect(() => useExportedRykvPage()).not.toThrow()
    })

    it('ロケールの差し込みが壊れていても落ちない', () => {
        put_json_element('gkill-export-data', JSON.stringify({
            schema_version: 1,
            title: 't',
            locale_name: 'en',
        }))
        put_json_element('gkill-export-i18n', '{ こわれている')
        expect(() => useExportedRykvPage()).not.toThrow()
    })
})
