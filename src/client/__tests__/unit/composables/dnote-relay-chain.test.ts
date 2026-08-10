/**
 * Dnote の中継チェーンが `requested_reload_kyou` を親まで通すことを、実際にマウントして確かめる。
 *
 * タグ/テキスト/通知の変更は `updated_kyou` を出さず、唯一の信号が `requested_reload_kyou`。
 * Dnote チェーンは6ファイルが揃ってこれを落としており、
 * KyouListViewDialog の中でタグを足しても Dnote にも上のページにも伝わらなかった。
 *
 * composable 単体のテストでは「束に全イベントが入っているか」までしか見えない。
 * テンプレートで `v-on="crudRelayHandlers"` を実際に張り忘れていないかは
 * マウントして子から emit してみないと分からないので、ここは shallowMount で見る。
 */
import { describe, expect, it, vi } from 'vitest'
import { shallowMount } from '@vue/test-utils'

// 末端の子は Vuetify を引き込む（vitest は node_modules の .css を解決できない）。
// shallowMount のスタブは描画時にしか効かず import は走ってしまうので、モジュールごと差し替える
vi.mock('@/pages/views/dnote-item-view.vue', () => ({
    default: { name: 'dnote-item-view', template: '<div />' },
}))
vi.mock('@/pages/views/dnote-list-view.vue', () => ({
    default: { name: 'dnote-list-view', template: '<div />' },
}))

import DnoteItemListView from '@/pages/views/dnote-item-list-view.vue'
import DnoteItemTableView from '@/pages/views/dnote-item-table-view.vue'
import DnoteListTableView from '@/pages/views/dnote-list-table-view.vue'

/** チェーンで落ちていた3つ + 本命の updated_kyou */
const RELAYED_EVENTS = [
    'requested_reload_kyou',
    'requested_reload_list',
    'requested_update_check_kyous',
    'updated_kyou',
] as const

const base_props = {
    gkill_api: {},
    application_config: {},
    editable: false,
}

function make_dnote_item(id: string) {
    return { id: id, title: 'item', prefix: '', suffix: '', aggregate_target: null }
}

function make_dnote_list_query(id: string) {
    return { id: id, title: 'list', aggregate_target: null, key_getter: null, predicate: null }
}

describe('Dnote の中継チェーン', () => {
    it('DnoteItemListView は子の中継イベントを親へ通す', async () => {
        const wrapper = shallowMount(DnoteItemListView, {
            props: { ...base_props, dnd_list_index: 0, modelValue: [make_dnote_item('item-1')] } as never,
        })

        const child = wrapper.findComponent({ name: 'dnote-item-view' })
        expect(child.exists(), 'DnoteItemView が描画されていない').toBe(true)

        for (const event_name of RELAYED_EVENTS) {
            child.vm.$emit(event_name, { id: 'kyou-1' })
            await wrapper.vm.$nextTick()
            expect(wrapper.emitted(event_name), `${event_name} が親へ届いていない`).toBeTruthy()
        }
    })

    it('DnoteItemTableView は子の中継イベントを親へ通す', async () => {
        const wrapper = shallowMount(DnoteItemTableView, {
            props: { ...base_props, modelValue: [[make_dnote_item('item-1')]] } as never,
        })

        const child = wrapper.findComponent({ name: 'dnote-item-list-view' })
        expect(child.exists(), 'DnoteItemListView が描画されていない').toBe(true)

        for (const event_name of RELAYED_EVENTS) {
            child.vm.$emit(event_name, { id: 'kyou-1' })
            await wrapper.vm.$nextTick()
            expect(wrapper.emitted(event_name), `${event_name} が親へ届いていない`).toBeTruthy()
        }
    })

    it('DnoteListTableView は子の中継イベントを親へ通す', async () => {
        const wrapper = shallowMount(DnoteListTableView, {
            props: { ...base_props, modelValue: [make_dnote_list_query('list-1')] } as never,
        })

        const child = wrapper.findComponent({ name: 'dnote-list-view' })
        expect(child.exists(), 'DnoteListView が描画されていない').toBe(true)

        for (const event_name of RELAYED_EVENTS) {
            child.vm.$emit(event_name, { id: 'kyou-1' })
            await wrapper.vm.$nextTick()
            expect(wrapper.emitted(event_name), `${event_name} が親へ届いていない`).toBeTruthy()
        }
    })

    // v-on と @ を同じ要素に併記すると両方登録されて2回発火する。
    // 手書きの @ を v-on に畳む改修で消し忘れると起きる
    it('フォーカスは二重に発火しない', async () => {
        const wrapper = shallowMount(DnoteItemListView, {
            props: { ...base_props, dnd_list_index: 0, modelValue: [make_dnote_item('item-1')] } as never,
        })

        const child = wrapper.findComponent({ name: 'dnote-item-view' })
        child.vm.$emit('focused_kyou', { id: 'kyou-1' })
        await wrapper.vm.$nextTick()

        expect(wrapper.emitted('focused_kyou')).toHaveLength(1)
    })
})
