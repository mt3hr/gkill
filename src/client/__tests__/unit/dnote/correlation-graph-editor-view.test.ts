/**
 * 相関グラフの編集画面（use-dnote-correlation-graph-editor-view.ts）の保存時の正規化と入力検査。
 *
 * 「記録が無い期間を 0 とみなす」（missing_as_zero）は件数・合計の集計対象でだけ意味がある
 * （平均・時刻の 0 は観測値ではない）。集計対象を平均へ切り替えたあともチェックだけ残ると、
 * 集計側の is_missing_as_zero_effective が無視するので画面上の設定と挙動がずれたまま保存される。
 * 集計側は correlation-aggregator.test.ts が守る。ここは保存時に落とすこと・入力検査を固定する。
 */
import { describe, expect, test, vi } from 'vitest'
import '../../helpers/setup-i18n'
import { nextTick, ref } from 'vue'
import { DnoteCorrelationGraphQuery, DnoteCorrelationMetric } from '@/classes/dnote/dnote-correlation'
import AggregateCountKyou from '@/classes/dnote/dnote-aggregate-target/aggregate-count-kyou'
import AggregateAverageKCNumValue from '@/classes/dnote/dnote-aggregate-target/aggregate-average-kc-num-value'
import { useDnoteCorrelationGraphEditorView } from '@/classes/use-dnote-correlation-graph-editor-view'

function make_query(): DnoteCorrelationGraphQuery {
    const query = new DnoteCorrelationGraphQuery()
    query.id = 'graph-1'
    query.title = 'テスト'
    const count = new DnoteCorrelationMetric()
    count.id = 'metric-count'
    count.title = '件数'
    count.aggregate_target = new AggregateCountKyou()
    count.missing_as_zero = true
    const average = new DnoteCorrelationMetric()
    average.id = 'metric-average'
    average.title = '平均'
    average.aggregate_target = new AggregateAverageKCNumValue()
    query.metrics = [count, average]
    return query
}

function build() {
    const emits = vi.fn()
    const initial_query = ref(make_query())
    const view = useDnoteCorrelationGraphEditorView({
        props: { gkill_api: { generate_uuid: () => 'new-metric' } } as never,
        emits: emits as never,
        initial_query,
    })
    return { view, emits, initial_query }
}

function saved_query(emits: ReturnType<typeof vi.fn>): DnoteCorrelationGraphQuery {
    const call = emits.mock.calls.find(args => args[0] === 'saved')
    expect(call, 'saved が emit されていない').toBeDefined()
    return call![1] as DnoteCorrelationGraphQuery
}

describe('useDnoteCorrelationGraphEditorView', () => {
    test('missing_as_zero は件数・合計の集計対象でだけ選べる', () => {
        const { view } = build()
        expect(view.supports_missing_as_zero(view.metrics.value[0])).toBe(true)
        expect(view.supports_missing_as_zero(view.metrics.value[1])).toBe(false)
        view.metrics.value[1].aggregate_target = 'AggregateSumNlogAmount'
        expect(view.supports_missing_as_zero(view.metrics.value[1])).toBe(true)
    })

    test('集計対象が平均のまま残っていたチェックは保存時に落とす', () => {
        const { view, emits } = build()
        expect(view.metrics.value[0].missing_as_zero).toBe(true)
        // 画面上はチェックが残った状態で平均のまま保存する
        view.metrics.value[1].missing_as_zero = true

        view.save()

        const query = saved_query(emits)
        expect(query.id).toBe('graph-1')
        expect(query.metrics.map(metric => metric.missing_as_zero)).toEqual([true, false])
        expect(query.metrics.map(metric => metric.aggregate_target.to_json().type)).toEqual(['AggregateCountKyou', 'AggregateAverageKCNumValue'])
        expect(query.metrics.map(metric => metric.timeis_span_policy)).toEqual(['split', 'split'])
    })

    test('件数を合計へ切り替えてもチェックは残り、計上先の指定も保存される', () => {
        const { view, emits } = build()
        view.metrics.value[0].aggregate_target = 'AggregateSumTimeIsTime'
        view.metrics.value[0].timeis_span_policy = 'start'

        view.save()

        const query = saved_query(emits)
        expect(query.metrics[0].missing_as_zero).toBe(true)
        expect(query.metrics[0].timeis_span_policy).toBe('start')
    })

    test('指標は2〜10本。名前の空・重複、lag の非整数は保存しない', () => {
        const { view, emits } = build()
        view.delete_metric(0)
        expect(view.metrics.value, '2本を下回る削除はできない').toHaveLength(2)

        view.metrics.value[1].title = '  '
        view.save()
        expect(view.validation_message.value).not.toBe('')
        expect(emits).not.toHaveBeenCalledWith('saved', expect.anything())

        view.metrics.value[1].title = '件数'
        view.save()
        expect(view.validation_message.value, '重複した名前').not.toBe('')

        view.metrics.value[1].title = '平均'
        view.lag.value = 1.5
        view.save()
        expect(view.validation_message.value, '非整数の lag').not.toBe('')
        expect(emits).not.toHaveBeenCalledWith('saved', expect.anything())

        view.lag.value = 2
        view.save()
        expect(view.validation_message.value).toBe('')
        expect(saved_query(emits).lag).toBe(2)
    })

    test('add_metric は10本まで。追加した指標は件数・未チェック・split で始まる', () => {
        const { view } = build()
        view.add_metric()
        const added = view.metrics.value[2]
        expect(added.id).toBe('new-metric')
        expect(added.aggregate_target).toBe('AggregateCountKyou')
        expect(added.missing_as_zero).toBe(false)
        expect(added.timeis_span_policy).toBe('split')
        for (let i = view.metrics.value.length; i < 12; i++) {
            view.add_metric()
        }
        expect(view.metrics.value).toHaveLength(10)
    })

    test('initial_query が差し替わったら編集中の値を捨てて読み直す', async () => {
        const { view, initial_query } = build()
        view.title.value = '書きかけ'
        const next = make_query()
        next.id = 'graph-2'
        next.title = '別のグラフ'
        initial_query.value = next
        await nextTick()

        expect(view.title.value).toBe('別のグラフ')
    })
})
