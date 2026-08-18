import { i18n, set_locale } from '@/i18n'
import { computed, nextTick, type Ref, ref, watch } from 'vue'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { GetMiBoardRequest } from '@/classes/api/req_res/get-mi-board-request'
import { UpdateApplicationConfigRequest } from '@/classes/api/req_res/update-application-config-request'
import { LogoutRequest } from '@/classes/api/req_res/logout-request'
import router from '@/router'
import { ReloadRepositoriesRequest } from '@/classes/api/req_res/reload-repositories-request'
import { useTheme } from 'vuetify'
import delete_gkill_kyou_cache, { delete_gkill_config_cache } from '@/classes/delete-gkill-cache'
import { GkillMessage } from '@/classes/api/gkill-message'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'
import { reset_dialog_history } from '@/classes/use-dialog-history-stack'
import type { ApplicationConfigViewProps } from '@/pages/views/application-config-view-props'
import type { ApplicationConfigViewEmits } from '@/pages/views/application-config-view-emits'
import type { GkillError } from '@/classes/api/gkill-error'
import type { DeviceStructElementData } from '@/classes/datas/config/device-struct-element-data'
import type { RepStructElementData } from '@/classes/datas/config/rep-struct-element-data'
import type { RepTypeStructElementData } from '@/classes/datas/config/rep-type-struct-element-data'
import type { KFTLTemplateElementData } from '@/classes/datas/kftl-template-element-data'
import type { TagStructElementData } from '@/classes/datas/config/tag-struct-element-data'
import type { MiBoardStructElementData } from '@/classes/datas/config/mi-board-struct-element-data'
import type { ComponentRef } from '@/classes/component-ref'
import { DashboardConfig } from '@/classes/datas/config/dashboard-config'
import { PlaingTimeIsConfig } from '@/classes/datas/config/plaing-time-is-config'
import { SavedFindQueryConfig } from '@/classes/datas/config/saved-find-query-config'
import { sort_mi_board_names_by_config_order } from '@/classes/mi-board-names'

export function useApplicationConfigView(options: {
    props: ApplicationConfigViewProps,
    emits: ApplicationConfigViewEmits,
}) {
    const { props, emits } = options

    const theme = useTheme()

    // ── Template refs ──
    const new_board_name_dialog = ref<ComponentRef | null>(null)
    const edit_device_struct_dialog = ref<ComponentRef | null>(null)
    const edit_rep_struct_dialog = ref<ComponentRef | null>(null)
    const edit_rep_type_struct_dialog = ref<ComponentRef | null>(null)
    const edit_tag_struct_dialog = ref<ComponentRef | null>(null)
    const edit_mi_board_struct_dialog = ref<ComponentRef | null>(null)
    const edit_kftl_template_dialog = ref<ComponentRef | null>(null)
    const edit_dnote_dialog = ref<ComponentRef | null>(null)
    const edit_ryuu_dialog = ref<ComponentRef | null>(null)
    const edit_dashboard_dialog = ref<ComponentRef | null>(null)
    const edit_plaing_time_is_dialog = ref<ComponentRef | null>(null)
    const edit_saved_find_query_dialog = ref<ComponentRef | null>(null)
    const server_config_dialog = ref<ComponentRef | null>(null)

    // ── State refs ──
    const pages = ref([
        { app_name: i18n.global.t('RYKV_APP_NAME'), page_name: 'rykv' },
        { app_name: i18n.global.t('MI_APP_NAME'), page_name: 'mi' },
        { app_name: i18n.global.t('KFTL_APP_NAME'), page_name: 'kftl' },
        { app_name: i18n.global.t('PLAING_TIMEIS_APP_NAME'), page_name: 'plaing' },
        { app_name: i18n.global.t('MKFL_APP_NAME'), page_name: 'mkfl' },
        { app_name: i18n.global.t('RUDBECKIA_APP_NAME'), page_name: 'rudbeckia' },
    ])

    const is_loading = ref(false)

    const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())

    const locale_name: Ref<'ja' | 'en' | 'zh' | 'ko' | 'es' | 'fr' | 'de'> = ref(i18n.global.locale)
    const google_map_api_key: Ref<string> = ref(cloned_application_config.value.google_map_api_key)
    const rykv_image_list_column_number: Ref<number> = ref(cloned_application_config.value.rykv_image_list_column_number)
    const rykv_hot_reload: Ref<boolean> = ref(cloned_application_config.value.rykv_hot_reload)
    const show_tags_in_list: Ref<boolean> = ref(cloned_application_config.value.show_tags_in_list)
    const mi_default_board: Ref<string> = ref(cloned_application_config.value.mi_default_board)
    // APIが返した生の板名一覧。表示順は mi_board_names（設定順に並べ替えたcomputed）を使う
    const mi_board_names_source: Ref<Array<string>> = ref([])
    // 板の並び順は ApplicationConfig（板構成の「上へ / 下へ」）が正。
    // get_mi_board_list はマップ反復順で返すので、素で渡すと呼ぶたびに並びが変わる。
    // 基準が props ではなく clone なのは、子の板構成ダイアログの「適用」が
    // clone にだけ並べ替え済みの struct を書くため（props を見ると並べ替え直後の
    // ここのプルダウンだけ古い順で取り残される）
    const mi_board_names = computed(() => sort_mi_board_names_by_config_order(mi_board_names_source.value, cloned_application_config.value.mi_board_struct))
    const rykv_default_period: Ref<number> = ref(cloned_application_config.value.rykv_default_period)
    const mi_default_period: Ref<number> = ref(cloned_application_config.value.mi_default_period)
    const is_checked_use_rykv_period: Ref<boolean> = ref(cloned_application_config.value.rykv_default_period !== -1)
    const is_checked_use_mi_period: Ref<boolean> = ref(cloned_application_config.value.mi_default_period !== -1)
    const use_dark_theme: Ref<boolean> = ref(theme.global.name.value === 'gkill_dark_theme')
    const is_show_share_footer: Ref<boolean> = ref(cloned_application_config.value.is_show_share_footer)
    const default_page: Ref<string> = ref(cloned_application_config.value.default_page)

    // ロケールとダークテーマは入力の都度プレビューする（見せないと選べない）。
    // 「適用」を押さずに閉じたときは開いた時点の状態へ戻すので、そのために控えておく
    const locale_name_on_open: Ref<'ja' | 'en' | 'zh' | 'ko' | 'es' | 'fr' | 'de'> = ref(locale_name.value)
    const use_dark_theme_on_open: Ref<boolean> = ref(use_dark_theme.value)

    // 表示の作り直し中は、チェックボックスの watcher に日数を上書きさせない
    let is_restoring_view_state = false

    // 子ダイアログの「適用」はサーバへ送らず clone に溜めるだけ。
    // 溜めている間に props が差し替わっても捨てない（捨てると未適用の編集が消える）
    let has_pending_child_edits = false

    // ── Watchers ──
    watch(() => props.application_config, async () => {
        if (has_pending_child_edits) {
            return
        }
        await reload_cloned_application_config()
    })

    watch(() => locale_name.value, async () => {
        let locale: 'ja' | 'en' | 'zh' | 'ko' | 'es' | 'fr' | 'de'
        switch (locale_name.value) {
            case 'ja':
            case 'en':
            case 'zh':
            case 'ko':
            case 'es':
            case 'fr':
            case 'de':
                locale = locale_name.value
                break
            default:
                locale = 'ja'
        }
        props.gkill_api.set_locale_name_to_cookie(locale)
        await set_locale(locale)
    })

    watch(() => is_checked_use_rykv_period.value, () => {
        // 保存済みの日数を復元しているだけのときは既定値で潰さない
        if (is_restoring_view_state) {
            return
        }
        if (is_checked_use_rykv_period.value) {
            rykv_default_period.value = 31
        } else {
            rykv_default_period.value = -1
        }
    })

    watch(() => is_checked_use_mi_period.value, () => {
        if (is_restoring_view_state) {
            return
        }
        if (is_checked_use_mi_period.value) {
            mi_default_period.value = 31
        } else {
            mi_default_period.value = -1
        }
    })

    watch(() => use_dark_theme.value, () => {
        if (use_dark_theme.value) {
            theme.global.name.value = 'gkill_dark_theme'
        } else {
            theme.global.name.value = 'gkill_theme'
        }
    })

    // ── Computed ──
    const urlog_bookmarklet = computed(() => {
        return (`
javascript: (function () {
	function genURLog() {
		let description = '';
		let image_url = '';

		if (new URL(location.href).host == "www.youtube.com") {
			let youtubeDescriptionTag = document.querySelector('#description > yt-formatted-string');
			if (youtubeDescriptionTag !== null) {
				description = youtubeDescriptionTag.textContent;
			}
		}
		if (description == '') {
			let descriptionTag = document.querySelector("meta[name='description']");
			if (descriptionTag !== null) {
				description = descriptionTag.getAttribute('content');
			} else {
				descriptionTag = document.querySelector("meta[property='og:description']");
				if (descriptionTag !== null) {
					description = descriptionTag.getAttribute('content');
				}
			}
		}

		if (new URL(location.href).host == "www.amazon.co.jp" || new URL(location.href).host == "www.amazon.com") {
			let amazonImageTag = document.querySelector('#landingImage');
			if (amazonImageTag !== null) {
				image_url = amazonImageTag.getAttribute('src');
			}
		}
		if (image_url == '') {
			let imageOGTag = document.querySelector('meta[property="og:image"]');
			if (imageOGTag !== null) {
				image_url = imageOGTag.getAttribute('content');
			}
		}

		return {
			url: location.href,
			title: document.title,
			time: new Date().toISOString(),
			favicon_url: 'https://www.google.com/s2/favicons?domain=' + new URL(location.href).host,
			description: description,
			image_url: image_url,
			session_id: '`+ props.application_config.urlog_bookmarklet_session + `',
		};
	};
	function sendURLog() {
		var p = genURLog();
		var q = Object.keys(p).map(function(k){ return encodeURIComponent(k)+'='+encodeURIComponent(p[k]); }).join('&');
		window.open('`  + location.protocol + "//" + location.host + props.gkill_api.urlog_bookmarklet_page_address + `?'+q, '_blank', 'width=420,height=160');
	};
	sendURLog();
}());`).split("\n").join("").split("\t").join("")
    })

    // ── Business logic ──
    const sleep = (time: number) => new Promise<void>((r) => setTimeout(r, time))

    async function reload_cloned_application_config(): Promise<void> {
        is_restoring_view_state = true
        has_pending_child_edits = false
        cloned_application_config.value = props.application_config.clone()
        google_map_api_key.value = cloned_application_config.value.google_map_api_key
        rykv_image_list_column_number.value = cloned_application_config.value.rykv_image_list_column_number
        rykv_hot_reload.value = cloned_application_config.value.rykv_hot_reload
        show_tags_in_list.value = cloned_application_config.value.show_tags_in_list
        mi_default_board.value = cloned_application_config.value.mi_default_board
        mi_board_names_source.value = []
        rykv_default_period.value = cloned_application_config.value.rykv_default_period
        mi_default_period.value = cloned_application_config.value.mi_default_period
        is_checked_use_rykv_period.value = cloned_application_config.value.rykv_default_period !== -1
        is_checked_use_mi_period.value = cloned_application_config.value.mi_default_period !== -1
        is_show_share_footer.value = cloned_application_config.value.is_show_share_footer
        default_page.value = cloned_application_config.value.default_page

        // ロケールとテーマは ApplicationConfig ではなく「いま効いているもの」が正。
        // 前回キャンセルで戻してあるので、実際に効いている値へ揃え直せばよい
        locale_name.value = i18n.global.locale
        use_dark_theme.value = theme.global.name.value === 'gkill_dark_theme'
        locale_name_on_open.value = locale_name.value
        use_dark_theme_on_open.value = use_dark_theme.value

        // watcher(pre flush)を消化してからフラグを下ろす
        await nextTick()
        is_restoring_view_state = false

        load_mi_board_names()
    }

    /**
     * 「適用」を押さずに閉じたときに、プレビューだけしていた変更を開く前の状態へ戻す。
     * ×・Escape・キャンセルのどれで閉じても呼ばれるよう、
     * application-config-dialog.vue の hide() から呼ぶ。
     */
    function cancel_pending_changes(): void {
        // 同じ値の代入では watcher が動かないので、戻す必要があるときだけ実際に走る
        use_dark_theme.value = use_dark_theme_on_open.value
        locale_name.value = locale_name_on_open.value
    }

    async function load_mi_board_names(): Promise<void> {
        const req = new GetMiBoardRequest()

        const res = await props.gkill_api.get_mi_board_list(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            // emits('received_messages', res.messages)
        }
        mi_board_names_source.value = res.boards
    }

    async function update_application_config(): Promise<void> {
        const application_config = new ApplicationConfig()
        application_config.google_map_api_key = google_map_api_key.value
        application_config.rykv_image_list_column_number = parseInt(rykv_image_list_column_number.value.toString())
        application_config.rykv_hot_reload = rykv_hot_reload.value
        application_config.mi_default_board = mi_default_board.value
        application_config.rykv_default_period = rykv_default_period.value
        application_config.show_tags_in_list = show_tags_in_list.value
        application_config.mi_default_period = mi_default_period.value
        application_config.use_dark_theme = use_dark_theme.value
        application_config.is_show_share_footer = is_show_share_footer.value
        application_config.default_page = default_page.value
        application_config.tag_struct = cloned_application_config.value.tag_struct
        application_config.rep_struct = cloned_application_config.value.rep_struct
        application_config.rep_type_struct = cloned_application_config.value.rep_type_struct
        application_config.device_struct = cloned_application_config.value.device_struct
        application_config.kftl_template_struct = cloned_application_config.value.kftl_template_struct
        application_config.ryuu_json_data = cloned_application_config.value.ryuu_json_data
        application_config.dnote_json_data = cloned_application_config.value.dnote_json_data
        application_config.dashboard_json_data = cloned_application_config.value.dashboard_json_data
        application_config.plaing_timeis_json_data = cloned_application_config.value.plaing_timeis_json_data
        application_config.saved_find_query_json_data = cloned_application_config.value.saved_find_query_json_data
        application_config.mi_board_struct = cloned_application_config.value.mi_board_struct
        // この画面で編集しない永続化フィールドも詰め直す。
        // 落とすとJSONから欠落し、サーバ側でゼロ値に巻き戻って保存される
        application_config.show_tutorial_on_startup = cloned_application_config.value.show_tutorial_on_startup

        const req = new UpdateApplicationConfigRequest()
        req.application_config = application_config

        const res = await props.gkill_api.update_application_config(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
        props.gkill_api.set_default_page_to_cookie(application_config.default_page)
        is_loading.value = true

        // 再読み込み
        const page_reload_message = new GkillMessage()
        page_reload_message.message = i18n.global.t("DO_RELOAD_TITLE")
        page_reload_message.message_code = GkillMessageCodes.do_reload
        emits('received_messages', [page_reload_message])
        await sleep(1500)

        location.reload()
    }

    async function logout(close_database: boolean): Promise<void> {
        const req = new LogoutRequest()
        req.close_database = close_database
        const res = await props.gkill_api.logout(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
        await sleep(1500)

        await props.gkill_api.clear_browser_datas()

        await reset_dialog_history()
        router.replace("/")
    }

    async function reload_repositories(clear_file_caches: boolean): Promise<void> {
        const requested_reload_message = new GkillMessage()
        requested_reload_message.message = i18n.global.t("REQUESTED_RELOAD_TITLE")
        requested_reload_message.message_code = GkillMessageCodes.requested_reload
        requested_reload_message.show_keep = true
        emits('received_messages', [requested_reload_message])

        is_loading.value = true
        const req = new ReloadRepositoriesRequest()
        req.clear_thumb_cache = clear_file_caches
        req.clear_video_cache = clear_file_caches
        req.clear_zip_cache = clear_file_caches
        const res = await props.gkill_api.reload_repositories(req)
        await delete_gkill_config_cache()
        await delete_gkill_kyou_cache(null)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
        is_loading.value = false

        const page_reload_message = new GkillMessage()
        page_reload_message.message = i18n.global.t("DO_RELOAD_TITLE")
        page_reload_message.message_code = GkillMessageCodes.do_reload
        emits('received_messages', [page_reload_message])
        await sleep(1500)

        location.reload()
    }

    // ── Dialog show methods ──
    function show_edit_device_dialog() {
        edit_device_struct_dialog.value?.show()
    }
    function show_edit_rep_dialog() {
        edit_rep_struct_dialog.value?.show()
    }
    function show_edit_tag_dialog() {
        edit_tag_struct_dialog.value?.show()
    }
    function show_edit_rep_type_dialog() {
        edit_rep_type_struct_dialog.value?.show()
    }
    function show_edit_mi_board_dialog() {
        edit_mi_board_struct_dialog.value?.show()
    }
    function show_edit_kftl_template_dialog() {
        edit_kftl_template_dialog.value?.show()
    }
    function show_edit_dnote_dialog() {
        edit_dnote_dialog.value?.show()
    }
    function show_edit_ryuu_dialog() {
        edit_ryuu_dialog.value?.show()
    }
    function show_edit_dashboard_dialog() {
        let dnote_query = undefined
        let mi_query = undefined
        if (cloned_application_config.value.dashboard_json_data) {
            const config = DashboardConfig.parse(cloned_application_config.value.dashboard_json_data)
            dnote_query = config.dashboard_dnote_find_kyou_query ?? undefined
            mi_query = config.dashboard_mi_find_kyou_query ?? undefined
        }
        edit_dashboard_dialog.value?.show(dnote_query, mi_query)
    }
    function show_edit_plaing_time_is_dialog() {
        let plaing_timeis_query = undefined
        if (cloned_application_config.value.plaing_timeis_json_data) {
            const config = PlaingTimeIsConfig.parse(cloned_application_config.value.plaing_timeis_json_data)
            plaing_timeis_query = config.plaing_timeis_find_kyou_query ?? undefined
        }
        edit_plaing_time_is_dialog.value?.show(plaing_timeis_query)
    }
    function show_edit_saved_find_query_dialog() {
        const config = SavedFindQueryConfig.parse(cloned_application_config.value.saved_find_query_json_data)
        edit_saved_find_query_dialog.value?.show(config)
    }
    function show_new_board_name_dialog(): void {
        new_board_name_dialog.value?.show()
    }
    function show_server_config_dialog(): void {
        server_config_dialog.value?.show()
    }

    // ── Event handlers ──
    function update_board_name(board_name: string): void {
        mi_board_names_source.value.push(board_name)
        mi_default_board.value = board_name
    }

    // 子ダイアログの「適用」はどれもサーバへ送らず、この画面の clone に組み立てるだけ。
    // 実際の送信は update_application_config()（この画面の「適用」）1箇所に閉じている
    function onRequestedApplyDeviceStruct(device_struct_element_data: DeviceStructElementData): void {
        has_pending_child_edits = true
        cloned_application_config.value.device_struct = device_struct_element_data
    }

    function onRequestedApplyKftlTemplateStruct(kftl_template_struct_element_data: KFTLTemplateElementData): void {
        has_pending_child_edits = true
        cloned_application_config.value.kftl_template_struct = kftl_template_struct_element_data
    }

    function onRequestedApplyRepStruct(rep_struct_element_data: RepStructElementData): void {
        has_pending_child_edits = true
        cloned_application_config.value.rep_struct = rep_struct_element_data
    }

    function onRequestedApplyRepTypeStruct(rep_type_struct_element_data: RepTypeStructElementData): void {
        has_pending_child_edits = true
        cloned_application_config.value.rep_type_struct = rep_type_struct_element_data
    }

    function onRequestedApplyTagStruct(tag_struct_element_data: TagStructElementData): void {
        has_pending_child_edits = true
        cloned_application_config.value.tag_struct = tag_struct_element_data
    }

    function onRequestedApplyMiBoardStruct(mi_board_struct_element_data: MiBoardStructElementData): void {
        has_pending_child_edits = true
        cloned_application_config.value.mi_board_struct = mi_board_struct_element_data
    }

    // 以下4つは struct 系と同じく clone にだけ書く。
    // props.application_config を直接書くと、この画面の「キャンセル」を押しても
    // 子ダイアログでの編集が残ってしまう（保存はされないが表示は変わったまま）。
    // props が差し替わっても編集が消えないようにするのは has_pending_child_edits の役目で、
    // props への書き戻しでやってはいけない
    function onRequestedApplyDnote(dnote_data: Record<string, unknown>): void {
        has_pending_child_edits = true
        cloned_application_config.value.dnote_json_data = dnote_data
    }

    function onRequestedApplyRyuuStruct(ryuu_data: Record<string, unknown>): void {
        has_pending_child_edits = true
        cloned_application_config.value.ryuu_json_data = ryuu_data
    }

    function onRequestedApplyDashboardStruct(dashboard_data: Record<string, unknown>): void {
        has_pending_child_edits = true
        cloned_application_config.value.dashboard_json_data = dashboard_data
    }

    function onRequestedApplyPlaingTimeIs(plaing_timeis_data: Record<string, unknown>): void {
        has_pending_child_edits = true
        cloned_application_config.value.plaing_timeis_json_data = plaing_timeis_data
    }

    function onRequestedApplySavedFindQueryStruct(saved_find_query_data: Record<string, unknown>): void {
        has_pending_child_edits = true
        cloned_application_config.value.saved_find_query_json_data = saved_find_query_data
    }

    // ── Event relay objects ──
    const errorMessageRelayHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
    }

    // ── Init ──
    load_mi_board_names()

    // ── Return ──
    return {
        // Template refs
        new_board_name_dialog,
        edit_device_struct_dialog,
        edit_rep_struct_dialog,
        edit_rep_type_struct_dialog,
        edit_tag_struct_dialog,
        edit_mi_board_struct_dialog,
        edit_kftl_template_dialog,
        edit_dnote_dialog,
        edit_ryuu_dialog,
        edit_dashboard_dialog,
        edit_plaing_time_is_dialog,
        edit_saved_find_query_dialog,
        server_config_dialog,

        // State
        is_loading,
        cloned_application_config,
        locale_name,
        google_map_api_key,
        rykv_image_list_column_number,
        rykv_hot_reload,
        show_tags_in_list,
        mi_default_board,
        mi_board_names,
        rykv_default_period,
        mi_default_period,
        is_checked_use_rykv_period,
        is_checked_use_mi_period,
        use_dark_theme,
        is_show_share_footer,
        default_page,
        pages,

        // Computed
        urlog_bookmarklet,

        // Business logic
        reload_cloned_application_config,
        cancel_pending_changes,
        update_application_config,
        logout,
        reload_repositories,

        // Dialog show methods
        show_edit_device_dialog,
        show_edit_rep_dialog,
        show_edit_tag_dialog,
        show_edit_rep_type_dialog,
        show_edit_mi_board_dialog,
        show_edit_kftl_template_dialog,
        show_edit_dnote_dialog,
        show_edit_ryuu_dialog,
        show_edit_dashboard_dialog,
        show_edit_plaing_time_is_dialog,
        show_edit_saved_find_query_dialog,
        show_new_board_name_dialog,
        show_server_config_dialog,

        // Event handlers
        update_board_name,
        onRequestedApplyDeviceStruct,
        onRequestedApplyKftlTemplateStruct,
        onRequestedApplyRepStruct,
        onRequestedApplyRepTypeStruct,
        onRequestedApplyTagStruct,
        onRequestedApplyMiBoardStruct,
        onRequestedApplyDnote,
        onRequestedApplyRyuuStruct,
        onRequestedApplyDashboardStruct,
        onRequestedApplyPlaingTimeIs,
        onRequestedApplySavedFindQueryStruct,

        // Event relay objects
        errorMessageRelayHandlers,
    }
}
