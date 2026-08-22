
// 初期取得の完了を待たずに可視化する理由と却下案:
// documents/adr/0035-visualize-before-initial-search.md
import router from '@/router'
import { computed, onMounted, onUnmounted, ref, type Ref } from 'vue'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { GkillAPI } from '@/classes/api/gkill-api'
import { GetApplicationConfigRequest } from '@/classes/api/req_res/get-application-config-request'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { useTheme } from 'vuetify'
import { useRoute } from 'vue-router'
import { reset_dialog_history } from '@/classes/use-dialog-history-stack'
import { LogoutRequest } from '@/classes/api/req_res/logout-request'
import type { ComponentRef } from '@/classes/component-ref'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag'
import { useConfigStructSync } from '@/classes/use-config-struct-sync'

/**
 * ダッシュボードのページ側。
 *
 * rykv / mi と同じく「ページは薄いラッパ」。中身は use-dashboard-view.ts にある。
 * ここが持つのは ApplicationConfig の取得とテーマ、ウィンドウのリサイズ購読、
 * メッセージ表示、板ツリー/タグツリーの追随、ログアウトだけ。
 */
export function useDashboardPage() {
    const theme = useTheme()
    // useRoute() は setup の中で1回だけ呼ぶ。
    // load_application_config の中で呼ぶと、再試行ボタンから呼ばれたときには
    // コンポーネントインスタンスが無く inject が undefined を返して落ちる
    // （＝「永久スピナーにしない」導線がそこで壊れる）
    const route = useRoute()

    // ── Template refs ──
    const application_config_dialog = ref<ComponentRef | null>(null)
    const confirm_logout_dialog = ref<ComponentRef | null>(null)

    // ── State refs ──
    const actual_height: Ref<number> = ref(0)
    const element_height: Ref<number> = ref(0)
    const browser_url_bar_height: Ref<number> = ref(0)
    const app_title_bar_height: Ref<number> = ref(50)
    const gkill_api = computed(() => GkillAPI.get_instance())
    const application_config: Ref<ApplicationConfig> = ref(new ApplicationConfig())
    // 設定取得に失敗した。取得できないと is_loaded が立たず画面の初期化が走らないので、
    // オーバーレイをスピナーからエラー表示＋再試行へ差し替えるために使う
    const application_config_load_failed: Ref<boolean> = ref(false)
    const app_content_height: Ref<number> = ref(0)
    const app_content_width: Ref<number> = ref(0)

    const messages: Ref<Array<{ code: string, message: string, id: string, show_snackbar: boolean, closable: boolean, auto_close_duration_milli_seconds: number | null, is_error: boolean }>> = ref([])

    // ── 板ツリー/タグツリーの追随 ──
    const { check_tag_update, check_mi_board_update, resync_structs } = useConfigStructSync({
        application_config,
        gkill_api: () => gkill_api.value,
        write_errors: (errors) => write_errors(errors),
    })

    // ── Helpers ──
    const sleep = (time: number) => new Promise<void>((r) => setTimeout(r, time))

    function resize_content(): void {
        const inner_element = document.querySelector('#control-height')
        actual_height.value = window.innerHeight
        element_height.value = inner_element ? inner_element.clientHeight : actual_height.value
        browser_url_bar_height.value = Number(element_height.value) - Number(actual_height.value)
        app_content_height.value = Number(element_height.value) - (Number(browser_url_bar_height.value) + Number(app_title_bar_height.value))
        app_content_width.value = window.innerWidth
    }

    async function load_application_config(): Promise<void> {
        const req = new GetApplicationConfigRequest()
        const loaded_raw_value = route.query.loaded
        const loaded = loaded_raw_value && (loaded_raw_value == 'true')
        req.force_reget = !loaded // メニューから遷移したときにはApplicationConfig再取得はしない（キャッシュから取得する）
        application_config_load_failed.value = false
        return gkill_api.value.get_application_config(req)
            .then(res => {
                if (res.errors && res.errors.length != 0) {
                    write_errors(res.errors)
                    // 設定が来ないと画面の初期化(is_loadedのwatch)が一度も走らない。
                    // 黙って戻ると読み込み中オーバーレイのまま永久に固まるので、
                    // 失敗を画面へ伝えて再試行できるようにする
                    application_config_load_failed.value = true
                    return
                }
                const use_dark_theme = res.application_config.use_dark_theme
                if (use_dark_theme) {
                    theme.global.name.value = 'gkill_dark_theme'
                } else {
                    theme.global.name.value = 'gkill_theme'
                }
                gkill_api.value.set_use_dark_theme(use_dark_theme)
                application_config.value = res.application_config
                GkillAPI.get_instance().set_saved_application_config(res.application_config)
                if (res.messages && res.messages.length != 0) {
                    write_messages(res.messages)
                    return
                }
            })
            .catch((err: unknown) => {
                // 通信例外もここで受ける。catchが無いと呼び出し元がawaitしていないぶん
                // unhandled rejectionになり、やはり画面が固まったままになる
                console.error(err)
                application_config_load_failed.value = true
            })
    }

    function write_errors(errors_: Array<GkillError>): void {
        const received_errors = new Array<{ code: string, message: string, id: string, show_snackbar: boolean, closable: boolean, auto_close_duration_milli_seconds: number | null, is_error: boolean }>()
        for (let i = 0; i < errors_.length; i++) {
            if (errors_[i] && errors_[i].error_message) {
                received_errors.push({
                    code: errors_[i].error_code,
                    message: errors_[i].error_message,
                    id: GkillAPI.get_instance().generate_uuid(),
                    show_snackbar: true,
                    closable: errors_[i].show_keep,
                    auto_close_duration_milli_seconds: errors_[i].show_keep ? null : 2500,
                    is_error: true,
                })
            }
        }
        messages.value.push(...received_errors)
        for (let j = 0; j < received_errors.length; j++) {
            const auto_close_duration_milli_seconds = received_errors[j].auto_close_duration_milli_seconds
            if (auto_close_duration_milli_seconds) {
                sleep(auto_close_duration_milli_seconds).then(() => {
                    close_message(received_errors[j].id)
                })
            }
        }
    }

    function write_messages(messages_: Array<GkillMessage>): void {
        const received_messages = new Array<{ code: string, message: string, id: string, show_snackbar: boolean, closable: boolean, auto_close_duration_milli_seconds: number | null, is_error: boolean }>()
        for (let i = 0; i < messages_.length; i++) {
            if (messages_[i] && messages_[i].message) {
                received_messages.push({
                    code: messages_[i].message_code,
                    message: messages_[i].message,
                    id: GkillAPI.get_instance().generate_uuid(),
                    show_snackbar: true,
                    closable: messages_[i].show_keep,
                    auto_close_duration_milli_seconds: messages_[i].show_keep ? null : 2500,
                    is_error: false,
                })
            }
        }
        messages.value.push(...received_messages)
        for (let j = 0; j < received_messages.length; j++) {
            const auto_close_duration_milli_seconds = received_messages[j].auto_close_duration_milli_seconds
            if (auto_close_duration_milli_seconds) {
                sleep(auto_close_duration_milli_seconds).then(() => {
                    close_message(received_messages[j].id)
                })
            }
        }
    }

    function close_message(message_id: string): void {
        for (let i = 0; i < messages.value.length; i++) {
            if (messages.value[i].id === message_id) {
                messages.value.splice(i, 1)
                return
            }
        }
    }

    function show_application_config_dialog(): void {
        application_config_dialog.value?.show()
    }

    function show_confirm_logout_dialog(close_database: boolean): void {
        confirm_logout_dialog.value?.show(close_database)
    }

    // ── Logout ──
    async function logout(close_database: boolean): Promise<void> {
        const req = new LogoutRequest()
        req.close_database = close_database
        const res = await gkill_api.value.logout(req)
        if (res.errors && res.errors.length !== 0) {
            write_errors(res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            write_messages(res.messages)
        }
        await sleep(1500)
        await gkill_api.value.clear_browser_datas()
        await reset_dialog_history()
        router.replace("/")
    }

    // ── DashboardView event handlers ──
    // rykv / mi のページと同じく、ページ側の仕事は板ツリー/タグツリーの追随と
    // メッセージ表示と設定ダイアログだけ。一覧の更新はビューが自分で済ませている
    const dashboardViewHandlers = {
        'deleted_kyou': () => { /* no-op in page */ },
        'deleted_tag': () => { /* no-op in page */ },
        'deleted_text': () => { /* no-op in page */ },
        'deleted_notification': () => { /* no-op in page */ },
        'registered_kyou': (kyou: Kyou) => check_mi_board_update(kyou),
        'registered_tag': (tag: Tag) => check_tag_update(tag),
        'registered_text': () => { /* no-op in page */ },
        'registered_notification': () => { /* no-op in page */ },
        'updated_kyou': (kyou: Kyou) => check_mi_board_update(kyou),
        'updated_tag': (tag: Tag) => check_tag_update(tag),
        'updated_text': () => { /* no-op in page */ },
        'updated_notification': () => { /* no-op in page */ },
        'requested_show_application_config_dialog': () => show_application_config_dialog(),
        'received_errors': (errors: Array<GkillError>) => write_errors(errors),
        'received_messages': (received_messages: Array<GkillMessage>) => write_messages(received_messages),
        'requested_reload_application_config': () => load_application_config(),
        // KFTL/MKFL はタグを registered_tag で上げてこないので、保存完了で両方取り直す
        'saved_kyou_by_kftl': () => resync_structs(),
        // 単独ページではビューが自分で router.replace するので、ここへは上がってこない
        'requested_navigate_page': () => { /* ポートでのみ使う */ },
    }

    // ── Lifecycle ──
    onMounted(async () => {
        await reset_dialog_history()
    })

    const onResize = () => {
        resize_content()
    }
    window.addEventListener('resize', onResize)
    onUnmounted(() => {
        window.removeEventListener('resize', onResize)
    })

    // ── Init ──
    resize_content()
    load_application_config()

    return {
        // Template refs
        application_config_dialog,
        confirm_logout_dialog,

        // State
        actual_height,
        app_title_bar_height,
        gkill_api,
        application_config,
        application_config_load_failed,
        app_content_height,
        app_content_width,
        messages,

        // Methods
        write_errors,
        write_messages,
        close_message,
        load_application_config,
        show_application_config_dialog,
        show_confirm_logout_dialog,
        logout,

        // CRUD relay
        dashboardViewHandlers,
    }
}
