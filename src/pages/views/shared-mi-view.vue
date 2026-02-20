<template>
    <div class="mi_view_wrap">
        <v-app-bar :height="app_title_bar_height" class="app_bar" color="primary" app flat>
            <v-toolbar-title> {{ share_title }} </v-toolbar-title>
        </v-app-bar>
        <v-main class="main">
            <div class="overlay_target">
                <v-overlay v-model="is_loading" class="align-center justify-center" persistent contained>
                    <v-progress-circular indeterminate color="primary" />
                </v-overlay>
            </div>
            <table class="mi_view_table" v-show="!is_loading">
                <tr>
                    <td valign="top">
                        <v-card>
                            <v-card-title>{{ share_title }}</v-card-title>
                            <KyouListView :kyou_height="56 + 35" :width="400" :show_timeis_plaing_end_button="false"
                                :list_height="kyou_list_view_height.valueOf() - 48"
                                :application_config="application_config" :gkill_api="gkill_api"
                                :matched_kyous="match_kyous" :query="new FindKyouQuery()" :last_added_tag="''"
                                :is_focused_list="true" :closable="false" :is_readonly_mi_check="true"
                                :show_checkbox="false" :show_footer="false" :enable_context_menu="false"
                                :enable_dialog="false" :show_content_only="false"
                                :is_show_doc_image_toggle_button="false" :is_show_arrow_button="false"
                                :show_rep_name="false" :force_show_latest_kyou_info="true"
                                @requested_reload_kyou="(...kyou: any[]) => reload_kyou(kyou[0] as Kyou)"
                                @clicked_kyou="(...kyou: any[]) => { focused_kyou = kyou[0] as Kyou }"
                                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
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
                                @requested_open_rykv_dialog="(...params: any[]) => open_rykv_dialog(params[0], params[1], params[2])"
                                ref="kyou_list_view" />
                        </v-card>
                    </td>
                    <td valign="top" v-if="is_show_kyou_detail_view">
                        <table>
                            <tr>
                                <td valign="top">
                                    <KyouCountCalendar v-show="is_show_kyou_count_calendar"
                                        :application_config="application_config" :gkill_api="gkill_api"
                                        :kyous="match_kyous" :for_mi="true" class="kyou_list_calendar_in_share_mi_view"
                                        @requested_focus_time="(...time: any[]) => { focused_time = time[0] as Date }" />
                                </td>
                            </tr>
                            <tr>
                                <td valign="top" v-if="is_show_kyou_detail_view">
                                    <div class="kyou_detail_view dummy">
                                        <KyouView v-if="focused_kyou && is_show_kyou_detail_view" :is_image_request_to_thumb_size="false"
                                            :application_config="application_config" :gkill_api="gkill_api"
                                            :highlight_targets="[]" :is_image_view="false" :kyou="focused_kyou"
                                            :last_added_tag="''" :show_checkbox="false" :show_content_only="false"
                                            :show_mi_create_time="true" :show_mi_estimate_end_time="true"
                                            :show_mi_estimate_start_time="true" :show_mi_limit_time="true"
                                            :show_attached_timeis="true" :show_timeis_elapsed_time="false"
                                            :show_timeis_plaing_end_button="true" :height="app_content_height.valueOf()"
                                            :is_readonly_mi_check="true" :width="400" :enable_context_menu="false"
                                            :show_rep_name="false" :force_show_latest_kyou_info="true"
                                            :enable_dialog="false" :show_update_time="false" :show_related_time="true"
                                            class="kyou_detail_view" :show_attached_tags="true"
                                            :show_attached_texts="true" :show_attached_notifications="true"
                                            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                                            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                                            @requested_open_rykv_dialog="(...params: any[]) => open_rykv_dialog(params[0], params[1], params[2])" />
                                    </div>
                                </td>
                            </tr>
                        </table>
                    </td>
                </tr>
            </table>
            <RykvDialogHost :application_config="application_config" :gkill_api="gkill_api" :dialogs="opened_dialogs"
                :last_added_tag="''" :enable_context_menu="false" :enable_dialog="false"
                @closed="(...id: any[]) => close_rykv_dialog(id[0] as string)"
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
                @requested_reload_kyou="(...kyou: any[]) => reload_kyou(kyou[0] as Kyou)"
                @requested_reload_list="() => { }"
                @requested_open_rykv_dialog="(...params: any[]) => open_rykv_dialog(params[0], params[1], params[2])" />
        </v-main>
    </div>
</template>
<script setup lang="ts">
import type { SharedMiViewProps } from './shared-mi-view-props'

import { computed, nextTick, type Ref, ref, watch } from 'vue'
import KyouListView from './kyou-list-view.vue'
import KyouView from './kyou-view.vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import KyouCountCalendar from './kyou-count-calendar.vue'
import type { Kyou } from '@/classes/datas/kyou'
import type { KyouViewEmits } from './kyou-view-emits'
import { GetKyousRequest } from '@/classes/api/req_res/get-kyous-request'
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import RykvDialogHost from './rykv-dialog-host.vue'
import type { OpenedRykvDialog, RykvDialogKind, RykvDialogPayload } from './rykv-dialog-kind'

const kyou_list_view = ref();

const props = defineProps<SharedMiViewProps>()
const emits = defineEmits<KyouViewEmits>()

const match_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const focused_time: Ref<Date> = ref(new Date())

const share_title: Ref<string> = ref(props.share_title)
const is_loading: Ref<boolean> = ref(true)

const kyou_list_view_height = computed(() => props.app_content_height)
const is_show_kyou_detail_view: Ref<boolean> = ref(true)
const is_show_kyou_count_calendar: Ref<boolean> = ref(true)

const focused_kyou: Ref<Kyou | null> = ref(null)
const opened_dialogs: Ref<Array<OpenedRykvDialog>> = ref([])

async function load_content(): Promise<void> {
    const get_kyous_req = new GetKyousRequest()
    await props.gkill_api.delete_updated_gkill_caches()
    const res = await props.gkill_api.get_kyous(get_kyous_req)
    const wait_promises = new Array<Promise<any>>()
    for (let i = 0; i < res.kyous.length; i++) {
        wait_promises.push(res.kyous[i].load_all())
    }
    await Promise.all(wait_promises)
    match_kyous.value = res.kyous
    is_loading.value = false
}

async function reload_kyou(kyou: Kyou): Promise<void> {
    const kyous_list = match_kyous.value
    for (let j = 0; j < kyous_list.length; j++) {
        const kyou_in_list = kyous_list[j]
        if (kyou.id === kyou_in_list.id) {
            const updated_kyou = kyou.clone()
            await updated_kyou.reload(false, true)
            await updated_kyou.load_all()
            kyous_list.splice(j, 1, updated_kyou)
        }
    }
    if (focused_kyou.value && focused_kyou.value.id === kyou.id) {
        const updated_kyou = kyou.clone()
        await updated_kyou.reload(false, true)
        await updated_kyou.load_all()
        focused_kyou.value = updated_kyou
    }
}

watch(() => focused_time.value, () => {
    if (!kyou_list_view.value) {
        return
    }
    let target_kyou: Kyou | null = null
    for (let i = 0; i < match_kyous.value.length; i++) {
        const kyou = match_kyous.value[i]
        if (kyou.related_time.getTime() >= focused_time.value.getTime()) {
            target_kyou = kyou
            break
        }
    }
    (kyou_list_view as any).value.scroll_to_kyou(target_kyou)
})

nextTick(() => load_content())

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
</script>
<style lang="css" scoped>
.overlay_target {
    z-index: -10000;
    position: absolute;
    min-height: calc(v-bind('app_content_height.toString().concat("px")'));
    min-width: calc(100vw);
}
</style>
<style lang="css">
.mi_view_wrap .v-calendar-weekly__head {
    width: unset !important;
}
</style>
