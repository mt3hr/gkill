/**
 * GkillAPIForExportedKyou の上書き。
 *
 * 共有ページ用の GkillAPIForSharedKyou を継承しているので、**上書きを忘れた点は
 * 共有版の挙動のまま静かに通る**。とくに次の2つは画面が動いているのに内容が違う形で壊れる。
 *   - get_kyous … 共有版はクエリを見ずに全件返す（出力では全列が同じ中身になる）
 *   - get_plugin_content_html … 共有版は上書きしていないので super の fetch に落ちる（通信する）
 *
 * このテストは上書き1点ずつを固定するもので、退行すると「出力した HTML の中身が違う」
 * または「出力した HTML が通信する」になる。
 */
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { GkillAPIForExportedKyou } from '@/classes/api/gkill-api'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { Kyou } from '@/classes/datas/kyou'
import { GPSLog } from '@/classes/datas/gps-log'
import { Notification } from '@/classes/datas/notification'
import { GetKyousRequest } from '@/classes/api/req_res/get-kyous-request'
import { GetApplicationConfigRequest } from '@/classes/api/req_res/get-application-config-request'
import { GetNotificationsByTargetIDRequest } from '@/classes/api/req_res/get-notifications-by-target-id-request'
import { GetNotificationHistoryByNotificationIDRequest } from '@/classes/api/req_res/get-notification-history-by-notification-id-request'
import { GetPluginContentHTMLRequest } from '@/classes/api/req_res/get-plugin-content-html-request'

function make_kyou(id: string): Kyou {
    const kyou = new Kyou()
    kyou.id = id
    return kyou
}

function make_query(query_id: string): FindKyouQuery {
    const query = new FindKyouQuery()
    query.query_id = query_id
    return query
}

/** 焼き込み前の素の状態へ戻す（シングルトンなのでテスト間で持ち越す） */
function reset(api: GkillAPIForExportedKyou): void {
    api.columns = []
    api.kyous = []
    api.attached_notifications = []
    api.attached_timeis_kyous = []
    api.plugin_html = new Map()
    api.media = new Map()
    api.idf_text_bytes = new Map()
    api.exported_application_config = new ApplicationConfig()
}

describe('GkillAPIForExportedKyou', () => {
    let api: GkillAPIForExportedKyou

    beforeEach(() => {
        api = GkillAPIForExportedKyou.get_instance_for_exported_kyou()
        reset(api)
        window.localStorage.clear()
    })

    describe('get_kyous は列ごとに違う結果を返す', () => {
        it('query_id で列を引く（共有版のように全件返さない）', async () => {
            const query1 = make_query('q1')
            const query2 = make_query('q2')
            api.kyous = [make_kyou('a'), make_kyou('b'), make_kyou('c')]
            api.columns = [
                { query: query1, kyous: [api.kyous[0]] },
                { query: query2, kyous: [api.kyous[1], api.kyous[2]] },
            ]

            const req1 = new GetKyousRequest()
            req1.query = query1
            const res1 = await api.get_kyous(req1)
            expect(res1.kyous.map(k => k.id), '列をまたいで同じ中身を返している').toEqual(['a'])

            const req2 = new GetKyousRequest()
            req2.query = query2
            const res2 = await api.get_kyous(req2)
            expect(res2.kyous.map(k => k.id)).toEqual(['b', 'c'])
        })

        it('焼き込まれていない列は空で返す（サーバへは取りに行かない）', async () => {
            api.kyous = [make_kyou('a')]
            api.columns = [{ query: make_query('q1'), kyous: api.kyous }]

            const req = new GetKyousRequest()
            req.query = make_query('q-unknown')
            const res = await api.get_kyous(req)
            expect(res.kyous).toEqual([])
        })

        // 絞り込みは注入されるが、起点は必ずその列の焼いた配列。
        // 「注入が何を返すか」ではなく「注入に渡るのが焼いた配列か」を見る
        it('絞り込みの注入には、その列の焼いた配列が渡る', async () => {
            api.kyous = [make_kyou('a'), make_kyou('b')]
            const query1 = make_query('q1')
            api.columns = [
                { query: query1, kyous: [api.kyous[0]] },
                { query: make_query('q2'), kyous: [api.kyous[1]] },
            ]
            const seen: Array<Array<string>> = []
            api.narrow_column_kyous = (kyous, _query, _baked) => {
                seen.push(kyous.map(k => k.id))
                return kyous
            }

            const req = new GetKyousRequest()
            req.query = query1
            await api.get_kyous(req)
            expect(seen, '別の列の記録まで渡している').toEqual([['a']])
        })

        it('絞り込みに渡るのは、画面の今の条件と焼いた条件の2つ', async () => {
            const baked = make_query('q1')
            baked.tags = ['タグA']
            api.kyous = [make_kyou('a')]
            api.columns = [{ query: baked, kyous: api.kyous }]
            let passed: { now: string[] | null, baked: string[] | null } | null = null
            api.narrow_column_kyous = (kyous, query, baked_query) => {
                passed = { now: query.tags, baked: baked_query.tags }
                return kyous
            }

            const req = new GetKyousRequest()
            req.query = make_query('q1')
            req.query.tags = ['タグB']
            await api.get_kyous(req)
            expect(passed).toEqual({ now: ['タグB'], baked: ['タグA'] })
        })

        it('注入が無ければ焼いたまま返す（サイドバーを含めずに出力したとき）', async () => {
            api.kyous = [make_kyou('a'), make_kyou('b')]
            const query = make_query('q1')
            api.columns = [{ query, kyous: api.kyous }]
            api.narrow_column_kyous = null

            const req = new GetKyousRequest()
            req.query = make_query('q1')
            req.query.tags = ['どこにも無いタグ']
            const res = await api.get_kyous(req)
            expect(res.kyous.map(k => k.id)).toEqual(['a', 'b'])
        })

        it('playing 検索は共有版の実装へ委譲する', async () => {
            api.attached_timeis_kyous = []
            const req = new GetKyousRequest()
            req.query = make_query('q1')
            req.query.playing_time = new Date(2026, 0, 1, 12, 0, 0, 0)
            const res = await api.get_kyous(req)
            // 打刻が1件も無いので0件。列の引き当てには行かない
            expect(res.kyous).toEqual([])
        })
    })

    it('get_application_config は焼き込んだ設定を返す（集計の定義が入っている）', async () => {
        api.exported_application_config.dnote_json_data = '[{"title":"定義A"}]'
        const res = await api.get_application_config(new GetApplicationConfigRequest())
        expect(res.application_config.dnote_json_data, '共有版の空の設定を返している（集計が空になる）')
            .toBe('[{"title":"定義A"}]')
        expect(res.application_config.for_export_kyou).toBe(true)
    })

    it('get_notifications_by_target_id は焼き込んだ通知から絞る（共有版は空のスタブ）', async () => {
        const notification = new Notification()
        notification.id = 'n1'
        notification.target_id = 'a'
        const other = new Notification()
        other.id = 'n2'
        other.target_id = 'b'
        api.attached_notifications = [notification, other]

        const req = new GetNotificationsByTargetIDRequest()
        req.target_id = 'a'
        const res = await api.get_notifications_by_target_id(req)
        expect(res.notifications.map(n => n.id)).toEqual(['n1'])
    })

    it('get_notification_history_by_notification_id も焼き込みから引く', async () => {
        const notification = new Notification()
        notification.id = 'n1'
        notification.target_id = 'a'
        api.attached_notifications = [notification]

        const req = new GetNotificationHistoryByNotificationIDRequest()
        req.id = 'n1'
        const res = await api.get_notification_history_by_notification_id(req)
        expect(res.notification_histories.map(n => n.id)).toEqual(['n1'])
    })

    describe('get_plugin_content_html は焼き込みから引く（共有版は未上書きで通信する）', () => {
        it('焼き込まれた本文を返す', async () => {
            api.plugin_html.set(GkillAPIForExportedKyou.plugin_html_key('PluginRep', 'p1'), '<p>本文</p>')
            const req = new GetPluginContentHTMLRequest()
            req.rep_name = 'PluginRep'
            req.kyou_id = 'p1'
            const res = await api.get_plugin_content_html(req)
            expect(res.html).toBe('<p>本文</p>')
        })

        it('焼き込みが無ければ空で返す（fetch へ落ちない）', async () => {
            const fetch_spy = vi.spyOn(globalThis, 'fetch')
            const req = new GetPluginContentHTMLRequest()
            req.rep_name = 'PluginRep'
            req.kyou_id = 'unknown'
            const res = await api.get_plugin_content_html(req)
            expect(res.html).toBe('')
            expect(fetch_spy, '出力 HTML が通信している').not.toHaveBeenCalled()
            fetch_spy.mockRestore()
        })
    })

    describe('Google Maps の API キーは閲覧者が入れる', () => {
        it('localStorage へ保存し、読み戻せる（Cookie は file:// で書けない）', () => {
            expect(api.get_google_map_api_key(), '既定で空でない').toBe('')
            api.set_google_map_api_key('viewer-key')
            expect(api.get_google_map_api_key()).toBe('viewer-key')
            expect(window.localStorage.getItem(GkillAPIForExportedKyou.google_map_api_key_storage_key))
                .toBe('viewer-key')
        })

        // /files/ 経由で開かれると allow-same-origin の無い sandbox（opaque origin）になり、
        // localStorage は例外を投げる。ここで投げると地図ごと画面が落ちる
        it('localStorage が例外を投げてもメモリへ退避して落ちない', () => {
            const set_spy = vi.spyOn(window.localStorage, 'setItem').mockImplementation(() => {
                throw new Error('opaque origin')
            })
            const get_spy = vi.spyOn(window.localStorage, 'getItem').mockImplementation(() => {
                throw new Error('opaque origin')
            })

            expect(() => api.set_google_map_api_key('viewer-key')).not.toThrow()
            expect(api.get_google_map_api_key(), '退避先から読めていない').toBe('viewer-key')

            set_spy.mockRestore()
            get_spy.mockRestore()
        })
    })


    describe('メディアの URL は焼き込みへ差し替える', () => {
        it('焼き込んだ URL をそのまま引く（build_media_url が渡す形）', () => {
            api.media.set('/files/rep/a.jpg?thumb=400x400', 'data:image/jpeg;base64,AAAA')
            expect(api.resolve_media_url('/files/rep/a.jpg?thumb=400x400')).toBe('data:image/jpeg;base64,AAAA')
        })

        // markdown-to-html.ts は base_url で解決した**絶対 URL**を渡してくる。
        // 前方一致だけで判定すると、ここを素通りして通信する
        it('オリジン付きの絶対 URL でも、パス + クエリで引き直す', () => {
            api.media.set('/files/rep/a.jpg', 'data:image/jpeg;base64,AAAA')
            const absolute = new URL('/files/rep/a.jpg', document.baseURI).toString()
            expect(api.resolve_media_url(absolute)).toBe('data:image/jpeg;base64,AAAA')
        })

        // 一覧は is_image_request_to_thumb_size=true でサムネの鍵を引くが、
        // 詳細ペインとダイアログは原寸の鍵を引く（rykv-view.vue / kyou-dialog.vue が false を渡す）。
        // サーバは「サムネイルのみ」で出力したとき原寸を焼かないので、落とさないと
        // 一覧には出るのに詳細とダイアログだけ画像が消える
        it('原寸が焼かれていなければサムネの鍵へ落とす', () => {
            api.media.set('/files/rep/a.jpg?thumb=400x400', 'data:image/jpeg;base64,AAAA')
            expect(api.resolve_media_url('/files/rep/a.jpg')).toBe('data:image/jpeg;base64,AAAA')
        })

        it('動画のポスターも同じように落とす', () => {
            api.media.set('/files/rep/a.mp4?is_video=true&thumb=400x400', 'data:image/jpeg;base64,BBBB')
            expect(api.resolve_media_url('/files/rep/a.mp4')).toBe('data:image/jpeg;base64,BBBB')
        })

        it('オリジン付きの絶対 URL でもサムネへ落とす', () => {
            api.media.set('/files/rep/a.jpg?thumb=400x400', 'data:image/jpeg;base64,AAAA')
            const absolute = new URL('/files/rep/a.jpg', document.baseURI).toString()
            expect(api.resolve_media_url(absolute)).toBe('data:image/jpeg;base64,AAAA')
        })

        // 原寸が焼かれているならそれを使う（サムネへ落とさない）
        it('原寸が焼かれていればサムネへ落とさない', () => {
            api.media.set('/files/rep/a.jpg', 'data:image/jpeg;base64,FULL')
            api.media.set('/files/rep/a.jpg?thumb=400x400', 'data:image/jpeg;base64,THUMB')
            expect(api.resolve_media_url('/files/rep/a.jpg')).toBe('data:image/jpeg;base64,FULL')
        })

        it('焼き込まれていない gkill の経路は about:blank にする', () => {
            expect(api.resolve_media_url('/files/rep/missing.jpg')).toBe('about:blank')
            expect(api.resolve_media_url('/zip_cache/rep/h/x.jpg')).toBe('about:blank')
            expect(api.resolve_media_url(new URL('/files/rep/missing.jpg', document.baseURI).toString()))
                .toBe('about:blank')
        })

        it('外部サイトの URL は残す（記録の中身を失わない）', () => {
            expect(api.resolve_media_url('https://example.com/x.png')).toBe('https://example.com/x.png')
        })

        it('data: や mailto: はそのまま', () => {
            expect(api.resolve_media_url('data:image/png;base64,AAAA')).toBe('data:image/png;base64,AAAA')
            expect(api.resolve_media_url('mailto:someone@example.com')).toBe('mailto:someone@example.com')
        })
    })

    // 地図の時間窓は rykv の別状態で初期値が「今」。列の期間へ揃えるのは
    // 期間フィルタが両端とも非 null のときだけなので、期間を指定していない列を出力すると
    // 「今〜今」のまま 0 件になる。焼いたものが必ず見えるように、端から端を返す
    describe('焼き込んだ GPSLog の範囲を返す（地図の時間窓の初期値）', () => {
        it('端から端を返す', () => {
            const a = new GPSLog()
            a.related_time = new Date('2026-07-01T09:00:00.000+09:00')
            const b = new GPSLog()
            b.related_time = new Date('2026-07-01T18:00:00.000+09:00')
            const c = new GPSLog()
            c.related_time = new Date('2026-07-01T12:00:00.000+09:00')
            api.gps_logs = [b, a, c]

            const window = api.get_exported_gps_log_window()
            expect(window).not.toBeNull()
            expect(window?.start_time.getTime()).toBe(a.related_time.getTime())
            expect(window?.end_time.getTime()).toBe(b.related_time.getTime())
        })

        it('0件なら null（窓を触らせない）', () => {
            api.gps_logs = []
            expect(api.get_exported_gps_log_window()).toBeNull()
        })
    })

    describe('IDF のテキスト本文は焼き込みから返す', () => {
        it('焼き込んだバイト列を返す（文字コードの判定は呼び出し側のまま）', () => {
            const bytes = new Uint8Array([0x61, 0x62, 0x63])
            api.idf_text_bytes.set('k1', bytes)
            expect(api.get_exported_idf_text_bytes('k1')).toBe(bytes)
        })

        it('焼き込みが無ければ null（基底と同じ＝従来どおり fetch する側へ倒す）', () => {
            expect(api.get_exported_idf_text_bytes('unknown')).toBeNull()
        })
    })
    describe('get_saved_rykv_find_kyou_querys は列の復元経路', () => {
        it('焼き込んだ列の検索条件を返す（共有版は既定クエリ1本）', () => {
            api.columns = [
                { query: make_query('q1'), kyous: [] },
                { query: make_query('q2'), kyous: [] },
            ]
            expect(api.get_saved_rykv_find_kyou_querys().map(q => q.query_id)).toEqual(['q1', 'q2'])
        })

        it('列が無ければ共有版の既定（1本）に委譲する', () => {
            expect(api.get_saved_rykv_find_kyou_querys()).toHaveLength(1)
        })
    })
})
