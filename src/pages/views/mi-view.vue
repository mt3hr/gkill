<template>
    <div class="mi_view_wrap" ref="mi_root">
        <v-app-bar :height="app_title_bar_height.valueOf()" class="app_bar" color="primary" app flat>
            <v-app-bar-nav-icon @click.stop="() => { if (inited) { drawer = !drawer } }" />
            <v-toolbar-title>
                <div>
                    <span>
                        {{ i18n.global.t("MI_APP_TITLE") }}
                    </span>
                    <v-menu activator="parent">
                        <v-list>
                            <v-list-item :key="index" :value="index" v-for="page, index in [
                                { app_name: i18n.global.t('RYKV_APP_NAME'), page_name: 'rykv' },
                                { app_name: i18n.global.t('MI_APP_NAME'), page_name: 'mi' },
                                { app_name: i18n.global.t('KFTL_APP_NAME'), page_name: 'kftl' },
                                { app_name: i18n.global.t('PLAING_TIMEIS_APP_NAME'), page_name: 'plaing' },
                                { app_name: i18n.global.t('MKFL_APP_NAME'), page_name: 'mkfl' },
                                { app_name: i18n.global.t('SAIHATE_APP_NAME'), page_name: 'saihate' },
                            ]">
                                <v-list-item-title
                                    @click="async () => { await resetDialogHistory(); router.replace('/' + page.page_name + '?loaded=true') }">
                                    {{ page.app_name }}</v-list-item-title>
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
        <v-navigation-drawer v-model="drawer" app :height="app_content_height" :mobile="drawer_mode_is_mobile"
            :width="312" :touchless="!drawer_mode_is_mobile">
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
                }" @inited="() => { if (!received_init_request) { init() }; received_init_request = true }"
                @request_open_focus_board="(board_name: string) => open_or_focus_board(board_name)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                ref="query_editor_sidebar" />
        </v-navigation-drawer>
        <v-main class="main" :class="(drawer_mode_is_mobile) ? 'scroll_snap_container' : ''">
            <div class="overlay_target">
                <v-overlay v-model="is_loading" class="align-center justify-center" persistent contained>
                    <v-progress-circular indeterminate color="primary" />
                </v-overlay>
            </div>
            <table class="mi_view_table" v-show="inited">
                <tr>
                    <td valign="top" v-for="query, index in querys" :key="query.query_id"
                        :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                        <v-card dropzone="true" @dragenter.prevent.stop="(...args: any[]) => { }"
                            @drop.prevent.stop="(...args: any[]) => on_drop_board_task(args[0] as DragEvent, query)"
                            @dragover.prevent.stop="(...args: any[]) => on_dragover_board_task(args[0] as DragEvent, query)">
                            <v-card-title v-if="query.use_mi_board_name">{{ query.mi_board_name }}</v-card-title>
                            <v-card-title v-if="!query.use_mi_board_name">{{ i18n.global.t("MI_ALL_TITLE")
                            }}</v-card-title>
                            <KyouListView :kyou_height="56 + 35" :width="400" :draggable="true"
                                :list_height="kyou_list_view_height.valueOf() - 48"
                                :application_config="application_config" :gkill_api="gkill_api"
                                :matched_kyous="match_kyous_list[index]" :query="query" :last_added_tag="last_added_tag"
                                :is_focused_list="focused_column_index === index" :closable="querys.length !== 1"
                                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                                :is_readonly_mi_check="false" :show_checkbox="false" :show_footer="true"
                                :is_show_doc_image_toggle_button="false" :is_show_arrow_button="false"
                                :show_rep_name="false" :force_show_latest_kyou_info="true" :show_content_only="false"
                                :show_timeis_plaing_end_button="false" @scroll_list="(...scroll_top: any[]) => {
                                    match_kyous_list_top_list[index] = scroll_top[0] as number
                                    if (inited) {
                                        props.gkill_api.set_saved_mi_scroll_indexs(match_kyous_list_top_list)
                                    }
                                }" @clicked_list_view="() => {
                                    skip_search_this_tick = true
                                    focused_column_index = index
                                    focused_query = querys[index]
                                    focused_column_index = index
                                    if (is_show_kyou_count_calendar) {
                                        update_focused_kyous_list(index)
                                    }
                                    nextTick(() => skip_search_this_tick = false)
                                }" @clicked_kyou="(...kyou: any[]) => {
                                    focused_column_index = index
                                    skip_search_this_tick = true
                                    focused_query = querys[index]
                                    clicked_kyou_in_list_view(index, kyou[0] as Kyou)
                                }" @focused_kyou="(...kyou: any[]) => {
                                    focused_column_index = index
                                    skip_search_this_tick = true
                                    focused_query = querys[index]
                                    clicked_kyou_in_list_view(index, kyou[0] as Kyou)
                                }"
                                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                                @requested_reload_kyou="(...kyou: any[]) => reload_kyou(kyou[0] as Kyou)"
                                @requested_reload_list="() => reload_list(index)"
                                @requested_update_check_kyous="(...params: any[]) => update_check_kyous(params[0] as Array<Kyou>, params[1] as boolean)"
                                @requested_change_focus_kyou="(...is_focus_kyou: any[]) => {
                                    focused_column_index = index
                                    skip_search_this_tick = true
                                    const query = querys[index].clone()
                                    query.is_focus_kyou_in_list_view = is_focus_kyou[0] as boolean
                                    querys.splice(index, 1, query)
                                    querys_backup.splice(index, 1, query)
                                }" @requested_search="() => {
                                    focused_column_index = index
                                    skip_search_this_tick = true
                                    const query = querys[index].clone()
                                    query.query_id = gkill_api.generate_uuid()
                                    querys[index] = query
                                    querys.splice(index, 1, query)
                                    querys_backup.splice(index, 1, query)
                                    reload_list(index)
                                }" ref="kyou_list_views" @requested_change_is_image_only_view="(...is_image_only_view: any[]) => {
                                    focused_column_index = index
                                    skip_search_this_tick = true
                                    focused_kyous_list = match_kyous_list[index]
                                    const query = querys[index].clone()
                                    query.query_id = gkill_api.generate_uuid()
                                    query.is_image_only = is_image_only_view[0] as boolean
                                    querys[index] = query
                                    querys.splice(index, 1, query)
                                    querys_backup.splice(index, 1, query)
                                    reload_list(index)
                                }" @requested_close_column="close_list_view(index)"
                                @deleted_kyou="(...deleted_kyou: any[]) => onDeletedKyou(deleted_kyou[0] as Kyou)"
                                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                                @registered_kyou="(...registered_kyou: any[]) => { emits('registered_kyou', registered_kyou[0] as Kyou) }"
                                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                                @updated_kyou="(...updated_kyou: any[]) => { reload_kyou(updated_kyou[0]); emits('updated_kyou', updated_kyou[0] as Kyou) }"
                                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                                @requested_open_rykv_dialog="(...params: any[]) => open_rykv_dialog(params[0], params[1], params[2])" />
                        </v-card>
                    </td>
                    <td valign="top" v-if="is_show_kyou_detail_view"
                        :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                        <div class="kyou_detail_view dummy">
                            <KyouView v-if="focused_kyou && is_show_kyou_detail_view"
                                :is_image_request_to_thumb_size="false" :application_config="application_config"
                                :gkill_api="gkill_api" :highlight_targets="[]" :is_image_view="false"
                                :kyou="focused_kyou" :last_added_tag="last_added_tag" :show_checkbox="false"
                                :show_content_only="false" :show_mi_create_time="true" :show_mi_estimate_end_time="true"
                                :show_mi_estimate_start_time="true" :show_mi_limit_time="true"
                                :show_timeis_elapsed_time="true" :show_timeis_plaing_end_button="true" :height="'unset'"
                                :is_readonly_mi_check="false" :width="'unset'"
                                :enable_context_menu="enable_context_menu" :show_update_time="false"
                                :show_related_time="true" :show_rep_name="true" :force_show_latest_kyou_info="true"
                                :enable_dialog="enable_dialog" :show_attached_timeis="true" class="kyou_detail_view"
                                :show_attached_tags="false" :show_attached_texts="false"
                                :show_attached_notifications="false"
                                @deleted_kyou="(...deleted_kyou: any[]) => onDeletedKyou(deleted_kyou[0] as Kyou)"
                                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                                @registered_kyou="(...registered_kyou: any[]) => { emits('registered_kyou', registered_kyou[0] as Kyou) }"
                                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                                @updated_kyou="(...updated_kyou: any[]) => { reload_kyou(updated_kyou[0]); emits('updated_kyou', updated_kyou[0] as Kyou) }"
                                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                                @requested_reload_kyou="(...kyou: any[]) => reload_kyou(kyou[0] as Kyou)"
                                @requested_reload_list="() => { }"
                                @requested_update_check_kyous="(...params: any[]) => update_check_kyous(params[0] as Array<Kyou>, params[1] as boolean)"
                                @requested_open_rykv_dialog="(...params: any[]) => open_rykv_dialog(params[0], params[1], params[2])" />
                        </div>
                    </td>
                    <td valign="top" :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                        <KyouCountCalendar v-show="is_show_kyou_count_calendar" :application_config="application_config"
                            :gkill_api="gkill_api" :kyous="focused_kyous_list" :for_mi="true"
                            @requested_focus_time="(...time: any[]) => { focused_time = time[0] as Date }" />
                    </td>
                </tr>
            </table>
            <AddKCDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :last_added_tag="''" :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => onDeletedKyou(deleted_kyou[0] as Kyou)"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => { emits('registered_kyou', registered_kyou[0] as Kyou) }"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => { reload_kyou(updated_kyou[0]); emits('updated_kyou', updated_kyou[0] as Kyou) }"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                ref="add_kc_dialog" />
            <AddTimeisDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :last_added_tag="last_added_tag" :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => onDeletedKyou(deleted_kyou[0] as Kyou)"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => { emits('registered_kyou', registered_kyou[0] as Kyou) }"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => { reload_kyou(updated_kyou[0]); emits('updated_kyou', updated_kyou[0] as Kyou) }"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @focused_kyou="(...kyou: any[]) => { focused_kyou = kyou[0] as Kyou }"
                @clicked_kyou="(...kyou: any[]) => { focused_kyou = kyou[0] as Kyou }"
                @requested_reload_kyou="(...kyou: any[]) => reload_kyou(kyou[0] as Kyou)"
                @requested_reload_list="() => { }"
                @requested_update_check_kyous="(...params: any[]) => update_check_kyous(params[0] as Array<Kyou>, params[1] as boolean)"
                ref="add_timeis_dialog" />
            <AddLantanaDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :last_added_tag="last_added_tag" :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => onDeletedKyou(deleted_kyou[0] as Kyou)"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => { emits('registered_kyou', registered_kyou[0] as Kyou) }"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => { reload_kyou(updated_kyou[0]); emits('updated_kyou', updated_kyou[0] as Kyou) }"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_reload_kyou="(...kyou: any[]) => reload_kyou(kyou[0] as Kyou)"
                @requested_reload_list="() => { }"
                @requested_update_check_kyous="(...params: any[]) => update_check_kyous(params[0] as Array<Kyou>, params[1] as boolean)"
                ref="add_lantana_dialog" />
            <AddUrlogDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :last_added_tag="last_added_tag" :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => onDeletedKyou(deleted_kyou[0] as Kyou)"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => { emits('registered_kyou', registered_kyou[0] as Kyou) }"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => { reload_kyou(updated_kyou[0]); emits('updated_kyou', updated_kyou[0] as Kyou) }"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_reload_kyou="(...kyou: any[]) => reload_kyou(kyou[0] as Kyou)"
                @requested_reload_list="() => { }"
                @requested_update_check_kyous="(...params: any[]) => update_check_kyous(params[0] as Array<Kyou>, params[1] as boolean)"
                ref="add_urlog_dialog" />
            <AddMiDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :last_added_tag="last_added_tag" :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => onDeletedKyou(deleted_kyou[0] as Kyou)"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => { emits('registered_kyou', registered_kyou[0] as Kyou) }"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => { reload_kyou(updated_kyou[0]); emits('updated_kyou', updated_kyou[0] as Kyou) }"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_reload_kyou="(...kyou: any[]) => reload_kyou(kyou[0] as Kyou)"
                @requested_reload_list="() => { }"
                @requested_update_check_kyous="(...params: any[]) => update_check_kyous(params[0] as Array<Kyou>, params[1] as boolean)"
                ref="add_mi_dialog" />
            <AddNlogDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :last_added_tag="last_added_tag" :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => onDeletedKyou(deleted_kyou[0] as Kyou)"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => { emits('registered_kyou', registered_kyou[0] as Kyou) }"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => { reload_kyou(updated_kyou[0]); emits('updated_kyou', updated_kyou[0] as Kyou) }"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_reload_kyou="(...kyou: any[]) => reload_kyou(kyou[0] as Kyou)"
                @requested_reload_list="() => { }"
                @requested_update_check_kyous="(...params: any[]) => update_check_kyous(params[0] as Array<Kyou>, params[1] as boolean)"
                ref="add_nlog_dialog" />
            <kftlDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :last_added_tag="last_added_tag" :kyou="new Kyou()" :app_content_height="app_content_height"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                :app_content_width="app_content_width"
                @deleted_kyou="(...deleted_kyou: any[]) => onDeletedKyou(deleted_kyou[0] as Kyou)"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => { emits('registered_kyou', registered_kyou[0] as Kyou) }"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => { reload_kyou(updated_kyou[0]); emits('updated_kyou', updated_kyou[0] as Kyou) }"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_reload_kyou="(...kyou: any[]) => reload_kyou(kyou[0] as Kyou)"
                @requested_reload_list="() => { }"
                @requested_update_check_kyous="(...params: any[]) => update_check_kyous(params[0] as Array<Kyou>, params[1] as boolean)"
                ref="kftl_dialog" />
            <mkflDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :last_added_tag="last_added_tag" :kyou="new Kyou()" :app_content_height="app_content_height"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                :app_content_width="app_content_width"
                @deleted_kyou="(...deleted_kyou: any[]) => onDeletedKyou(deleted_kyou[0] as Kyou)"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => { emits('registered_kyou', registered_kyou[0] as Kyou) }"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => { reload_kyou(updated_kyou[0]); emits('updated_kyou', updated_kyou[0] as Kyou) }"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_reload_kyou="(...kyou: any[]) => reload_kyou(kyou[0] as Kyou)"
                @requested_reload_list="() => { }"
                @requested_update_check_kyous="(...params: any[]) => update_check_kyous(params[0] as Array<Kyou>, params[1] as boolean)"
                ref="mkfl_dialog" />
            <UploadFileDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
                :application_config="application_config" :gkill_api="gkill_api" :last_added_tag="''"
                @deleted_kyou="(...deleted_kyou: any[]) => onDeletedKyou(deleted_kyou[0] as Kyou)"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => { emits('registered_kyou', registered_kyou[0] as Kyou) }"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => { reload_kyou(updated_kyou[0]); emits('updated_kyou', updated_kyou[0] as Kyou) }"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                ref="upload_file_dialog" />
            <RykvDialogHost :application_config="application_config" :gkill_api="gkill_api" :dialogs="opened_dialogs"
                :last_added_tag="last_added_tag" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @closed="(...id: any[]) => close_rykv_dialog(id[0] as string)"
                @deleted_kyou="(...deleted_kyou: any[]) => onDeletedKyou(deleted_kyou[0] as Kyou)"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => { emits('registered_kyou', registered_kyou[0] as Kyou) }"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => { reload_kyou(updated_kyou[0]); emits('updated_kyou', updated_kyou[0] as Kyou) }"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_reload_kyou="(...kyou: any[]) => reload_kyou(kyou[0] as Kyou)"
                @requested_reload_list="() => { }"
                @requested_update_check_kyous="(...params: any[]) => update_check_kyous(params[0] as Array<Kyou>, params[1] as boolean)"
                @requested_open_rykv_dialog="(...params: any[]) => open_rykv_dialog(params[0], params[1], params[2])" />
            <v-avatar :style="floatingActionButtonStyle()" color="primary" class="position-fixed">
                <v-menu :style="add_kyou_menu_style" transition="slide-x-transition">
                    <template v-slot:activator="{ props }">
                        <v-btn color="white" v-long-press="() => show_kftl_dialog()" icon="mdi-plus" variant="text"
                            v-bind="props" />
                    </template>
                    <v-list>
                        <v-list-item @click="show_kftl_dialog()">
                            <v-list-item-title>{{ i18n.global.t("KFTL_APP_NAME") }}</v-list-item-title>
                        </v-list-item>
                        <v-list-item @click="show_mkfl_dialog()">
                            <v-list-item-title>{{ i18n.global.t("MKFL_APP_NAME") }}</v-list-item-title>
                        </v-list-item>
                        <v-list-item @click="show_add_kc_dialog()">
                            <v-list-item-title>{{ i18n.global.t("KC_APP_NAME") }}</v-list-item-title>
                        </v-list-item>
                        <v-list-item @click="show_urlog_dialog()">
                            <v-list-item-title>{{ i18n.global.t("URLOG_APP_NAME") }}</v-list-item-title>
                        </v-list-item>
                        <v-list-item @click="show_timeis_dialog()">
                            <v-list-item-title>{{ i18n.global.t("TIMEIS_APP_NAME") }}</v-list-item-title>
                        </v-list-item>
                        <v-list-item @click="show_mi_dialog()">
                            <v-list-item-title>{{ i18n.global.t("MI_APP_NAME") }}</v-list-item-title>
                        </v-list-item>
                        <v-list-item @click="show_nlog_dialog()">
                            <v-list-item-title>{{ i18n.global.t("NLOG_APP_NAME") }}</v-list-item-title>
                        </v-list-item>
                        <v-list-item @click="show_lantana_dialog()">
                            <v-list-item-title>{{ i18n.global.t("LANTANA_APP_NAME") }}</v-list-item-title>
                        </v-list-item>
                        <v-list-item @click="show_upload_file_dialog()">
                            <v-list-item-title>{{ i18n.global.t("UPLOAD_APP_NAME") }}</v-list-item-title>
                        </v-list-item>
                    </v-list>
                </v-menu>
            </v-avatar>
        </v-main>
    </div>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import router from '@/router'
import MiQueryEditorSidebar from './mi-query-editor-sidebar.vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { computed, nextTick, type Ref, ref, watch } from 'vue'
import { Kyou } from '@/classes/datas/kyou'
import AddKCDialog from '../dialogs/add-kc-dialog.vue'
import AddMiDialog from '../dialogs/add-mi-dialog.vue'
import AddNlogDialog from '../dialogs/add-nlog-dialog.vue'
import KyouCountCalendar from './kyou-count-calendar.vue'
import KyouListView from './kyou-list-view.vue'
import KyouView from './kyou-view.vue'
import kftlDialog from '../dialogs/kftl-dialog.vue'
import mkflDialog from '../dialogs/mkfl-dialog.vue'
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
import { useScopedEnterForKFTL } from '@/classes/use-scoped-enter-for-kftl'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { Tag } from '@/classes/datas/tag'
import { Mi } from '@/classes/datas/mi'
import { UpdateMiRequest } from '@/classes/api/req_res/update-mi-request'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import { GetKyousResponse } from '@/classes/api/req_res/get-kyous-response'
import { resetDialogHistory } from '@/classes/use-dialog-history-stack'
import RykvDialogHost from './rykv-dialog-host.vue'
import type { OpenedRykvDialog, RykvDialogKind, RykvDialogPayload } from './rykv-dialog-kind'

const enable_context_menu = ref(true)
const enable_dialog = ref(true)

const query_editor_sidebar = ref<InstanceType<typeof MiQueryEditorSidebar> | null>(null);
const add_mi_dialog = ref<InstanceType<typeof AddMiDialog> | null>(null);
const add_nlog_dialog = ref<InstanceType<typeof AddNlogDialog> | null>(null);
const add_lantana_dialog = ref<InstanceType<typeof AddLantanaDialog> | null>(null);
const add_timeis_dialog = ref<InstanceType<typeof AddTimeisDialog> | null>(null);
const add_urlog_dialog = ref<InstanceType<typeof AddUrlogDialog> | null>(null);
const kftl_dialog = ref<InstanceType<typeof KftlDialog> | null>(null);
const add_kc_dialog = ref<InstanceType<typeof AddKCDialog> | null>(null);
const mkfl_dialog = ref<InstanceType<typeof mkflDialog> | null>(null);
const upload_file_dialog = ref<InstanceType<typeof UploadFileDialog> | null>(null);
const kyou_list_views = ref();
const mi_root = ref<HTMLElement | null>(null);
const opened_dialogs: Ref<Array<OpenedRykvDialog>> = ref([])

const querys: Ref<Array<FindKyouQuery>> = ref([new FindKyouQuery()])
const querys_backup: Ref<Array<FindKyouQuery>> = ref(new Array<FindKyouQuery>()) // 更新検知用バックアップ
const match_kyous_list: Ref<Array<Array<Kyou>>> = ref(new Array<Array<Kyou>>())
const match_kyous_list_top_list: Ref<Array<number>> = ref(new Array<number>())
const focused_query: Ref<FindKyouQuery> = ref(new FindKyouQuery())
const focused_column_index: Ref<number> = ref(0)
const focused_kyous_list: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const focused_kyou: Ref<Kyou | null> = ref(null)
const focused_time: Ref<Date> = ref(moment().toDate())
const is_show_kyou_detail_view: Ref<boolean> = ref(false)
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
    if (!match_kyous_list.value || match_kyous_list.value.length === 0) {
        return
    }
    focused_kyous_list.value = match_kyous_list.value[column_index]
}

function removeKyouFromListById(list: Array<Kyou>, deletedId: string): void {
    for (let i = list.length - 1; i >= 0; i--) {
        if (list[i].id === deletedId) {
            list.splice(i, 1)
        }
    }
}

function removeKyouFromMultiColumnLists(lists: Array<Array<Kyou>>, deletedId: string): void {
    for (let i = 0; i < lists.length; i++) {
        removeKyouFromListById(lists[i], deletedId)
    }
}

function onDeletedKyou(deletedKyou: Kyou): void {
    removeKyouFromMultiColumnLists(match_kyous_list.value, deletedKyou.id)
    removeKyouFromListById(focused_kyous_list.value, deletedKyou.id)
    if (focused_kyou.value?.id === deletedKyou.id) {
        focused_kyou.value = null
    }
    emits('deleted_kyou', deletedKyou)
}

async function reload_kyou(kyou: Kyou): Promise<void> {
    (async (): Promise<void> => {
        for (let i = 0; i < match_kyous_list.value.length; i++) {
            const kyous_list = match_kyous_list.value[i]
            for (let j = 0; j < kyous_list.length; j++) {
                const kyou_in_list = kyous_list[j]
                if (kyou.id === kyou_in_list.id) {
                    const updated_kyou = kyou.clone()
                    await updated_kyou.reload(false, true, querys.value[i])
                    await updated_kyou.load_all()
                    kyous_list.splice(j, 1, updated_kyou)
                }
            }
        }
    })();
    (async (): Promise<void> => {
        if (focused_kyou.value && focused_kyou.value.id === kyou.id) {
            const updated_kyou = kyou.clone()
            await updated_kyou.reload(false, true)
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
                drawer_mode_is_mobile.value = !(props.app_content_width.valueOf() >= 430)
                drawer.value = props.app_content_width.valueOf() >= 430
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

        querys.value.splice(column_index, 1)
        querys_backup.value.splice(column_index, 1)

        if (abort_controllers.value[column_index]) {
            abort_controllers.value[column_index].abort()
            abort_controllers.value[column_index] = new AbortController()
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
            abort_controllers.value[column_index] = new AbortController()
        }

        if (match_kyous_list.value[column_index]) {
            match_kyous_list.value[column_index] = []
        }

        nextTick(() => {
            const kyou_list_view = kyou_list_views.value.filter((kyou_list_view: any) => kyou_list_view.get_query_id() === query.query_id)[0] as any
            if (kyou_list_view) {
                if (inited.value) {
                    kyou_list_view.scroll_to(0)
                }
                ((async () => kyou_list_view.set_loading(true))());
            }
        })

        const waitPromises = new Array<Promise<any>>()

        const req = new GetKyousRequest()
        abort_controllers.value[column_index] = req.abort_controller
        req.query = query.clone()
        req.query.parse_words_and_not_words()
        if (update_cache) {
            waitPromises.push(delete_gkill_kyou_cache(null))
            req.query.update_cache = true
        } else {
            waitPromises.push(props.gkill_api.delete_updated_gkill_caches())
        }

        let res = new GetKyousResponse()
        waitPromises.push(props.gkill_api.get_kyous(req).then(response => res = response))

        await Promise.all(waitPromises)

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
            const kyou_list_view = kyou_list_views.value.filter((kyou_list_view: any) => kyou_list_view.get_query_id() === query.query_id)[0] as any
            if (kyou_list_view) {
                ((async () => kyou_list_view.set_loading(false))());
            }

            if (inited.value) {
                kyou_list_view.scroll_to(0)
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
}


function open_or_focus_board(board_name: string): void {
    if (board_name === "") {
        board_name = i18n.global.t("MI_ALL_TITLE")
    }

    let opened = false
    for (let i = 0; i < querys.value.length; i++) {
        const query = querys.value[i]
        if (query.mi_board_name === board_name) {
            focused_query.value = querys.value[i].clone()

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
    if (query.mi_board_name !== i18n.global.t("MI_ALL_TITLE")) {
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

function show_add_kc_dialog(): void {
    add_kc_dialog.value?.show()
}

function show_mkfl_dialog(): void {
    mkfl_dialog.value?.show()
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

const enable_enter_shortcut = ref(true)
useScopedEnterForKFTL(mi_root, show_kftl_dialog, enable_enter_shortcut);

function isTargetMiKyou(kyou: Kyou, miId: string): boolean {
    return kyou.typed_mi?.id === miId || kyou.id === miId
}

function findKyouInstancesByMiId(miId: string): Array<{ columnIndex: number, rowIndex: number, kyou: Kyou }> {
    const instances: Array<{ columnIndex: number, rowIndex: number, kyou: Kyou }> = []
    for (let columnIndex = 0; columnIndex < match_kyous_list.value.length; columnIndex++) {
        const column = match_kyous_list.value[columnIndex]
        for (let rowIndex = 0; rowIndex < column.length; rowIndex++) {
            const kyou = column[rowIndex]
            if (isTargetMiKyou(kyou, miId)) {
                instances.push({ columnIndex, rowIndex, kyou })
            }
        }
    }
    return instances
}

function removeKyouFromColumnById(columnIndex: number, kyouId: string): void {
    const column = match_kyous_list.value[columnIndex]
    if (!column) {
        return
    }
    for (let i = column.length - 1; i >= 0; i--) {
        if (column[i].id === kyouId) {
            column.splice(i, 1)
        }
    }
}

function insertKyouIntoColumnIfAbsent(columnIndex: number, kyou: Kyou): void {
    const column = match_kyous_list.value[columnIndex]
    if (!column) {
        return
    }
    for (let i = 0; i < column.length; i++) {
        if (column[i].id === kyou.id) {
            return
        }
    }
    column.push(kyou)
}

function patchKyouMiBoardName(kyou: Kyou, updatedMi: Mi): void {
    if (!kyou.typed_mi) {
        kyou.typed_mi = new Mi()
    }
    kyou.typed_mi.id = updatedMi.id
    kyou.typed_mi.board_name = updatedMi.board_name
    kyou.typed_mi.update_app = updatedMi.update_app
    kyou.typed_mi.update_device = updatedMi.update_device
    kyou.typed_mi.update_user = updatedMi.update_user
    kyou.typed_mi.update_time = updatedMi.update_time
}

function applyBoardMoveLocally(miId: string, beforeBoard: string, afterBoard: string, updatedMi: Mi): void {
    const instances = findKyouInstancesByMiId(miId)
    if (instances.length === 0) {
        return
    }

    // 既存インスタンスにボード更新を反映
    for (let i = 0; i < instances.length; i++) {
        patchKyouMiBoardName(instances[i].kyou, updatedMi)
    }
    const targetKyou = instances[0].kyou

    for (let i = 0; i < querys.value.length; i++) {
        const query = querys.value[i]
        if (query.use_mi_board_name && query.mi_board_name === beforeBoard) {
            removeKyouFromColumnById(i, targetKyou.id)
        }
        if (query.use_mi_board_name && query.mi_board_name === afterBoard) {
            insertKyouIntoColumnIfAbsent(i, targetKyou)
        }
    }

    if (focused_kyou.value && isTargetMiKyou(focused_kyou.value, miId)) {
        patchKyouMiBoardName(focused_kyou.value, updatedMi)
    }
    if (is_show_kyou_count_calendar.value) {
        update_focused_kyous_list(focused_column_index.value)
    }
}

async function on_drop_board_task(e: DragEvent, find_kyou_query: FindKyouQuery) {
    let mi: Mi
    try {
        const json_mi = JSON.parse(e.dataTransfer!.getData("gkill_mi"))
        const parsed_mi = new Mi()
        for (const key in json_mi) {
            (parsed_mi as any)[key] = (json_mi as any)[key]

            // 時刻はDate型に変換
            if (key.endsWith("time") && (parsed_mi as any)[key]) {
                (parsed_mi as any)[key] = new Date((parsed_mi as any)[key])
            }
        }
        mi = parsed_mi
    } catch (e: any) {
        console.error(e)
        return
    }

    if (!mi.id || mi.id == "") {
        return
    }

    e!.preventDefault()
    e!.stopPropagation()

    const before_board_name = mi.board_name
    const after_board_name = find_kyou_query.mi_board_name
    if (before_board_name === after_board_name || !find_kyou_query.use_mi_board_name) {
        return
    }

    mi.board_name = find_kyou_query.mi_board_name
    mi.update_app = "gkill"
    mi.update_device = props.application_config.device
    mi.update_time = new Date(Date.now())
    mi.update_user = props.application_config.user_id

    const req = new UpdateMiRequest()
    req.mi = mi
    req.want_response_kyou = true
    const res = await props.gkill_api.update_mi(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }

    const updatedMi = (res.updated_mi && res.updated_mi.id !== "") ? res.updated_mi : mi
    applyBoardMoveLocally(mi.id, before_board_name, after_board_name, updatedMi)

    if (res.updated_kyou) {
        emits('updated_kyou', res.updated_kyou)
    }
}
function on_dragover_board_task(e: DragEvent, _find_kyou_query: FindKyouQuery) {
    e!.dataTransfer!.dropEffect = "move"
    e!.preventDefault()
    e!.stopPropagation()
}

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
.kyou_detail_view.dummy {
    resize: horizontal;
    overflow-x: hidden;
    overflow-y: scroll;
    height: calc(v-bind('app_content_height.toString().concat("px")'));
    width: 408px;
}

.scroll_snap_container {
    overflow-x: auto;
    scroll-snap-type: x proximity;
    width: 100vw;
}

.scroll_snap_area {
    scroll-snap-align: start;
    width: 100vw;
}
</style>
<style lang="css" scoped>
.overlay_target {
    z-index: -10000;
    position: absolute;
    min-height: calc(v-bind('app_content_height.toString().concat("px")'));
    min-width: v-bind("is_loading ? 'calc(100vw)' : '0px'");
}
</style>
