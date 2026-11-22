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
import {
    build_exported_api, parse_exported_payload, exported_payload_schema_version,
} from '@/classes/exported-kyou-payload'

function reset(): GkillAPIForExportedKyou {
    const api = GkillAPIForExportedKyou.get_instance_for_exported_kyou()
    api.columns = []
    api.kyous = []
    api.attached_notifications = []
    api.plugin_html = new Map()
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

    it('配列の欄が欠けていても空で埋める（古い出力でも落ちない）', () => {
        const api = build_exported_api({ schema_version: 1 })
        expect(api.kyous).toEqual([])
        expect(api.columns).toEqual([])
        expect(api.attached_tags).toEqual([])
        expect(api.gps_logs).toEqual([])
        expect(api.plugin_html.size).toBe(0)
    })
})
