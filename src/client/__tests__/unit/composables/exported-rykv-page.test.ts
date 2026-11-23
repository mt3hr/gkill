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
import type { Ref } from 'vue'

function put_json_element(element_id: string, text: string): void {
    const element = document.createElement('script')
    element.type = 'application/json'
    element.id = element_id
    element.textContent = text
    document.body.appendChild(element)
}

/**
 * `is_ready` が立つまで待つ。
 *
 * 焼き込みを流し込んだあとサイドバーのチェックツリーを組み終えてから立つので、
 * 読み取りだけなら同期でも、`is_ready` は数マイクロタスク後になる
 */
async function wait_ready(is_ready: Ref<boolean>): Promise<void> {
    for (let i = 0; i < 50 && !is_ready.value; i++) {
        await Promise.resolve()
    }
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

    it('schema_version のある差し込みを読み、タイトルを取り出す', async () => {
        put_json_element('gkill-export-data', JSON.stringify({
            schema_version: 1,
            title: '出力したタイトル',
            locale_name: 'ja',
        }))
        const { is_ready, share_title } = useExportedRykvPage()
        await wait_ready(is_ready)
        expect(is_ready.value).toBe(true)
        expect(share_title.value).toBe('出力したタイトル')
    })

    it('タイトルが欠けていても読める（空文字になる）', async () => {
        put_json_element('gkill-export-data', JSON.stringify({ schema_version: 1 }))
        const { is_ready, share_title } = useExportedRykvPage()
        await wait_ready(is_ready)
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

// サイドバーのチェックツリーは application_config の木から描かれる。
// 利用者の設定した木をそのまま焼くと、**出力に含めなかったタグ名・rep 名まで共有先に見える**。
// ここでは焼いたデータから組み直すので、木に出るのは焼いたものだけになる。
describe('サイドバーのチェックツリーは焼いたデータから組む', () => {
    function put_payload(): void {
        put_json_element('gkill-export-data', JSON.stringify({
            schema_version: 1,
            title: 'サンプル',
            locale_name: 'ja',
            with_sidebar: true,
            kyous: [
                { id: 'a', rep_name: 'RepA', data_type: 'kmemo', related_time: '2026-07-01T09:00:00.000+09:00' },
            ],
            attached_tags: [
                { id: 't1', target_id: 'a', tag: 'タグA', related_time: '2026-07-01T09:00:00.000+09:00' },
            ],
            columns: [{
                query: { query_id: 'q1', tags: ['タグA', 'no tags'], reps: ['RepA'] },
                kyou_ids: ['a'],
            }],
        }))
    }

    function tag_names(config: { tag_struct: { children: Array<{ tag_name: string }> | null } }): Array<string> {
        return (config.tag_struct.children ?? []).map(child => child.tag_name)
    }

    it('木が組み終わるまで is_ready は立たない', async () => {
        put_payload()
        const { is_ready, application_config } = useExportedRykvPage()
        // 立った瞬間には木がもう組まれていること。
        // 先に画面を描くと、サイドバーの各節が「チェック0個」を生成して emit し、
        // tags = [] / reps = []（非 null の空配列 = 0件指定）で**列が全部空になる**
        expect(is_ready.value, '木を組む前に画面を描こうとしている').toBe(false)
        await wait_ready(is_ready)
        expect(tag_names(application_config.value).length, '立ったのに木が空').not.toBe(0)
    })

    it('焼いたタグと「タグ無し」が木に出る', async () => {
        put_payload()
        const { is_ready, application_config } = useExportedRykvPage()
        await wait_ready(is_ready)
        const names = tag_names(application_config.value)
        expect(names).toContain('タグA')
        expect(names, '「タグ無し」が無いと、タグの付いていない記録が一覧から消える').toContain('no tags')
        expect(names, '焼いていないタグが木に出ている').not.toContain('タグB')
    })

    it('木のルートは入れ物にする（葉のままだと子が描かれない）', async () => {
        put_payload()
        const { is_ready, application_config } = useExportedRykvPage()
        await wait_ready(is_ready)
        for (const root of [
            application_config.value.tag_struct,
            application_config.value.rep_struct,
            application_config.value.device_struct,
            application_config.value.rep_type_struct,
        ]) {
            expect(root.is_dir, 'ルートが葉のまま').toBe(true)
        }
    })

    // 木は利用者が設定画面で並べて初めて作られる。一度も触っていない利用者の設定には
    // 入っておらず、サーバは null を返す。焼き込みの JSON にも null で乗り、
    // `hydrate` は source の全キーを写すので `new ApplicationConfig()` の既定ごと潰れる。
    // 直前まで実機では再現しなかった（設定を触っている利用者では木があるため）
    it('木が null で焼かれていても組み直す（設定を一度も触っていない利用者）', async () => {
        put_json_element('gkill-export-data', JSON.stringify({
            schema_version: 1,
            title: 'サンプル',
            locale_name: 'ja',
            with_sidebar: true,
            application_config: {
                tag_struct: null,
                rep_struct: null,
                device_struct: null,
                rep_type_struct: null,
            },
            kyous: [
                { id: 'a', rep_name: 'RepA', data_type: 'kmemo', related_time: '2026-07-01T09:00:00.000+09:00' },
            ],
            attached_tags: [
                { id: 't1', target_id: 'a', tag: 'タグA', related_time: '2026-07-01T09:00:00.000+09:00' },
            ],
            columns: [{
                query: { query_id: 'q1', tags: ['タグA', 'no tags'], reps: ['RepA'] },
                kyou_ids: ['a'],
            }],
        }))
        const { is_ready, application_config } = useExportedRykvPage()
        await wait_ready(is_ready)
        expect(is_ready.value, '木を組む途中で例外になっている').toBe(true)
        expect(tag_names(application_config.value)).toContain('タグA')
        expect((application_config.value.rep_struct.children ?? []).map(child => child.rep_name)).toEqual(['RepA'])
    })

    // 焼いた記録には、チェックしていないタグも付いている（タグAで引っかかった記録が
    // タグBも持っている、など）。そのまま選択肢に出すと、**出力に含めていないタグ名が
    // 渡した相手に見える**
    it('チェックされていなかったタグ・記録保管場所は選択肢に出さない', async () => {
        put_json_element('gkill-export-data', JSON.stringify({
            schema_version: 1,
            title: 'サンプル',
            locale_name: 'ja',
            with_sidebar: true,
            kyous: [
                { id: 'a', rep_name: 'RepA', data_type: 'kmemo', related_time: '2026-07-01T09:00:00.000+09:00' },
                { id: 'b', rep_name: 'RepB', data_type: 'kmemo', related_time: '2026-07-01T10:00:00.000+09:00' },
            ],
            attached_tags: [
                { id: 't1', target_id: 'a', tag: 'タグA', related_time: '2026-07-01T09:00:00.000+09:00' },
                { id: 't2', target_id: 'a', tag: 'タグB', related_time: '2026-07-01T09:00:00.000+09:00' },
            ],
            // チェックされていたのは タグA と RepA だけ
            columns: [{ query: { query_id: 'q1', tags: ['タグA'], reps: ['RepA'] }, kyou_ids: ['a', 'b'] }],
        }))
        const { is_ready, application_config } = useExportedRykvPage()
        await wait_ready(is_ready)

        const names = tag_names(application_config.value)
        expect(names).toContain('タグA')
        expect(names, 'チェックされていないタグが選択肢に出ている').not.toContain('タグB')
        expect(names, '「タグ無し」も外されていたのに出ている').not.toContain('no tags')

        const rep_names = (application_config.value.rep_struct.children ?? []).map(child => child.rep_name)
        expect(rep_names, 'チェックされていない記録保管場所が選択肢に出ている').toEqual(['RepA'])
    })

    it('ルートの id は木ごとに別（空のままだと4本が同じ id になる）', async () => {
        put_payload()
        const { is_ready, application_config } = useExportedRykvPage()
        await wait_ready(is_ready)
        const ids = [
            application_config.value.tag_struct,
            application_config.value.rep_struct,
            application_config.value.device_struct,
            application_config.value.rep_type_struct,
        ].map(root => root.id)
        expect(new Set(ids).size, 'ルートの id が重なっている').toBe(ids.length)
    })

    it('記録保管場所の木は、焼いた記録に現れる rep だけになる', async () => {
        put_payload()
        const { is_ready, application_config } = useExportedRykvPage()
        await wait_ready(is_ready)
        const rep_names = (application_config.value.rep_struct.children ?? []).map(child => child.rep_name)
        expect(rep_names).toEqual(['RepA'])
    })
})
