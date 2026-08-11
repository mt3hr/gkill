import { i18n } from '@/i18n'
import { ServerConfig } from '@/classes/datas/config/server-config'
import { computed, nextTick, ref, watch, type Ref } from 'vue'
import type { ServerConfigViewEmits } from '@/pages/views/server-config-view-emits'
import type { ServerConfigViewProps } from '@/pages/views/server-config-view-props'
import { GkillError } from '@/classes/api/gkill-error'
import { UpdateServerConfigsRequest } from '@/classes/api/req_res/update-server-configs-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import type { ComponentRef } from '@/classes/component-ref'

export function useServerConfigView(options: {
    props: ServerConfigViewProps,
    emits: ServerConfigViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const confirm_generate_tls_files_dialog = ref<ComponentRef | null>(null)
    const manage_account_dialog = ref<ComponentRef | null>(null)
    const new_device_name_dialog = ref<ComponentRef | null>(null)

    // ── State refs ──
    const is_loading = ref(false)
    const cloned_server_configs: Ref<Array<ServerConfig>> = ref(new Array<ServerConfig>())
    const server_config: Ref<ServerConfig> = ref(new ServerConfig())
    const device: Ref<string> = ref("")
    const devices: Ref<Array<string>> = ref(new Array<string>())

    /**
     * props の ServerConfig を要素ごと複製する。
     * `concat()` は配列だけの浅いコピーで、要素は props と同一参照になる。
     * 入力欄は `v-model="server_config.address"` のようにオブジェクトを直接書き換えるので、
     * 浅いコピーだと「適用」を押す前から props 側が変わってしまう
     */
    async function reload_cloned_server_configs(): Promise<void> {
        cloned_server_configs.value = await Promise.all(props.server_configs.map((config) => config.clone()))
    }

    /**
     * 有効な端末のプロファイルを選択中にする。
     * 見つからない場合はここでは何もしない。
     * 通知は load_current_server_config() の not_found_enable_device に任せる
     * （両方で出すとエラーが二重に飛ぶ）
     */
    function select_enable_device(): void {
        const enable_server_config = props.server_configs.find((server_config) => server_config.enable_this_device)
        if (!enable_server_config) {
            return
        }
        device.value = enable_server_config.device
    }

    /** props を編集用の状態へ取り込み直す。初期化とpropsの差し替えで同じ手順を通す */
    async function reload_from_props(): Promise<void> {
        await reload_cloned_server_configs()
        select_enable_device()
        load_devices()
        load_current_server_config()
    }

    /** 入力欄は秒で扱う。ServerConfig の保持はGoの time.Duration と同じナノ秒 */
    const urlog_timeout_sec = computed<number>({
        get: () => server_config.value.urlog_timeout / 1_000_000_000,
        set: (value) => {
            server_config.value.urlog_timeout = Math.round(Number(value) * 1_000_000_000)
        },
    })

    // ── Init ──
    nextTick(async () => {
        await reload_from_props()
    })

    // ── Watchers ──
    watch(() => props.server_configs, async () => {
        await reload_from_props()
    })

    watch(() => device.value, () => {
        update_enable_device()
        load_current_server_config()
    })

    // ── Business logic ──
    function update_enable_device(): void {
        for (let i = 0; i < cloned_server_configs.value.length; i++) {
            const server_config = cloned_server_configs.value[i]
            server_config.enable_this_device = server_config.device === device.value
        }
    }

    function add_device(device_name: string): void {
        for (let i = 0; i < cloned_server_configs.value.length; i++) {
            if (cloned_server_configs.value[i].device === device_name) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.device_is_already_exist
                error.error_message = i18n.global.t("ALEADY_EXISTS_PROFILE_MESSAGE")
                emits('received_errors', [error])
                return
            }
        }
        const new_server_config = new ServerConfig()
        new_server_config.is_local_only_access = server_config.value.is_local_only_access
        new_server_config.use_gkill_notification = server_config.value.use_gkill_notification
        new_server_config.address = server_config.value.address
        new_server_config.enable_tls = server_config.value.enable_tls
        new_server_config.tls_cert_file = server_config.value.tls_cert_file
        new_server_config.tls_key_file = server_config.value.tls_key_file
        new_server_config.open_directory_command = server_config.value.open_directory_command
        new_server_config.open_file_command = server_config.value.open_file_command
        new_server_config.urlog_timeout = server_config.value.urlog_timeout
        new_server_config.urlog_useragent = server_config.value.urlog_useragent
        new_server_config.upload_size_limit_month = server_config.value.upload_size_limit_month
        new_server_config.user_data_directory = server_config.value.user_data_directory
        new_server_config.google_map_api_key = server_config.value.google_map_api_key
        new_server_config.lan_hostname = server_config.value.lan_hostname
        new_server_config.global_hostname = server_config.value.global_hostname
        new_server_config.device = device_name
        cloned_server_configs.value.push(new_server_config)
        device.value = device_name
        load_devices()
    }

    function load_devices(): void {
        devices.value.splice(0)
        for (let i = 0; i < cloned_server_configs.value.length; i++) {
            devices.value.push(cloned_server_configs.value[i].device)
        }
    }

    async function load_current_server_config(): Promise<void> {
        // まだサーバ設定が届いていない。設定ダイアログは応答を待たずに開き、
        // このviewを v-show で隠すだけ（v-if ではない）なので初期化はここまで走る。
        // 0件を「有効なプロファイルが見つかりません」として通知すると、
        // ダイアログを開くたびに毎回そのエラーが出る。
        // 応答が届けば props の差し替えでウォッチャが動き、改めてここを通る
        if (cloned_server_configs.value.length === 0) {
            return
        }

        let current_server_config: ServerConfig | null = null
        for (let i = 0; i < cloned_server_configs.value.length; i++) {
            if (cloned_server_configs.value[i].enable_this_device) {
                current_server_config = cloned_server_configs.value[i]
                break
            }
        }
        if (!current_server_config) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_enable_device
            error.error_message = i18n.global.t("NOT_FOUND_ENABLE_DEVICE_MESSAGE")
            emits('received_errors', [error])
            return
        }
        server_config.value = current_server_config
    }

    // ── Template event handlers ──
    function show_manage_account_dialog(): void {
        manage_account_dialog.value?.show()
    }

    function show_confirm_generate_tls_files_dialog(): void {
        confirm_generate_tls_files_dialog.value?.show()
    }

    function show_new_device_name_dialog(): void {
        new_device_name_dialog.value?.show()
    }

    async function update_server_config(): Promise<void> {
        is_loading.value = true
        update_enable_device()
        const req = new UpdateServerConfigsRequest()
        req.server_configs = cloned_server_configs.value.concat()

        const res = await props.gkill_api.update_server_config(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            is_loading.value = false
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
        is_loading.value = false
        emits('requested_reload_server_config')
        emits('requested_close_dialog')
    }

    function delete_current_server_config(): void {
        // 最後の1件は消させない。サーバは「有効なプロファイルがちょうど1件」を前提に
        // 起動するので、0件にしてしまうと次の再起動で立ち上がらなくなる。
        // 削除ボタン自体も2件以上のときしか出していないので、ここは保険
        if (cloned_server_configs.value.length <= 1) {
            return
        }

        let delete_target_server_config_index = -1
        for (let i = 0; i < cloned_server_configs.value.length; i++) {
            if (cloned_server_configs.value[i].device === device.value) {
                delete_target_server_config_index = i
                break
            }
        }
        if (delete_target_server_config_index === -1) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_delete_target_device
            error.error_message = i18n.global.t("NOT_FOUND_DELETE_TARGET_DEVICE_MESSAGE")
            emits('received_errors', [error])
            return
        }
        cloned_server_configs.value.splice(delete_target_server_config_index, 1)
        server_config.value = cloned_server_configs.value[0]
        device.value = server_config.value.device
        load_devices()
    }

    function onRequestedCloseDialog(): void {
        emits('requested_close_dialog')
    }

    function onSettedNewDeviceName(device_name: string): void {
        add_device(device_name)
    }

    // ── Return ──
    return {
        // Template refs
        confirm_generate_tls_files_dialog,
        manage_account_dialog,
        new_device_name_dialog,

        // State
        is_loading,
        cloned_server_configs,
        server_config,
        device,
        devices,
        urlog_timeout_sec,

        // Template event handlers
        show_manage_account_dialog,
        show_confirm_generate_tls_files_dialog,
        show_new_device_name_dialog,
        update_server_config,
        delete_current_server_config,
        onRequestedCloseDialog,
        onSettedNewDeviceName,
    }
}
