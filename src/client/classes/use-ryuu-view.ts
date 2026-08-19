import { i18n } from '@/i18n'
import { ref, computed, type Ref, watch, nextTick, onUnmounted } from 'vue'
import RelatedKyouQuery from '@/classes/dnote/related-kyou-query'
import type RyuuViewProps from '@/pages/views/ryuu-view-props'
import type RyuuViewEmits from '@/pages/views/ryuu-view-emits'
import { build_dnote_predicate_from_json } from '@/classes/dnote/serialize/register-dictionary'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { ComponentRef } from '@/classes/component-ref'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'
import type { RelatedTimeMatchType } from '@/classes/dnote/related-time-match-type'

export interface RyuuDefinition {
    name: string
    queries: Array<RelatedKyouQuery>
}

export function useRyuuView(options: {
    props: RyuuViewProps,
    emits: RyuuViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const add_ryuu_item_dialog = ref<ComponentRef | null>(null)
    const related_kyou_list_item_views = ref<ComponentRef | null>(null)

    // ── State refs ──
    const ryuu_definitions: Ref<Array<RyuuDefinition>> = ref([])
    const current_definition_index = ref(0)
    const abort_controler: Ref<AbortController> = ref(new AbortController())

    // ── Computed ──
    const related_kyou_queries = computed({
        get: () => {
            if (ryuu_definitions.value.length === 0) return [] as Array<RelatedKyouQuery>
            const idx = current_definition_index.value
            const safe_idx = (idx >= 0 && idx < ryuu_definitions.value.length) ? idx : 0
            return ryuu_definitions.value[safe_idx].queries
        },
        set: (val: Array<RelatedKyouQuery>) => {
            if (ryuu_definitions.value.length === 0) return
            const idx = current_definition_index.value
            if (idx >= 0 && idx < ryuu_definitions.value.length) {
                ryuu_definitions.value[idx].queries = val
            }
        }
    })

    // ── Watchers ──
    watch(current_definition_index, (new_idx, old_idx) => {
        if (new_idx === old_idx) return
        if (!props.editable) {
            abort_controler.value.abort()
            abort_controler.value = new AbortController()
            nextTick(() => load_related_kyou())
        }
    })

    watch(() => props.target_kyou, () => {
        if (props.editable && !props.target_kyou) return
        abort_controler.value.abort()
        abort_controler.value = new AbortController()
        nextTick(() => { load_related_kyou() })
    })

    // ── Lifecycle ──
    nextTick(async () => {
        await load_from_application_config()
        if (props.editable) return

        abort_controler.value.abort()
        abort_controler.value = new AbortController()
        nextTick(() => load_related_kyou())
    })

    onUnmounted(() => {
        abort_controler.value.abort()
        abort_controler.value = new AbortController()
    })

    // ── Internal helpers ──
    async function load_related_kyou(): Promise<void> {
        if (!related_kyou_list_item_views.value) return
        const wait_promises = []
        for (let i = 0; i < related_kyou_list_item_views.value.length; i++) {
            wait_promises.push(related_kyou_list_item_views.value[i].load_related_kyou())
        }
        await Promise.all(wait_promises)
    }

    async function load_from_application_config(): Promise<void> {
        nextTick(() => {
            from_json(props.application_config.ryuu_json_data)
        })
    }

    // 保存済みJSONは外部由来なので unknown で受ける
    function parse_single_definition_queries(json: unknown): Array<RelatedKyouQuery> {
        const queries = new Array<RelatedKyouQuery>()
        if (!Array.isArray(json)) return queries
        for (const raw of json as Array<Record<string, unknown>>) {
            const related_kyou_query = new RelatedKyouQuery()
            related_kyou_query.id = String(raw.id ?? '')
            related_kyou_query.title = String(raw.title ?? '')
            related_kyou_query.prefix = String(raw.prefix ?? '')
            related_kyou_query.suffix = String(raw.suffix ?? '')
            related_kyou_query.predicate = build_dnote_predicate_from_json(raw.predicate as Record<string, unknown>)
            related_kyou_query.related_time_match_type = raw.related_time_match_type as RelatedTimeMatchType
            related_kyou_query.find_kyou_query = raw.find_kyou_query
                ? FindKyouQuery.parse_find_kyou_query(raw.find_kyou_query as Record<string, unknown>)
                : null
            related_kyou_query.find_duration_hour = Number(raw.find_duration_hour ?? 1)
            queries.push(related_kyou_query)
        }
        return queries
    }

    function from_json(json: unknown): void {
        let definitions_json: Array<Record<string, unknown>>
        if (Array.isArray(json) && json.length > 0 && json[0] !== null && typeof json[0] === 'object' && 'name' in json[0] && 'queries' in json[0]) {
            definitions_json = json
        } else if (Array.isArray(json)) {
            definitions_json = [{ name: i18n.global.t('RYUU_DEFINITION_DEFAULT_NAME'), queries: json }]
        } else {
            definitions_json = [{ name: i18n.global.t('RYUU_DEFINITION_DEFAULT_NAME'), queries: [] }]
        }
        ryuu_definitions.value = definitions_json.map((def_json) => ({
            name: String(def_json.name || i18n.global.t('RYUU_DEFINITION_DEFAULT_NAME')),
            queries: parse_single_definition_queries(def_json.queries),
        }))
        if (current_definition_index.value >= ryuu_definitions.value.length) {
            current_definition_index.value = 0
        }
    }

    function serialize_single_definition(def: RyuuDefinition): Record<string, unknown> {
        const json = []
        for (let i = 0; i < def.queries.length; i++) {
            const related_kyou_query = def.queries[i]
            json.push({
                id: related_kyou_query.id,
                title: related_kyou_query.title,
                prefix: related_kyou_query.prefix,
                suffix: related_kyou_query.suffix,
                predicate: related_kyou_query.predicate.predicate_struct_to_json(),
                related_time_match_type: related_kyou_query.related_time_match_type,
                find_kyou_query: related_kyou_query.find_kyou_query,
                find_duration_hour: related_kyou_query.find_duration_hour,
            })
        }
        return { name: def.name, queries: json }
    }

    function to_json(): Array<Record<string, unknown>> {
        return ryuu_definitions.value.map(serialize_single_definition)
    }

    // ── Business logic ──
    function add_definition(): void {
        const new_def: RyuuDefinition = {
            name: i18n.global.t('RYUU_DEFINITION_DEFAULT_NAME') + " " + (ryuu_definitions.value.length + 1),
            queries: new Array<RelatedKyouQuery>(),
        }
        ryuu_definitions.value.push(new_def)
        current_definition_index.value = ryuu_definitions.value.length - 1
    }

    function delete_current_definition(): void {
        if (ryuu_definitions.value.length <= 1) return
        ryuu_definitions.value.splice(current_definition_index.value, 1)
        if (current_definition_index.value >= ryuu_definitions.value.length) {
            current_definition_index.value = ryuu_definitions.value.length - 1
        }
        if (!props.editable) {
            abort_controler.value.abort()
            abort_controler.value = new AbortController()
            nextTick(() => load_related_kyou())
        }
    }

    function add_related_kyou_query(related_kyou_query: RelatedKyouQuery): void {
        related_kyou_queries.value.push(related_kyou_query)
    }

    /**
     * 編集内容を親へ渡すだけにする。
     * 以前は v-model で受け取った ApplicationConfig のプロパティを直接書き換えていたが、
     * それは設定画面の clone そのものなので、設定画面でキャンセルしても戻らなくなっていた。
     * 反映先は requested_apply_ryuu_struct を受けた側が決める
     */
    async function apply(): Promise<void> {
        const ryuu_json_data = to_json()
        emits('requested_apply_ryuu_struct', ryuu_json_data)
        nextTick(() => emits('requested_close_dialog'))
    }

    function floating_action_button_style() {
        return {
            bottom: '60px',
            right: '10px',
            height: '50px',
            width: '50px',
        }
    }

    function delete_related_kyou_query(id: string): void {
        let delete_target_index: number | null = null
        for (let i = 0; i < related_kyou_queries.value.length; i++) {
            if (related_kyou_queries.value[i].id === id) {
                delete_target_index = i
                break
            }
        }
        if (delete_target_index !== null) {
            related_kyou_queries.value.splice(delete_target_index, 1)
        }
    }

    /**
     * FoldableStruct式：上/下挿入で並び替え
     */
    function handle_move_related_kyou_query(src_id: string, target_id: string, drop_type: 'up' | 'down'): void {
        if (!props.editable) return

        const from = related_kyou_queries.value.findIndex(v => v.id === src_id)
        const target = related_kyou_queries.value.findIndex(v => v.id === target_id)
        if (from < 0 || target < 0) return
        if (from === target) return

        const [item] = related_kyou_queries.value.splice(from, 1)

        // remove後のtarget補正
        let t = target
        if (from < target) t = target - 1

        const insert_index = (drop_type === 'up') ? t : (t + 1)
        related_kyou_queries.value.splice(insert_index, 0, item)

        nextTick(() => load_related_kyou())
    }

    // ── Template event handlers (extracted from inline) ──
    function onRequestedMoveRelatedKyouQuery(id0: string, id1: string, direction: 'up' | 'down'): void {
        handle_move_related_kyou_query(id0, id1, direction)
    }

    function onRequestedDeleteRelatedKyouListQuery(id: string): void {
        delete_related_kyou_query(id)
    }

    function onReceivedErrors(errors: Array<GkillError>): void {
        emits('received_errors', errors)
    }

    function onReceivedMessages(messages: Array<GkillMessage>): void {
        emits('received_messages', messages)
    }

    function onRequestedAddRelatedKyouQuery(related_kyou_query: RelatedKyouQuery): void {
        add_related_kyou_query(related_kyou_query)
    }

    function onAddButtonClick(): void {
        add_ryuu_item_dialog.value?.show()
    }

    function onApplyClick(): void {
        apply()
    }

    function onCancelClick(): void {
        emits('requested_close_dialog')
    }

    // ── Event relay objects ──
    // RyuuItemViewから上がってくる18件をそのまま上位へ流す。
    // 手書きで並べるとkyou_view_relay_event_namesの網羅チェック(Exclude<>)の外に出てしまい、
    // KyouViewRelayArgsにイベントを足したときRyuuからだけ黙って落ちる。
    // requested_open_rykv_dialogもこの束に含まれる(Ryuu内ではDialogHostを持たないので、
    // rykv画面のDialogHostで開くように上位へ伝播する)。
    //
    // focused_kyou / clicked_kyou は含まない(view版を使う)。Ryuuの行クリックで
    // rykvのフォーカスKyouが変わるとRyuu自身のtarget_kyouが変わって再検索し続けるため、
    // RyuuItemView側も意図的に発火していない。
    const ryuuItemRelayHandlers = build_kyou_view_relay(emits)

    // ── Return ──
    return {
        // Template refs
        add_ryuu_item_dialog,
        related_kyou_list_item_views,

        // State
        ryuu_definitions,
        current_definition_index,
        abort_controler,
        related_kyou_queries,

        // Business logic
        add_definition,
        delete_current_definition,
        apply,
        floating_action_button_style,

        // Template event handlers
        onRequestedMoveRelatedKyouQuery,
        onRequestedDeleteRelatedKyouListQuery,
        onReceivedErrors,
        onReceivedMessages,
        onRequestedAddRelatedKyouQuery,
        onAddButtonClick,
        onApplyClick,
        onCancelClick,

        // Event relay objects
        ryuuItemRelayHandlers,
    }
}
