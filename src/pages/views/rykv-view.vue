<template>
    <v-app-bar :height="app_title_bar_height" class="app_bar" color="primary" app flat>
        <v-app-bar-nav-icon @click.stop="() => { drawer = !drawer }" />
        <v-toolbar-title>rykv</v-toolbar-title>
        <v-spacer />
        <v-btn icon="mdi-cog" @click="emits('requested_show_application_config_dialog')" />
    </v-app-bar>
    <v-navigation-drawer v-model="drawer" app :width="300">
        <RykvQueryEditorSideBar :application_config="application_config" :gkill_api="gkill_api"
            :app_title_bar_height="app_title_bar_height" :app_content_height="app_content_height"
            :app_content_width="app_content_width" :find_kyou_query="querys[focused_column_index]"
            @requested_search="search(focused_column_index)"
            @updated_query="new_query => querys.splice(focused_column_index, 1, new_query)"
            ref="query_editor_sidebar" />
    </v-navigation-drawer>
    <v-main class="main">
        <table class="rykv_view_table">
            <tr>
                <td valign="top">
                    <KyouListView :kyou_height="180" :width="400" :list_height="kyou_list_view_height"
                        v-for="query, index in querys" :application_config="application_config" :gkill_api="gkill_api"
                        :matched_kyous="match_kyous_list[index]" :query="query" :last_added_tag="last_added_tag"
                        @received_errors="(errors) => emits('received_errors', errors)"
                        @received_messages="(messages) => emits('received_messages', messages)"
                        @requested_reload_kyou="(kyou) => reload_kyou(kyou)" @requested_reload_list="reload_list(index)"
                        @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)" />
                </td>
                <td valign="top">
                    <KyouView :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                        :is_image_view="false" :kyou="focused_kyou" :last_added_tag="last_added_tag"
                        :show_checkbox="false" :show_content_only="false" :show_mi_create_time="true"
                        :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true" :show_mi_limit_time="true"
                        :show_timeis_plaing_end_button="true" :height="app_content_height.valueOf()" :width="400"
                        @received_errors="(errors) => emits('received_errors', errors)"
                        @received_messages="(messages) => emits('received_messages', messages)"
                        @requested_reload_kyou="(kyou) => reload_kyou(kyou)" @requested_reload_list="() => { }"
                        @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)" />
                </td>
                <td valign="top">
                    <KyouCountCalendar :application_config="application_config" :gkill_api="gkill_api"
                        :kyous="focused_column_kyous" @requested_focus_time="(time) => focused_time = time" />
                </td>
                <td valign="top">
                    <GPSLogMap :application_config="application_config" :gkill_api="gkill_api"
                        :start_date="focused_time" :end_date="focused_time"
                        @received_errors="(errors) => emits('received_errors', errors)"
                        @received_messages="(messages) => emits('received_messages', messages)"
                        @requested_focus_time="(time) => focused_time = time" />
                </td>
            </tr>
        </table>
        <Dnote :app_content_height="app_content_height" :app_content_width="app_content_width"
            :application_config="application_config" :gkill_api="gkill_api" :query="querys[focused_column_index]"
            :last_added_tag="last_added_tag" @received_messages="(messages) => emits('received_messages', messages)"
            @received_errors="(errors) => emits('received_errors', errors)" />
        <AggregateView :application_config="application_config" :gkill_api="gkill_api"
            :checked_kyous="focused_column_checked_kyous"
            @received_messages="(messages) => emits('received_messages', messages)"
            @received_errors="(errors) => emits('received_errors', errors)" />

        <AddMiDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
            :last_added_tag="last_added_tag" :kyou="new Kyou()"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => reload_kyou(kyou)" @requested_reload_list="() => { }"
            @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)"
            ref="add_mi_dialog" />
        <AddNlogDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
            :last_added_tag="last_added_tag" :kyou="new Kyou()"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => reload_kyou(kyou)" @requested_reload_list="() => { }"
            @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)"
            ref="add_nlog_dialog" />
        <EndTimeIsPlaingDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
            :last_added_tag="last_added_tag" :kyou="new Kyou()"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => reload_kyou(kyou)" @requested_reload_list="() => { }"
            @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)"
            ref="end_timeis_plaing_dialog" />
        <StartTimeIsDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
            :last_added_tag="last_added_tag" :kyou="new Kyou()"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => reload_kyou(kyou)" @requested_reload_list="() => { }"
            @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)"
            ref="start_timeis_dialog" />
        <kftlDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
            :last_added_tag="last_added_tag" :kyou="new Kyou()" :app_content_height="app_content_height"
            :app_content_width="app_content_width" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou: Kyou) => reload_kyou(kyou)" @requested_reload_list="() => { }"
            @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)"
            ref="kftl_dialog" />

        <v-avatar :style="floatingActionButtonStyle()" color="indigo" class="position-fixed">
            <v-menu :style="add_kyou_menu_style" transition="slide-x-transition">
                <template v-slot:activator="{ props }">
                    <v-btn color="white" icon="mdi-plus" variant="text" v-bind="props" />
                </template>
                <v-list>
                    <v-list-item @click="show_kftl_dialog()">
                        <v-list-item-title>kftl</v-list-item-title>
                    </v-list-item>
                    <v-list-item @click="show_timeis_start_dialog()">
                        <v-list-item-title>timeis start</v-list-item-title>
                    </v-list-item>
                    <v-list-item @click="show_mi_dialog()">
                        <v-list-item-title>mi</v-list-item-title>
                    </v-list-item>
                    <v-list-item @click="show_nlog_dialog()">
                        <v-list-item-title>nlog</v-list-item-title>
                    </v-list-item>
                    <v-list-item @click="show_lantana_dialog()">
                        <v-list-item-title>lantana</v-list-item-title>
                    </v-list-item>
                </v-list>
            </v-menu>
        </v-avatar>
    </v-main>
</template>
<script setup lang="ts">

import { computed, nextTick, type Ref, ref } from 'vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { Kyou } from '@/classes/datas/kyou'
import AddMiDialog from '../dialogs/add-mi-dialog.vue'
import AddNlogDialog from '../dialogs/add-nlog-dialog.vue'
import AggregateView from './aggregate-view.vue'
import Dnote from './dnote.vue'
import EndTimeIsPlaingDialog from '../dialogs/end-time-is-plaing-dialog.vue'
import GPSLogMap from './gps-log-map.vue'
import KyouCountCalendar from './kyou-count-calendar.vue'
import KyouListView from './kyou-list-view.vue'
import KyouView from './kyou-view.vue'
import StartTimeIsDialog from '../dialogs/start-time-is-dialog.vue'
import RykvQueryEditorSideBar from './rykv-query-editor-side-bar.vue'
import kftlDialog from '../dialogs/kftl-dialog.vue'
import type { rykvViewEmits } from './rykv-view-emits'
import type { rykvViewProps } from './rykv-view-props'
import { GkillAPI } from '@/classes/api/gkill-api'
import { GetKyousRequest } from '@/classes/api/req_res/get-kyous-request'
import type KftlDialog from '../dialogs/kftl-dialog.vue'

const query_editor_sidebar = ref<InstanceType<typeof RykvQueryEditorSideBar> | null>(null);
const add_mi_dialog = ref<InstanceType<typeof AddMiDialog> | null>(null);
const add_nlog_dialog = ref<InstanceType<typeof AddNlogDialog> | null>(null);
const start_timeis_dialog = ref<InstanceType<typeof StartTimeIsDialog> | null>(null);
const kftl_dialog = ref<InstanceType<typeof KftlDialog> | null>(null);

const querys: Ref<Array<FindKyouQuery>> = ref((() => { const queries = new Array<FindKyouQuery>(); queries.push(new FindKyouQuery()); return queries })())
const match_kyous_list: Ref<Array<Array<Kyou>>> = ref(new Array<Array<Kyou>>())
const focused_column_index: Ref<number> = ref(0)
const focused_kyou: Ref<Kyou> = ref(new Kyou())
const focused_list_views_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const focused_time: Ref<Date> = ref(new Date())
const focused_column_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const focused_column_checked_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const is_show_kyou_detail_view: Ref<boolean> = ref(false)
const is_show_kyou_count_calendar: Ref<boolean> = ref(false)
const is_show_gps_log_map: Ref<boolean> = ref(false)
const last_added_tag: Ref<string> = ref("")
const drawer: Ref<boolean | null> = ref(null)
const kyou_list_view_height = computed(() => props.app_content_height)
const is_show_add_kyou_menu: Ref<boolean> = ref(false)

const position_x: Ref<Number> = ref(0)
const position_y: Ref<Number> = ref(0)

const props = defineProps<rykvViewProps>()
const emits = defineEmits<rykvViewEmits>()

nextTick(() => query_editor_sidebar.value?.generate_query())

async function add_list_view(): Promise<void> {
    throw new Error('Not implemented')
}
async function close_list_view(query_index: Number): Promise<void> {
    throw new Error('Not implemented')
}
async function update_queries(query_index: Number, by_user: boolean): Promise<void> {
    throw new Error('Not implemented')
}
async function update_kyous(column_index: Number, kyous: Array<Kyou>): Promise<void> {
    throw new Error('Not implemented')
}

async function reload_kyou(kyou: Kyou): Promise<void> {
    throw new Error('Not implemented')
}

async function reload_list(column_index: number): Promise<void> {
    throw new Error('Not implemented')
}

async function update_check_kyous(kyous: Array<Kyou>, is_checked: boolean): Promise<void> {
    throw new Error('Not implemented')
}

async function search(column_index: number): Promise<void> {
    match_kyous_list.value = []
    const req = new GetKyousRequest()
    req.session_id = GkillAPI.get_instance().get_session_id()
    req.query = querys.value[column_index]
    const res = await GkillAPI.get_instance().get_kyous(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    match_kyous_list.value[column_index] = res.kyous
}

function floatingActionButtonStyle() {
    return {
        'bottom': '10px',
        'right': '10px',
        'height': '50px',
        'width': '50px'
    }
}

const add_kyou_menu_style = computed(() => `{ position: absolute; left: ${position_x.value}px; top: ${position_y.value}px; }`)

async function show_add_kyou_menu(e: PointerEvent): Promise<void> {
    position_x.value = e.clientX
    position_y.value = e.clientY
    is_show_add_kyou_menu.value = true
}

function show_kftl_dialog(): void {
    kftl_dialog.value?.show()
}

function show_timeis_start_dialog(): void {
    start_timeis_dialog.value?.show()
}
function show_mi_dialog(): void {
    add_mi_dialog.value?.show()
}

function show_nlog_dialog(): void {
    add_nlog_dialog.value?.show()
}

function show_lantana_dialog(): void {
    // TODO
}

</script>
<style lang="css">
.rykv_view_table {
    padding-top: 0px;
}
</style>