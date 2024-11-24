<template>
    <ConfirmDeleteShareTaskListDialog :application_config="application_config" :gkill_api="gkill_api"
        :share_mi_task_list_info="share_mi_task_list_info"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
    <KyouCountCalendar :application_config="application_config" :gkill_api="gkill_api" :kyous="focused_board_kyous"
        @requested_focus_time="(time) => focused_time = time" />
    <KyouView :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
        :is_image_view="false" :kyou="focused_mi_kyou" :last_added_tag="last_added_tag" :show_checkbox="false"
        :show_content_only="false" :show_mi_create_time="true" :show_mi_estimate_end_time="true"
        :show_mi_estimate_start_time="true" :show_mi_limit_time="true" :show_timeis_plaing_end_button="true"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => reload_kyou(kyou)" @requested_reload_list="() => { }"
        @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)" />
    <ManageShareTaskListDialog :application_config="application_config" :gkill_api="gkill_api"
        :share_mi_task_list_infos="share_mi_task_list_infos"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_show_confirm_delete_share_task_list_dialog="(share_mi_task_list_info) => show_confirm_delete_share_task_list_dialog(share_mi_task_list_info)"
        @requested_show_share_task_link_dialog="(share_mi_task_list_info) => show_share_task_link_dialog(share_mi_task_list_info)" />
    <ShareTaskListDialog :application_config="application_config" :gkill_api="gkill_api"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
    <ShareTaskListLinkDialog :application_config="application_config" :gkill_api="gkill_api"
        :share_mi_task_list_info="share_mi_task_list_info"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
    <MiBoardTaskListView v-for="find_mi_query, index in find_mi_queries" :application_config="application_config"
        :app_content_height="app_content_height" :app_content_width="app_content_width" :last_added_tag="last_added_tag"
        :matched_kyous="match_kyous_list[index]" :query="generate_find_kyou_query(find_mi_query)" :gkill_api="gkill_api"
        :is_show_close_button="true" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_close_board="(board_name) => close_board(board_name)"
        @requested_focus_kyou="(kyou) => update_focused_kyou(kyou)" @requested_reload_kyou="(kyou) => reload_kyou(kyou)"
        @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => update_check_kyous(kyous, is_checked)" />
    <miQueryEditorSidebar :application_config="application_config" :board_names="board_names" :gkill_api="gkill_api"
        :query="find_mi_queries[focused_board_index]" :sort_type="find_mi_queries[focused_board_index].sort_type"
        @updated_query="(query) => update_query(query)"
        @request_open_focus_board="(board_name) => open_focus_board(board_name)"
        @request_search="() => reload_board()" />
    <miShareFooter :application_config="application_config" :board_names="board_names" :gkill_api="gkill_api"
        :query="find_mi_queries[focused_board_index]" :sort_type="find_mi_queries[focused_board_index].sort_type"
        @request_open_manage_share_mi_dialog="() => show_manage_share_mi_dialog()"
        @request_open_share_mi_dialog="() => show_share_mi_dialog()" />
</template>
<script setup lang="ts">
import ConfirmDeleteShareTaskListDialog from '../dialogs/confirm-delete-share-task-list-dialog.vue'
import KyouCountCalendar from './kyou-count-calendar.vue'
import KyouView from './kyou-view.vue'
import ManageShareTaskListDialog from '../dialogs/manage-share-task-list-dialog.vue'
import ShareTaskListDialog from '../dialogs/share-task-list-dialog.vue'
import ShareTaskListLinkDialog from '../dialogs/share-task-list-link-dialog.vue'
import MiBoardTaskListView from './mi-board-task-list-view.vue'
import miQueryEditorSidebar from './mi-query-editor-sidebar.vue'
import miShareFooter from './mi-share-footer.vue'
import type { miViewEmits } from './mi-view-emits'
import type { miViewProps } from './mi-view-props'
import type { FindMiQuery } from '@/classes/api/find_query/find-mi-query'
import { ref, type Ref } from 'vue'
import { ShareMiTaskListInfo } from '@/classes/datas/share-mi-task-list-info'
import { Kyou } from '@/classes/datas/kyou'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'

const props = defineProps<miViewProps>()
const emits = defineEmits<miViewEmits>()

const find_mi_queries: Ref<Array<FindMiQuery>> = ref(new Array<FindMiQuery>())
const match_kyous_list: Ref<Array<Array<Kyou>>> = ref(new Array<Array<Kyou>>())
const share_mi_task_list_info: Ref<ShareMiTaskListInfo> = ref(new ShareMiTaskListInfo())
const share_mi_task_list_infos: Ref<Array<ShareMiTaskListInfo>> = ref(new Array<ShareMiTaskListInfo>())
const board_names: Ref<Array<string>> = ref(new Array<string>())
const focused_board_index: Ref<number> = ref(0)
const focused_mi_kyou: Ref<Kyou> = ref(new Kyou())
const focused_time: Ref<Date> = ref(new Date())
const focused_board_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const last_added_tag: Ref<string> = ref("")

async function reload_kyou(kyou: Kyou): Promise<void> {
    throw new Error('Not implemented')
}

async function reload_board(): Promise<void> {
    throw new Error('Not implemented')
}

async function update_check_kyous(kyou: Array<Kyou>, is_checked: boolean): Promise<void> {
    throw new Error('Not implemented')
}

async function show_confirm_delete_share_task_list_dialog(share_mi_task_list_info: ShareMiTaskListInfo): Promise<void> {
    throw new Error('Not implemented')
}

async function show_share_task_link_dialog(share_mi_task_list_info: ShareMiTaskListInfo): Promise<void> {
    throw new Error('Not implemented')
}

async function show_manage_share_mi_dialog(): Promise<void> {
    throw new Error('Not implemented')
}

async function show_share_mi_dialog(): Promise<void> {
    throw new Error('Not implemented')
}

async function update_query(query: FindMiQuery): Promise<void> {
    find_mi_queries.value.splice(focused_board_index.value, 1, query)
}

async function close_board(board_name: string): Promise<void> {
    let index = -1
    for (let i = 0; i < find_mi_queries.value.length; i++) {
        const query: FindMiQuery = find_mi_queries.value[i]
        if (query.board_name === board_name) {
            index = i
            break
        }
    }
    if (index !== -1) {
        find_mi_queries.value.splice(index, 1)
        match_kyous_list.value.splice(index, 1)
    }
}

async function update_focused_kyou(kyou: Kyou): Promise<void> {
    throw new Error('Not implemented')
}

function generate_find_kyou_query(find_mi_query: FindMiQuery): FindKyouQuery {
    return find_mi_query.generate_find_kyou_query()
}

async function open_focus_board(board_name: string): Promise<void> {
    throw new Error('Not implemented')
}
</script>
