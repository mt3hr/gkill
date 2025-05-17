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
                                :show_rep_name="false" :force_show_latest_kyou_info="false"
                                @requested_reload_kyou="(kyou) => reload_kyou(kyou)"
                                @clicked_kyou="(kyou) => { focused_kyou = kyou }"
                                @received_errors="(errors) => emits('received_errors', errors)"
                                @received_messages="(messages) => emits('received_messages', messages)"
                                @deleted_kyou="(deleted_kyou) => emits('deleted_kyou', deleted_kyou)"
                                @deleted_tag="(deleted_tag) => emits('deleted_tag', deleted_tag)"
                                @deleted_text="(deleted_text) => emits('deleted_text', deleted_text)"
                                @deleted_notification="(deleted_notification) => emits('deleted_notification', deleted_notification)"
                                @registered_kyou="(registered_kyou) => emits('registered_kyou', registered_kyou)"
                                @registered_tag="(registered_tag) => emits('registered_tag', registered_tag)"
                                @registered_text="(registered_text) => emits('registered_text', registered_text)"
                                @registered_notification="(registered_notification) => emits('registered_notification', registered_notification)"
                                @updated_kyou="(updated_kyou) => emits('updated_kyou', updated_kyou)"
                                @updated_tag="(updated_tag) => emits('updated_tag', updated_tag)"
                                @updated_text="(updated_text) => emits('updated_text', updated_text)"
                                @updated_notification="(updated_notification) => emits('updated_notification', updated_notification)"
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
                                        @requested_focus_time="(time) => { focused_time = time }" />
                                </td>
                            </tr>
                            <tr>
                                <td valign="top" v-if="is_show_kyou_detail_view">
                                    <div class="kyou_detail_view dummy">
                                        <KyouView v-if="focused_kyou && is_show_kyou_detail_view"
                                            :application_config="application_config" :gkill_api="gkill_api"
                                            :highlight_targets="[]" :is_image_view="false" :kyou="focused_kyou"
                                            :last_added_tag="''" :show_checkbox="false" :show_content_only="false"
                                            :show_mi_create_time="true" :show_mi_estimate_end_time="true"
                                            :show_mi_estimate_start_time="true" :show_mi_limit_time="true"
                                            :show_attached_timeis="true" :show_timeis_elapsed_time="false"
                                            :show_timeis_plaing_end_button="true" :height="app_content_height.valueOf()"
                                            :is_readonly_mi_check="true" :width="400" :enable_context_menu="false"
                                            :show_rep_name="false" :force_show_latest_kyou_info="true"
                                            :enable_dialog="false" :show_related_time="true" class="kyou_detail_view"
                                            @received_errors="(errors) => emits('received_errors', errors)"
                                            @received_messages="(messages) => emits('received_messages', messages)" />
                                    </div>
                                </td>
                            </tr>
                        </table>
                    </td>
                </tr>
            </table>
        </v-main>
    </div>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { SharedMiViewProps } from './shared-mi-view-props'

import { computed, nextTick, type Ref, ref, watch } from 'vue'
import KyouListView from './kyou-list-view.vue'
import KyouView from './kyou-view.vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import KyouCountCalendar from './kyou-count-calendar.vue'
import type { Kyou } from '@/classes/datas/kyou'
import type { KyouViewEmits } from './kyou-view-emits'
import { GetKyousRequest } from '@/classes/api/req_res/get-kyous-request'

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

async function load_content(): Promise<void> {
    const get_kyous_req = new GetKyousRequest()
    const res = await props.gkill_api.get_kyous(get_kyous_req)
    match_kyous.value = res.kyous
    is_loading.value = false
}

async function reload_kyou(kyou: Kyou): Promise<void> {
    const kyous_list = match_kyous.value
    for (let j = 0; j < kyous_list.length; j++) {
        const kyou_in_list = kyous_list[j]
        if (kyou.id === kyou_in_list.id) {
            const updated_kyou = kyou.clone()
            await updated_kyou.reload(false)
            await updated_kyou.load_all()
            kyous_list.splice(j, 1, updated_kyou)
        }
    }
    if (focused_kyou.value && focused_kyou.value.id === kyou.id) {
        const updated_kyou = kyou.clone()
        await updated_kyou.reload(false)
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
</script>
<style lang="css">
.mi_view_table {
    padding-top: 0px;
}

.kyou_detail_view .kyou_image {
    width: -webkit-fill-available !important;
    height: -webkit-fill-available !important;
    max-width: -webkit-fill-available !important;
    max-height: 100vh !important;
    object-fit: contain;
}

.kyou_detail_view .kyou_video {
    width: -webkit-fill-available !important;
    height: -webkit-fill-available !important;
    max-width: -webkit-fill-available !important;
    max-height: 100vh !important;
    object-fit: contain;
}

.mi_view_wrap {
    position: relative;
}

.kyou_list_calendar_in_share_mi_view {
    width: 416px;
}
</style>
<style lang="css" scoped>
.overlay_target {
    z-index: -10000;
    position: absolute;
    min-height: calc(v-bind('app_content_height.toString().concat("px")'));
    min-width: calc(100vw);
}
</style>