/**
 * Ryuu（流）の中継チェーンが RyuuItemView のイベントを親まで通すことを、実際にマウントして確かめる。
 *
 * Ryuu は rykv の詳細ペインに出るだけで DialogHost を持たないので、
 * ダブルクリックで KyouDialog を開くのも、タグを足した通知を列へ返すのも、
 * すべて上位への中継頼み。ここが1つでも欠けると
 * 「Ryuu からはダイアログが開かない」「Ryuu で編集しても一覧が古いまま」になる。
 *
 * 束を手書きで並べていた頃は kyou_view_relay_event_names の網羅チェック(Exclude<>)の
 * 外にあり、KyouViewRelayArgs へイベントを足しても Ryuu からだけ黙って落ちていた。
 * composable 単体では「束に全イベントが入っているか」までしか見えず、
 * テンプレートで v-on を張り忘れていないかはマウントしないと分からない。
 */
import { describe, expect, test, vi } from 'vitest'
import { shallowMount } from '@vue/test-utils'

// req_res は GkillAPIRequest を継承する。GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の
// 循環importがあるため、本番同様に gkill-api を先に評価させる
import '@/classes/api/gkill-api'

vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

// 末端の子は Vuetify を引き込む（vitest は node_modules の .css を解決できない）。
// shallowMount のスタブは描画時にしか効かず import は走ってしまうので、モジュールごと差し替える。
// テンプレート ref 経由で load_related_kyou() が呼ばれるので、そこだけ生やしておく
vi.mock('@/pages/views/ryuu-item-view.vue', () => ({
    default: {
        name: 'ryuu-item-view',
        template: '<div />',
        methods: {
            load_related_kyou(): Promise<void> { return Promise.resolve() },
        },
    },
}))
vi.mock('@/pages/dialogs/add-ryuu-item-dialog.vue', () => ({
    default: { name: 'add-ryuu-item-dialog', template: '<div />' },
}))

import RyuuView from '@/pages/views/ryuu-view.vue'
import { kyou_view_relay_event_names } from '@/classes/kyou-view-relay'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { Kyou } from '@/classes/datas/kyou'

function make_ryuu_json_data(): Array<Record<string, unknown>> {
    return [{
        name: '既定',
        queries: [{
            id: 'related-query-1',
            title: '前後',
            prefix: '',
            suffix: '',
            predicate: { logic: 'AND', predicates: [] },
            related_time_match_type: 0,
            find_kyou_query: null,
            find_duration_hour: 1,
        }],
    }]
}

async function mount_ryuu_view() {
    const wrapper = shallowMount(RyuuView, {
        props: {
            gkill_api: {},
            application_config: { ryuu_json_data: make_ryuu_json_data() },
            find_kyou_query_default: new FindKyouQuery(),
            target_kyou: { id: 'kyou-1' } as unknown as Kyou,
            matched_kyous: null,
            editable: false,
        } as never,
    })

    // ryuu_definitions は nextTick を2回またいで application_config から入る
    for (let i = 0; i < 4; i++) {
        await wrapper.vm.$nextTick()
    }

    const child = wrapper.findComponent({ name: 'ryuu-item-view' })
    expect(child.exists(), 'RyuuItemView が描画されていない').toBe(true)
    return { wrapper: wrapper, child: child }
}

describe('Ryuu の中継チェーン', () => {
    test.each(kyou_view_relay_event_names)('%s を親へ通す', async (event_name) => {
        const { wrapper, child } = await mount_ryuu_view()

        child.vm.$emit(event_name, { id: 'kyou-1' })
        await wrapper.vm.$nextTick()

        expect(wrapper.emitted(event_name), `${event_name} が親へ届いていない`).toBeTruthy()
    })

    // ダブルクリックで開く KyouDialog は rykv 画面の DialogHost が持つ。
    // Ryuu 側で握りつぶすとどこにも開かない
    test('requested_open_rykv_dialog は kind と payload ごと通す', async () => {
        const { wrapper, child } = await mount_ryuu_view()

        const kyou = { id: 'kyou-1' }
        child.vm.$emit('requested_open_rykv_dialog', 'kyou', kyou, null)
        await wrapper.vm.$nextTick()

        const emitted = wrapper.emitted('requested_open_rykv_dialog')
        expect(emitted, 'KyouDialog を開く要求が上位へ届いていない').toBeTruthy()
        expect(emitted![0]).toEqual(['kyou', kyou, null])
    })

    // Ryuu の行クリックで rykv のフォーカス Kyou が変わると、
    // Ryuu 自身の target_kyou が変わって再検索し続ける。だからここだけは通さない
    test('フォーカスは通さない', async () => {
        const { wrapper, child } = await mount_ryuu_view()

        child.vm.$emit('focused_kyou', { id: 'kyou-1' })
        child.vm.$emit('clicked_kyou', { id: 'kyou-1' })
        await wrapper.vm.$nextTick()

        expect(wrapper.emitted('focused_kyou'), 'Ryuu のクリックで rykv のフォーカスを奪っている').toBeFalsy()
        expect(wrapper.emitted('clicked_kyou'), 'Ryuu のクリックで rykv のフォーカスを奪っている').toBeFalsy()
    })

    // v-on と @ を同じ要素に併記すると両方登録されて2回発火する。
    // 手書きの @ を v-on に畳む改修で消し忘れると起きる
    test('二重に発火しない', async () => {
        const { wrapper, child } = await mount_ryuu_view()

        child.vm.$emit('updated_kyou', { id: 'kyou-1' })
        await wrapper.vm.$nextTick()

        expect(wrapper.emitted('updated_kyou')).toHaveLength(1)
    })
})
