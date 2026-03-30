import { i18n } from '@/i18n'
import { computed, type Ref, ref, watch } from 'vue'
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
import { resetDialogHistory } from '@/classes/use-dialog-history-stack'
import type { ApplicationConfigViewProps } from '@/pages/views/application-config-view-props'
import type { ApplicationConfigViewEmits } from '@/pages/views/application-config-view-emits'
import type { GkillError } from '@/classes/api/gkill-error'
import type { DeviceStructElementData } from '@/classes/datas/config/device-struct-element-data'
import type { RepStructElementData } from '@/classes/datas/config/rep-struct-element-data'
import type { RepTypeStructElementData } from '@/classes/datas/config/rep-type-struct-element-data'
import type { KFTLTemplateElementData } from '@/classes/datas/kftl-template-element-data'
import type { TagStructElementData } from '@/classes/datas/config/tag-struct-element-data'
import type { ComponentRef } from '@/classes/component-ref'

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
    const edit_kftl_template_dialog = ref<ComponentRef | null>(null)
    const edit_dnote_dialog = ref<ComponentRef | null>(null)
    const edit_ryuu_dialog = ref<ComponentRef | null>(null)
    const server_config_dialog = ref<ComponentRef | null>(null)

    // ── State refs ──
    const pages = ref([
        { app_name: i18n.global.t('RYKV_APP_NAME'), page_name: 'rykv' },
        { app_name: i18n.global.t('MI_APP_NAME'), page_name: 'mi' },
        { app_name: i18n.global.t('KFTL_APP_NAME'), page_name: 'kftl' },
        { app_name: i18n.global.t('PLAING_TIMEIS_APP_NAME'), page_name: 'plaing' },
        { app_name: i18n.global.t('MKFL_APP_NAME'), page_name: 'mkfl' },
    ])

    const is_loading = ref(false)

    const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())

    const locale_name: Ref<'ja' | 'en' | 'zh' | 'ko' | 'es' | 'fr' | 'de'> = ref(i18n.global.locale)
    const google_map_api_key: Ref<string> = ref(cloned_application_config.value.google_map_api_key)
    const rykv_image_list_column_number: Ref<number> = ref(cloned_application_config.value.rykv_image_list_column_number)
    const rykv_hot_reload: Ref<boolean> = ref(cloned_application_config.value.rykv_hot_reload)
    const show_tags_in_list: Ref<boolean> = ref(cloned_application_config.value.show_tags_in_list)
    const mi_default_board: Ref<string> = ref(cloned_application_config.value.mi_default_board)
    const mi_board_names: Ref<Array<string>> = ref([])
    const rykv_default_period: Ref<number> = ref(cloned_application_config.value.rykv_default_period)
    const mi_default_period: Ref<number> = ref(cloned_application_config.value.mi_default_period)
    const is_checked_use_rykv_period: Ref<boolean> = ref(cloned_application_config.value.rykv_default_period !== -1)
    const is_checked_use_mi_period: Ref<boolean> = ref(cloned_application_config.value.mi_default_period !== -1)
    const use_dark_theme: Ref<boolean> = ref(theme.global.name.value === 'gkill_dark_theme')
    const is_show_share_footer: Ref<boolean> = ref(cloned_application_config.value.is_show_share_footer)
    const default_page: Ref<string> = ref(cloned_application_config.value.default_page)

    // ── Watchers ──
    watch(() => props.application_config, async () => {
        cloned_application_config.value = props.application_config.clone()
    })

    watch(() => locale_name.value, () => {
        let locale: 'ja' | 'en' | 'zh' | 'ko' | 'es' | 'fr' | 'de' = 'ja'
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
        i18n.global.locale = locale
    })

    watch(() => is_checked_use_rykv_period.value, () => {
        if (is_checked_use_rykv_period.value) {
            rykv_default_period.value = 31
        } else {
            rykv_default_period.value = -1
        }
    })

    watch(() => is_checked_use_mi_period.value, () => {
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
		let urlog = JSON.stringify(genURLog());
		fetch('`  + location.protocol + "//" + location.host + props.gkill_api.urlog_bookmarklet_address + `', {
			method: '`+ props.gkill_api.urlog_bookmarklet_method + `',
            mode: 'no-cors',
			headers: { 'Content-Type': 'application/json' },
				body: urlog
			}
		)
	};
	addEventListener('onload', sendURLog());
}());`).split("\n").join("").split("\t").join("")
    })

    // ── Business logic ──
    const sleep = (time: number) => new Promise<void>((r) => setTimeout(r, time))

    async function reload_cloned_application_config(): Promise<void> {
        cloned_application_config.value = props.application_config.clone()
        google_map_api_key.value = cloned_application_config.value.google_map_api_key
        rykv_image_list_column_number.value = cloned_application_config.value.rykv_image_list_column_number
        rykv_hot_reload.value = cloned_application_config.value.rykv_hot_reload
        show_tags_in_list.value = cloned_application_config.value.show_tags_in_list
        mi_default_board.value = cloned_application_config.value.mi_default_board
        mi_board_names.value = []
        rykv_default_period.value = cloned_application_config.value.rykv_default_period
        mi_default_period.value = cloned_application_config.value.mi_default_period
        is_show_share_footer.value = cloned_application_config.value.is_show_share_footer
        default_page.value = cloned_application_config.value.default_page
        load_mi_board_names()
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
        mi_board_names.value = res.boards
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
        application_config.mi_board_struct = cloned_application_config.value.mi_board_struct

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

        props.gkill_api.set_session_id("")

        await resetDialogHistory()
        router.replace("/")
    }

    async function reload_repositories(clear_thumb_cache: boolean): Promise<void> {
        const requested_reload_message = new GkillMessage()
        requested_reload_message.message = i18n.global.t("REQUESTED_RELOAD_TITLE")
        requested_reload_message.message_code = GkillMessageCodes.requested_reload
        requested_reload_message.show_keep = true
        emits('received_messages', [requested_reload_message])

        is_loading.value = true
        const req = new ReloadRepositoriesRequest()
        req.clear_thumb_cache = clear_thumb_cache
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
    function show_edit_kftl_template_dialog() {
        edit_kftl_template_dialog.value?.show()
    }
    function show_edit_dnote_dialog() {
        edit_dnote_dialog.value?.show()
    }
    function show_edit_ryuu_dialog() {
        edit_ryuu_dialog.value?.show()
    }
    function show_new_board_name_dialog(): void {
        new_board_name_dialog.value?.show()
    }
    function show_server_config_dialog(): void {
        server_config_dialog.value?.show()
    }

    // ── Event handlers ──
    function update_board_name(board_name: string): void {
        mi_board_names.value.push(board_name)
        mi_default_board.value = board_name
    }

    function onRequestedApplyDeviceStruct(device_struct_element_data: DeviceStructElementData): void {
        cloned_application_config.value.device_struct = device_struct_element_data
    }

    function onRequestedApplyKftlTemplateStruct(kftl_template_struct_element_data: KFTLTemplateElementData): void {
        cloned_application_config.value.kftl_template_struct = kftl_template_struct_element_data
    }

    function onRequestedApplyRepStruct(rep_struct_element_data: RepStructElementData): void {
        cloned_application_config.value.rep_struct = rep_struct_element_data
    }

    function onRequestedApplyRepTypeStruct(rep_type_struct_element_data: RepTypeStructElementData): void {
        cloned_application_config.value.rep_type_struct = rep_type_struct_element_data
    }

    function onRequestedApplyTagStruct(tag_struct_element_data: TagStructElementData): void {
        cloned_application_config.value.tag_struct = tag_struct_element_data
    }

    function onRequestedApplyDnote(dnote_data: Record<string, unknown>): void {
        cloned_application_config.value.dnote_json_data = dnote_data
    }

    function onRequestedApplyRyuuStruct(ryuu_data: Record<string, unknown>): void {
        cloned_application_config.value.ryuu_json_data = ryuu_data
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
        edit_kftl_template_dialog,
        edit_dnote_dialog,
        edit_ryuu_dialog,
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
        update_application_config,
        logout,
        reload_repositories,

        // Dialog show methods
        show_edit_device_dialog,
        show_edit_rep_dialog,
        show_edit_tag_dialog,
        show_edit_rep_type_dialog,
        show_edit_kftl_template_dialog,
        show_edit_dnote_dialog,
        show_edit_ryuu_dialog,
        show_new_board_name_dialog,
        show_server_config_dialog,

        // Event handlers
        update_board_name,
        onRequestedApplyDeviceStruct,
        onRequestedApplyKftlTemplateStruct,
        onRequestedApplyRepStruct,
        onRequestedApplyRepTypeStruct,
        onRequestedApplyTagStruct,
        onRequestedApplyDnote,
        onRequestedApplyRyuuStruct,

        // Event relay objects
        errorMessageRelayHandlers,
    }
}
