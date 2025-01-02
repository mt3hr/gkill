<template>
    <div class="mi_view_wrap">
        <v-overlay v-model="is_loading" class="align-center justify-center" persistent>
            <v-progress-circular indeterminate color="primary" />
        </v-overlay>
        <v-app-bar :height="app_title_bar_height.toString().concat('px')" class="app_bar" color="primary" app flat>
            <v-toolbar-title> {{ share_title }} </v-toolbar-title>
        </v-app-bar>
        <v-main class="main" v-show="inited">
            <table class="mi_view_table">
                <tr>
                    <td valign="top">
                        <v-card>
                            <v-card-title>{{ share_title }}</v-card-title>
                            <KyouListView :kyou_height="56 + 25" :width="400"
                                :list_height="kyou_list_view_height.valueOf() - 48"
                                :application_config="application_config" :gkill_api="gkill_api"
                                :matched_kyous="match_kyous" :query="new FindKyouQuery()" :last_added_tag="''"
                                :is_focused_list="true" :closable="false" :is_readonly_mi_check="true"
                                :show_checkbox="false" :show_footer="false" :enable_context_menu="false"
                                :enable_dialog="false" :show_content_only="false"
                                @requested_reload_kyou="(kyou) => reload_kyou(kyou)"
                                @clicked_kyou="(kyou) => { focused_kyou = kyou }"
                                @received_errors="(errors) => emits('received_errors', errors)"
                                @received_messages="(messages) => emits('received_messages', messages)"
                                ref="kyou_list_views" />
                        </v-card>
                    </td>
                    <td valign="top" v-if="is_show_kyou_detail_view">
                        <div class="kyou_detail_view dummy">
                            <KyouView v-if="focused_kyou && is_show_kyou_detail_view"
                                :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                                :is_image_view="false" :kyou="focused_kyou" :last_added_tag="''" :show_checkbox="false"
                                :show_content_only="false" :show_mi_create_time="true" :show_mi_estimate_end_time="true"
                                :show_mi_estimate_start_time="true" :show_mi_limit_time="true"
                                :show_timeis_plaing_end_button="true" :height="app_content_height.valueOf()"
                                :is_readonly_mi_check="true" :width="400" :enable_context_menu="false"
                                :enable_dialog="false" class="kyou_detail_view"
                                @received_errors="(errors) => emits('received_errors', errors)"
                                @received_messages="(messages) => emits('received_messages', messages)" />
                        </div>
                    </td>
                    <td valign="top">
                        <KyouCountCalendar v-show="is_show_kyou_count_calendar" :application_config="application_config"
                            :gkill_api="gkill_api" :kyous="match_kyous"
                            @requested_focus_time="(time) => { focused_time = time }" />
                    </td>
                </tr>
            </table>
        </v-main>
    </div>
</template>
<script setup lang="ts">
import type { miSharedTaskViewEmits } from './mi-shared-task-view-emits'
import type { miSharedTaskViewProps } from './mi-shared-task-view-props'

import { computed, nextTick, type Ref, ref } from 'vue'
import KyouListView from './kyou-list-view.vue'
import KyouView from './kyou-view.vue'
import { GkillAPI, GkillAPIForSharedMi } from '@/classes/api/gkill-api'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import KyouCountCalendar from './kyou-count-calendar.vue'
import type { Kyou } from '@/classes/datas/kyou'
import { GetSharedMiTasksRequest } from '@/classes/api/req_res/get-shared-mi-tasks-request'

const props = defineProps<miSharedTaskViewProps>()
const emits = defineEmits<miSharedTaskViewEmits>()

const match_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const focused_time: Ref<Date> = ref(new Date())

const share_title: Ref<string> = ref("")
const is_loading: Ref<boolean> = ref(true)
const inited = ref(false)

const kyou_list_view_height = computed(() => props.app_content_height)
const is_show_kyou_detail_view: Ref<boolean> = ref(true)
const is_show_kyou_count_calendar: Ref<boolean> = ref(true)

const focused_kyou: Ref<Kyou | null> = ref(null)

async function load_content(): Promise<void> {
    const req = new GetSharedMiTasksRequest()
    req.shared_id = props.share_id
    const res = await props.gkill_api.get_mi_shared_tasks(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }

    inited.value = true
    is_loading.value = false

    // GkillAPIForSharedMiを設定ここから
    const gkill_api_for_shared_mi = GkillAPIForSharedMi.get_instance_for_share_mi()
    gkill_api_for_shared_mi.kyous = res.mi_kyous
    gkill_api_for_shared_mi.mis = res.mis
    gkill_api_for_shared_mi.tags = res.tags
    gkill_api_for_shared_mi.texts = res.texts
    gkill_api_for_shared_mi.timeiss = res.timeiss
    GkillAPI.set_gkill_api(gkill_api_for_shared_mi)
    // GkillAPIForSharedMiを設定ここまで

    share_title.value = res.title
    match_kyous.value.splice(0)
    for (let i = 0; i < res.mi_kyous.length; i++) {
        match_kyous.value.push(res.mi_kyous[i])
    }
}

async function reload_kyou(kyou: Kyou): Promise<void> {
    const kyous_list = match_kyous.value
    for (let j = 0; j < kyous_list.length; j++) {
        const kyou_in_list = kyous_list[j]
        if (kyou.id === kyou_in_list.id) {
            const updated_kyou = kyou.clone()
            await updated_kyou.reload()
            await updated_kyou.load_all()
            kyous_list.splice(j, 1, updated_kyou)
        }
    }
    if (focused_kyou.value && focused_kyou.value.id === kyou.id) {
        const updated_kyou = kyou.clone()
        await updated_kyou.reload()
        await updated_kyou.load_all()
        focused_kyou.value = updated_kyou
    }
}

nextTick(() => load_content())
</script>
<style lang="css">
.mi_view_table {
    padding-top: 0px;
}

.kyou_detail_view {
    width: calc(400px - 8px);
    max-width: calc(400px - 8px);
    min-width: calc(400px - 8px);
}

.kyou_dialog img.kyou_image,
.kyou_detail_view img.kyou_image {
    width: unset !important;
    height: unset !important;
    max-width: 85vw !important;
    max-height: 85vh !important;
}

.mi_view_wrap {
    position: relative;
}
</style>