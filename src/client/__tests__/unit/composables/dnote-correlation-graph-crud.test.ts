/**
 * 相関グラフの CRUD 導線が、トレンドグラフと同じ形で端まで繋がっていることを確かめる。
 *
 * 以前は相関グラフだけが
 *   dnote-view → (ref) → 表ビュー → (ref) → 追加も編集も削除も兼ねる1つのダイアログ
 * という2段のテンプレート ref を経由していた。
 * どの段が null でも `?.` で無言に落ちるだけなので、
 * 「＋メニューを押しても何も起きない」に気づけず、実機で初めて発覚した。
 *
 * いまは他の集計要素と同じく
 *   追加ダイアログは dnote-view.vue が直接持つ
 *   編集・削除ダイアログとコンテキストメニューは各グラフのビューが持つ
 * という形に揃えてある。この2点を固定する。
 */
import { describe, expect, it, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { mount } from '@vue/test-utils'

// 末端のグラフビューは Vuetify と統計計算を引き込むので、モジュールごと差し替える。
// ここで見たいのは「子が上げたイベントを表ビューが定義へ反映するか」だけ。
vi.mock('@/pages/views/dnote-correlation-graph-view.vue', () => ({
    default: { name: 'dnote-correlation-graph-view', template: '<div />' },
}))

import DnoteCorrelationGraphTableView from '@/pages/views/dnote-correlation-graph-table-view.vue'

const base_props = {
    gkill_api: {},
    application_config: {},
    editable: true,
}

function make_query(id: string, title: string) {
    return { id: id, title: title, granularity: 'day', method: 'pearson', lag: 0, metrics: [] }
}

describe('相関グラフの表ビュー', () => {
    it('子の削除要求で定義から取り除かれる', async () => {
        const model = [make_query('a', 'A'), make_query('b', 'B')]
        const wrapper = mount(DnoteCorrelationGraphTableView, {
            props: { ...base_props, modelValue: model } as never,
        })

        const child = wrapper.findComponent({ name: 'dnote-correlation-graph-view' })
        expect(child.exists(), 'グラフのビューが描画されていない').toBe(true)

        child.vm.$emit('requested_delete_dnote_correlation_graph', 'a')
        await wrapper.vm.$nextTick()

        expect(model.map(q => q.id), '削除要求が定義に反映されていない').toEqual(['b'])
    })

    it('子の更新要求で同じ位置に差し替わる', async () => {
        const model = [make_query('a', 'A'), make_query('b', 'B')]
        const wrapper = mount(DnoteCorrelationGraphTableView, {
            props: { ...base_props, modelValue: model } as never,
        })

        const child = wrapper.findComponent({ name: 'dnote-correlation-graph-view' })
        child.vm.$emit('requested_update_dnote_correlation_graph', make_query('a', 'A2'))
        await wrapper.vm.$nextTick()

        expect(model.map(q => q.id), '並び順が変わってしまっている').toEqual(['a', 'b'])
        expect(model[0].title, '更新要求が定義に反映されていない').toBe('A2')
    })

    it('子の移動要求で並び順が入れ替わる', async () => {
        const model = [make_query('a', 'A'), make_query('b', 'B')]
        const wrapper = mount(DnoteCorrelationGraphTableView, {
            props: { ...base_props, modelValue: model } as never,
        })

        const child = wrapper.findComponent({ name: 'dnote-correlation-graph-view' })
        child.vm.$emit('requested_move_dnote_correlation_graph', 'b', 'a', 'left')
        await wrapper.vm.$nextTick()

        expect(model.map(q => q.id), '移動要求が定義に反映されていない').toEqual(['b', 'a'])
    })
})

describe('Dnoteの追加メニュー', () => {
    it('4種類とも自分が持つダイアログの show() を直接呼ぶ', () => {
        const source = readFileSync(resolve(__dirname, '../../../pages/views/dnote-view.vue'), 'utf8')

        // 相関だけ別経路(ref越しのref)にすると、押しても無反応になっても気づけない
        for (const dialog of [
            'add_dnote_item_dialog',
            'add_dnote_list_dialog',
            'add_dnote_trend_graph_dialog',
            'add_dnote_correlation_graph_dialog',
        ]) {
            expect(source, `${dialog} が ＋メニューから直接呼ばれていない`)
                .toContain(`@click="${dialog}?.show()"`)
        }
    })

    it('追加ダイアログは4つとも dnote-view.vue に mount されている', () => {
        const source = readFileSync(resolve(__dirname, '../../../pages/views/dnote-view.vue'), 'utf8')

        for (const dialog of [
            'AddDnoteListDialog',
            'AddDnoteItemDialog',
            'AddDnoteTrendGraphDialog',
            'AddDnoteCorrelationGraphDialog',
        ]) {
            expect(source, `${dialog} が dnote-view.vue に mount されていない`).toContain(`<${dialog}`)
        }
    })
})
