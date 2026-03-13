import { i18n } from '@/i18n'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { computed, nextTick, ref, watch, type Ref } from 'vue'
import DnoteItem from '@/classes/dnote/dnote-item'
import DnoteListQuery from '@/pages/views/dnote-list-query'
import type { DnoteEmits } from '@/pages/views/dnote-emits'
import type { DnoteViewProps } from '@/pages/views/dnote-view-props'
import regist_dictionary, { build_dnote_aggregate_target_from_json, build_dnote_key_getter_from_json, build_dnote_predicate_from_json } from '@/classes/dnote/serialize/regist-dictionary'
import moment from 'moment'
import { saveAs } from '@/classes/save-as'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type { GkillError } from '@/classes/api/gkill-error'
import { GkillMessage } from '@/classes/api/gkill-message'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'
import { toExportKyouDto } from '@/classes/dto/export_dto'
import { pruneEmpty } from '@/classes/dto/export_prune'

export interface DnoteDefinition {
    name: string
    items: Array<Array<DnoteItem>>
    lists: Array<DnoteListQuery>
}

export function useDnoteView(options: {
    props: DnoteViewProps,
    emits: DnoteEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const add_dnote_list_dialog = ref<any>(null)
    const add_dnote_item_dialog = ref<any>(null)

    // ── View refs (Map-based, for dynamic :ref bindings) ──
    const item_view_refs = new Map<number, any>()
    const list_view_refs = new Map<number, any>()

    function set_item_table_ref(i: number, el: any): void {
        if (el) item_view_refs.set(i, el)
        else item_view_refs.delete(i)
    }
    function set_list_table_ref(i: number, el: any): void {
        if (el) list_view_refs.set(i, el)
        else list_view_refs.delete(i)
    }

    // ── State refs ──
    const dnote_definitions: Ref<Array<DnoteDefinition>> = ref([])
    const current_definition_index = ref(0)
    const abort_controller = ref(new AbortController())
    const is_loading = ref(true)

    const target_kyous_count = ref(0)
    const getted_kyous_count = ref(0)
    const estimate_aggregate_task = ref(0)
    const finished_aggregate_task = ref(0)

    const first_kyou_date_str = ref("")
    const last_kyou_date_str = ref("")

    const loaded_kyous: Ref<Array<Kyou> | null> = ref(null)
    const last_reload_query: Ref<FindKyouQuery> = ref(new FindKyouQuery())

    // ── Computed ──
    const start_date_str: Ref<string> = computed(() => props.query.use_calendar ? (moment(props.query.calendar_start_date ? props.query.calendar_start_date : moment().toDate()).format("YYYY-MM-DD")) : first_kyou_date_str.value)
    const end_date_str: Ref<string> = computed(() => props.query.use_calendar ? (moment(props.query.calendar_end_date ? props.query.calendar_end_date : moment().toDate()).format("YYYY-MM-DD")) : last_kyou_date_str.value)

    const dnote_item_table_view_data = computed({
        get: () => {
            if (dnote_definitions.value.length === 0) return [[]] as Array<Array<DnoteItem>>
            const idx = current_definition_index.value
            const safeIdx = (idx >= 0 && idx < dnote_definitions.value.length) ? idx : 0
            return dnote_definitions.value[safeIdx].items
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
            const safeIdx = (idx >= 0 && idx < dnote_definitions.value.length) ? idx : 0
            return dnote_definitions.value[safeIdx].lists
        },
        set: (val: Array<DnoteListQuery>) => {
            if (dnote_definitions.value.length === 0) return
            const idx = current_definition_index.value
            if (idx >= 0 && idx < dnote_definitions.value.length) {
                dnote_definitions.value[idx].lists = val
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

    watch(current_definition_index, async (newIdx, oldIdx) => {
        if (newIdx === oldIdx) return
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
        })
    }

    async function load_aggregated_value(ac: AbortController, kyous: Array<Kyou>, query: FindKyouQuery, kyou_is_loaded: boolean) {
        return item_view_refs.get(current_definition_index.value)?.load_aggregated_value(ac, kyous, query, kyou_is_loaded)
    }

    async function load_aggregate_grouping_list(ac: AbortController, kyous: Array<Kyou>, find_kyou_query: FindKyouQuery, kyou_is_loaded: boolean): Promise<void> {
        return await list_view_refs.get(current_definition_index.value)?.load_aggregate_grouping_list(ac, kyous, find_kyou_query, kyou_is_loaded)
    }

    function parse_single_definition_json(def_json: any): DnoteDefinition {
        regist_dictionary()
        const name = def_json.name || i18n.global.t('DNOTE_DEFINITION_DEFAULT_NAME')
        const items: Array<Array<DnoteItem>> = ((def_json && def_json.dnote_item_table_view_data ? def_json.dnote_item_table_view_data : []) || []).map((col: any[]) =>
            col.map((itemJson: any) => {
                const item = new DnoteItem()
                item.id = itemJson.id
                item.prefix = itemJson.prefix
                item.suffix = itemJson.suffix
                item.title = itemJson.title
                item.agregate_target = build_dnote_aggregate_target_from_json(itemJson.aggregate_target)
                item.predicate = build_dnote_predicate_from_json(itemJson.predicate)
                return item
            })
        )
        const lists: Array<DnoteListQuery> = ((def_json && def_json.dnote_list_item_table_view_data ? def_json.dnote_list_item_table_view_data : []) || []).map((queryJson: any) => {
            const query = new DnoteListQuery()
            query.id = queryJson.id
            query.prefix = queryJson.prefix
            query.suffix = queryJson.suffix
            query.title = queryJson.title
            query.aggregate_target = build_dnote_aggregate_target_from_json(queryJson.aggregate_target)
            query.predicate = build_dnote_predicate_from_json(queryJson.predicate)
            query.key_getter = build_dnote_key_getter_from_json(queryJson.key_getter)
            return query
        })
        if (items.length === 0) {
            items.push(new Array<DnoteItem>())
        }
        return { name, items, lists }
    }

    function serialize_single_definition(def: DnoteDefinition): any {
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
                    aggregate_target: dnote_item.agregate_target.to_json(),
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

        return {
            name: def.name,
            dnote_item_table_view_data: dnote_item_table_view_data_serialized,
            dnote_list_item_table_view_data: dnote_list_item_table_view_data_serialized,
        }
    }

    function to_json(): any {
        return dnote_definitions.value.map(serialize_single_definition)
    }

    function from_json(json: any): void {
        regist_dictionary()
        let definitions_json: any[]
        if (Array.isArray(json)) {
            definitions_json = json
        } else if (json && (json.dnote_item_table_view_data || json.dnote_list_item_table_view_data)) {
            definitions_json = [json]
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
        target_kyous_count.value = loaded_kyous.value.length
        getted_kyous_count.value = loaded_kyous.value.length

        abort_controller.value.abort()
        abort_controller.value = new AbortController()
        await nextTick()
        await item_view_refs.get(current_definition_index.value)?.reset()
        await list_view_refs.get(current_definition_index.value)?.reset()

        const kyou_is_loaded = true
        const waitPromises = new Array<Promise<any>>()
        waitPromises.push(load_aggregated_value(abort_controller.value, loaded_kyous.value, last_reload_query.value, kyou_is_loaded))
        waitPromises.push(load_aggregate_grouping_list(abort_controller.value, loaded_kyous.value, last_reload_query.value, kyou_is_loaded))
        await Promise.all(waitPromises)
        is_loading.value = false
    }

    // ── Business logic ──
    async function reload(kyous: Array<Kyou>, query: FindKyouQuery): Promise<void> {
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

        const cloned_kyou = await load_kyous(abort_controller.value, trimed_kyous, true, true)
        const kyou_is_loaded = true
        const waitPromises = new Array<Promise<any>>()
        waitPromises.push(load_aggregated_value(abort_controller.value, cloned_kyou, query, kyou_is_loaded))
        waitPromises.push(load_aggregate_grouping_list(abort_controller.value, cloned_kyou, query, kyou_is_loaded))
        await Promise.all(waitPromises)
        is_loading.value = false
        loaded_kyous.value = cloned_kyou
    }

    async function abort(): Promise<any> {
        abort_controller.value.abort()
        abort_controller.value = new AbortController()
        return reset_view()
    }

    function add_definition(): void {
        const new_def: DnoteDefinition = {
            name: i18n.global.t('DNOTE_DEFINITION_DEFAULT_NAME') + " " + (dnote_definitions.value.length + 1),
            items: [new Array<DnoteItem>()],
            lists: new Array<DnoteListQuery>(),
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

    function floatingActionButtonStyle() {
        return {
            'bottom': '60px',
            'right': '10px',
            'height': '50px',
            'width': '50px',
        }
    }

    async function apply(): Promise<void> {
        const dnote_json_data = to_json()
        emits('requested_apply_dnote', dnote_json_data)
        nextTick(() => emits('requested_close_dialog'))
    }

    // 進捗表示のためかか共通からコピー
    async function load_kyous(ac: AbortController, kyous: Array<Kyou>, get_latest_data: boolean, clone: boolean): Promise<Array<Kyou>> {
        const cloned_kyous = new Array<Kyou>()
        for (let i = 0; i < kyous.length; i++) {
            let kyou: Kyou = kyous[i]
            const waitPromises = []
            if (clone) {
                kyou = kyous[i].clone()
                kyou.abort_controller = ac
            }
            if (get_latest_data) {
                await kyou.reload(false, true)
            }
            if (clone || get_latest_data) {
                waitPromises.push(kyou.load_typed_datas())
                waitPromises.push(kyou.load_attached_tags())
                waitPromises.push(kyou.load_attached_texts())
            }
            await Promise.all(waitPromises)
            cloned_kyous.push(kyou)
            getted_kyous_count.value++
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
            await streamSaveJsonArray(kyous, filename)
            return
        }

        const jsonStr = JSON.stringify(kyous)
        const blob = new Blob([jsonStr], { type: "application/json;charset=utf-8" })
        saveAs(blob, filename)
    }

    async function streamSaveJsonArray(items: any[], filename: string): Promise<void> {
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
                const dto = toExportKyouDto(items[i]);
                const pruned = pruneEmpty(dto);
                if (pruned === undefined) continue;

                const seen = new WeakSet<object>()
                const replacer = (_k: string, v: any) => {
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

    function incrementFinishedAggregateTask(): void {
        finished_aggregate_task.value++
    }

    // ── Event relay objects ──
    const crudRelayHandlers = {
        'deleted_kyou': (...args: any[]) => emits('deleted_kyou', args[0] as Kyou),
        'deleted_tag': (...args: any[]) => emits('deleted_tag', args[0] as Tag),
        'deleted_text': (...args: any[]) => emits('deleted_text', args[0] as Text),
        'deleted_notification': (...args: any[]) => emits('deleted_notification', args[0] as Notification),
        'registered_kyou': (...args: any[]) => emits('registered_kyou', args[0] as Kyou),
        'registered_tag': (...args: any[]) => emits('registered_tag', args[0] as Tag),
        'registered_text': (...args: any[]) => emits('registered_text', args[0] as Text),
        'registered_notification': (...args: any[]) => emits('registered_notification', args[0] as Notification),
        'updated_kyou': (...args: any[]) => emits('updated_kyou', args[0] as Kyou),
        'updated_tag': (...args: any[]) => emits('updated_tag', args[0] as Tag),
        'updated_text': (...args: any[]) => emits('updated_text', args[0] as Text),
        'updated_notification': (...args: any[]) => emits('updated_notification', args[0] as Notification),
    }

    const focusClickRelayHandlers = {
        'focused_kyou': (...args: any[]) => emits('focused_kyou', args[0] as Kyou),
        'clicked_kyou': (...args: any[]) => { emits('focused_kyou', args[0] as Kyou); emits('clicked_kyou', args[0] as Kyou) },
    }

    const rykvDialogHandler = {
        'requested_open_rykv_dialog': (...args: any[]) => emits('requested_open_rykv_dialog', args[0], args[1], args[2]),
    }

    const errorsMessagesRelayHandlers = {
        'received_errors': (...args: any[]) => emits('received_errors', args[0] as Array<GkillError>),
        'received_messages': (...args: any[]) => emits('received_messages', args[0] as Array<GkillMessage>),
    }

    // ── Return ──
    return {
        // Template refs
        add_dnote_list_dialog,
        add_dnote_item_dialog,

        // View ref helpers
        item_view_refs,
        list_view_refs,
        set_item_table_ref,
        set_list_table_ref,

        // State
        dnote_definitions,
        current_definition_index,
        abort_controller,
        is_loading,
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

        // Business logic (exposed for defineExpose)
        reload,
        abort,

        // Template event handlers
        add_definition,
        delete_current_definition,
        floatingActionButtonStyle,
        apply,
        download_kyous_json,
        onRequestedAddDnoteListQuery,
        onRequestedAddDnoteItem,
        incrementFinishedAggregateTask,

        // Event relay objects
        crudRelayHandlers,
        focusClickRelayHandlers,
        rykvDialogHandler,
        errorsMessagesRelayHandlers,
    }
}
