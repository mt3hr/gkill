<template>
    <v-card class="pa-0 ma-0 related_kyou_list_item" :draggable="editable" :class="{ draggable: editable }"
        @dragstart="drag_start" @dragover="dragover" @drop="drop"
        @contextmenu.prevent.stop="(e: any) => { if (editable) { show_context_menu(e) } }"
        @dblclick="() => { if (editable) { show_edit_ryuu_item_dialog() } }">
        <table>
            <tr>
                <td>
                    <span>
                        {{ model_value?.title }}
                    </span>
                </td>
                <td>
                    <span>:</span>
                </td>
                <td>
                    <v-row class="pa-0 ma-0">
                        <v-col class="pa-0 ma-0" cols="auto">
                            <table>
                                <tr>
                                    <td>
                                        <span>{{ model_value?.prefix }}</span>
                                    </td>

                                    <td v-if="is_no_data">
                                        ---
                                    </td>

                                    <td v-if="match_kyou && !is_no_data">
                                        <span
                                            v-if="(match_kyou.data_type.startsWith('lantana') && match_kyou.typed_lantana)">
                                            <LantanaFlowersView :gkill_api="gkill_api"
                                                :application_config="application_config"
                                                :mood="match_kyou.typed_lantana.mood" :editable="false"
                                                @dblclick="() => { if (editable) { show_edit_ryuu_item_dialog() } else { show_kyou_dialog() } }" />
                                        </span>

                                        <span v-if="(match_kyou.data_type.startsWith('kc') && match_kyou.typed_kc)"
                                            @dblclick="() => { if (editable) { show_edit_ryuu_item_dialog() } else { show_kyou_dialog() } }">
                                            {{ match_kyou.typed_kc.num_value }}
                                        </span>

                                        <KyouView :is_image_request_to_thumb_size="false"
                                            v-if="!(match_kyou.data_type.startsWith('lantana') && match_kyou.typed_lantana) && !(match_kyou.data_type.startsWith('kc') && match_kyou.typed_kc)"
                                            :application_config="application_config" :gkill_api="gkill_api"
                                            :highlight_targets="[]" :is_image_view="false" :kyou="match_kyou"
                                            :last_added_tag="''" :show_checkbox="false" :show_content_only="true"
                                            :show_mi_create_time="false" :show_mi_estimate_end_time="false"
                                            :show_mi_estimate_start_time="false" :show_mi_limit_time="false"
                                            :show_timeis_elapsed_time="true" :show_timeis_plaing_end_button="false"
                                            :height="'fit-content'" :enable_context_menu="enable_context_menu"
                                            :enable_dialog="enable_dialog" :show_attached_timeis="false"
                                            :show_update_time="false" :show_related_time="false" :width="'fit-content'"
                                            :is_readonly_mi_check="true" :show_rep_name="false"
                                            :force_show_latest_kyou_info="true" :show_attached_tags="true"
                                            :show_attached_texts="true" :show_attached_notifications="true"
                                            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                                            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" />
                                    </td>

                                    <td>
                                        <span>{{ model_value?.suffix }}</span>
                                    </td>
                                </tr>
                            </table>
                        </v-col>
                    </v-row>
                </td>
            </tr>
        </table>

        <KyouDialog v-if="match_kyou" :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="[]" :kyou="match_kyou" :last_added_tag="''" :enable_context_menu="enable_context_menu"
            :enable_dialog="enable_dialog" :is_readonly_mi_check="false" :show_timeis_plaing_end_button="true"
            @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0] as Kyou)"
            @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
            @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
            @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
            @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
            @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
            @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
            @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
            @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
            @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
            @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
            @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_reload_kyou="(...cloned_kyou: any[]) => emits('requested_reload_kyou', cloned_kyou[0] as Kyou)"
            @requested_reload_list="emits('requested_reload_list')"
            @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)"
            ref="kyou_dialog" />

        <RyuuListItemContextMenu :application_config="application_config" :gkill_api="gkill_api" v-model="model_value"
            @requested_delete_related_kyou_query="(...id: any[]) => emits('requested_delete_related_kyou_list_query', id[0] as string)"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            ref="contextmenu" />

        <EditRyuuItemDialog :application_config="application_config" :gkill_api="gkill_api" v-model="model_value"
            ref="edit_related_kyou_query_dialog" />
    </v-card>
</template>

<script lang="ts" setup>
import { i18n } from '@/i18n'
import EditRyuuItemDialog from '../dialogs/edit-ryuu-item-dialog.vue'
import { ref, type Ref } from 'vue';
import KyouView from './kyou-view.vue';
import type RyuuListItemViewEmits from './ryuu-list-item-view-emits';
import type RyuuListItemViewProps from './ryuu-list-item-view-props';
import KyouDialog from '../dialogs/kyou-dialog.vue';
import type { Kyou } from '@/classes/datas/kyou';
import { build_dnote_predicate_from_json } from '@/classes/dnote/serialize/regist-dictionary';
import AndPredicate from '@/classes/dnote/dnote-predicate/and-predicate';
import RelatedTimeBeforePredicate from '@/classes/dnote/dnote-predicate/related-time-before-predicate';
import RelatedTimeAfterPredicate from '@/classes/dnote/dnote-predicate/related-time-after-predicate';
import FilterTopKyous from '@/classes/dnote/dnote-filter/filter-top-kyous';
import LantanaFlowersView from './lantana-flowers-view.vue';
import { GetKyousRequest } from '@/classes/api/req_res/get-kyous-request';
import FilterBottomKyous from '@/classes/dnote/dnote-filter/filter-bottom-kyous';
import { DnoteMatcher } from '@/classes/dnote/dnote-matcher';
import load_kyous from '@/classes/dnote/kyou-loader';
import { RelatedTimeMatchType } from '@/classes/dnote/related-time-match-type';
import moment from 'moment';
import type RelatedKyouQuery from '../../classes/dnote/related-kyou-query';
import RyuuListItemContextMenu from './ryuu-list-item-context-menu.vue';
import type Predicate from '@/classes/dnote/predicate';
import type PredicateGroupType from '@/classes/dnote/predicate-group-type';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';
import EqualTagsTargetKyouPredicate from '@/classes/dnote/dnote-predicate/target-kyou-predicate/equal-tags-target-kyou-predicate';
import EqualTitleTargetKyouPredicate from '@/classes/dnote/dnote-predicate/target-kyou-predicate/equal-title-target-kyou-predicate';
import type DnotePredicate from '@/classes/dnote/dnote-predicate';
import OrPredicate from '@/classes/dnote/dnote-predicate/or-predicate';

const kyou_dialog = ref<InstanceType<typeof KyouDialog> | null>(null);
const contextmenu = ref<InstanceType<typeof RyuuListItemContextMenu> | null>(null);
const edit_related_kyou_query_dialog = ref<InstanceType<typeof EditRyuuItemDialog> | null>(null);
const match_kyou: Ref<Kyou | null> = ref(null)
const is_no_data = ref(false)

const model_value = defineModel<RelatedKyouQuery>()
const props = defineProps<RyuuListItemViewProps>()
const emits = defineEmits<RyuuListItemViewEmits>()
defineExpose({ load_related_kyou })

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
    // ここから下はあなたの元コードそのまま（変更なし）
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

function predicate_struct_to_json(group: PredicateGroupType | Predicate): any {
    if (is_group(group)) {
        return {
            logic: group.logic,
            predicates: group.predicates.map(p => predicate_struct_to_json(p))
        }
    } else {
        return { type: group.type, value: group.value }
    }
}

function is_group(p: Predicate | PredicateGroupType): p is PredicateGroupType {
    return 'logic' in p && Array.isArray(p.predicates)
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
</script>

<style lang="css" scoped>
.related_kyou_list_item {
    border-top: 1px solid silver;
}

.related_kyou_list_item.draggable {
    cursor: grab;
}

.related_kyou_list_item.draggable:active {
    cursor: grabbing;
}
</style>