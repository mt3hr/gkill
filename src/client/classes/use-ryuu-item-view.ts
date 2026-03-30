import { ref, type Ref } from 'vue'
import type { RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import type { Kyou } from '@/classes/datas/kyou'
import { build_dnote_predicate_from_json } from '@/classes/dnote/serialize/regist-dictionary'
import AndPredicate from '@/classes/dnote/dnote-predicate/and-predicate'
import RelatedTimeBeforePredicate from '@/classes/dnote/dnote-predicate/related-time-before-predicate'
import RelatedTimeAfterPredicate from '@/classes/dnote/dnote-predicate/related-time-after-predicate'
import FilterTopKyous from '@/classes/dnote/dnote-filter/filter-top-kyous'
import { GetKyousRequest } from '@/classes/api/req_res/get-kyous-request'
import FilterBottomKyous from '@/classes/dnote/dnote-filter/filter-bottom-kyous'
import { DnoteMatcher } from '@/classes/dnote/dnote-matcher'
import load_kyous from '@/classes/dnote/kyou-loader'
import { RelatedTimeMatchType } from '@/classes/dnote/related-time-match-type'
import moment from 'moment'
import type RelatedKyouQuery from '@/classes/dnote/related-kyou-query'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import EqualTagsTargetKyouPredicate from '@/classes/dnote/dnote-predicate/target-kyou-predicate/equal-tags-target-kyou-predicate'
import EqualTitleTargetKyouPredicate from '@/classes/dnote/dnote-predicate/target-kyou-predicate/equal-title-target-kyou-predicate'
import type DnotePredicate from '@/classes/dnote/dnote-predicate'
import type RyuuItemViewEmits from '@/pages/views/ryuu-item-view-emits'
import type RyuuItemViewProps from '@/pages/views/ryuu-item-view-props'
import type { ComponentRef } from '@/classes/component-ref'

export function useRyuuItemView(options: {
    props: RyuuItemViewProps,
    emits: RyuuItemViewEmits,
    model_value: Ref<RelatedKyouQuery | undefined>,
}) {
    const { props, emits, model_value } = options

    // ── Template refs ──
    const kyou_dialog = ref<ComponentRef | null>(null)
    const contextmenu = ref<ComponentRef | null>(null)
    const edit_related_kyou_query_dialog = ref<ComponentRef | null>(null)

    // ── State refs ──
    const match_kyou: Ref<Kyou | null> = ref(null)
    const is_no_data = ref(false)

    // ── Constants ──
    const _enable_context_menu = props.enable_context_menu
    const _enable_dialog = props.enable_dialog

    /**
     * D&D: FoldableStruct式（上/下判定）
     */
    type DropTypeRyuu = 'up' | 'down'

    function drag_start(e: DragEvent): void {
        if (!props.editable) return
        const id = model_value.value?.id ?? ''
        if (!id) return

        // Firefox対策で何かしら setData が必要なことがある
        e.dataTransfer?.setData('gkill_ryuu_query_id', id)
        if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move'
        e.stopPropagation()
    }

    function dragover(e: DragEvent): void {
        if (!props.editable) return
        if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
        e.preventDefault()      // dropを許可する
        e.stopPropagation()
    }

    function drop(e: DragEvent): void {
        if (!props.editable) return

        const srcId = e.dataTransfer?.getData('gkill_ryuu_query_id')
        const targetId = model_value.value?.id ?? ''
        if (!srcId || !targetId) return
        if (srcId === targetId) return

        // currentTarget基準で上/下を判定（子要素に落ちても安定）
        const el = e.currentTarget as HTMLElement | null
        if (!el) return
        const rect = el.getBoundingClientRect()
        const y = e.clientY - rect.top
        const dropType: DropTypeRyuu = (y <= rect.height * 0.5) ? 'up' : 'down'

        emits('requested_move_related_kyou_query', srcId, targetId, dropType)

        e.preventDefault()
        e.stopPropagation()
    }

    async function load_related_kyou(): Promise<void> {
        match_kyou.value = null
        is_no_data.value = false

        let related_time = new Date(Date.now())
        if (props.target_kyou) {
            related_time = props.target_kyou.related_time
        }
        const ryuu_predicate = build_dnote_predicate_from_json(model_value.value!.predicate.predicate_struct_to_json())
        const related_time_match_type = model_value.value!.related_time_match_type
        const predicate_for_before = new AndPredicate([
            ryuu_predicate,
            new RelatedTimeBeforePredicate(related_time),
        ])
        const matcher_for_before = new DnoteMatcher(predicate_for_before)
        const predicate_for_after = new AndPredicate([
            ryuu_predicate,
            new RelatedTimeAfterPredicate(related_time),
        ])
        const matcher_for_after = new DnoteMatcher(predicate_for_after)
        const find_kyou_query = model_value.value?.find_kyou_query ? model_value.value.find_kyou_query.clone() : props.find_kyou_query_default.clone()
        find_kyou_query.use_calendar = true
        find_kyou_query.apply_rep_summary_to_detaul(props.application_config)

        switch (related_time_match_type) {
            case RelatedTimeMatchType.NEAR_RELATED_TIME: {
                find_kyou_query.calendar_start_date = new Date(related_time.getTime() - (model_value.value!.find_duration_hour * 60 * 60 * 1000))
                find_kyou_query.calendar_end_date = new Date(related_time.getTime() + (model_value.value!.find_duration_hour * 60 * 60 * 1000))
                break
            }
            case RelatedTimeMatchType.NEAR_RELATED_TIME_BEFORE: {
                find_kyou_query.calendar_start_date = new Date(related_time.getTime() - (model_value.value!.find_duration_hour * 60 * 60 * 1000))
                find_kyou_query.calendar_end_date = props.target_kyou && props.target_kyou?.related_time ? props.target_kyou.related_time : null
                break
            }
            case RelatedTimeMatchType.NEAR_RELATED_TIME_AFTER: {
                find_kyou_query.calendar_start_date = props.target_kyou && props.target_kyou?.related_time ? props.target_kyou.related_time : null
                find_kyou_query.calendar_end_date = new Date(related_time.getTime() + (model_value.value!.find_duration_hour * 60 * 60 * 1000))
                break
            }
        }

        // Titleが同じ であれば検索条件に入れる
        if (ryuu_predicate && ryuu_predicate instanceof AndPredicate) {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            (ryuu_predicate as any).predicates.forEach((predicate: DnotePredicate) => {
                if (predicate && predicate instanceof EqualTitleTargetKyouPredicate) {
                    const get_title_func = (kyou: Kyou | null): string | null => {
                        if (kyou === null) return null
                        if (kyou.data_type.startsWith("kmemo")) return kyou.typed_kmemo ? kyou.typed_kmemo.content : null
                        if (kyou.data_type.startsWith("kc")) return kyou.typed_kc ? kyou.typed_kc.title : null
                        if (kyou.data_type.startsWith("urlog")) return kyou.typed_urlog ? kyou.typed_urlog.url : null
                        if (kyou.data_type.startsWith("nlog")) return kyou.typed_nlog ? kyou.typed_nlog.title : null
                        if (kyou.data_type.startsWith("timeis")) return kyou.typed_timeis ? kyou.typed_timeis.title : null
                        if (kyou.data_type.startsWith("mi")) return kyou.typed_mi ? kyou.typed_mi.title : null
                        if (kyou.data_type.startsWith("lantana")) return null
                        if (kyou.data_type.startsWith("idf")) return kyou.typed_idf_kyou ? kyou.typed_idf_kyou.file_name : null
                        if (kyou.data_type.startsWith("git")) return kyou.typed_git_commit_log ? kyou.typed_git_commit_log.commit_message : null
                        if (kyou.data_type.startsWith("rekyou")) return null
                        return null
                    }
                    const title = get_title_func(props.target_kyou)
                    if (title && title !== "") {
                        find_kyou_query.use_words = true
                        find_kyou_query.words = [title]
                    }
                }
            })
            if (ryuu_predicate && ryuu_predicate instanceof AndPredicate) {
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                (ryuu_predicate as any).predicates.forEach((predicate: DnotePredicate) => {
                    if (predicate && predicate instanceof EqualTagsTargetKyouPredicate) {
                        find_kyou_query.use_tags = true
                        find_kyou_query.tags_and = predicate["and"] ? Boolean(predicate["and"]) : false
                        find_kyou_query.tags = props.target_kyou ? props.target_kyou.attached_tags.map(tag => tag.tag) : []
                    }
                })
            }
        }

        const get_kyous_req = new GetKyousRequest()
        get_kyous_req.abort_controller = props.abort_controller
        get_kyous_req.query = find_kyou_query
        await props.gkill_api.delete_updated_gkill_caches()
        const res = await props.gkill_api.get_kyous(get_kyous_req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }

        const trimed_kyous_map = new Map<string, Kyou>()
        for (let i = 0; i < res.kyous.length; i++) {
            trimed_kyous_map.set(res.kyous[i].id, res.kyous[i])
        }
        const trimed_kyous = new Array<Kyou>()
        trimed_kyous_map.forEach((kyou) => trimed_kyous.push(kyou))

        const clone = true
        const get_latest_data = true
        let kyous = new Array<Kyou>()
        switch (related_time_match_type) {
            case RelatedTimeMatchType.NEAR_RELATED_TIME: {
                kyous = await load_kyous(props.abort_controller, trimed_kyous, get_latest_data, clone)
                break
            }
            case RelatedTimeMatchType.NEAR_RELATED_TIME_BEFORE: {
                kyous = await load_kyous(props.abort_controller, trimed_kyous, get_latest_data, clone, predicate_for_before, props.target_kyou, 1)
                break
            }
            case RelatedTimeMatchType.NEAR_RELATED_TIME_AFTER: {
                kyous = await load_kyous(props.abort_controller, trimed_kyous, get_latest_data, clone, predicate_for_after, props.target_kyou, 1)
                break
            }
        }

        const kyou_is_loaded = true
        const limit_count = 1
        const match_kyous_before = await (new FilterTopKyous(limit_count).filter_kyous(
            (await matcher_for_before.match(props.abort_controller, kyous, find_kyou_query, props.target_kyou, kyou_is_loaded)),
            find_kyou_query
        ))
        const match_kyous_after = await (new FilterBottomKyous(limit_count).filter_kyous(
            (await matcher_for_after.match(props.abort_controller, kyous, find_kyou_query, props.target_kyou, kyou_is_loaded)),
            find_kyou_query
        ))

        switch (related_time_match_type) {
            case RelatedTimeMatchType.NEAR_RELATED_TIME: {
                let match_kyou_before: Kyou | null = null
                if (match_kyous_before.length !== 0) {
                    match_kyou_before = match_kyous_before[0]
                }
                let match_kyou_after: Kyou | null = null
                if (match_kyous_after.length !== 0) {
                    match_kyou_after = match_kyous_after[0]
                }
                if (match_kyou_before && !match_kyou_after) {
                    match_kyou.value = match_kyou_before
                } else if (!match_kyou_before && match_kyou_after) {
                    match_kyou.value = match_kyou_after
                } else if (match_kyou_before && match_kyou_after) {
                    if (Math.abs(moment(match_kyou_before.related_time).diff(related_time)) < Math.abs(moment(match_kyou_after.related_time).diff(related_time))) {
                        await match_kyou_before.load_all()
                        match_kyou.value = match_kyou_before
                    } else {
                        await match_kyou_after.load_all()
                        match_kyou.value = match_kyou_after
                    }
                }
                break
            }
            case RelatedTimeMatchType.NEAR_RELATED_TIME_BEFORE: {
                let match_kyou_before: Kyou | null = null
                if (match_kyous_before.length !== 0) {
                    match_kyou_before = match_kyous_before[0]
                    await match_kyou_before.load_all()
                }
                match_kyou.value = match_kyou_before
                break
            }
            case RelatedTimeMatchType.NEAR_RELATED_TIME_AFTER: {
                let match_kyou_after: Kyou | null = null
                if (match_kyous_after.length !== 0) {
                    match_kyou_after = match_kyous_after[0]
                    await match_kyou_after.load_all()
                }
                match_kyou.value = match_kyou_after
                break
            }
        }

        if (!match_kyou.value) {
            is_no_data.value = true
        }
    }

    function show_kyou_dialog(): void {
        if (props.enable_dialog) {
            kyou_dialog.value?.show()
        }
    }

    async function show_context_menu(e: PointerEvent): Promise<void> {
        if (props.editable) {
            contextmenu.value?.show(e)
        }
    }

    async function show_edit_ryuu_item_dialog(): Promise<void> {
        edit_related_kyou_query_dialog.value?.show()
    }

    // ── Event relay objects ──
    const kyouViewRelayHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'focused_kyou': (kyou: Kyou) => emits('focused_kyou', kyou),
        'clicked_kyou': (kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) },
        'requested_open_rykv_dialog': (kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => emits('requested_open_rykv_dialog', kind, kyou, payload),
    }

    const kyouDialogRelayHandlers = {
        'deleted_kyou': (kyou: Kyou) => emits('deleted_kyou', kyou),
        'deleted_tag': (tag: Tag) => emits('deleted_tag', tag),
        'deleted_text': (text: Text) => emits('deleted_text', text),
        'deleted_notification': (notification: Notification) => emits('deleted_notification', notification),
        'registered_kyou': (kyou: Kyou) => emits('registered_kyou', kyou),
        'registered_tag': (tag: Tag) => emits('registered_tag', tag),
        'registered_text': (text: Text) => emits('registered_text', text),
        'registered_notification': (notification: Notification) => emits('registered_notification', notification),
        'updated_kyou': (kyou: Kyou) => emits('updated_kyou', kyou),
        'updated_tag': (tag: Tag) => emits('updated_tag', tag),
        'updated_text': (text: Text) => emits('updated_text', text),
        'updated_notification': (notification: Notification) => emits('updated_notification', notification),
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'focused_kyou': (kyou: Kyou) => emits('focused_kyou', kyou),
        'clicked_kyou': (kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) },
        'requested_reload_kyou': (kyou: Kyou) => emits('requested_reload_kyou', kyou),
        'requested_reload_list': () => emits('requested_reload_list'),
        'requested_update_check_kyous': (kyous: Array<Kyou>, checked: boolean) => emits('requested_update_check_kyous', kyous, checked),
        'requested_open_rykv_dialog': (kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => emits('requested_open_rykv_dialog', kind, kyou, payload),
    }

    const contextMenuRelayHandlers = {
        'requested_delete_related_kyou_query': (value: string) => emits('requested_delete_related_kyou_list_query', value),
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
    }

    // ── Return ──
    return {
        // Template refs
        kyou_dialog,
        contextmenu,
        edit_related_kyou_query_dialog,

        // State
        match_kyou,
        is_no_data,

        // Methods
        drag_start,
        dragover,
        drop,
        load_related_kyou,
        show_kyou_dialog,
        show_context_menu,
        show_edit_ryuu_item_dialog,

        // Event relay objects
        kyouViewRelayHandlers,
        kyouDialogRelayHandlers,
        contextMenuRelayHandlers,
    }
}
