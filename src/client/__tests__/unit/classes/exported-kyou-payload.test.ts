/**
 * 焼き込み JSON の実体化。
 *
 * JSON.parse の結果はただのオブジェクトでメソッドが生えていないので、hydrate で
 * データクラスへ詰め替える（日時のキーは Date に戻る）。列は kyou_ids の参照で
 * 持っているので、ここで ID から本体へ解決する —— 列ごとに本体を持つと、
 * 列をまたいで同じ記録が出るぶんだけ出力が膨らむ。
 */
import { describe, it, expect, beforeEach } from 'vitest'
import { GkillAPIForExportedKyou } from '@/classes/api/gkill-api'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import {
    build_exported_api, parse_exported_payload, exported_payload_schema_version,
} from '@/classes/exported-kyou-payload'

function reset(): GkillAPIForExportedKyou {
    const api = GkillAPIForExportedKyou.get_instance_for_exported_kyou()
    api.columns = []
    api.kyous = []
    api.attached_notifications = []
    api.plugin_html = new Map()
    api.media = new Map()
    api.idf_text_bytes = new Map()
    api.narrow_column_kyous = null
    api.exported_application_config = new ApplicationConfig()
    return api
}

beforeEach(() => {
    reset()
})

describe('parse_exported_payload', () => {
    it('schema_version が数値でなければ null（ガワだけの template.html は {}）', () => {
        expect(parse_exported_payload({})).toBeNull()
        expect(parse_exported_payload(null)).toBeNull()
        expect(parse_exported_payload('文字列')).toBeNull()
        expect(parse_exported_payload({ schema_version: '1' })).toBeNull()
    })

    // 出力ページがマウント先を選ぶのはこの印だけ。
    // 知らない値と欠損を rykv へ寄せるので、**印を足す前の出力も読める**。
    // これのために schema_version を上げないという判断がここに依存する
    it('画面の印は mi だけを読み、欠損と未知の値は rykv へ寄せる', () => {
        const base = { schema_version: exported_payload_schema_version }
        expect(parse_exported_payload(base)?.screen, '印が無い出力が rykv にならない').toBe('rykv')
        expect(parse_exported_payload({ ...base, screen: 'rykv' })?.screen).toBe('rykv')
        expect(parse_exported_payload({ ...base, screen: 'mi' })?.screen).toBe('mi')
        expect(parse_exported_payload({ ...base, screen: 'dashboard' })?.screen,
            '知らない画面をそのまま返している').toBe('rykv')
        expect(parse_exported_payload({ ...base, screen: 42 })?.screen).toBe('rykv')
    })

    it('説明の欄が欠けていても既定で埋める', () => {
        const payload = parse_exported_payload({ schema_version: exported_payload_schema_version })
        expect(payload).not.toBeNull()
        expect(payload?.title).toBe('')
        expect(payload?.locale_name, 'ロケールの既定が ja でない').toBe('ja')
        expect(payload?.with_dnote).toBe(false)
        expect(payload?.with_map).toBe(false)
        expect(payload?.with_sidebar).toBe(false)
        expect(payload?.warnings).toEqual([])
    })

    it('画面のオプションは true のときだけ true（文字列の "true" で通さない）', () => {
        const payload = parse_exported_payload({
            schema_version: 1,
            with_dnote: true,
            with_map: 'true',
            with_sidebar: 1,
        })
        expect(payload?.with_dnote).toBe(true)
        expect(payload?.with_map).toBe(false)
        expect(payload?.with_sidebar).toBe(false)
    })
})

describe('build_exported_api', () => {
    it('Kyou を実体化し、日時のキーを Date へ戻す', () => {
        const api = build_exported_api({
            schema_version: 1,
            kyous: [{ id: 'a', related_time: '2026-07-01T09:00:00.000Z', data_type: 'kmemo' }],
        })
        expect(api.kyous).toHaveLength(1)
        expect(api.kyous[0].related_time, '日時が文字列のまま入っている').toBeInstanceOf(Date)
        expect(api.kyous[0].related_time.getTime()).toBe(new Date('2026-07-01T09:00:00.000Z').getTime())
        // メソッドが生えていること（素のオブジェクトをキャストしただけでは生えない）
        expect(typeof api.kyous[0].clone).toBe('function')
    })

    it('列は kyou_ids から本体へ解決する', () => {
        const api = build_exported_api({
            schema_version: 1,
            kyous: [{ id: 'a' }, { id: 'b' }, { id: 'c' }],
            columns: [
                { query: { query_id: 'q1' }, kyou_ids: ['a'] },
                { query: { query_id: 'q2' }, kyou_ids: ['b', 'c'] },
            ],
        })
        expect(api.columns).toHaveLength(2)
        expect(api.columns[0].query.query_id).toBe('q1')
        expect(api.columns[0].kyous.map(k => k.id)).toEqual(['a'])
        expect(api.columns[1].kyous.map(k => k.id)).toEqual(['b', 'c'])
        // 本体は和集合で1回だけ持つので、列は同じインスタンスを指す
        expect(api.columns[0].kyous[0]).toBe(api.kyous[0])
    })

    it('列をまたいで同じ記録を参照できる（本体は重複して持たない）', () => {
        const api = build_exported_api({
            schema_version: 1,
            kyous: [{ id: 'a' }],
            columns: [
                { query: { query_id: 'q1' }, kyou_ids: ['a'] },
                { query: { query_id: 'q2' }, kyou_ids: ['a'] },
            ],
        })
        expect(api.kyous).toHaveLength(1)
        expect(api.columns[0].kyous[0]).toBe(api.columns[1].kyous[0])
    })

    it('kyou_ids に実体の無い ID が混ざっていても落ちない（取りこぼすだけ）', () => {
        const api = build_exported_api({
            schema_version: 1,
            kyous: [{ id: 'a' }],
            columns: [{ query: { query_id: 'q1' }, kyou_ids: ['a', 'missing', 42] }],
        })
        expect(api.columns[0].kyous.map(k => k.id)).toEqual(['a'])
    })

    it('設定を実体化する（集計の定義も入る）', () => {
        const api = build_exported_api({
            schema_version: 1,
            application_config: { dnote_json_data: '[{"title":"定義A"}]', rykv_image_list_column_number: 4 },
        })
        expect(api.exported_application_config.dnote_json_data).toBe('[{"title":"定義A"}]')
        expect(api.exported_application_config.rykv_image_list_column_number).toBe(4)
    })

    it('プラグインの本文を辞書へ入れる（文字列以外は捨てる）', () => {
        const key = GkillAPIForExportedKyou.plugin_html_key('PluginRep', 'p1')
        const api = build_exported_api({
            schema_version: 1,
            plugin_html: { [key]: '<p>本文</p>', 'bad\u0000key': 42 },
        })
        expect(api.plugin_html.get(key)).toBe('<p>本文</p>')
        expect(api.plugin_html.has('bad\u0000key')).toBe(false)
    })

    it('付随データと GPSLog を実体化する', () => {
        const api = build_exported_api({
            schema_version: 1,
            attached_tags: [{ id: 't1', target_id: 'a', tag: 'タグA' }],
            attached_texts: [{ id: 'x1', target_id: 'a', text: 'テキスト' }],
            attached_notifications: [{ id: 'n1', target_id: 'a' }],
            gps_logs: [{ latitude: 35.1234, longitude: 139.5678, related_time: '2026-07-01T09:00:00.000Z' }],
        })
        expect(api.attached_tags.map(t => t.tag)).toEqual(['タグA'])
        expect(api.attached_texts.map(t => t.text)).toEqual(['テキスト'])
        expect(api.attached_notifications.map(n => n.id)).toEqual(['n1'])
        expect(api.gps_logs).toHaveLength(1)
        expect(api.gps_logs[0].related_time).toBeInstanceOf(Date)
    })


    it('メディアの対応表を辞書へ入れる（src が文字列のものだけ）', () => {
        const api = build_exported_api({
            schema_version: 1,
            media: {
                '/files/rep/a.jpg?thumb=400x400': { src: 'data:image/jpeg;base64,AAAA', kind: 'thumb' },
                '/files/rep/b.jpg': { kind: 'full' },
            },
        })
        expect(api.media.get('/files/rep/a.jpg?thumb=400x400')).toBe('data:image/jpeg;base64,AAAA')
        expect(api.media.has('/files/rep/b.jpg'), 'src が無い項目を入れている').toBe(false)
    })

    // 文字コードの判定は detect_and_decode_text が唯一の判定者。
    // サーバ側でデコードして文字列を焼くと2実装になって必ずずれるので、バイト列のまま持つ
    it('IDF の本文はバイト列のまま持つ', () => {
        // "abc" の base64
        const api = build_exported_api({
            schema_version: 1,
            idf_text_contents: { k1: { bytes_base64: 'YWJj', truncated: false } },
        })
        const bytes = api.idf_text_bytes.get('k1')
        expect(bytes).toBeInstanceOf(Uint8Array)
        expect(Array.from(bytes ?? [])).toEqual([0x61, 0x62, 0x63])
    })

    it('壊れた base64 は飛ばす（その記録の本文が出ないだけで画面は出す）', () => {
        const api = build_exported_api({
            schema_version: 1,
            idf_text_contents: {
                ok: { bytes_base64: 'YWJj' },
                broken: { bytes_base64: '!!! not base64 !!!' },
                wrong_type: { bytes_base64: 42 },
            },
        })
        expect(api.idf_text_bytes.has('ok')).toBe(true)
        expect(api.idf_text_bytes.has('broken')).toBe(false)
        expect(api.idf_text_bytes.has('wrong_type')).toBe(false)
    })
    it('配列の欄が欠けていても空で埋める（古い出力でも落ちない）', () => {
        const api = build_exported_api({ schema_version: 1 })
        expect(api.kyous).toEqual([])
        expect(api.columns).toEqual([])
        expect(api.attached_tags).toEqual([])
        expect(api.gps_logs).toEqual([])
        expect(api.plugin_html.size).toBe(0)
        expect(api.media.size).toBe(0)
        expect(api.idf_text_bytes.size).toBe(0)
    })
})

// 出力先のサイドバーは「焼いた記録を絞ることしかできない」。
// どんな条件を入れても、焼かれていない記録が現れることは無い（これが共有の境界線）。
describe('build_exported_api が注入する絞り込み', () => {
    // tags / reps の既定は [] = 0件指定（null = 絞り込み未使用ではない）。
    // サイドバーは全チェック状態を配列で載せてくるので、素の {query_id} ではなく
    // 「全部チェック済み」をこのテストの起点にする。ここを {query_id} だけにすると
    // 本体と同じ理由で全件落ちる
    const all_checked = { query_id: 'q1', tags: ['タグA', 'no tags'], reps: ['RepA', 'RepB'] }

    function make_api() {
        return build_exported_api({
            schema_version: 1,
            kyous: [
                { id: 'a', rep_name: 'RepA', data_type: 'kmemo', related_time: '2026-07-01T09:00:00.000+09:00' },
                { id: 'b', rep_name: 'RepB', data_type: 'kmemo', related_time: '2026-07-01T20:00:00.000+09:00' },
            ],
            attached_tags: [
                { id: 't1', target_id: 'a', tag: 'タグA', related_time: '2026-07-01T09:00:00.000+09:00' },
            ],
            columns: [{ query: all_checked, kyou_ids: ['a', 'b'] }],
        })
    }

    function narrow(api: ReturnType<typeof build_exported_api>, raw: object): Array<string> {
        const baked = api.columns[0]
        const query = FindKyouQuery.parse_find_kyou_query(raw)
        return (api.narrow_column_kyous?.(baked.kyous, query, baked.query) ?? []).map(k => k.id)
    }

    it('タグは焼き込みから先に埋める（遅延ロード待ちだとタグ条件が全部落ちる）', () => {
        const api = make_api()
        const a = api.kyous.find(k => k.id === 'a')
        expect(a?.is_attached_tags_loaded, 'タグが未ロード扱いのまま').toBe(true)
        expect(a?.attached_tags.map(t => t.tag)).toEqual(['タグA'])
        expect(api.kyous.find(k => k.id === 'b')?.attached_tags).toEqual([])
    })

    it('出力した時点の条件そのままなら、焼いた配列をそのまま返す', () => {
        const api = make_api()
        const baked = api.columns[0]
        const same = FindKyouQuery.parse_find_kyou_query(all_checked)
        // サーバが本文一致で入れた記録を「この条件では一致しない」と落とさないための経路
        expect(api.narrow_column_kyous?.(baked.kyous, same, baked.query)).toBe(baked.kyous)
    })

    it('タグで絞ると減る', () => {
        expect(narrow(make_api(), { ...all_checked, tags: ['タグA'] })).toEqual(['a'])
    })

    it('記録保管場所で絞ると減る', () => {
        expect(narrow(make_api(), { ...all_checked, reps: ['RepB'] })).toEqual(['b'])
    })

    it('期間で絞ると減る', () => {
        expect(narrow(make_api(), {
            ...all_checked,
            calendar_start_time: '2026-07-01T00:00:00.000+09:00',
            calendar_end_time: '2026-07-01T12:00:00.000+09:00',
        })).toEqual(['a'])
    })

    // ワードは Kyou 単体では判定できない。見ないので、ワードを理由に記録が落ちることは無い
    it('ワード条件は見ない（ワードを理由に記録を落とさない）', () => {
        expect(narrow(make_api(), {
            ...all_checked, words: ['どこにも無い語'], not_words: ['タグA'],
        }), 'ワードで絞ってしまっている').toEqual(['a', 'b'])
    })

    // ここが共有の境界線。どんな条件でも焼いた配列の部分集合しか返さない
    it('どんな条件でも焼いた配列の部分集合しか返さない', () => {
        const api = make_api()
        const baked = api.columns[0]
        const queries: Array<object> = [
            all_checked,
            { ...all_checked, tags: [] },
            { ...all_checked, tags: ['どこにも無いタグ'] },
            { ...all_checked, reps: [] },
            { ...all_checked, hide_tags: ['タグA'] },
            { ...all_checked, words: ['x'] },
            { ...all_checked, period_of_time_start_time_second: 0, period_of_time_end_time_second: 3600 },
            { ...all_checked, calendar_start_time: '2000-01-01T00:00:00.000+09:00' },
            { query_id: 'q1' },
            {},
        ]
        for (const raw of queries) {
            const query = FindKyouQuery.parse_find_kyou_query(raw)
            const result = api.narrow_column_kyous?.(baked.kyous, query, baked.query) ?? []
            for (const kyou of result) {
                expect(baked.kyous, `焼いた配列に無い記録が出た: ${JSON.stringify(raw)}`).toContain(kyou)
            }
        }
    })
})
