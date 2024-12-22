<template>
    <div class="rykv_view_wrap">
        <v-overlay v-model="is_loading" class="align-center justify-center" persistent>
            <v-progress-circular indeterminate color="primary" />
        </v-overlay>
        <v-app-bar :height="app_title_bar_height" class="app_bar" color="primary" app flat>
            <v-app-bar-nav-icon @click.stop="() => { drawer = !drawer }" />
            <v-toolbar-title>rykv</v-toolbar-title>
            <v-spacer />
            <v-btn icon @click="is_show_kyou_detail_view = !is_show_kyou_detail_view">
                <v-icon>mdi-file-document</v-icon>
            </v-btn>
            <v-btn icon @click="is_show_kyou_count_calendar = !is_show_kyou_count_calendar">
                <v-icon>mdi-calendar</v-icon>
            </v-btn>
            <v-btn icon @click="is_show_gps_log_map = !is_show_gps_log_map">
                <v-icon>mdi-map</v-icon>
            </v-btn>
            <v-divider vertical />
            <v-btn icon="mdi-cog" @click="emits('requested_show_application_config_dialog')" />
        </v-app-bar>
        <v-navigation-drawer v-model="drawer" app :width="300" :height="app_content_height">
            <RykvQueryEditorSideBar v-show="!is_loading" class="rykv_query_editor_sidebar"
                :application_config="application_config" :gkill_api="gkill_api"
                :app_title_bar_height="app_title_bar_height" :app_content_height="app_content_height"
                :app_content_width="app_content_width" :find_kyou_query="focused_query"
                :inited="false /* これは見られないのでfalseのままでOK */"
                @requested_search="(update_cache: boolean) => { search(focused_column_index, querys[focused_column_index], true, update_cache) }"
                @updated_query="(new_query) => {
                    if (!inited) {
                        return
                    }
                    if (application_config.rykv_hot_reload) {
                        if (skip_search_this_tick) {
                            nextTick(() => skip_search_this_tick = false)
                            return
                        }
                        search(focused_column_index, new_query)
                    };
                    if (new_query.calendar_start_date && new_query.calendar_end_date) {
                        gps_log_map_start_time = new_query.calendar_start_date
                        gps_log_map_end_time = new_query.calendar_end_date
                    }
                }" @updated_query_clear="(new_query) => {
                    if (application_config.rykv_hot_reload) {
                        if (skip_search_this_tick) {
                            nextTick(() => skip_search_this_tick = false)
                            return
                        }
                        skip_search_this_tick = true // 使い方違うけど
                        search(focused_column_index, new_query, true)
                    };
                    if (new_query.calendar_start_date && new_query.calendar_end_date) {
                        gps_log_map_start_time = new_query.calendar_start_date
                        gps_log_map_end_time = new_query.calendar_end_date
                    }
                }" @inited="() => { if (!received_init_request) { init() }; received_init_request = true }"
                ref="query_editor_sidebar" />
        </v-navigation-drawer>
        <v-main class="main">
            <table class="rykv_view_table">
                <tr>
                    <td valign="top" v-for="query, index in querys">
                        <KyouListView :kyou_height="180" :width="400" :list_height="kyou_list_view_height"
                            :application_config="application_config" :gkill_api="gkill_api"
                            :matched_kyous="match_kyous_list[index]" :query="query" :last_added_tag="last_added_tag"
                            :is_focused_list="focused_column_index === index" :closable="querys.length !== 1"
                            @click="() => { focused_column_index = index; focused_query = querys[index]; focused_kyous_list.splice(0); if (match_kyous_list.length === 0) { return }; for (let i = 0; i < match_kyous_list[index].length; i++) { focused_kyous_list.push(match_kyous_list[index][i]); } }"
                            @clicked_kyou="(kyou) => { clicked_kyou_in_list_view(index, kyou); gps_log_map_start_time = kyou.related_time; gps_log_map_end_time = kyou.related_time; gps_log_map_marker_time = kyou.related_time; }"
                            @received_errors="(errors) => emits('received_errors', errors)"
                            @received_messages="(messages) => emits('received_messages', messages)"
                            @requested_reload_kyou="(kyou) => reload_kyou(kyou)"
                            @requested_reload_list="reload_list(index)"
                            @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)"
                            @requested_change_focus_kyou="(is_focus_kyou) => query.is_focus_kyou = is_focus_kyou"
                            @requested_search="search(focused_column_index, querys[focused_column_index], true)"
                            ref="kyou_list_views" @requested_change_is_image_only_view="(is_image_only_view: boolean) => {
                                const query = querys[index].clone();
                                query.is_image_only = is_image_only_view;
                                querys.splice(index, 1, query);
                                search(focused_column_index, querys[focused_column_index], true);
                            }" @requested_close_column="close_list_view(index)" :key="match_kyous_list.length" />
                    </td>
                    <td valign="top">
                        <v-btn class="rounded-sm mx-auto" :height="app_content_height.valueOf()" :width="30"
                            :color="'primary'" @click="add_list_view()" icon="mdi-plus" variant="text" />
                    </td>
                    <td valign="top" v-if="is_show_kyou_detail_view">
                        <div class="kyou_detail_view dummy">
                            <KyouView v-if="focused_kyou && is_show_kyou_detail_view"
                                :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                                :is_image_view="false" :kyou="focused_kyou" :last_added_tag="last_added_tag"
                                :show_checkbox="false" :show_content_only="false" :show_mi_create_time="true"
                                :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                                :show_mi_limit_time="true" :show_timeis_plaing_end_button="true"
                                :height="app_content_height.valueOf()" :is_readonly_mi_check="false" :width="400"
                                class="kyou_detail_view" @received_errors="(errors) => emits('received_errors', errors)"
                                @received_messages="(messages) => emits('received_messages', messages)"
                                @requested_reload_kyou="(kyou) => reload_kyou(kyou)" @requested_reload_list="() => { }"
                                @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)" />
                        </div>
                    </td>
                    <td valign="top">
                        <KyouCountCalendar v-show="is_show_kyou_count_calendar" :application_config="application_config"
                            :gkill_api="gkill_api" :kyous="focused_kyous_list"
                            @requested_focus_time="(time) => { focused_time = time; gps_log_map_start_time = time; gps_log_map_end_time = time; gps_log_map_marker_time = time }" />
                    </td>
                    <td valign="top">
                        <GPSLogMap v-show="is_show_gps_log_map" :application_config="application_config"
                            :gkill_api="gkill_api" :start_date="gps_log_map_start_time" :end_date="gps_log_map_end_time"
                            :marker_time="gps_log_map_marker_time"
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

            <AddTimeisDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :last_added_tag="last_added_tag" :kyou="new Kyou()"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => reload_kyou(kyou)" @requested_reload_list="() => { }"
                @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)"
                ref="add_timeis_dialog" />
            <AddLantanaDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :last_added_tag="last_added_tag" :kyou="new Kyou()"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => reload_kyou(kyou)" @requested_reload_list="() => { }"
                @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)"
                ref="add_lantana_dialog" />
            <AddUrlogDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :last_added_tag="last_added_tag" :kyou="new Kyou()"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => reload_kyou(kyou)" @requested_reload_list="() => { }"
                @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)"
                ref="add_urlog_dialog" />
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
                        <v-list-item @click="show_urlog_dialog()">
                            <v-list-item-title>urlog</v-list-item-title>
                        </v-list-item>
                        <v-list-item @click="show_timeis_dialog()">
                            <v-list-item-title>timeis</v-list-item-title>
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
    </div>
</template>
<script setup lang="ts">

import { computed, nextTick, type Ref, ref, watch } from 'vue'
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
import AddLantanaDialog from '../dialogs/add-lantana-dialog.vue'
import AddTimeisDialog from '../dialogs/add-timeis-dialog.vue'
import AddUrlogDialog from '../dialogs/add-urlog-dialog.vue'
import moment from 'moment'
import { GetKyousResponse } from '@/classes/api/req_res/get-kyous-response'
import { deepEquals } from '@/classes/deep-equals'

const query_editor_sidebar = ref<InstanceType<typeof RykvQueryEditorSideBar> | null>(null);
const add_mi_dialog = ref<InstanceType<typeof AddMiDialog> | null>(null);
const add_nlog_dialog = ref<InstanceType<typeof AddNlogDialog> | null>(null);
const add_lantana_dialog = ref<InstanceType<typeof AddLantanaDialog> | null>(null);
const add_timeis_dialog = ref<InstanceType<typeof AddTimeisDialog> | null>(null);
const add_urlog_dialog = ref<InstanceType<typeof AddUrlogDialog> | null>(null);
const kftl_dialog = ref<InstanceType<typeof KftlDialog> | null>(null);
const kyou_list_views = ref();

const querys: Ref<Array<FindKyouQuery>> = ref([new FindKyouQuery()])
const querys_backup: Ref<Array<FindKyouQuery>> = ref(new Array<FindKyouQuery>()) // 更新検知用バックアップ
const match_kyous_list: Ref<Array<Array<Kyou>>> = ref(new Array<Array<Kyou>>())
const focused_query: Ref<FindKyouQuery> = ref(new FindKyouQuery())
const focused_column_index: Ref<number> = ref(0)
const focused_kyous_list: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const focused_kyou: Ref<Kyou | null> = ref(null)
const focused_time: Ref<Date> = ref(moment().toDate())
const focused_column_checked_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const gps_log_map_start_time: Ref<Date> = ref(moment().toDate())
const gps_log_map_end_time: Ref<Date> = ref(moment().toDate())
const gps_log_map_marker_time: Ref<Date> = ref(moment().toDate())
const is_show_kyou_detail_view: Ref<boolean> = ref(false)
const is_show_kyou_count_calendar: Ref<boolean> = ref(false)
const is_show_gps_log_map: Ref<boolean> = ref(false)
const last_added_tag: Ref<string> = ref("")
const drawer: Ref<boolean | null> = ref(null)
const kyou_list_view_height = computed(() => props.app_content_height)

const position_x: Ref<Number> = ref(0)
const position_y: Ref<Number> = ref(0)

const props = defineProps<rykvViewProps>()
const emits = defineEmits<rykvViewEmits>()

const skip_search_this_tick = ref(false)

watch(() => focused_column_index.value, () => {
    skip_search_this_tick.value = true
})

watch(() => focused_time.value, () => {
    if (!kyou_list_views) {
        return
    }
    const kyou_list_view = kyou_list_views.value[focused_column_index.value] as any
    kyou_list_view.scroll_to_time(focused_time.value)
})


const is_loading: Ref<boolean> = ref(true)

const inited = ref(false)
const received_init_request = ref(false)
function init(): void {
    nextTick(async () => {
        is_show_kyou_count_calendar.value = props.app_content_width.valueOf() >= 420
        is_show_gps_log_map.value = props.app_content_width.valueOf() >= 420

        // 前回開いていた列があれば復元する
        const saved_querys = GkillAPI.get_instance().get_saved_rykv_find_kyou_querys()
        if (saved_querys.length.valueOf() === 0) {
            await nextTick(() => { })
            search(0, query_editor_sidebar.value?.generate_query()!, true)
            inited.value = true
            return
        }
        close_list_view(0)
        await nextTick(() => { })
        for (let i = 0; i < saved_querys.length; i++) {
            add_list_view(saved_querys[i])
            await nextTick(() => { })
            search(i, querys.value[i], true)
            await nextTick(() => { })
        }
        focused_column_index.value = 0

        if (querys.value[focused_column_index.value].calendar_start_date && querys.value[focused_column_index.value].calendar_end_date) {
            gps_log_map_start_time.value = querys.value[focused_column_index.value].calendar_start_date!
            gps_log_map_end_time.value = querys.value[focused_column_index.value].calendar_end_date!
        }

        inited.value = true
        is_loading.value = false
    })
}

function close_list_view(column_index: number): void {
    skip_search_this_tick.value = true
    focused_column_index.value = 0
    querys.value.splice(column_index, 1);
    querys_backup.value.splice(column_index, 1);
    match_kyous_list.value.splice(column_index, 1);
    abort_controllers.value.splice(column_index, 1);
    GkillAPI.get_instance().set_saved_rykv_find_kyou_querys(querys.value);
}

async function add_list_view(query?: FindKyouQuery): Promise<void> {
    // 初期化されていないときはDefaultQueryがない。
    // その場合は初期値のFindKyouQueryをわたして初期化してもらう
    const default_query = query_editor_sidebar.value?.get_default_query()?.clone()
    if (query) {
        querys.value.push(query)
        focused_query.value = query
    } else if (default_query) {
        default_query.query_id = GkillAPI.get_instance().generate_uuid()
        querys.value.push(default_query)
        focused_query.value = default_query
    } else {
        const query = new FindKyouQuery()
        query.query_id = GkillAPI.get_instance().generate_uuid()
        querys.value.push(query)
        focused_query.value = query
    }
    match_kyous_list.value.push([])
    abort_controllers.value.push(new AbortController())
    focused_column_index.value = querys.value.length - 1
}

async function reload_kyou(kyou: Kyou): Promise<void> {
    for (let i = 0; i < match_kyous_list.value.length; i++) {
        const kyous_list = match_kyous_list.value[i]
        for (let j = 0; j < kyous_list.length; j++) {
            const kyou_in_list = kyous_list[j]
            if (kyou.id === kyou_in_list.id) {
                const updated_kyou = kyou.clone()
                await updated_kyou.reload()
                await updated_kyou.load_all()
                kyous_list.splice(j, 1, updated_kyou)
            }
        }
    }
    if (focused_kyou.value && focused_kyou.value.id === kyou.id) {
        const updated_kyou = kyou.clone()
        await updated_kyou.reload()
        await updated_kyou.load_all()
        focused_kyou.value = updated_kyou
    }
}

async function reload_list(column_index: number): Promise<void> {
    search(column_index, querys.value[column_index], true)
}

async function update_check_kyous(kyous: Array<Kyou>, is_checked: boolean): Promise<void> {
    throw new Error('Not implemented')
}

function clicked_kyou_in_list_view(column_index: number, kyou: Kyou) {
    focused_kyou.value = kyou
    focused_column_index.value = column_index

    const update_target_column_indexs = new Array<number>()
    for (let i = 0; i < querys.value.length; i++) {
        if (querys.value[i].is_focus_kyou) {
            update_target_column_indexs.push(i)
        }
    }

    let scrolled = false
    for (let i = 0; i < update_target_column_indexs.length; i++) {
        const target_column_index = update_target_column_indexs[i]
        for (let j = 0; j < match_kyous_list.value[target_column_index].length; j++) {
            const kyou_in_list = match_kyous_list.value[target_column_index][j]
            if (kyou.id = kyou_in_list.id) {
                kyou_list_views.value[target_column_index].scroll_to_time(kyou.related_time)
                scrolled = true
                break
            }
            if (scrolled) {
                break
            }
        }
    }
}

const abort_controllers: Ref<Array<AbortController>> = ref([])
async function search(column_index: number, query: FindKyouQuery, force_search?: boolean, update_cache?: boolean): Promise<void> {
    focused_query.value = query
    querys.value[column_index] = query
    GkillAPI.get_instance().set_saved_rykv_find_kyou_querys(querys.value)

    // 検索する。Tickでまとめる
    nextTick(async () => {
        if (!force_search) {
            if (querys_backup.value.length > column_index) {
                if (deepEquals(querys_backup.value[column_index], query)) {
                    return
                }
            } else {
                querys_backup.value.length = column_index + 1
            }
        }
        querys_backup.value[column_index] = query

        // 前の検索処理を中断する
        if (abort_controllers.value[column_index]) {
            abort_controllers.value[column_index].abort()
        }

        const kyou_list_view = kyou_list_views.value[column_index] as any
        kyou_list_view.set_loading(true)

        if (match_kyous_list.value[column_index]) {
            match_kyous_list.value[column_index] = []
        }

        const req = new GetKyousRequest()
        abort_controllers.value[column_index] = req.abort_controller
        req.session_id = GkillAPI.get_instance().get_session_id()
        req.query = query.clone()
        req.query.parse_words_and_not_words()
        if (update_cache) {
            req.query.update_cache = true
        }
        try {
            const res = await GkillAPI.get_instance().get_kyous(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }
            match_kyous_list.value[column_index] = res.kyous
            focused_kyous_list.value.splice(0)
            for (let i = 0; i < match_kyous_list.value[column_index].length; i++) {
                focused_kyous_list.value.push(match_kyous_list.value[column_index][i])
            }

            kyou_list_view.set_loading(false)
        } catch (err: any) {
            // abortは握りつぶす
            if (!(err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request"))) {
                // abort以外はエラー出力する
                console.error(err)
            }
        }
    })
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

function show_kftl_dialog(): void {
    kftl_dialog.value?.show()
}

function show_timeis_dialog(): void {
    add_timeis_dialog.value?.show()
}
function show_mi_dialog(): void {
    add_mi_dialog.value?.show()
}

function show_nlog_dialog(): void {
    add_nlog_dialog.value?.show()
}

function show_lantana_dialog(): void {
    add_lantana_dialog.value?.show()
}

function show_urlog_dialog(): void {
    add_urlog_dialog.value?.show()
}

</script>
<style lang="css">
.rykv_view_table {
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
    max-width: -webkit-fill-available !important;
    max-height: 85vh !important;
}

.rykv_view_wrap {
    position: relative;
}
</style>