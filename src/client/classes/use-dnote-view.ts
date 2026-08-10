import { i18n } from '@/i18n'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { computed, nextTick, ref, watch, type Ref } from 'vue'
import DnoteItem from '@/classes/dnote/dnote-item'
import DnoteListQuery from '@/pages/views/dnote-list-query'
import DnoteTrendGraphQuery from '@/pages/views/dnote-trend-graph-query'
import type { DnoteEmits } from '@/pages/views/dnote-emits'
import type { DnoteViewProps } from '@/pages/views/dnote-view-props'
import register_dictionary, { build_dnote_aggregate_target_from_json, build_dnote_key_getter_from_json, build_dnote_predicate_from_json } from '@/classes/dnote/serialize/register-dictionary'
import moment from 'moment'
import { save_as } from '@/classes/save-as'
import type { Kyou } from '@/classes/datas/kyou'
import { build_kyou_dialog_relay } from '@/classes/kyou-view-relay'
import type { GkillError } from '@/classes/api/gkill-error'
import { GkillMessage } from '@/classes/api/gkill-message'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'
import { to_export_kyou_dto } from '@/classes/dto/export_dto'
import { prune_empty } from '@/classes/dto/export_prune'
import type { ComponentRef } from '@/classes/component-ref'

export interface DnoteDefinition {
    name: string
    items: Array<Array<DnoteItem>>
    lists: Array<DnoteListQuery>
    trends: Array<DnoteTrendGraphQuery>
}

export function useDnoteView(options: {
    props: DnoteViewProps,
    emits: DnoteEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const add_dnote_list_dialog = ref<ComponentRef | null>(null)
    const add_dnote_item_dialog = ref<ComponentRef | null>(null)
    const add_dnote_trend_graph_dialog = ref<ComponentRef | null>(null)

    // ── View refs (Map-based, for dynamic :ref bindings) ──
    const item_view_refs = new Map<number, ComponentRef>()
    const list_view_refs = new Map<number, ComponentRef>()
    const trend_view_refs = new Map<number, ComponentRef>()

    function set_item_table_ref(i: number, el: ComponentRef | null): void {
        if (el) item_view_refs.set(i, el)
        else item_view_refs.delete(i)
    }
    function set_list_table_ref(i: number, el: ComponentRef | null): void {
        if (el) list_view_refs.set(i, el)
        else list_view_refs.delete(i)
    }
    function set_trend_table_ref(i: number, el: ComponentRef | null): void {
        if (el) trend_view_refs.set(i, el)
        else trend_view_refs.delete(i)
    }

    // ── State refs ──
    const dnote_definitions: Ref<Array<DnoteDefinition>> = ref([])
    const current_definition_index = ref(0)
    const abort_controller = ref(new AbortController())
    const is_loading = ref(true)
    const is_fetching_from_api = ref(false)

    const target_kyous_count = ref(0)
    const getted_kyous_count = ref(0)
    const estimate_aggregate_task = ref(0)
    const finished_aggregate_task = ref(0)

    const first_kyou_date_str = ref("")
    const last_kyou_date_str = ref("")

    const loaded_kyous: Ref<Array<Kyou> | null> = ref(null)
    const last_reload_query: Ref<FindKyouQuery> = ref(new FindKyouQuery())

    // ── Computed ──
    const start_date_str: Ref<string> = computed(() => props.query.calendar_start_date !== null ? moment(props.query.calendar_start_date).format("YYYY-MM-DD") : first_kyou_date_str.value)
    const end_date_str: Ref<string> = computed(() => props.query.calendar_end_date !== null ? moment(props.query.calendar_end_date).format("YYYY-MM-DD") : last_kyou_date_str.value)

    const dnote_item_table_view_data = computed({
        get: () => {
            if (dnote_definitions.value.length === 0) return [[]] as Array<Array<DnoteItem>>
            const idx = current_definition_index.value
            const safe_idx = (idx >= 0 && idx < dnote_definitions.value.length) ? idx : 0
            return dnote_definitions.value[safe_idx].items
        },
        set: (val: Array<Array<DnoteItem>>) => {
            if (dnote_definitions.value.length === 0) return
            const idx = current_definition_index.value
            if (idx >= 0 && idx < dnote_definitions.value.length) {
                dnote_definitions.value[idx].items = val
            }
        }
    })

    const dnote_list_item_table_view_data = computed({
        get: () => {
            if (dnote_definitions.value.length === 0) return [] as Array<DnoteListQuery>
            const idx = current_definition_index.value
            const safe_idx = (idx >= 0 && idx < dnote_definitions.value.length) ? idx : 0
            return dnote_definitions.value[safe_idx].lists
        },
        set: (val: Array<DnoteListQuery>) => {
            if (dnote_definitions.value.length === 0) return
            const idx = current_definition_index.value
            if (idx >= 0 && idx < dnote_definitions.value.length) {
                dnote_definitions.value[idx].lists = val
            }
        }
    })

    const dnote_trend_graph_view_data = computed({
        get: () => {
            if (dnote_definitions.value.length === 0) return [] as Array<DnoteTrendGraphQuery>
            const idx = current_definition_index.value
            const safe_idx = (idx >= 0 && idx < dnote_definitions.value.length) ? idx : 0
            return dnote_definitions.value[safe_idx].trends
        },
        set: (val: Array<DnoteTrendGraphQuery>) => {
            if (dnote_definitions.value.length === 0) return
            const idx = current_definition_index.value
            if (idx >= 0 && idx < dnote_definitions.value.length) {
                dnote_definitions.value[idx].trends = val
            }
        }
    })

    // ── Watchers ──
    watch(() => props.application_config, () => {
        load_from_application_config()
    })

    nextTick(() => {
        load_from_application_config()
    })

    watch(current_definition_index, async (new_idx, old_idx) => {
        if (new_idx === old_idx) return
        if (!props.editable && loaded_kyous.value && loaded_kyous.value.length > 0) {
            await re_aggregate_current_definition()
        }
    })

    // ── Internal helpers ──
    async function reset_view(): Promise<void> {
        return nextTick(async () => {
            for (const ref of item_view_refs.values()) {
                await ref.reset()
            }
            for (const ref of list_view_refs.values()) {
                await ref.reset()
            }
            for (const ref of trend_view_refs.values()) {
                await ref.reset()
            }
        })
    }

    async function load_aggregated_value(ac: AbortController, kyous: Array<Kyou>, query: FindKyouQuery, kyou_is_loaded: boolean) {
        return item_view_refs.get(current_definition_index.value)?.load_aggregated_value(ac, kyous, query, kyou_is_loaded)
    }

    async function load_aggregate_grouping_list(ac: AbortController, kyous: Array<Kyou>, find_kyou_query: FindKyouQuery, kyou_is_loaded: boolean): Promise<void> {
        return await list_view_refs.get(current_definition_index.value)?.load_aggregate_grouping_list(ac, kyous, find_kyou_query, kyou_is_loaded)
    }

    async function load_trend_graphs(ac: AbortController, kyous: Array<Kyou>, find_kyou_query: FindKyouQuery, kyou_is_loaded: boolean): Promise<void> {
        return await trend_view_refs.get(current_definition_index.value)?.load_trend_graph(ac, kyous, find_kyou_query, kyou_is_loaded)
    }

    function parse_single_definition_json(def_json: Record<string, unknown>): DnoteDefinition {
        register_dictionary()
        const name = (def_json.name as string) || i18n.global.t('DNOTE_DEFINITION_DEFAULT_NAME')
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const items: Array<Array<DnoteItem>> = ((def_json && def_json.dnote_item_table_view_data ? def_json.dnote_item_table_view_data : []) as Array<Array<any>> || []).map((col: Array<any>) =>
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            col.map((item_json: any) => {
                const item = new DnoteItem()
                item.id = item_json.id
                item.prefix = item_json.prefix
                item.suffix = item_json.suffix
                item.title = item_json.title
                item.aggregate_target = build_dnote_aggregate_target_from_json(item_json.aggregate_target)
                item.predicate = build_dnote_predicate_from_json(item_json.predicate)
                return item
            })
        )
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const lists: Array<DnoteListQuery> = ((def_json && def_json.dnote_list_item_table_view_data ? def_json.dnote_list_item_table_view_data : []) as Array<any> || []).map((query_json: any) => {
            const query = new DnoteListQuery()
            query.id = query_json.id
            query.prefix = query_json.prefix
            query.suffix = query_json.suffix
            query.title = query_json.title
            query.aggregate_target = build_dnote_aggregate_target_from_json(query_json.aggregate_target)
            query.predicate = build_dnote_predicate_from_json(query_json.predicate)
            query.key_getter = build_dnote_key_getter_from_json(query_json.key_getter)
            return query
        })
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const trends: Array<DnoteTrendGraphQuery> = ((def_json && def_json.dnote_trend_graph_view_data ? def_json.dnote_trend_graph_view_data : []) as Array<any> || []).map((query_json: any) => {
            const query = new DnoteTrendGraphQuery()
            query.id = query_json.id
            query.title = query_json.title
            query.aggregate_target = build_dnote_aggregate_target_from_json(query_json.aggregate_target)
            query.predicate = build_dnote_predicate_from_json(query_json.predicate)
            query.granularity = (query_json.granularity === 'week' || query_json.granularity === 'month') ? query_json.granularity : 'day'
            query.chart_type = query_json.chart_type === 'bar' ? 'bar' : 'line'
            return query
        })
        if (items.length === 0) {
            items.push(new Array<DnoteItem>())
        }
        return { name, items, lists, trends }
    }

    function serialize_single_definition(def: DnoteDefinition): Record<string, unknown> {
        const dnote_item_table_view_data_serialized = []
        for (let i = 0; i < def.items.length; i++) {
            const list = []
            const dnote_item_col = def.items[i]
            for (let j = 0; j < dnote_item_col.length; j++) {
                const dnote_item = dnote_item_col[j]
                const record = {
                    id: dnote_item.id,
                    prefix: dnote_item.prefix,
                    suffix: dnote_item.suffix,
                    title: dnote_item.title,
                    aggregate_target: dnote_item.aggregate_target.to_json(),
                    predicate: dnote_item.predicate.predicate_struct_to_json(),
                }
                list.push(record)
            }
            dnote_item_table_view_data_serialized.push(list)
        }

        const dnote_list_item_table_view_data_serialized = []
        for (let i = 0; i < def.lists.length; i++) {
            const list_find_query = def.lists[i]
            const record = {
                id: list_find_query.id,
                prefix: list_find_query.prefix,
                suffix: list_find_query.suffix,
                title: list_find_query.title,
                aggregate_target: list_find_query.aggregate_target.to_json(),
                predicate: list_find_query.predicate.predicate_struct_to_json(),
                key_getter: list_find_query.key_getter.to_json(),
            }
            dnote_list_item_table_view_data_serialized.push(record)
        }

        const dnote_trend_graph_view_data_serialized = []
        for (let i = 0; i < def.trends.length; i++) {
            const trend_graph_query = def.trends[i]
            const record = {
                id: trend_graph_query.id,
                title: trend_graph_query.title,
                aggregate_target: trend_graph_query.aggregate_target.to_json(),
                predicate: trend_graph_query.predicate.predicate_struct_to_json(),
                granularity: trend_graph_query.granularity,
                chart_type: trend_graph_query.chart_type,
            }
            dnote_trend_graph_view_data_serialized.push(record)
        }

        return {
            name: def.name,
            dnote_item_table_view_data: dnote_item_table_view_data_serialized,
            dnote_list_item_table_view_data: dnote_list_item_table_view_data_serialized,
            dnote_trend_graph_view_data: dnote_trend_graph_view_data_serialized,
        }
    }

    function to_json(): Array<Record<string, unknown>> {
        return dnote_definitions.value.map(serialize_single_definition)
    }

    function from_json(json: unknown): void {
        register_dictionary()
        let definitions_json: Array<Record<string, unknown>>
        if (Array.isArray(json)) {
            definitions_json = json as Array<Record<string, unknown>>
        } else if (json && typeof json === 'object' && ((json as Record<string, unknown>).dnote_item_table_view_data || (json as Record<string, unknown>).dnote_list_item_table_view_data)) {
            definitions_json = [json as Record<string, unknown>]
        } else {
            definitions_json = []
        }
        if (definitions_json.length === 0) {
            definitions_json = [{ name: i18n.global.t('DNOTE_DEFINITION_DEFAULT_NAME'), dnote_item_table_view_data: [[]], dnote_list_item_table_view_data: [] }]
        }
        dnote_definitions.value = definitions_json.map(parse_single_definition_json)
        if (current_definition_index.value >= dnote_definitions.value.length) {
            current_definition_index.value = 0
        }
    }

    function load_from_application_config(): void {
        from_json(props.application_config.dnote_json_data)
    }

    async function re_aggregate_current_definition(): Promise<void> {
        if (!loaded_kyous.value) return
        is_loading.value = true
        finished_aggregate_task.value = 0
        estimate_aggregate_task.value = 0
        for (let i = 0; i < dnote_item_table_view_data.value.length; i++) {
            estimate_aggregate_task.value += dnote_item_table_view_data.value[i].length
        }
        estimate_aggregate_task.value += dnote_list_item_table_view_data.value.length
        estimate_aggregate_task.value += dnote_trend_graph_view_data.value.length
        target_kyous_count.value = loaded_kyous.value.length
        getted_kyous_count.value = loaded_kyous.value.length

        abort_controller.value.abort()
        abort_controller.value = new AbortController()
        await nextTick()
        await item_view_refs.get(current_definition_index.value)?.reset()
        await list_view_refs.get(current_definition_index.value)?.reset()
        await trend_view_refs.get(current_definition_index.value)?.reset()

        const kyou_is_loaded = true
        const wait_promises = new Array<Promise<unknown>>()
        wait_promises.push(load_aggregated_value(abort_controller.value, loaded_kyous.value, last_reload_query.value, kyou_is_loaded))
        wait_promises.push(load_aggregate_grouping_list(abort_controller.value, loaded_kyous.value, last_reload_query.value, kyou_is_loaded))
        wait_promises.push(load_trend_graphs(abort_controller.value, loaded_kyous.value, last_reload_query.value, kyou_is_loaded))
        await Promise.all(wait_promises)
        is_loading.value = false
    }

    // ── Business logic ──
    async function reload(kyous: Array<Kyou>, query: FindKyouQuery): Promise<void> {
        is_fetching_from_api.value = false
        loaded_kyous.value = null
        is_loading.value = true
        last_reload_query.value = query
        first_kyou_date_str.value = kyous && kyous.length > 0 ? moment(kyous[kyous.length - 1].related_time).format("YYYY-MM-DD") : ""
        last_kyou_date_str.value = kyous && kyous.length > 0 ? moment(kyous[0].related_time).format("YYYY-MM-DD") : ""

        reset_view()
        if (dnote_item_table_view_data.value.length === 0) {
            dnote_item_table_view_data.value.push(new Array<DnoteItem>())
        }
        await abort()

        const trimed_kyous_map = new Map<string, Kyou>()
        for (let i = 0; i < kyous.length; i++) {
            if (trimed_kyous_map.has(kyous[i].id) && trimed_kyous_map.get(kyous[i].id)!.update_time.getTime() > kyous[i].update_time.getTime()) {
                continue
            }
            trimed_kyous_map.set(kyous[i].id, kyous[i])
        }
        const trimed_kyous = new Array<Kyou>()
        trimed_kyous_map.forEach((kyou) => trimed_kyous.push(kyou))

        target_kyous_count.value = trimed_kyous.length
        getted_kyous_count.value = 0
        finished_aggregate_task.value = 0
        estimate_aggregate_task.value = 0
        for (let i = 0; i < dnote_item_table_view_data.value.length; i++) {
            estimate_aggregate_task.value += dnote_item_table_view_data.value[i].length
        }
        estimate_aggregate_task.value += dnote_list_item_table_view_data.value.length
        estimate_aggregate_task.value += dnote_trend_graph_view_data.value.length

        const cloned_kyou = await load_kyous(abort_controller.value, trimed_kyous)
        const kyou_is_loaded = true
        const wait_promises = new Array<Promise<unknown>>()
        wait_promises.push(load_aggregated_value(abort_controller.value, cloned_kyou, query, kyou_is_loaded))
        wait_promises.push(load_aggregate_grouping_list(abort_controller.value, cloned_kyou, query, kyou_is_loaded))
        wait_promises.push(load_trend_graphs(abort_controller.value, cloned_kyou, query, kyou_is_loaded))
        await Promise.all(wait_promises)
        is_loading.value = false
        loaded_kyous.value = cloned_kyou
    }

    async function abort(): Promise<void> {
        abort_controller.value.abort()
        abort_controller.value = new AbortController()
        return reset_view()
    }

    function set_loading(loading: boolean): void {
        is_loading.value = loading
        if (loading) {
            is_fetching_from_api.value = true
        }
    }

    function add_definition(): void {
        const new_def: DnoteDefinition = {
            name: i18n.global.t('DNOTE_DEFINITION_DEFAULT_NAME') + " " + (dnote_definitions.value.length + 1),
            items: [new Array<DnoteItem>()],
            lists: new Array<DnoteListQuery>(),
            trends: new Array<DnoteTrendGraphQuery>(),
        }
        dnote_definitions.value.push(new_def)
        current_definition_index.value = dnote_definitions.value.length - 1
    }

    function delete_current_definition(): void {
        if (dnote_definitions.value.length <= 1) return
        dnote_definitions.value.splice(current_definition_index.value, 1)
        if (current_definition_index.value >= dnote_definitions.value.length) {
            current_definition_index.value = dnote_definitions.value.length - 1
        }
    }

    function floating_action_button_style() {
        return {
            'bottom': '60px',
            'right': '10px',
            'height': '50px',
            'width': '50px',
        }
    }

    async function apply(): Promise<void> {
        const dnote_json_data = to_json()
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        emits('requested_apply_dnote', dnote_json_data as any)
        nextTick(() => emits('requested_close_dialog'))
    }

    // 進捗表示のためかか共通からコピー。
    // 呼び出し元はここ1箇所で「複製して関連データを読む」用途しかないため、
    // 元実装にあった get_latest_data / clone の分岐は落としてある
    async function load_kyous(ac: AbortController, kyous: Array<Kyou>): Promise<Array<Kyou>> {
        // 1件ずつ待つと件数×RTTかかる(1,000件 × RTT20ms で約20秒)ので
        // 一定数ずつ並列で読む。件数はサーバのgoroutineプールを
        // 埋め尽くさない程度に抑える。
        const load_concurrency = 8
        const cloned_kyous = new Array<Kyou>()
        for (let start = 0; start < kyous.length; start += load_concurrency) {
            const chunk = kyous.slice(start, start + load_concurrency)
            const prepared = await Promise.all(chunk.map(async source => {
                const kyou: Kyou = source.clone()
                kyou.abort_controller = ac
                await Promise.all([
                    kyou.load_typed_datas(),
                    kyou.load_attached_tags(),
                    kyou.load_attached_texts(),
                ])
                // 進捗表示用。1件終わるごとに増やす
                getted_kyous_count.value++
                return kyou
            }))
            cloned_kyous.push(...prepared)
        }
        return cloned_kyous
    }

    async function download_kyous_json(): Promise<void> {
        const kyous = loaded_kyous.value
        if (!kyous || kyous.length === 0) return

        const start_date = new Date(kyous[kyous.length - 1].related_time)
        const end_date = new Date(kyous[0].related_time)
        const now = new Date(Date.now())
        const pad2 = (n: number) => String(n).padStart(2, "0")
        const format_date_string = (d: Date) => `${d.getFullYear()}${pad2(d.getMonth() + 1)}${pad2(d.getDate())}`
        const format_date_time_string = (d: Date) => `${d.getFullYear()}${pad2(d.getMonth() + 1)}${pad2(d.getDate())}${pad2(d.getHours())}${pad2(d.getMinutes())}${pad2(d.getSeconds())}`
        const filename = `gkill_export_data_${format_date_string(start_date)}_${format_date_string(end_date)}_exported_${format_date_time_string(now)}.json`

        if ("showSaveFilePicker" in window) {
            await stream_save_json_array(kyous, filename)
            return
        }

        const json_str = JSON.stringify(kyous)
        const blob = new Blob([json_str], { type: "application/json;charset=utf-8" })
        save_as(blob, filename)
    }

    async function stream_save_json_array(items: Kyou[], filename: string): Promise<void> {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const handle = await (window as any).showSaveFilePicker({
            suggestedName: filename,
            types: [{ description: "JSON", accept: { "application/json": [".json"] } }],
        })

        const writable = await handle.createWritable()

        const start_message = new GkillMessage()
        start_message.message_code = GkillMessageCodes.start_export_kyous
        start_message.message = i18n.global.t('START_EXPORT_KYOUS_MESSAGE')
        emits('received_messages', [start_message])

        try {
            await writable.write("[\n")
            for (let i = 0; i < items.length; i++) {
                const dto = to_export_kyou_dto(items[i]);
                const pruned = prune_empty(dto);
                if (pruned === undefined) continue;

                const seen = new WeakSet<object>()
                const replacer = (_k: string, v: unknown) => {
                    if (typeof v === "bigint") return v.toString()
                    if (v && typeof v === "object") {
                        if (seen.has(v)) return "[Circular]"
                        seen.add(v)
                    }
                    return v
                }

                if (i > 0) await writable.write(",\n")

                // 1要素ずつstringifyする
                const s = JSON.stringify(pruned, replacer, 0)
                await writable.write(s)
            }
            await writable.write("\n]\n")
        } finally {
            await writable.close()
            const finish_message = new GkillMessage()
            finish_message.message_code = GkillMessageCodes.start_export_kyous
            finish_message.message = i18n.global.t('FINISH_EXPORT_KYOUS_MESSAGE')
            emits('received_messages', [finish_message])
        }
    }

    // ── Template event handlers ──
    function onRequestedAddDnoteListQuery(dnote_list_query: DnoteListQuery): void {
        dnote_list_item_table_view_data.value.push(dnote_list_query)
        load_aggregated_value(abort_controller.value, [], new FindKyouQuery(), true)
        load_aggregate_grouping_list(abort_controller.value, [], new FindKyouQuery(), true)
    }

    function onRequestedAddDnoteItem(dnote_item: DnoteItem): void {
        dnote_item_table_view_data.value[0].push(dnote_item)
        load_aggregated_value(abort_controller.value, [], new FindKyouQuery(), true)
        load_aggregate_grouping_list(abort_controller.value, [], new FindKyouQuery(), true)
    }

    function onRequestedAddDnoteTrendGraph(dnote_trend_graph_query: DnoteTrendGraphQuery): void {
        dnote_trend_graph_view_data.value.push(dnote_trend_graph_query)
        load_trend_graphs(abort_controller.value, [], new FindKyouQuery(), true)
    }

    function increment_finished_aggregate_task(): void {
        finished_aggregate_task.value++
    }

    // ── Event relay objects ──
    // 以前は crud / focusClick / rykvDialog の3束に分かれていて、
    // どの束にも requested_reload_kyou / requested_reload_list /
    // requested_update_check_kyous が入っていなかった。
    // rykv-view.vue は allColumnsRequestHandlers を渡してくれているのに、
    // Dnote側がこれらをemitしないので死んでいた
    const crudRelayHandlers = build_kyou_dialog_relay(emits, {
        // クリックはフォーカス移動も伴う
        'clicked_kyou': (kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) },
    })

    // errors/messagesしかemitしない子（AddDnote*Dialog等）には20件束を渡す意味がない
    const errorsMessagesRelayHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
    }

    // ── Return ──
    return {
        // Template refs
        add_dnote_list_dialog,
        add_dnote_item_dialog,
        add_dnote_trend_graph_dialog,

        // View ref helpers
        item_view_refs,
        list_view_refs,
        trend_view_refs,
        set_item_table_ref,
        set_list_table_ref,
        set_trend_table_ref,

        // State
        dnote_definitions,
        current_definition_index,
        abort_controller,
        is_loading,
        is_fetching_from_api,
        target_kyous_count,
        getted_kyous_count,
        estimate_aggregate_task,
        finished_aggregate_task,
        loaded_kyous,

        // Computed
        start_date_str,
        end_date_str,
        dnote_item_table_view_data,
        dnote_list_item_table_view_data,
        dnote_trend_graph_view_data,

        // Business logic (exposed for defineExpose)
        reload,
        abort,
        set_loading,

        // Template event handlers
        add_definition,
        delete_current_definition,
        floating_action_button_style,
        apply,
        download_kyous_json,
        onRequestedAddDnoteListQuery,
        onRequestedAddDnoteItem,
        onRequestedAddDnoteTrendGraph,
        increment_finished_aggregate_task,

        // Event relay objects
        crudRelayHandlers,
        errorsMessagesRelayHandlers,
    }
}
