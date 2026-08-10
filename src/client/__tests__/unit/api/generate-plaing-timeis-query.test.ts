import { describe, test, expect } from 'vitest'
import { generate_plaing_timeis_query } from '@/classes/api/find_query/generate-plaing-timeis-query'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'

// generate_plaing_timeis_query が読むフィールドだけを持つ最小のApplicationConfig。
// 実クラスを使うと gkill-api.ts のimport副作用を引き込むため、構造互換で代用する
function make_application_config(overrides: Record<string, unknown> = {}): ApplicationConfig {
    return {
        for_share_kyou: false,
        plaing_timeis_json_data: undefined,
        rep_struct: {
            rep_name: '', children: [
                { rep_name: 'Kmemo_device1_all', children: [] },
                {
                    rep_name: 'TimeIs_device1_all', children: [
                        { rep_name: 'TimeIs_nested_all', children: [] },
                    ],
                },
            ],
        },
        tag_struct: {
            tag_name: '', check_when_inited: false, is_force_hide: false, children: [
                { tag_name: 'tag_checked', check_when_inited: true, is_force_hide: false, children: [] },
                { tag_name: 'tag_unchecked', check_when_inited: false, is_force_hide: false, children: [] },
                { tag_name: 'tag_hidden', check_when_inited: false, is_force_hide: true, children: [] },
            ],
        },
        ...overrides,
    } as unknown as ApplicationConfig
}

// PlaingTimeIsConfig.parse が受け取る形のJSONを実クラスの直列化で作る。
// グループの有効/無効は値の null 判定で表す（words=[] でキーワードグループ有効化など）
function make_saved_query_json(setup: (query: FindKyouQuery) => void): Record<string, unknown> {
    const query = new FindKyouQuery()
    setup(query)
    return { plaing_timeis_find_kyou_query: JSON.parse(JSON.stringify(query)) }
}

describe('generate_plaing_timeis_query', () => {
    const plaing_time = new Date('2026-08-09T12:00:00Z')

    test('application_config が null なら基礎クエリのみ(従来のバグ互換)', () => {
        const query = generate_plaing_timeis_query(null, plaing_time)
        // 非nullの plaing_time が実行中検索を表す
        expect(query.plaing_time).toBe(plaing_time)
        // タグフィルタは未使用（null。旧 use_tags=false と等価）
        expect(query.tags).toBeNull()
        // reps はコンストラクタ既定の []（有効・チェック0個=0件）のまま
        expect(query.reps).toEqual([])
    })

    test('記録タイプはカスタム条件の有無によらずTimeIs固定', () => {
        for (const config of [null, make_application_config(), make_application_config({
            plaing_timeis_json_data: make_saved_query_json(saved => { saved.words = [] }),
        })]) {
            const query = generate_plaing_timeis_query(config, plaing_time)
            expect(query.rep_types).toEqual(['timeis'])
        }
    })

    test('for_share_kyou ならrep/tag構造にもカスタム条件にも触れない', () => {
        // rep_struct / tag_struct を持たないモック。触ればthrowする
        const config = {
            for_share_kyou: true,
            plaing_timeis_json_data: make_saved_query_json(saved => {
                saved.words = []
                saved.keywords = 'should-be-ignored'
            }),
        } as unknown as ApplicationConfig
        const query = generate_plaing_timeis_query(config, plaing_time)
        expect(query.plaing_time).toBe(plaing_time)
        expect(query.tags).toBeNull()
        expect(query.reps).toEqual([])
        // キーワードグループは未使用（null）のまま
        expect(query.words).toBeNull()
        expect(query.keywords).toBe('')
    })

    test('未設定なら従来の既定動作(全rep + タグフィルタ未使用、非表示タグなし)', () => {
        const query = generate_plaing_timeis_query(make_application_config(), plaing_time)
        expect(query.plaing_time).toBe(plaing_time)
        expect(query.tags).toBeNull()
        expect(query.reps).toEqual(['Kmemo_device1_all', 'TimeIs_device1_all', 'TimeIs_nested_all'])
        expect(query.hide_tags).toEqual([])
    })

    test('保存クエリが明示的にnullでも既定動作', () => {
        const config = make_application_config({
            plaing_timeis_json_data: { plaing_timeis_find_kyou_query: null },
        })
        const query = generate_plaing_timeis_query(config, plaing_time)
        expect(query.reps).toEqual(['Kmemo_device1_all', 'TimeIs_device1_all', 'TimeIs_nested_all'])
        expect(query.tags).toBeNull()
    })

    test('保存クエリありなら対象フィールドだけコピーされる', () => {
        const config = make_application_config({
            plaing_timeis_json_data: make_saved_query_json(saved => {
                saved.words = [] // キーワードグループを有効化（null=未使用）
                saved.keywords = 'foo -bar'
                saved.words_and = true
                saved.tags = ['tag_checked']
                saved.tags_and = true
            }),
        })
        const query = generate_plaing_timeis_query(config, plaing_time)
        expect(query.keywords).toBe('foo -bar')
        expect(query.words_and).toBe(true)
        // parse_words_and_not_words が keywords から導出する
        expect(query.words).toEqual(['foo'])
        expect(query.not_words).toEqual(['bar'])
        expect(query.tags).toEqual(['tag_checked'])
        expect(query.tags_and).toBe(true)
        // 非表示タグは現在の設定から再適用される
        expect(query.hide_tags).toEqual(['tag_hidden'])
    })

    test('保存クエリの対象外フィールドは既定値のまま', () => {
        const config = make_application_config({
            plaing_timeis_json_data: make_saved_query_json(saved => {
                saved.calendar_start_date = new Date('2026-01-01T00:00:00Z')
                saved.calendar_end_date = new Date('2026-01-31T00:00:00Z')
                saved.map_latitude = 35.1
                saved.map_longitude = 139.2
                saved.map_radius = 300
                saved.timeis_words = ['作業']
                saved.for_mi = true
                saved.is_image_only = true
            }),
        })
        const query = generate_plaing_timeis_query(config, plaing_time)
        const defaults = new FindKyouQuery()
        expect(query.calendar_start_date).toBeNull()
        expect(query.calendar_end_date).toBeNull()
        expect(query.map_latitude).toBeNull()
        expect(query.timeis_words).toBeNull()
        expect(query.for_mi).toBe(defaults.for_mi)
        expect(query.is_image_only).toBe(defaults.is_image_only)
    })

    test('保存クエリの記録タイプ指定は無視されてTimeIsに固定される', () => {
        const config = make_application_config({
            plaing_timeis_json_data: make_saved_query_json(saved => {
                saved.rep_types = ['mi']
            }),
        })
        const query = generate_plaing_timeis_query(config, plaing_time)
        expect(query.rep_types).toEqual(['timeis'])
    })

    test('保存クエリ由来の plaing_time は常に強制上書きされる', () => {
        const config = make_application_config({
            plaing_timeis_json_data: make_saved_query_json(saved => {
                saved.plaing_time = new Date('2000-01-01T00:00:00Z')
            }),
        })
        const query = generate_plaing_timeis_query(config, plaing_time)
        expect(query.plaing_time).toBe(plaing_time)
    })

    /**
     * 記録保管場所はエディタから消えたので、保存JSONに残っていても使わない。
     * サーバの意味論は「reps=null は絞らない / reps=[] は0件」で、
     * new FindKyouQuery() の既定は reps=[]（有効・チェック0個=0件）。
     * カスタム条件適用時に放置するとrep名絞り込みで常に0件になるため、
     * 必ず null（絞らない）へ倒す。
     */
    test('保存クエリのrep指定は無視され、rep名絞り込みは未使用(null=絞らない)になる', () => {
        const config = make_application_config({
            plaing_timeis_json_data: make_saved_query_json(saved => {
                saved.reps = ['TimeIs_device1_all']
            }),
        })
        const query = generate_plaing_timeis_query(config, plaing_time)
        expect(query.reps).toBeNull()
    })
})
