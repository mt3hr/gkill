import { i18n } from '@/i18n'
import { ref, computed, type Ref, watch, nextTick, onUnmounted } from 'vue'
import RelatedKyouQuery from '@/classes/dnote/related-kyou-query'
import type RyuuViewProps from '@/pages/views/ryuu-view-props'
import type RyuuViewEmits from '@/pages/views/ryuu-view-emits'
import { build_dnote_predicate_from_json } from '@/classes/dnote/serialize/regist-dictionary'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { OpenedRykvDialog, RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import type { ComponentRef } from '@/classes/component-ref'

export interface RyuuDefinition {
    name: string
    queries: Array<RelatedKyouQuery>
}

export function useRyuuView(options: {
    props: RyuuViewProps,
    emits: RyuuViewEmits,
    model_value: Ref<ApplicationConfig | undefined>,
}) {
    const { props, emits, model_value } = options

    // ── Template refs ──
    const add_ryuu_item_dialog = ref<ComponentRef | null>(null)
    const related_kyou_list_item_views = ref<ComponentRef | null>(null)

    // ── State refs ──
    const ryuu_definitions: Ref<Array<RyuuDefinition>> = ref([])
    const current_definition_index = ref(0)
    const opened_dialogs: Ref<Array<OpenedRykvDialog>> = ref([])
    const abort_controler: Ref<AbortController> = ref(new AbortController())

    // ── Computed ──
    const related_kyou_queries = computed({
        get: () => {
            if (ryuu_definitions.value.length === 0) return [] as Array<RelatedKyouQuery>
            const idx = current_definition_index.value
            const safeIdx = (idx >= 0 && idx < ryuu_definitions.value.length) ? idx : 0
            return ryuu_definitions.value[safeIdx].queries
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
    watch(current_definition_index, (newIdx, oldIdx) => {
        if (newIdx === oldIdx) return
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

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    function parse_single_definition_queries(json: any): Array<RelatedKyouQuery> {
        const queries = new Array<RelatedKyouQuery>()
        if (!json) return queries
        for (let i = 0; i < json.length; i++) {
            const related_kyou_query = new RelatedKyouQuery()
            related_kyou_query.id = json[i].id
            related_kyou_query.title = json[i].title
            related_kyou_query.prefix = json[i].prefix
            related_kyou_query.suffix = json[i].suffix
            related_kyou_query.predicate = build_dnote_predicate_from_json(json[i].predicate)
            related_kyou_query.related_time_match_type = json[i].related_time_match_type
            related_kyou_query.find_kyou_query = json[i].find_kyou_query ? FindKyouQuery.parse_find_kyou_query(json[i].find_kyou_query) : null
            related_kyou_query.find_duration_hour = json[i].find_duration_hour
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
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        ryuu_definitions.value = definitions_json.map((def_json: any) => ({
            name: def_json.name || i18n.global.t('RYUU_DEFINITION_DEFAULT_NAME'),
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

    async function apply(): Promise<void> {
        if (!model_value.value) return
        const ryuu_json_data = to_json()
        model_value.value.ryuu_json_data = ryuu_json_data
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        emits('requested_apply_ryuu_struct', ryuu_json_data as any)
        nextTick(() => emits('requested_close_dialog'))
    }

    function floatingActionButtonStyle() {
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
    function handle_move_related_kyou_query(srcId: string, targetId: string, dropType: 'up' | 'down'): void {
        if (!props.editable) return

        const from = related_kyou_queries.value.findIndex(v => v.id === srcId)
        const target = related_kyou_queries.value.findIndex(v => v.id === targetId)
        if (from < 0 || target < 0) return
        if (from === target) return

        const [item] = related_kyou_queries.value.splice(from, 1)

        // remove後のtarget補正
        let t = target
        if (from < target) t = target - 1

        const insertIndex = (dropType === 'up') ? t : (t + 1)
        related_kyou_queries.value.splice(insertIndex, 0, item)

        nextTick(() => load_related_kyou())
    }

    function open_rykv_dialog(kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload): void {
        opened_dialogs.value.push({
            id: props.gkill_api.generate_uuid(),
            kind,
            kyou: kyou.clone(),
            payload: payload ?? null,
            opened_at: Date.now(),
        })
    }

    function close_rykv_dialog(dialog_id: string): void {
        for (let i = 0; i < opened_dialogs.value.length; i++) {
            if (opened_dialogs.value[i].id === dialog_id) {
                opened_dialogs.value.splice(i, 1)
                break
            }
        }
    }

    // ── Template event handlers (extracted from inline) ──
    function onRequestedMoveRelatedKyouQuery(id0: string, id1: string, direction: 'up' | 'down'): void {
        handle_move_related_kyou_query(id0, id1, direction)
    }

    function onRequestedDeleteRelatedKyouListQuery(id: string): void {
        delete_related_kyou_query(id)
    }

    function onDeletedKyou(deleted_kyou: Kyou): void {
        emits('deleted_kyou', deleted_kyou)
    }

    function onDeletedTag(deleted_tag: Tag): void {
        emits('deleted_tag', deleted_tag)
    }

    function onDeletedText(deleted_text: Text): void {
        emits('deleted_text', deleted_text)
    }

    function onDeletedNotification(deleted_notification: Notification): void {
        emits('deleted_notification', deleted_notification)
    }

    function onRegisteredKyou(registered_kyou: Kyou): void {
        emits('registered_kyou', registered_kyou)
    }

    function onRegisteredTag(registered_tag: Tag): void {
        emits('registered_tag', registered_tag)
    }

    function onRegisteredText(registered_text: Text): void {
        emits('registered_text', registered_text)
    }

    function onRegisteredNotification(registered_notification: Notification): void {
        emits('registered_notification', registered_notification)
    }

    function onUpdatedKyou(updated_kyou: Kyou): void {
        emits('updated_kyou', updated_kyou)
    }

    function onUpdatedTag(updated_tag: Tag): void {
        emits('updated_tag', updated_tag)
    }

    function onUpdatedText(updated_text: Text): void {
        emits('updated_text', updated_text)
    }

    function onUpdatedNotification(updated_notification: Notification): void {
        emits('updated_notification', updated_notification)
    }

    function onReceivedErrors(errors: Array<GkillError>): void {
        emits('received_errors', errors)
    }

    function onReceivedMessages(messages: Array<GkillMessage>): void {
        emits('received_messages', messages)
    }

    function onRequestedReloadKyou(kyou: Kyou): void {
        emits('requested_reload_kyou', kyou)
    }

    function onRequestedReloadList(): void {
        emits('requested_reload_list')
    }

    function onRequestedUpdateCheckKyous(kyous: Array<Kyou>, is_checked: boolean): void {
        emits('requested_update_check_kyous', kyous, is_checked)
    }

    function onFocusedKyou(kyou: Kyou): void {
        emits('focused_kyou', kyou)
    }

    function onClickedKyou(kyou: Kyou): void {
        emits('focused_kyou', kyou)
        emits('clicked_kyou', kyou)
    }

    function onRequestedOpenRykvDialog(kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload): void {
        open_rykv_dialog(kind, kyou, payload)
    }

    function onRequestedAddRelatedKyouQuery(related_kyou_query: RelatedKyouQuery): void {
        add_related_kyou_query(related_kyou_query)
    }

    function onDialogHostClosed(dialog_id: string): void {
        close_rykv_dialog(dialog_id)
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
    const ryuuListItemCrudRelayHandlers = {
        'deleted_kyou': (kyou: Kyou) => onDeletedKyou(kyou),
        'deleted_tag': (tag: Tag) => onDeletedTag(tag),
        'deleted_text': (text: Text) => onDeletedText(text),
        'deleted_notification': (notification: Notification) => onDeletedNotification(notification),
        'registered_kyou': (kyou: Kyou) => onRegisteredKyou(kyou),
        'registered_tag': (tag: Tag) => onRegisteredTag(tag),
        'registered_text': (text: Text) => onRegisteredText(text),
        'registered_notification': (notification: Notification) => onRegisteredNotification(notification),
        'updated_kyou': (kyou: Kyou) => onUpdatedKyou(kyou),
        'updated_tag': (tag: Tag) => onUpdatedTag(tag),
        'updated_text': (text: Text) => onUpdatedText(text),
        'updated_notification': (notification: Notification) => onUpdatedNotification(notification),
        'received_errors': (errors: Array<GkillError>) => onReceivedErrors(errors),
        'received_messages': (messages: Array<GkillMessage>) => onReceivedMessages(messages),
    }

    const ryuuListItemRequestHandlers = {
        'requested_reload_kyou': (kyou: Kyou) => onRequestedReloadKyou(kyou),
        'requested_reload_list': () => onRequestedReloadList(),
        'requested_update_check_kyous': (kyous: Array<Kyou>, checked: boolean) => onRequestedUpdateCheckKyous(kyous, checked),
    }

    const ryuuListItemFocusHandlers = {
        'focused_kyou': (kyou: Kyou) => onFocusedKyou(kyou),
        'clicked_kyou': (kyou: Kyou) => onClickedKyou(kyou),
    }

    const rykvDialogHandler = {
        'requested_open_rykv_dialog': (kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => onRequestedOpenRykvDialog(kind, kyou, payload),
    }

    // ── Return ──
    return {
        // Template refs
        add_ryuu_item_dialog,
        related_kyou_list_item_views,

        // State
        ryuu_definitions,
        current_definition_index,
        opened_dialogs,
        abort_controler,
        related_kyou_queries,

        // Business logic
        add_definition,
        delete_current_definition,
        apply,
        floatingActionButtonStyle,

        // Template event handlers
        onRequestedMoveRelatedKyouQuery,
        onRequestedDeleteRelatedKyouListQuery,
        onDeletedKyou,
        onDeletedTag,
        onDeletedText,
        onDeletedNotification,
        onRegisteredKyou,
        onRegisteredTag,
        onRegisteredText,
        onRegisteredNotification,
        onUpdatedKyou,
        onUpdatedTag,
        onUpdatedText,
        onUpdatedNotification,
        onReceivedErrors,
        onReceivedMessages,
        onRequestedReloadKyou,
        onRequestedReloadList,
        onRequestedUpdateCheckKyous,
        onFocusedKyou,
        onClickedKyou,
        onRequestedOpenRykvDialog,
        onRequestedAddRelatedKyouQuery,
        onDialogHostClosed,
        onAddButtonClick,
        onApplyClick,
        onCancelClick,

        // Event relay objects
        ryuuListItemCrudRelayHandlers,
        ryuuListItemRequestHandlers,
        ryuuListItemFocusHandlers,
        rykvDialogHandler,
    }
}
