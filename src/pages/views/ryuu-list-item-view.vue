<template>
    <v-card class="pa-0 ma-0 related_kyou_list_item"
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
                    <span>
                        :
                    </span>
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
                                        <KyouView
                                            v-if="!(match_kyou.data_type.startsWith('lantana') && match_kyou.typed_lantana) && !(match_kyou.data_type.startsWith('kc') && match_kyou.typed_kc)"
                                            :application_config="application_config" :gkill_api="gkill_api"
                                            :highlight_targets="[]" :is_image_view="false" :kyou="match_kyou"
                                            :last_added_tag="''" :show_checkbox="false" :show_content_only="true"
                                            :show_mi_create_time="false" :show_mi_estimate_end_time="false"
                                            :show_mi_estimate_start_time="false" :show_mi_limit_time="false"
                                            :show_timeis_elapsed_time="false" :show_timeis_plaing_end_button="false"
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

const kyou_dialog = ref<InstanceType<typeof KyouDialog> | null>(null);
const contextmenu = ref<InstanceType<typeof RyuuListItemContextMenu> | null>(null);
const edit_related_kyou_query_dialog = ref<InstanceType<typeof EditRyuuItemDialog> | null>(null);
const match_kyou: Ref<Kyou | null> = ref(null)
const is_no_data = ref(false)

const model_value = defineModel<RelatedKyouQuery>()
const props = defineProps<RyuuListItemViewProps>()
const emits = defineEmits<RyuuListItemViewEmits>()
defineExpose({ load_related_kyou })

async function load_related_kyou(): Promise<void> {
    match_kyou.value = null
    is_no_data.value = false

    let related_time = new Date(Date.now())
    if (props.target_kyou) {
        related_time = props.target_kyou.related_time
    }
    const related_time_match_type = model_value.value!.related_time_match_type
    const predicate_for_before = new AndPredicate([
        build_dnote_predicate_from_json(model_value.value!.predicate.predicate_struct_to_json()),
        new RelatedTimeBeforePredicate(related_time),
    ])
    const matcher_for_before = new DnoteMatcher(predicate_for_before)
    const predicate_for_after = new AndPredicate([
        build_dnote_predicate_from_json(model_value.value!.predicate.predicate_struct_to_json()),
        new RelatedTimeAfterPredicate(related_time),
    ])
    const matcher_for_after = new DnoteMatcher(predicate_for_after)
    const find_kyou_query = model_value.value?.find_kyou_query ? model_value.value.find_kyou_query : props.find_kyou_query_default
    find_kyou_query.use_calendar = true
    find_kyou_query.apply_rep_summary_to_detaul(props.application_config)
    find_kyou_query.calendar_start_date = new Date(related_time.getTime() - (model_value.value!.find_duration_hour * 60 * 60 * 1000))
    find_kyou_query.calendar_end_date = new Date(related_time.getTime() + (model_value.value!.find_duration_hour * 60 * 60 * 1000))

    const get_kyous_req = new GetKyousRequest()
    get_kyous_req.abort_controller = props.abort_controller
    get_kyous_req.query = find_kyou_query
    await props.gkill_api.delete_updated_gkill_caches()
    const res = await props.gkill_api.get_kyous(get_kyous_req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    const clone = true
    const get_latest_data = true
    const kyous = await load_kyous(props.abort_controller, res.kyous, get_latest_data, clone)

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
</style>
<style lang="css">
.related_kyou_list_item .lantana_icon {
    position: relative;
    width: 20px !important;
    height: 20px !important;
    max-width: 20px !important;
    min-width: 20px !important;
    max-height: 20px !important;
    min-height: 20px !important;
    display: inline-block;
}

.related_kyou_list_item .lantana_icon_fill {
    width: 20px !important;
    height: 20px !important;
    max-width: 20px !important;
    min-width: 20px !important;
    max-height: 20px !important;
    min-height: 20px !important;
    display: inline-block;
    z-index: 10;
}

.related_kyou_list_item .lantana_icon_harf_left {
    position: absolute;
    left: 0px;
    width: 10px !important;
    height: 20px !important;
    max-width: 10px !important;
    min-width: 10px !important;
    max-height: 20px !important;
    min-height: 20px !important;
    object-fit: cover;
    object-position: 0 0;
    display: inline-block;
    z-index: 10;
}

.related_kyou_list_item .lantana_icon_harf_right {
    position: absolute;
    left: 0px;
    width: 20px !important;
    height: 20px !important;
    max-width: 20px !important;
    min-width: 20px !important;
    max-height: 20px !important;
    min-height: 20px !important;
    display: inline-block;
    z-index: 9;
}

.related_kyou_list_item .lantana_icon_none {
    width: 20px !important;
    height: 20px !important;
    max-width: 20px !important;
    min-width: 20px !important;
    max-height: 20px !important;
    min-height: 20px !important;
    display: inline-block;
    z-index: 10;
}

.related_kyou_list_item .gray {
    filter: grayscale(100%);
}

.related_kyou_list_item .lantana_icon_tr {
    width: calc(20px * 5);
    max-width: calc(20px * 5);
    min-width: calc(20px * 5);
}

.related_kyou_list_item .lantana_icon_td {
    width: 20px !important;
    height: 20px !important;
    max-width: 20px !important;
    min-width: 20px !important;
    max-height: 20px !important;
    min-height: 20px !important;
    display: inline-block;
}
</style>