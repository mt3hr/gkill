<template>
    <div class="rykv_view_wrap" ref="rykv_root">
        <v-app-bar :height="app_title_bar_height.valueOf()" class="app_bar" color="primary" app flat>
            <v-app-bar-nav-icon v-if="!is_shared_rykv_view" @click.stop="() => { if (inited) { drawer = !drawer } }" />
            <v-toolbar-title>
                <div>
                    <span v-if="!is_shared_rykv_view">
                        {{ i18n.global.t("RYKV_APP_TITLE") }}
                    </span>
                    <span v-if="is_shared_rykv_view">
                        {{ share_title }}
                    </span>
                    <v-menu v-if="!is_shared_rykv_view" activator="parent">
                        <v-list>
                            <v-list-item :key="index" :value="index" v-for="page, index in [
                                { app_name: i18n.global.t('RYKV_APP_NAME'), page_name: 'rykv' },
                                { app_name: i18n.global.t('MI_APP_NAME'), page_name: 'mi' },
                                { app_name: i18n.global.t('KFTL_APP_NAME'), page_name: 'kftl' },
                                { app_name: i18n.global.t('PLAING_TIMEIS_APP_NAME'), page_name: 'plaing' },
                                { app_name: i18n.global.t('MKFL_APP_NAME'), page_name: 'mkfl' },
                                { app_name: i18n.global.t('SAIHATE_APP_NAME'), page_name: 'saihate' },
                            ]">
                                <v-list-item-title @click="router.replace('/' + page.page_name + '?loaded=true')">
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
            <v-btn v-if="!is_shared_rykv_view" icon
                @click="async () => { await dnote_view?.abort(); is_show_dnote = !is_show_dnote }">
                <v-icon>mdi-file-chart-outline</v-icon>
            </v-btn>
            <v-btn icon @click="is_show_kyou_count_calendar = !is_show_kyou_count_calendar">
                <v-icon>mdi-calendar</v-icon>
            </v-btn>
            <v-btn icon @click="is_show_gps_log_map = !is_show_gps_log_map">
                <v-icon>mdi-map</v-icon>
            </v-btn>
            <v-divider vertical />
            <v-btn v-if="!is_shared_rykv_view" icon="mdi-cog" :disabled="!application_config.is_loaded"
                @click="emits('requested_show_application_config_dialog')" />
        </v-app-bar>
        <v-navigation-drawer v-if="!is_shared_rykv_view" v-model="drawer" app :height="app_content_height"
            :mobile="drawer_mode_is_mobile" :width="312" :touchless="!drawer_mode_is_mobile">
            <RykvQueryEditorSideBar v-show="inited" class="rykv_query_editor_sidebar"
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
                    if (new_query.use_calendar && new_query.calendar_start_date && new_query.calendar_end_date) {
                        gps_log_map_start_time = new_query.calendar_start_date
                        gps_log_map_end_time = new_query.calendar_end_date
                    }
                }" @inited="() => { if (!received_init_request) { init() }; received_init_request = true }"
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
            <table class="rykv_view_table" v-show="inited">
                <tr>
                    <td valign="top" v-for="query, index in querys" :key="query.query_id"
                        :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                        <KyouListView :kyou_height="180" :width="400" :list_height="kyou_list_view_height"
                            :application_config="application_config" :gkill_api="gkill_api"
                            :matched_kyous="match_kyous_list[index]" :query="query" :last_added_tag="last_added_tag"
                            :is_focused_list="focused_column_index === index" :closable="querys.length !== 1"
                            :enable_context_menu="!is_shared_rykv_view" :enable_dialog="!is_shared_rykv_view"
                            :is_readonly_mi_check="false" :show_checkbox="true" :show_footer="!is_shared_rykv_view"
                            :is_show_doc_image_toggle_button="true" :is_show_arrow_button="true"
                            :show_rep_name="!is_shared_rykv_view" :force_show_latest_kyou_info="true"
                            :show_content_only="false" :show_timeis_plaing_end_button="false" @scroll_list="(...scroll_top: any[]) => {
                                match_kyous_list_top_list[index] = scroll_top[0] as number
                                if (inited) {
                                    props.gkill_api.set_saved_rykv_scroll_indexs(match_kyous_list_top_list)
                                }
                            }" @clicked_list_view="() => {
                                if (is_shared_rykv_view) {
                                    return
                                }
                                skip_search_this_tick = true
                                focused_query = querys[index]
                                if (is_show_kyou_count_calendar || is_show_dnote) {
                                    update_focused_kyous_list(index)
                                }
                                focused_column_index = index
                                nextTick(() => skip_search_this_tick = false)
                            }" @clicked_kyou="(...kyou: any[]) => {
                                const typed_kyou = kyou[0] as Kyou
                                skip_search_this_tick = true
                                focused_column_index = index
                                focused_query = querys[index]
                                clicked_kyou_in_list_view(index, typed_kyou)
                                gps_log_map_start_time = typed_kyou.related_time
                                gps_log_map_end_time = typed_kyou.related_time
                                gps_log_map_marker_time = typed_kyou.related_time
                            }"
                            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                            @requested_reload_kyou="(...kyou: any[]) => reload_kyou(kyou[0] as Kyou)"
                            @requested_reload_list="() => {
                                const query = querys[index].clone()
                                query.query_id = gkill_api.generate_uuid()
                                querys[index] = query
                                reload_list(index)
                            }" @requested_change_focus_kyou="(...is_focus_kyou: any[]) => {
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
                                const query = querys[index].clone()
                                query.query_id = gkill_api.generate_uuid()
                                query.is_image_only = is_image_only_view[0] as boolean
                                querys[index] = query
                                querys.splice(index, 1, query)
                                querys_backup.splice(index, 1, query)
                                search(index, query, true)
                            }" @requested_close_column="close_list_view(index)"
                            @deleted_kyou="(...deleted_kyou: any[]) => { reload_list(index); reload_kyou(deleted_kyou[0] as Kyou); focused_kyou?.reload(true, false); emits('deleted_kyou', deleted_kyou[0] as Kyou) }"
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
                            @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)" />
                    </td>
                    <td valign="top" v-if="!is_shared_rykv_view"
                        :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                        <v-btn class="rykv_add_column_button rounded-sm mx-auto" :height="app_content_height.valueOf()"
                            :width="30" :color="'primary'" @click="async () => {
                                add_list_view()
                                skip_search_this_tick = true
                                if (application_config.rykv_hot_reload) {
                                    search(querys.length - 1, querys[querys.length - 1], true)
                                }
                            }" icon="mdi-plus" variant="text"
                            :style="{ background: 'rgb(var(--v-theme-background))' }" />
                    </td>
                    <td valign="top" v-if="is_show_kyou_detail_view"
                        :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                        <div class="kyou_detail_view dummy">
                            <KyouView v-if="focused_kyou && is_show_kyou_detail_view" :is_image_request_to_thumb_size="false"
                                :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                                :is_image_view="false" :kyou="focused_kyou" :last_added_tag="last_added_tag"
                                :show_checkbox="false" :show_content_only="false" :show_mi_create_time="true"
                                :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                                :show_mi_limit_time="true" :show_timeis_elapsed_time="true"
                                :show_timeis_plaing_end_button="!is_shared_rykv_view" :height="'auto'"
                                :is_readonly_mi_check="is_shared_rykv_view" :width="'auto'"
                                :enable_context_menu="!is_shared_rykv_view" :enable_dialog="!is_shared_rykv_view"
                                :show_attached_timeis="true" :show_rep_name="true" :force_show_latest_kyou_info="true"
                                class="kyou_detail_view" :show_update_time="false" :show_related_time="true"
                                :show_attached_tags="true" :show_attached_texts="true"
                                :show_attached_notifications="true"
                                @deleted_kyou="(...deleted_kyou: any[]) => { reload_kyou(deleted_kyou[0] as Kyou); focused_kyou?.reload(true, false); emits('deleted_kyou', deleted_kyou[0] as Kyou) }"
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
                                @requested_reload_kyou="(...kyou: any[]) => reload_kyou(kyou[0] as Kyou)"
                                @requested_reload_list="() => { }" />
                        </div>
                        <div class="ryuu_view dummy">
                            <RyuuListView v-if="focused_kyou && default_query" :application_config="application_config"
                                :gkill_api="gkill_api" :target_kyou="focused_kyou" :editable="false"
                                :find_kyou_query_default="default_query"
                                @deleted_kyou="(...deleted_kyou: any[]) => { reload_kyou(deleted_kyou[0] as Kyou); focused_kyou?.reload(true, false); emits('deleted_kyou', deleted_kyou[0] as Kyou) }"
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
                                @requested_reload_kyou="(...kyou: any[]) => reload_kyou(kyou[0] as Kyou)"
                                @requested_reload_list="() => { }" />
                        </div>
                    </td>
                    <td valign="top" v-if="is_show_dnote && !is_shared_rykv_view"
                        :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                        <Dnote class="rykv_dnote_wrap" :app_content_height="app_content_height"
                            :app_content_width="app_content_width" :application_config="application_config"
                            :gkill_api="gkill_api" :query="focused_query" :checked_kyous="focused_column_checked_kyous"
                            :last_added_tag="last_added_tag" :editable="false"
                            @deleted_kyou="(...deleted_kyou: any[]) => { reload_kyou(deleted_kyou[0] as Kyou); focused_kyou?.reload(true, false); emits('deleted_kyou', deleted_kyou[0] as Kyou) }"
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
                            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                            @requested_reload_kyou="(...kyou: any[]) => reload_kyou(kyou[0] as Kyou)"
                            @requested_reload_list="() => { }" ref="dnote_view" />
                    </td>
                    <td valign="top" :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                        <KyouCountCalendar v-show="is_show_kyou_count_calendar" :application_config="application_config"
                            :gkill_api="gkill_api" :kyous="focused_kyous_list" :for_mi="false"
                            @requested_focus_time="(...time: any[]) => { focused_time = time[0]; gps_log_map_start_time = time[0]; gps_log_map_end_time = time[0]; gps_log_map_marker_time = time[0] }" />
                    </td>
                    <td valign="top" :class="(drawer_mode_is_mobile) ? 'scroll_snap_area' : ''">
                        <GPSLogMap v-show="is_show_gps_log_map" :application_config="application_config"
                            :gkill_api="gkill_api" :start_date="gps_log_map_start_time" :end_date="gps_log_map_end_time"
                            :marker_time="gps_log_map_marker_time"
                            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                            @requested_focus_time="(...time: any[]) => { focused_time = time[0] as Date }" />
                    </td>
                </tr>
            </table>
            <AddKCDialog v-if="!is_shared_rykv_view" :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="[]" :last_added_tag="''" :kyou="new Kyou()"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => { reload_kyou(deleted_kyou[0] as Kyou); focused_kyou?.reload(true, false); emits('deleted_kyou', deleted_kyou[0] as Kyou) }"
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
            <AddTimeisDialog v-if="!is_shared_rykv_view" :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="[]" :last_added_tag="last_added_tag" :kyou="new Kyou()"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => { reload_kyou(deleted_kyou[0] as Kyou); focused_kyou?.reload(true, false); emits('deleted_kyou', deleted_kyou[0] as Kyou) }"
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
                @requested_reload_list="() => { }" ref="add_timeis_dialog" />
            <AddLantanaDialog v-if="!is_shared_rykv_view" :application_config="application_config"
                :gkill_api="gkill_api" :highlight_targets="[]" :last_added_tag="last_added_tag" :kyou="new Kyou()"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => { reload_kyou(deleted_kyou[0] as Kyou); focused_kyou?.reload(true, false); emits('deleted_kyou', deleted_kyou[0] as Kyou) }"
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
                @requested_reload_list="() => { }" ref="add_lantana_dialog" />
            <AddUrlogDialog v-if="!is_shared_rykv_view" :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="[]" :last_added_tag="last_added_tag" :kyou="new Kyou()"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => { reload_kyou(deleted_kyou[0] as Kyou); focused_kyou?.reload(true, false); emits('deleted_kyou', deleted_kyou[0] as Kyou) }"
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
                @requested_reload_list="() => { }" ref="add_urlog_dialog" />
            <AddMiDialog v-if="!is_shared_rykv_view" :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="[]" :last_added_tag="last_added_tag" :kyou="new Kyou()"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => { reload_kyou(deleted_kyou[0] as Kyou); focused_kyou?.reload(true, false); emits('deleted_kyou', deleted_kyou[0] as Kyou) }"
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
                @requested_reload_list="() => { }" ref="add_mi_dialog" />
            <AddNlogDialog v-if="!is_shared_rykv_view" :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="[]" :last_added_tag="last_added_tag" :kyou="new Kyou()"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @deleted_kyou="(...deleted_kyou: any[]) => { reload_kyou(deleted_kyou[0] as Kyou); focused_kyou?.reload(true, false); emits('deleted_kyou', deleted_kyou[0] as Kyou) }"
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
                @requested_reload_list="() => { }" ref="add_nlog_dialog" />
            <kftlDialog v-if="!is_shared_rykv_view" :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="[]" :last_added_tag="last_added_tag" :kyou="new Kyou()"
                :app_content_height="app_content_height" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" :app_content_width="app_content_width"
                @deleted_kyou="(...deleted_kyou: any[]) => { reload_kyou(deleted_kyou[0] as Kyou); focused_kyou?.reload(true, false); emits('deleted_kyou', deleted_kyou[0] as Kyou) }"
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
                @requested_reload_list="() => { }" ref="kftl_dialog" />
            <mkflDialog v-if="!is_shared_rykv_view" :application_config="application_config" :gkill_api="gkill_api"
                :highlight_targets="[]" :last_added_tag="last_added_tag" :kyou="new Kyou()"
                :app_content_height="app_content_height" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" :app_content_width="app_content_width"
                @deleted_kyou="(...deleted_kyou: any[]) => { reload_kyou(deleted_kyou[0] as Kyou); focused_kyou?.reload(true, false); emits('deleted_kyou', deleted_kyou[0] as Kyou) }"
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
                @requested_reload_list="() => { }" ref="mkfl_dialog" />
            <UploadFileDialog v-if="!is_shared_rykv_view" :app_content_height="app_content_height"
                :app_content_width="app_content_width" :application_config="application_config" :gkill_api="gkill_api"
                :last_added_tag="''"
                @deleted_kyou="(...deleted_kyou: any[]) => { reload_kyou(deleted_kyou[0] as Kyou); focused_kyou?.reload(true, false); emits('deleted_kyou', deleted_kyou[0] as Kyou) }"
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
            <v-avatar v-if="!is_shared_rykv_view" :style="floatingActionButtonStyle()" color="primary"
                class="position-fixed">
                <v-menu :style="add_kyou_menu_style" transition="slide-x-transition">
                    <template v-slot:activator="{ props }">
                        <v-btn color="white" icon="mdi-plus" v-long-press="() => show_kftl_dialog()" variant="text"
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
import { computed, nextTick, type Ref, ref, watch } from 'vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { Kyou } from '@/classes/datas/kyou'
import AddKCDialog from '../dialogs/add-kc-dialog.vue'
import AddMiDialog from '../dialogs/add-mi-dialog.vue'
import AddNlogDialog from '../dialogs/add-nlog-dialog.vue'
import Dnote from './dnote-view.vue'
import GPSLogMap from './gps-log-map.vue'
import KyouCountCalendar from './kyou-count-calendar.vue'
import KyouListView from './kyou-list-view.vue'
import KyouView from './kyou-view.vue'
import RykvQueryEditorSideBar from './rykv-query-editor-side-bar.vue'
import kftlDialog from '../dialogs/kftl-dialog.vue'
import mkflDialog from '../dialogs/mkfl-dialog.vue'
import type { rykvViewEmits } from './rykv-view-emits'
import type { rykvViewProps } from './rykv-view-props'
import { GetKyousRequest } from '@/classes/api/req_res/get-kyous-request'
import type KftlDialog from '../dialogs/kftl-dialog.vue'
import AddLantanaDialog from '../dialogs/add-lantana-dialog.vue'
import AddTimeisDialog from '../dialogs/add-timeis-dialog.vue'
import AddUrlogDialog from '../dialogs/add-urlog-dialog.vue'
import UploadFileDialog from '../dialogs/upload-file-dialog.vue'
import moment from 'moment'
import { deepEquals } from '@/classes/deep-equals'
import RyuuListView from './ryuu-list-view.vue'
import { useScopedEnterForKFTL } from '@/classes/use-scoped-enter-for-kftl'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { Tag } from '@/classes/datas/tag'

const enable_context_menu = ref(true)
const enable_dialog = ref(true)
useDialogHistoryStack(enable_dialog)

const query_editor_sidebar = ref<InstanceType<typeof RykvQueryEditorSideBar> | null>(null);
const add_mi_dialog = ref<InstanceType<typeof AddMiDialog> | null>(null);
const add_nlog_dialog = ref<InstanceType<typeof AddNlogDialog> | null>(null);
const add_lantana_dialog = ref<InstanceType<typeof AddLantanaDialog> | null>(null);
const add_timeis_dialog = ref<InstanceType<typeof AddTimeisDialog> | null>(null);
const add_urlog_dialog = ref<InstanceType<typeof AddUrlogDialog> | null>(null);
const kftl_dialog = ref<InstanceType<typeof KftlDialog> | null>(null);
const add_kc_dialog = ref<InstanceType<typeof AddKCDialog> | null>(null);
const mkfl_dialog = ref<InstanceType<typeof mkflDialog> | null>(null);
const upload_file_dialog = ref<InstanceType<typeof UploadFileDialog> | null>(null);
const dnote_view = ref<InstanceType<typeof Dnote> | null>(null);
const kyou_list_views = ref();
const rykv_root = ref<HTMLElement | null>(null);

const querys: Ref<Array<FindKyouQuery>> = ref([new FindKyouQuery()])
const querys_backup: Ref<Array<FindKyouQuery>> = ref(new Array<FindKyouQuery>()) // 更新検知用バックアップ
const match_kyous_list: Ref<Array<Array<Kyou>>> = ref(new Array<Array<Kyou>>())
const match_kyous_list_top_list: Ref<Array<number>> = ref(new Array<number>())
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
const is_show_dnote: Ref<boolean> = ref(false)
const last_added_tag: Ref<string> = ref("")
const drawer: Ref<boolean | null> = ref(false)
const drawer_mode_is_mobile: Ref<boolean | null> = ref(false)
const kyou_list_view_height = computed(() => props.app_content_height)
const default_query: Ref<FindKyouQuery> = ref(new FindKyouQuery())

const position_x: Ref<Number> = ref(0)
const position_y: Ref<Number> = ref(0)

const props = defineProps<rykvViewProps>()
const emits = defineEmits<rykvViewEmits>()

const skip_search_this_tick = ref(false)


watch(() => focused_time.value, () => {
    if (!kyou_list_views.value) {
        return
    }
    const kyou_list_view = kyou_list_views.value[focused_column_index.value] as any
    if (!kyou_list_view) {
        return
    }
    if (inited.value) {
        kyou_list_view.scroll_to_time(focused_time.value)
    }
})

watch(() => is_show_kyou_count_calendar.value, () => {
    if (props.is_shared_rykv_view) {
        return
    }
    if (is_show_kyou_count_calendar.value) {
        update_focused_kyous_list(focused_column_index.value)
    }
})

watch(() => is_show_dnote.value, async () => {
    if (props.is_shared_rykv_view) {
        return
    }
    dnote_view.value?.abort()
    if (is_show_dnote.value) {
        update_focused_kyous_list(focused_column_index.value)

        const kyou_list_view = kyou_list_views.value[focused_column_index.value] as any
        if (!kyou_list_view) {
            return
        }
        while (kyou_list_view.get_is_loading()) {
            await sleep(500)
        }
        nextTick(() => {
            dnote_view.value?.reload(focused_kyous_list.value, focused_query.value)
        })
    } else {
        dnote_view.value?.abort()
    }
})

if (props.is_shared_rykv_view) {
    nextTick(async () => {
        is_loading.value = false
        inited.value = true
        await props.gkill_api.delete_updated_gkill_caches()
        const kyous = (await props.gkill_api.get_kyous(new GetKyousRequest())).kyous
        const wait_promises = new Array<Promise<any>>()
        for (let i = 0; i < kyous.length; i++) {
            wait_promises.push(kyous[i].load_all())
        }
        await Promise.all(wait_promises)
        match_kyous_list.value = [kyous]
        focused_kyous_list.value = kyous
        focused_column_index.value = 0
    })
}

const is_loading: Ref<boolean> = ref(true)

const inited = ref(false)
const received_init_request = ref(false)
async function init(): Promise<void> {
    if (inited.value) {
        return
    }
    return nextTick(async () => {
        const waitPromises = new Array<Promise<void>>()
        try {
            // スクロール位置の復元
            match_kyous_list_top_list.value = props.gkill_api.get_saved_rykv_scroll_indexs()

            // 前回開いていた列があれば復元する
            skip_search_this_tick.value = true
            const saved_querys = props.gkill_api.get_saved_rykv_find_kyou_querys()
            default_query.value = query_editor_sidebar.value!.get_default_query()!.clone()
            default_query.value.query_id = props.gkill_api.generate_uuid()
            if (saved_querys.length.valueOf() === 0) {
                const cloned_default_query = default_query.value.clone()
                cloned_default_query.query_id = props.gkill_api.generate_uuid()
                saved_querys.push(cloned_default_query)
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
                if (querys.value[focused_column_index.value].use_calendar && querys.value[focused_column_index.value].calendar_start_date && querys.value[focused_column_index.value].calendar_end_date) {
                    gps_log_map_start_time.value = querys.value[focused_column_index.value].calendar_start_date!
                    gps_log_map_end_time.value = querys.value[focused_column_index.value].calendar_end_date!
                }

                inited.value = true
                drawer_mode_is_mobile.value = !(props.app_content_width.valueOf() >= 430)
                drawer.value = props.app_content_width.valueOf() >= 430
                is_loading.value = false
                skip_search_this_tick.value = false
            })
            nextTick(() => default_query.value = query_editor_sidebar.value!.get_default_query()!.clone())
        }
    })
}

function update_focused_kyous_list(column_index: number): void {
    if (props.is_shared_rykv_view) {
        return
    }
    if (!match_kyous_list.value || match_kyous_list.value.length === 0) {
        return
    }
    focused_kyous_list.value = match_kyous_list.value[column_index]
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
        props.gkill_api.set_saved_rykv_find_kyou_querys(querys.value)
        props.gkill_api.set_saved_rykv_scroll_indexs(match_kyous_list_top_list.value)
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
    props.gkill_api.set_saved_rykv_find_kyou_querys(querys.value)
    props.gkill_api.set_saved_rykv_scroll_indexs(match_kyous_list_top_list.value)
}

async function reload_kyou(kyou: Kyou): Promise<void> {
    (async (): Promise<void> => {
        for (let i = 0; i < match_kyous_list.value.length; i++) {
            const kyous_list = match_kyous_list.value[i]
            for (let j = 0; j < kyous_list.length; j++) {
                const kyou_in_list = kyous_list[j]
                if (kyou.id === kyou_in_list.id) {
                    const updated_kyou = kyou.clone()
                    await updated_kyou.reload(false, true)
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

async function reload_list(column_index: number): Promise<void> {
    return search(column_index, querys.value[column_index], true)
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
        if (inited.value && column_index !== target_column_index) {
            kyou_list_views.value[target_column_index].scroll_to_time(kyou.related_time)
        }
    }
}

const abort_controllers: Ref<Array<AbortController>> = ref([])
async function search(column_index: number, query: FindKyouQuery, force_search?: boolean, update_cache?: boolean): Promise<void> {
    const query_id = query.query_id
    await dnote_view.value?.abort()
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

        props.gkill_api.set_saved_rykv_find_kyou_querys(querys.value)

        focused_column_checked_kyous.value = []

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

        const req = new GetKyousRequest()
        abort_controllers.value[column_index] = req.abort_controller
        req.query = query.clone()
        req.query.parse_words_and_not_words()
        if (update_cache) {
            req.query.update_cache = true
        }
        await props.gkill_api.delete_updated_gkill_caches()
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
        if (!props.is_shared_rykv_view) {
            if (is_show_kyou_count_calendar.value || is_show_dnote.value) {
                update_focused_kyous_list(column_index)
            }
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
            dnote_view.value?.reload(focused_kyous_list.value, focused_query.value)
        })
    } catch (err: any) {
        // abortは握りつぶす
        if (!(err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request"))) {
            // abort以外はエラー出力する
            console.error(err)
        }
    }
}
function floatingActionButtonStyle() {
    return {
        'bottom': '60px',
        'right': '10px',
        'height': '50px',
        'width': '50px'
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
useScopedEnterForKFTL(rykv_root, show_kftl_dialog, enable_enter_shortcut);

const sleep = (time: number) => new Promise<void>((r) => setTimeout(r, time))
</script>
<style lang="css" scoped>
.kyou_detail_view.dummy {
    resize: horizontal;
    overflow-x: hidden;
    overflow-y: scroll;
    height: calc(v-bind('app_content_height.toString().concat("px")') - v-bind('(!props.application_config.ryuu_json_data || props.application_config.ryuu_json_data.length === 0) ? "0px" : "100vh * 0.2"'));
    width: 400px;
    min-width: 400px;
}

.ryuu_view.dummy {
    overflow-x: hidden;
    overflow-y: auto;
    height: calc(100vh * 0.2);
    height: calc(v-bind('(!props.application_config.ryuu_json_data || props.application_config.ryuu_json_data.length === 0) ? "0px" : "100vh * 0.2"'));
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
<style lang="css">
.rykv_view_table {
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

.kyou_dialog .kyou_image {
    width: unset !important;
    height: unset !important;
    max-width: 85vw !important;
    max-height: 85vh !important;
}

.kyou_dialog .kyou_video {
    width: unset !important;
    height: unset !important;
    max-width: 85vw !important;
    max-height: 85vh !important;
}

.rykv_view_wrap {
    position: relative;
}

.v-navigation-drawer {
    transition: transform 0.3s ease-out !important;
    will-change: transform;
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
<style lang="css">
.rykv_dnote_wrap {
    width: fit-content;
}
</style>