/**
 * useRepQuery の「プロファイル(device)×記録分類(rep_type)→記録先(rep)詳細」算出の検証。
 *
 * 再計算は update_check_devices / update_check_rep_types の
 * `if (!loading.value)` ガードの内側にあり、loading は props 同期 watcher が
 * 立てて finally で必ず倒す。壊れたクエリインスタンス(古い世代のビルドが保存した
 * JSON由来でフィールドが欠落)で clone() が throw しても loading が復帰しないと、
 * 算出機構だけが無言で死ぬ(2026-08-10 の回帰)。
 */
import { describe, expect, test, vi } from 'vitest'

// req_res は GkillAPIRequest を継承する。GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の
// 循環importがあるため、本番同様に gkill-api を先に評価させないと class extends が undefined になる
import '@/classes/api/gkill-api'

import { createApp, defineComponent, h, nextTick, reactive } from 'vue'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { CheckState } from '@/pages/views/check-state'
import { is_struct_container_node, type FoldableStructModel } from '@/pages/views/foldable-struct-model'
import { useRepQuery } from '@/classes/use-rep-query'
import type { RepQueryProps } from '@/pages/views/rep-query-props'
import type { RepQueryEmits } from '@/pages/views/rep-query-emits'
import {
    makeDeviceStructElement,
    makeRepStructElement,
    makeRepTypeStructElement,
} from '../../helpers/factory'

interface StructNode { key: string, is_checked: boolean, is_dir: boolean, children: Array<StructNode> | null }

// FoldableStruct.get_selected_items() の写し。
// 本番と同じく入れ物(ルート / フォルダ)は返さない。ここがずれると、
// 本番では起きえないキー集合でコンポーザブルを検証してしまう
function collect_checked_keys(root: StructNode): Array<string> {
    const checked: Array<string> = []
    const walk = (node: StructNode): void => {
        if (node.is_checked && !is_struct_container_node(node as unknown as FoldableStructModel)) {
            checked.push(node.key)
        }
        node.children?.forEach((child) => walk(child))
    }
    root.children?.forEach((child) => walk(child))
    return checked
}

function make_config(): ApplicationConfig {
    const config = new ApplicationConfig()
    config.device_struct = makeDeviceStructElement({
        name: 'root', key: '__root__',
        children: [makeDeviceStructElement({ name: 'なし', device_name: 'なし', key: 'なし' })],
    }) as unknown as typeof config.device_struct
    config.rep_type_struct = makeRepTypeStructElement({
        name: 'root', key: '__root__',
        children: [
            makeRepTypeStructElement({ name: 'Kmemo', rep_type_name: 'Kmemo', key: 'Kmemo' }),
            makeRepTypeStructElement({ name: 'URLog', rep_type_name: 'URLog', key: 'URLog' }),
        ],
    }) as unknown as typeof config.rep_type_struct
    config.rep_struct = makeRepStructElement({
        name: 'root', key: '__root__',
        children: [
            makeRepStructElement({ name: 'Kmemo', rep_name: 'Kmemo', key: 'Kmemo' }),
            makeRepStructElement({ name: 'URLog', rep_name: 'URLog', key: 'URLog' }),
        ],
    }) as unknown as typeof config.rep_struct
    return config
}

function createView() {
    const emitted: Array<{ event: string, args: Array<unknown> }> = []
    const emits = (event: string, ...args: Array<unknown>) => {
        emitted.push({ event: event, args: args })
    }
    const props = reactive({
        application_config: make_config(),
        gkill_api: {},
        find_kyou_query: new FindKyouQuery(),
        inited: true,
    }) as unknown as RepQueryProps & { find_kyou_query: FindKyouQuery }

    // ミニアプリにマウントして watcher 例外を errorHandler で受ける。
    // 素で composable を呼ぶと、意図的に投げさせた例外が Vue の再throwで
    // unhandled rejection になり vitest がエラー扱いする
    let view!: ReturnType<typeof useRepQuery>
    const app = createApp(defineComponent({
        setup() {
            view = useRepQuery({ props, emits: emits as unknown as RepQueryEmits })
            return () => h('div')
        },
    }))
    const captured_errors: Array<unknown> = []
    app.config.errorHandler = (err) => { captured_errors.push(err) }
    app.mount(document.createElement('div'))

    // FoldableStruct の defineExpose 相当の fake。cloned_application_config の
    // モデルツリーからチェック済みキーを読む
    const struct_of = {
        reps: () => view.cloned_application_config.value.rep_struct as unknown as StructNode,
        devices: () => view.cloned_application_config.value.device_struct as unknown as StructNode,
        rep_types: () => view.cloned_application_config.value.rep_type_struct as unknown as StructNode,
    }
    view.foldable_struct_reps.value = {
        get_selected_items: () => collect_checked_keys(struct_of.reps()),
        update_check: vi.fn(),
    } as unknown as typeof view.foldable_struct_reps.value
    view.foldable_struct_devices.value = {
        get_selected_items: () => collect_checked_keys(struct_of.devices()),
        update_check: vi.fn(),
    } as unknown as typeof view.foldable_struct_devices.value
    view.foldable_struct_rep_types.value = {
        get_selected_items: () => collect_checked_keys(struct_of.rep_types()),
        update_check: vi.fn(),
    } as unknown as typeof view.foldable_struct_rep_types.value

    return { view, props, emitted, struct_of, app, captured_errors }
}

async function flush(times = 8): Promise<void> {
    for (let i = 0; i < times; i++) {
        await nextTick()
        await Promise.resolve()
    }
}

function make_column_query(): FindKyouQuery {
    const query = new FindKyouQuery()
    query.query_id = 'col-1'
    query.devices_in_sidebar = ['なし']
    query.rep_types_in_sidebar = ['Kmemo', 'URLog']
    query.reps = ['Kmemo', 'URLog']
    return query
}

describe('useRepQuery サマリ→記録先詳細の算出', () => {
    test('記録分類のチェック変更で記録先詳細が再計算される', async () => {
        const { view, props, emitted, struct_of, app } = createView()
        props.find_kyou_query = make_column_query()
        await flush()
        expect(collect_checked_keys(struct_of.reps()), '同期の前提が崩れている').toEqual(['Kmemo', 'URLog'])

        // ユーザーが記録分類を Kmemo だけに絞る
        await view.update_check_rep_types(['Kmemo'], CheckState.checked, true)

        expect(collect_checked_keys(struct_of.rep_types())).toEqual(['Kmemo'])
        expect(collect_checked_keys(struct_of.reps()), '記録先詳細が再計算されていない').toEqual(['Kmemo'])
        const reps_emissions = emitted.filter((emit) => emit.event === 'request_update_checked_reps' && emit.args[1] === true)
        expect(reps_emissions, '再計算結果がユーザー編集として親へ届いていない').toHaveLength(1)
        expect(reps_emissions[0].args[0]).toEqual(['Kmemo'])
        app.unmount()
    })

    test('壊れたクエリの同期で例外が出ても、loadingが復帰して次の算出は動く', async () => {
        const { view, props, struct_of, app, captured_errors } = createView()
        props.find_kyou_query = make_column_query()
        await flush()

        // 古い世代のJSON由来を模した壊れたインスタンス(clone()がthrowする)
        const broken = make_column_query();
        (broken as unknown as { period_of_time_week_of_days: unknown }).period_of_time_week_of_days = undefined
        props.find_kyou_query = broken
        await flush()
        expect(captured_errors.length, '前提: cloneの例外がwatcherで発生していること').toBeGreaterThan(0)

        // 例外後もloadingがfinallyで倒れているので、算出は生きている。
        // try/finallyが無いとloadingが立ちっぱなしになり、ここが再計算されない
        await view.update_check_rep_types(['URLog'], CheckState.checked, true)
        expect(collect_checked_keys(struct_of.reps()), '例外後に算出機構が死んでいる').toEqual(['URLog'])
        app.unmount()
    })
})
