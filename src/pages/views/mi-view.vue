<template>
    <div class="mi_view_wrap">
        <v-app-bar :height="app_title_bar_height.valueOf()" class="app_bar" color="primary" app flat>
            <v-app-bar-nav-icon @click.stop="() => { if (inited) { drawer = !drawer } }" />
            <v-toolbar-title>
                <div>
                    <span>
                        mi
                    </span>
                    <v-menu activator="parent">
                        <v-list>
                            <v-list-item v-for="page, index in ['rykv', 'mi', 'kftl', 'plaing', 'mkfl', 'saihate']"
                                :key="index" :value="index">
                                <v-list-item-title @click="router.replace('/' + page)">{{ page
                                }}</v-list-item-title>
                            </v-list-item>
                        </v-list>
                    </v-menu>
                </div>
            </v-toolbar-title>
            <v-spacer />
            <v-btn icon @click="is_show_kyou_detail_view = !is_show_kyou_detail_view">
                <v-icon>mdi-file-document</v-icon>
            </v-btn>
            <v-btn icon @click="is_show_kyou_count_calendar = !is_show_kyou_count_calendar">
                <v-icon>mdi-calendar</v-icon>
            </v-btn>

            <v-divider vertical />
            <v-btn icon="mdi-cog" :disabled="!application_config.is_loaded"
                @click="emits('requested_show_application_config_dialog')" />
        </v-app-bar>
        <v-navigation-drawer v-model="drawer" app :width="300" :height="app_content_height"
            :mobile="drawer_mode_is_mobile" :touchless="!drawer_mode_is_mobile">
            <MiQueryEditorSidebar v-show="inited" class="mi_query_editor_sidebar"
                :application_config="application_config" :gkill_api="gkill_api"
                :app_title_bar_height="app_title_bar_height" :app_content_height="app_content_height"
                :app_content_width="app_content_width" :find_kyou_query="focused_query"
                :inited="false /* これは見られないのでfalseのままでOK */" @requested_search="(update_cache: boolean) => {
                    nextTick(() => search(focused_column_index, querys[focused_column_index], true, update_cache))
                }" @updated_query="(new_query: FindKyouQuery) => {
                    if (!inited) {
                        return
                    }
                    if (skip_search_this_tick || !application_config.rykv_hot_reload) {
                        nextTick(() => skip_search_this_tick = false)
                        return
                    }
                    search(focused_column_index, new_query)
                }" @updated_query_clear="(new_query: FindKyouQuery) => {
                    if (skip_search_this_tick || !application_config.rykv_hot_reload) {
                        nextTick(() => skip_search_this_tick = false)
                        return
                    }
                    skip_search_this_tick = true // 使い方違うけど
                    search(focused_column_index, new_query, application_config.rykv_hot_reload)
                }" @inited="() => { if (!received_init_request) { init() }; received_init_request = true }"
                @request_open_focus_board="(board_name: string) => open_or_focus_board(board_name)"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)" ref="query_editor_sidebar" />
        </v-navigation-drawer>
        <v-main class="main">
            <div class="overlay_target">
                <v-overlay v-model="is_loading" class="align-center justify-center" persistent contained>
                    <v-progress-circular indeterminate color="primary" />
                </v-overlay>
            </div>
            <table class="mi_view_table" v-show="inited">
                <tr>
                    <td valign="top" v-for="query, index in querys" :key="query.query_id">
                        <v-card>
                            <v-card-title v-if="query.use_mi_board_name">{{ query.mi_board_name }}</v-card-title>
                            <v-card-title v-if="!query.use_mi_board_name">すべて</v-card-title>
                            <KyouListView :kyou_height="56 + 35" :width="400"
                                :list_height="kyou_list_view_height.valueOf() - 48"
                                :application_config="application_config" :gkill_api="gkill_api"
                                :matched_kyous="match_kyous_list[index]" :query="query" :last_added_tag="last_added_tag"
                                :is_focused_list="focused_column_index === index" :closable="querys.length !== 1"
                                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                                :is_readonly_mi_check="false" :show_checkbox="false" :show_footer="true"
                                :show_content_only="false" :show_timeis_plaing_end_button="false" @scroll_list="(scroll_top: number) => {
                                    match_kyous_list_top_list[index] = scroll_top
                                    if (inited) {
                                        props.gkill_api.set_saved_mi_scroll_indexs(match_kyous_list_top_list)
                                    }
                                }" @clicked_list_view="() => {
                                    skip_search_this_tick = true
                                    focused_query = querys[index]

                                    if (is_show_kyou_count_calendar) {
                                        update_focused_kyous_list(index)
                                    }
                                    focused_column_index = index
                                    nextTick(() => skip_search_this_tick = false)
                                }" @clicked_kyou="(kyou) => {
                                    skip_search_this_tick = true
                                    focused_query = querys[index]
                                    clicked_kyou_in_list_view(index, kyou)
                                }" @received_errors="(errors) => emits('received_errors', errors)"
                                @received_messages="(messages) => emits('received_messages', messages)"
                                @requested_reload_kyou="(kyou) => reload_kyou(kyou)"
                                @requested_reload_list="() => nextTick(() => reload_list(index))"
                                @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)"
                                @requested_change_focus_kyou="(is_focus_kyou) => {
                                    skip_search_this_tick = true
                                    focused_column_index = index

                                    const query = querys[index].clone()
                                    query.is_focus_kyou_in_list_view = is_focus_kyou
                                    querys.splice(index, 1, query)
                                    querys_backup.splice(index, 1, query)
                                }" @requested_search="() => {
                                    focused_column_index = index
                                    nextTick(() => search(focused_column_index, querys[focused_column_index], true))
                                }" ref="kyou_list_views" @requested_change_is_image_only_view="(is_image_only_view: boolean) => {
                                    focused_column_index = index
                                    focused_kyous_list = match_kyous_list[index]
                                    const query = querys[index].clone()
                                    query.is_image_only = is_image_only_view
                                    querys[index] = query
                                    search(index, query, true)
                                }" @requested_close_column="close_list_view(index)"
                                @deleted_kyou="(deleted_kyou) => { reload_list(index); reload_kyou(deleted_kyou); focused_kyou?.reload() }"
                                @deleted_tag="(deleted_tag) => { }" @deleted_text="(deleted_text) => { }"
                                @deleted_notification="(deleted_notification) => { }"
                                @registered_kyou="(registered_kyou) => { }" @registered_tag="(registered_tag) => { }"
                                @registered_text="(registered_text) => { }"
                                @registered_notification="(registered_notification) => { }"
                                @updated_kyou="(updated_kyou) => reload_kyou(updated_kyou)"
                                @updated_tag="(updated_tag) => { }" @updated_text="(updated_text) => { }"
                                @updated_notification="(updated_notification) => { }" />
                        </v-card>
                    </td>
                    <td valign="top" v-if="is_show_kyou_detail_view">
                        <div class="kyou_detail_view dummy">
                            <KyouView v-if="focused_kyou && is_show_kyou_detail_view"
                                :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                                :is_image_view="false" :kyou="focused_kyou" :last_added_tag="last_added_tag"
                                :show_checkbox="false" :show_content_only="false" :show_mi_create_time="true"
                                :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                                :show_mi_limit_time="true" :show_timeis_elapsed_time="true"
                                :show_timeis_plaing_end_button="true" :height="app_content_height.valueOf()"
                                :is_readonly_mi_check="false" :width="400" :enable_context_menu="enable_context_menu"
                                :enable_dialog="enable_dialog" :show_attached_timeis="true" class="kyou_detail_view"
                                @received_errors="(errors) => emits('received_errors', errors)"
                                @received_messages="(messages) => emits('received_messages', messages)"
                                @requested_reload_kyou="(kyou) => reload_kyou(kyou)" @requested_reload_list="() => { }"
                                @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)" />
                        </div>
                    </td>
                    <td valign="top">
                        <KyouCountCalendar v-show="is_show_kyou_count_calendar" :application_config="application_config"
                            :gkill_api="gkill_api" :kyous="focused_kyous_list" :for_mi="true"
                            @requested_focus_time="(time) => { focused_time = time }" />
                    </td>
                </tr>
            </table>
            <AddTimeisDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :last_added_tag="last_added_tag" :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" @received_errors="(errors) => emits('received_errors', errors)"
                @deleted_kyou="(deleted_kyou) => { reload_kyou(deleted_kyou); focused_kyou?.reload() }"
                @deleted_text="(deleted_text) => { }" @deleted_notification="(deleted_notification) => { }"
                @registered_kyou="(registered_kyou) => { }" @registered_tag="(registered_tag) => { }"
                @registered_text="(registered_text) => { }" @registered_notification="(registered_notification) => { }"
                @updated_kyou="(updated_kyou) => reload_kyou(updated_kyou)" @updated_tag="(updated_tag) => { }"
                @updated_text="(updated_text) => { }" @updated_notification="(updated_notification) => { }"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => reload_kyou(kyou)" @requested_reload_list="() => { }"
                @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)"
                ref="add_timeis_dialog" />
            <AddLantanaDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :last_added_tag="last_added_tag" :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" @received_errors="(errors) => emits('received_errors', errors)"
                @deleted_kyou="(deleted_kyou) => { reload_kyou(deleted_kyou); focused_kyou?.reload() }"
                @deleted_text="(deleted_text) => { }" @deleted_notification="(deleted_notification) => { }"
                @registered_kyou="(registered_kyou) => { }" @registered_tag="(registered_tag) => { }"
                @registered_text="(registered_text) => { }" @registered_notification="(registered_notification) => { }"
                @updated_kyou="(updated_kyou) => reload_kyou(updated_kyou)" @updated_tag="(updated_tag) => { }"
                @updated_text="(updated_text) => { }" @updated_notification="(updated_notification) => { }"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => reload_kyou(kyou)" @requested_reload_list="() => { }"
                @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)"
                ref="add_lantana_dialog" />
            <AddUrlogDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :last_added_tag="last_added_tag" :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" @received_errors="(errors) => emits('received_errors', errors)"
                @deleted_kyou="(deleted_kyou) => { reload_kyou(deleted_kyou); focused_kyou?.reload() }"
                @deleted_text="(deleted_text) => { }" @deleted_notification="(deleted_notification) => { }"
                @registered_kyou="(registered_kyou) => { }" @registered_tag="(registered_tag) => { }"
                @registered_text="(registered_text) => { }" @registered_notification="(registered_notification) => { }"
                @updated_kyou="(updated_kyou) => reload_kyou(updated_kyou)" @updated_tag="(updated_tag) => { }"
                @updated_text="(updated_text) => { }" @updated_notification="(updated_notification) => { }"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => reload_kyou(kyou)" @requested_reload_list="() => { }"
                @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)"
                ref="add_urlog_dialog" />
            <AddMiDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :last_added_tag="last_added_tag" :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" @received_errors="(errors) => emits('received_errors', errors)"
                @deleted_kyou="(deleted_kyou) => { reload_kyou(deleted_kyou); focused_kyou?.reload() }"
                @deleted_text="(deleted_text) => { }" @deleted_notification="(deleted_notification) => { }"
                @registered_kyou="(registered_kyou) => { }" @registered_tag="(registered_tag) => { }"
                @registered_text="(registered_text) => { }" @registered_notification="(registered_notification) => { }"
                @updated_kyou="(updated_kyou) => reload_kyou(updated_kyou)" @updated_tag="(updated_tag) => { }"
                @updated_text="(updated_text) => { }" @updated_notification="(updated_notification) => { }"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => reload_kyou(kyou)" @requested_reload_list="() => { }"
                @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)"
                ref="add_mi_dialog" />
            <AddNlogDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :last_added_tag="last_added_tag" :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" @received_errors="(errors) => emits('received_errors', errors)"
                @deleted_kyou="(deleted_kyou) => { reload_kyou(deleted_kyou); focused_kyou?.reload() }"
                @deleted_text="(deleted_text) => { }" @deleted_notification="(deleted_notification) => { }"
                @registered_kyou="(registered_kyou) => { }" @registered_tag="(registered_tag) => { }"
                @registered_text="(registered_text) => { }" @registered_notification="(registered_notification) => { }"
                @updated_kyou="(updated_kyou) => reload_kyou(updated_kyou)" @updated_tag="(updated_tag) => { }"
                @updated_text="(updated_text) => { }" @updated_notification="(updated_notification) => { }"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => reload_kyou(kyou)" @requested_reload_list="() => { }"
                @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)"
                ref="add_nlog_dialog" />
            <kftlDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :last_added_tag="last_added_tag" :kyou="new Kyou()" :app_content_height="app_content_height"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                :app_content_width="app_content_width" @received_errors="(errors) => emits('received_errors', errors)"
                @registered_tag="(registered_tag) => { }" @registered_text="(registered_text) => { }"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou: Kyou) => reload_kyou(kyou)" @requested_reload_list="() => { }"
                @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)"
                ref="kftl_dialog" />
            <UploadFileDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
                :application_config="application_config" :gkill_api="gkill_api" :last_added_tag="''"
                @deleted_kyou="(deleted_kyou) => { reload_kyou(deleted_kyou); focused_kyou?.reload() }"
                @deleted_text="(deleted_text) => { }" @deleted_notification="(deleted_notification) => { }"
                @registered_kyou="(registered_kyou) => { }" @registered_tag="(registered_tag) => { }"
                @registered_text="(registered_text) => { }" @registered_notification="(registered_notification) => { }"
                @updated_kyou="(updated_kyou) => reload_kyou(updated_kyou)" @updated_tag="(updated_tag) => { }"
                @updated_text="(updated_text) => { }" @updated_notification="(updated_notification) => { }"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)" ref="upload_file_dialog" />
            <v-avatar :style="floatingActionButtonStyle()" color="primary" class="position-fixed">
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
                        <v-list-item @click="show_upload_file_dialog()">
                            <v-list-item-title>アップロード</v-list-item-title>
                        </v-list-item>
                    </v-list>
                </v-menu>
            </v-avatar>
        </v-main>
    </div>
</template>
<script setup lang="ts">
import router from '@/router'
import MiQueryEditorSidebar from './mi-query-editor-sidebar.vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { computed, nextTick, type Ref, ref, watch } from 'vue'
import { Kyou } from '@/classes/datas/kyou'
import AddMiDialog from '../dialogs/add-mi-dialog.vue'
import AddNlogDialog from '../dialogs/add-nlog-dialog.vue'
import KyouCountCalendar from './kyou-count-calendar.vue'
import KyouListView from './kyou-list-view.vue'
import KyouView from './kyou-view.vue'
import kftlDialog from '../dialogs/kftl-dialog.vue'
import type { miViewEmits } from './mi-view-emits'
import type { miViewProps } from './mi-view-props'
import { GetKyousRequest } from '@/classes/api/req_res/get-kyous-request'
import type KftlDialog from '../dialogs/kftl-dialog.vue'
import AddLantanaDialog from '../dialogs/add-lantana-dialog.vue'
import AddTimeisDialog from '../dialogs/add-timeis-dialog.vue'
import AddUrlogDialog from '../dialogs/add-urlog-dialog.vue'
import UploadFileDialog from '../dialogs/upload-file-dialog.vue'
import moment from 'moment'
import { deepEquals } from '@/classes/deep-equals'

const enable_context_menu = ref(true)
const enable_dialog = ref(true)

const query_editor_sidebar = ref<InstanceType<typeof MiQueryEditorSidebar> | null>(null);
const add_mi_dialog = ref<InstanceType<typeof AddMiDialog> | null>(null);
const add_nlog_dialog = ref<InstanceType<typeof AddNlogDialog> | null>(null);
const add_lantana_dialog = ref<InstanceType<typeof AddLantanaDialog> | null>(null);
const add_timeis_dialog = ref<InstanceType<typeof AddTimeisDialog> | null>(null);
const add_urlog_dialog = ref<InstanceType<typeof AddUrlogDialog> | null>(null);
const kftl_dialog = ref<InstanceType<typeof KftlDialog> | null>(null);
const upload_file_dialog = ref<InstanceType<typeof UploadFileDialog> | null>(null);
const kyou_list_views = ref();

const querys: Ref<Array<FindKyouQuery>> = ref([new FindKyouQuery()])
const querys_backup: Ref<Array<FindKyouQuery>> = ref(new Array<FindKyouQuery>()) // 更新検知用バックアップ
const match_kyous_list: Ref<Array<Array<Kyou>>> = ref(new Array<Array<Kyou>>())
const match_kyous_list_top_list: Ref<Array<number>> = ref(new Array<number>())
const focused_query: Ref<FindKyouQuery> = ref(new FindKyouQuery())
const focused_column_index: Ref<number> = ref(0)
const focused_kyous_list: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const focused_kyou: Ref<Kyou | null> = ref(null)
const focused_time: Ref<Date> = ref(moment().toDate())
const is_show_kyou_detail_view: Ref<boolean> = ref(true)
const is_show_kyou_count_calendar: Ref<boolean> = ref(false)
const last_added_tag: Ref<string> = ref("")
const drawer: Ref<boolean | null> = ref(false)
const drawer_mode_is_mobile: Ref<boolean | null> = ref(false)
const kyou_list_view_height = computed(() => props.app_content_height)

const position_x: Ref<Number> = ref(0)
const position_y: Ref<Number> = ref(0)

const props = defineProps<miViewProps>()
const emits = defineEmits<miViewEmits>()

watch(() => is_show_kyou_count_calendar.value, () => {
    focused_kyous_list.value.splice(0)
    if (is_show_kyou_count_calendar.value) {
        update_focused_kyous_list(focused_column_index.value)
    }
})

watch(() => focused_time.value, () => {
    if (!kyou_list_views.value) {
        return
    }
    const kyou_list_view = kyou_list_views.value[focused_column_index.value]
    if (!kyou_list_view) {
        return
    }
    let target_kyou: Kyou | null = null
    for (let i = 0; i < focused_kyous_list.value.length; i++) {
        const kyou = focused_kyous_list.value[i]
        if (kyou.related_time.getTime() >= focused_time.value.getTime()) {
            target_kyou = kyou
            break
        }
    }
    if (inited.value) {
        kyou_list_view.scroll_to_kyou(target_kyou)
    }
})

function update_focused_kyous_list(column_index: number): void {
    focused_kyous_list.value.splice(0)
    for (let i = 0; i < match_kyous_list.value[column_index].length; i++) {
        focused_kyous_list.value.push(match_kyous_list.value[column_index][i])
    }
}

async function reload_kyou(kyou: Kyou): Promise<void> {
    (async (): Promise<void> => {
        for (let i = 0; i < match_kyous_list.value.length; i++) {
            const kyous_list = match_kyous_list.value[i]
            for (let j = 0; j < kyous_list.length; j++) {
                const kyou_in_list = kyous_list[j]
                if (kyou.id === kyou_in_list.id) {
                    const updated_kyou = kyou.clone()
                    await updated_kyou.reload()
                    await updated_kyou.load_all()
                    kyous_list.splice(j, 1, updated_kyou)
                    break
                }
            }
        }
    })();
    (async (): Promise<void> => {
        if (focused_kyou.value && focused_kyou.value.id === kyou.id) {
            const updated_kyou = kyou.clone()
            await updated_kyou.reload()
            await updated_kyou.load_all()
            focused_kyou.value = updated_kyou
        }
    })();
}

async function update_check_kyous(_kyou: Array<Kyou>, _is_checked: boolean): Promise<void> {
    throw new Error('Not implemented')
}

async function reload_list(column_index: number): Promise<void> {
    return search(column_index, querys.value[column_index], true)
}

const is_loading: Ref<boolean> = ref(true)
const inited = ref(false)
const received_init_request = ref(false)
const skip_search_this_tick = ref(false)
const abort_controllers: Ref<Array<AbortController>> = ref([])
async function init(): Promise<void> {
    if (inited.value) {
        return
    }
    return nextTick(async () => {
        const waitPromises = new Array<Promise<void>>()
        try {
            // スクロール位置の復元
            match_kyous_list_top_list.value = props.gkill_api.get_saved_mi_scroll_indexs()

            // 前回開いていた列があれば復元する
            skip_search_this_tick.value = true
            const saved_querys = props.gkill_api.get_saved_mi_find_kyou_querys()
            if (saved_querys.length.valueOf() === 0) {
                const default_query = query_editor_sidebar.value!.get_default_query()!.clone()
                default_query.query_id = props.gkill_api.generate_uuid()
                saved_querys.push(default_query)
            }

            if (props.application_config.rykv_hot_reload) {
                for (let i = 0; i < saved_querys.length; i++) {
                    await nextTick(() => {
                        skip_search_this_tick.value = true
                        waitPromises.push(search(i, saved_querys[i], true).then(async () => {
                            return nextTick(() => {
                                kyou_list_views.value[i].scroll_to(match_kyous_list_top_list.value[i])
                                kyou_list_views.value[i].set_loading(false)
                            })
                        }))
                    })
                }
            } else {
                querys.value = saved_querys.concat()
                querys_backup.value = saved_querys.concat()
                for (let i = 0; i < saved_querys.length; i++) {
                    match_kyous_list.value.push([])
                }
            }
        } finally {
            Promise.all(waitPromises).then(async () => {
                focused_column_index.value = 0
                inited.value = true
                drawer_mode_is_mobile.value = !(props.app_content_width.valueOf() >= 420)
                drawer.value = props.app_content_width.valueOf() >= 420
                is_loading.value = false
                skip_search_this_tick.value = false
            })
        }
    })
}

async function close_list_view(column_index: number): Promise<void> {
    return nextTick(() => {
        skip_search_this_tick.value = true
        focused_column_index.value = -1
        focused_query.value = querys.value[focused_column_index.value]
        focused_kyous_list.value.splice(0)

        querys.value.splice(column_index, 1)
        querys_backup.value.splice(column_index, 1)

        if (abort_controllers.value[column_index]) {
            abort_controllers.value[column_index].abort()
        }

        match_kyous_list.value.splice(column_index, 1)
        match_kyous_list_top_list.value.splice(column_index, 1)
        abort_controllers.value.splice(column_index, 1)

        match_kyous_list_top_list.value.splice(column_index, 1)
        for (let i = column_index; i < querys.value.length; i++) {
            const kyou_list_view = kyou_list_views.value[i] as any
            if (!kyou_list_view) {
                continue
            }
            if (inited.value) {
                kyou_list_view.scroll_to(match_kyous_list_top_list.value[i])
            }
        }
        props.gkill_api.set_saved_mi_find_kyou_querys(querys.value)
        props.gkill_api.set_saved_mi_scroll_indexs(match_kyous_list_top_list.value)
        nextTick(() => {
            skip_search_this_tick.value = true
            focused_column_index.value = 0
        })
    })
}

function add_list_view(query?: FindKyouQuery): void {
    match_kyous_list.value.push(new Array<Kyou>())
    match_kyous_list_top_list.value.push(0)
    // 初期化されていないときはDefaultQueryがない。
    // その場合は初期値のFindKyouQueryをわたして初期化してもらう
    const default_query = query_editor_sidebar.value?.get_default_query()?.clone()
    if (query) {
        querys.value.push(query)
        focused_query.value = query
    } else if (default_query) {
        default_query.query_id = props.gkill_api.generate_uuid()
        querys.value.push(default_query)
        focused_query.value = default_query
    } else {
        const query = new FindKyouQuery()
        query.query_id = props.gkill_api.generate_uuid()
        querys.value.push(query)
        focused_query.value = query
    }
    if (inited.value) {
        focused_column_index.value = querys.value.length - 1
    }
    props.gkill_api.set_saved_mi_find_kyou_querys(querys.value)
    props.gkill_api.set_saved_mi_scroll_indexs(match_kyous_list_top_list.value)
}

function floatingActionButtonStyle() {
    return {
        'bottom': '60px',
        'right': '10px',
        'height': '50px',
        'width': '50px'
    }
}

async function clicked_kyou_in_list_view(column_index: number, kyou: Kyou): Promise<void> {
    focused_kyou.value = kyou
    focused_column_index.value = column_index

    const update_target_column_indexs = new Array<number>()
    for (let i = 0; i < querys.value.length; i++) {
        if (querys.value[i].is_focus_kyou_in_list_view) {
            update_target_column_indexs.push(i)
        }
    }

    for (let i = 0; i < update_target_column_indexs.length; i++) {
        const target_column_index = update_target_column_indexs[i]
        if (inited.value) {
            kyou_list_views.value[target_column_index].scroll_to_time(kyou.related_time)
        }
    }
}

async function search(column_index: number, query: FindKyouQuery, force_search?: boolean, update_cache?: boolean): Promise<void> {


    const query_id = query.query_id
    // 検索する。Tickでまとめる
    return nextTick(async () => {
        try {
            if (!force_search) {
                if (deepEquals(querys_backup.value[column_index], query)) {
                    return
                }
            }
            querys.value[column_index] = query
            querys_backup.value[column_index] = query
            focused_query.value = query

            props.gkill_api.set_saved_mi_find_kyou_querys(querys.value)

            // 前の検索処理を中断する
            if (abort_controllers.value[column_index]) {
                abort_controllers.value[column_index].abort()
            }

            if (match_kyous_list.value[column_index]) {
                match_kyous_list.value[column_index] = []
            }

            const kyou_list_view = kyou_list_views.value[column_index] as any
            if (kyou_list_view && inited.value) {
                kyou_list_view.scroll_to(0)
            }
            await nextTick(async () => {
                const kyou_list_view = kyou_list_views.value[column_index] as any
                if (!kyou_list_view) {
                    return
                }
                kyou_list_view.set_loading(true)
                return nextTick(() => { }) // loading表記切り替え待ち
            })

            const req = new GetKyousRequest()
            abort_controllers.value[column_index] = req.abort_controller
            req.query = query.clone()
            req.query.parse_words_and_not_words()
            if (update_cache) {
                req.query.update_cache = true
            }
            const res = await props.gkill_api.get_kyous(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }

            // 検索後の列位置を取得する
            column_index = -1
            for (let i = 0; i < querys.value.length; i++) {
                const query = querys.value[i]
                if (query.query_id === query_id) {
                    column_index = i
                    break
                }
            }

            if (column_index === -1) {
                return
            }

            match_kyous_list.value[column_index] = res.kyous
            if (is_show_kyou_count_calendar.value) {
                update_focused_kyous_list(column_index)
            }

            await nextTick(() => {
                const kyou_list_view = kyou_list_views.value[column_index] as any
                if (!kyou_list_view) {
                    return
                }
                if (inited.value) {
                    kyou_list_view.scroll_to(0)
                    kyou_list_view.set_loading(false)
                    skip_search_this_tick.value = false
                }
            })
        } catch (err: any) {
            // abortは握りつぶす
            if (!(err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request"))) {
                // abort以外はエラー出力する
                console.error(err)
            }
        }
    })
}

function open_or_focus_board(board_name: string): void {
    if (board_name === "") {
        board_name = "すべて"
    }

    let opened = false
    for (let i = 0; i < querys.value.length; i++) {
        const query = querys.value[i]
        if (query.mi_board_name === board_name) {
            focused_query.value = querys.value[i].clone()

            focused_kyous_list.value.splice(0)
            for (let j = 0; j < match_kyous_list.value[i].length; j++) {
                focused_kyous_list.value.push(match_kyous_list.value[i][j])
            }
            focused_column_index.value = i
            opened = true
            break
        }
    }
    if (opened) {
        return
    }

    let query = query_editor_sidebar.value!.get_default_query()!.clone()
    query.query_id = props.gkill_api.generate_uuid()
    query.mi_board_name = board_name
    if (query.mi_board_name !== "すべて") {
        query.use_mi_board_name = true
    } else {
        query.use_mi_board_name = false
    }

    skip_search_this_tick.value = true
    add_list_view(query)
    if (props.application_config.rykv_hot_reload) {
        search(querys.value.length - 1, query, true)
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

function show_upload_file_dialog(): void {
    upload_file_dialog.value?.show()
}
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
</style>
<style lang="css" scoped>
.overlay_target {
    z-index: -10000;
    position: absolute;
    min-height: calc(v-bind('app_content_height.toString().concat("px")'));
    min-width: calc(100vw);
}
</style>