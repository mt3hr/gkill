import router from '@/router'
import { computed, onMounted, onUnmounted, ref, type Ref } from 'vue'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { GkillAPI } from '@/classes/api/gkill-api'
import { GetApplicationConfigRequest } from '@/classes/api/req_res/get-application-config-request'
import type { GkillError } from '@/classes/api/gkill-error'
import { GkillMessage } from '@/classes/api/gkill-message'
import { useTheme } from 'vuetify'
import { useRoute } from 'vue-router'
import { reset_dialog_history } from '@/classes/use-dialog-history-stack'
import type { ComponentRef } from '@/classes/component-ref'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag'
import { useConfigStructSync } from '@/classes/use-config-struct-sync'
import { useScopedEnterForKFTL } from '@/classes/use-scoped-enter-for-kftl'
import { useScopedCtrlVForClipboard } from '@/classes/use-scoped-ctrl-v-for-clipboard'
import { build_kyou_dialog_host_handlers } from '@/classes/kyou-view-relay'
import { gkill_page_list, rudbeckia_page_list } from '@/classes/gkill-page-list'
import type { RudbeckiaPageKind } from '@/pages/views/rudbeckia-page-kind'
import { create_kyou_change_bus, type KyouChange } from '@/classes/kyou-change-bus'
import { new_reload_batch } from '@/classes/kyou-reload'

/**
 * ポート自身（FABの追加系ダイアログ）が出す通知の発生元。
 * どのウィンドウの id とも一致しないので、全ウィンドウが受け取る
 */
const RUDBECKIA_PAGE_ORIGIN_ID = 'rudbeckia-page'

/**
 * ポート（開発コード rudbeckia）。
 *
 * さいはてと同じ「背景とFABだけ」の器に、画面ウィンドウのホストを1つ足したもの。
 * ApplicationConfig の取得・テーマ・メッセージ表示・板/タグツリーの追随は
 * **ここが1つだけ持つ**（ホストした各画面のビューは設定を取りに行かない）。
 */
export function useRudbeckiaPage() {
    const theme = useTheme()
    // useRoute() は setup の中で1回だけ呼ぶ。関数の中で呼ぶと、
    // 再試行ボタンのようにインスタンス外から呼ばれたとき inject が undefined を返して落ちる
    const route = useRoute()

    // ── Template refs ──
    const rudbeckia_root = ref<HTMLElement | null>(null)
    const page_dialog_host = ref<{ show?: (kind: RudbeckiaPageKind) => void } | null>(null)
    const application_config_dialog = ref<ComponentRef | null>(null)
    const add_mi_dialog = ref<ComponentRef | null>(null)
    const add_nlog_dialog = ref<ComponentRef | null>(null)
    const add_lantana_dialog = ref<ComponentRef | null>(null)
    const add_timeis_dialog = ref<ComponentRef | null>(null)
    const add_urlog_dialog = ref<ComponentRef | null>(null)
    const kftl_dialog = ref<ComponentRef | null>(null)
    const add_kc_dialog = ref<ComponentRef | null>(null)
    const mkfl_dialog = ref<ComponentRef | null>(null)
    const upload_file_dialog = ref<ComponentRef | null>(null)
    const save_clipboard_to_file_dialog = ref<ComponentRef | null>(null)

    // ── State refs ──
    const enable_context_menu = ref(true)
    const enable_dialog = ref(false)
    const actual_height: Ref<number> = ref(0)
    const element_height: Ref<number> = ref(0)
    const browser_url_bar_height: Ref<number> = ref(0)
    const app_title_bar_height: Ref<number> = ref(50)
    const gkill_api = computed(() => GkillAPI.get_instance())
    const application_config: Ref<ApplicationConfig> = ref(new ApplicationConfig())
    // 設定取得に失敗した。取得できないと is_loaded が立たず、ホストした画面の初期化が
    // 一度も走らない。オーバーレイをスピナーからエラー表示＋再試行へ差し替えるために使う
    const application_config_load_failed: Ref<boolean> = ref(false)
    const app_content_height: Ref<number> = ref(0)
    const app_content_width: Ref<number> = ref(0)
    const is_loading: Ref<boolean> = ref(true)

    const messages: Ref<Array<{ code: string, message: string, id: string, show_snackbar: boolean, closable: boolean, auto_close_duration_milli_seconds: number | null, is_error: boolean }>> = ref([])

    // ── 板ツリー/タグツリーの追随 ──
    const { check_tag_update, check_mi_board_update, resync_structs } = useConfigStructSync({
        application_config,
        gkill_api: () => gkill_api.value,
        write_errors: (errors) => write_errors(errors),
    })

    // ── 画面間の変更通知 ──
    // ポートが1つだけ持ち、開いている全ウィンドウへ配る。
    // 各ウィンドウは自分の id を origin_id にして publish し、自分の通知は受けない
    const kyou_change_bus = create_kyou_change_bus()


    function publish_kyou_change(change: KyouChange): void {
        kyou_change_bus.publish(RUDBECKIA_PAGE_ORIGIN_ID, change, new_reload_batch())
    }

    // ── Computed ──
    const page_list = gkill_page_list
    /** FABの「画面」グループ。ポートの中でウィンドウとして開ける4つだけ */
    const screen_list = rudbeckia_page_list

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
        req.force_reget = !loaded // メニューから遷移したときにはApplicationConfig再取得はしない
        application_config_load_failed.value = false
        return gkill_api.value.get_application_config(req)
            .then(res => {
                if (res.errors && res.errors.length !== 0) {
                    write_errors(res.errors)
                    // 設定が来ないとホストした画面の初期化が一度も走らない。
                    // 黙って戻ると読み込み中オーバーレイのまま永久に固まる
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
                is_loading.value = false
                if (res.messages && res.messages.length !== 0) {
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

    // ── 画面ウィンドウ ──
    /** FABの「画面」グループと、ホストした画面の切替メニューの両方から呼ばれる */
    function open_page_dialog(kind: RudbeckiaPageKind): void {
        page_dialog_host.value?.show?.(kind)
    }

    /** ポートがウィンドウとして開けない画面（メモ帳・打刻メモ帳・さいはて・ポート自身）はページ遷移する */
    async function navigate_to_page(page_name: string): Promise<void> {
        await reset_dialog_history()
        router.replace('/' + page_name + '?loaded=true')
    }

    // ── Floating button ──
    function floating_action_button_style() {
        return {
            'bottom': '60px',
            'right': '10px',
            'height': '50px',
            'width': '50px',
        }
    }

    // ── Dialog show methods ──
    function show_application_config_dialog(): void {
        application_config_dialog.value?.show()
    }

    function show_kftl_dialog(): void {
        kftl_dialog.value?.show()
    }

    function show_mkfl_dialog(): void {
        mkfl_dialog.value?.show()
    }

    function show_add_kc_dialog(): void {
        add_kc_dialog.value?.show()
    }

    function show_urlog_dialog(): void {
        add_urlog_dialog.value?.show()
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

    function show_upload_file_dialog(): void {
        upload_file_dialog.value?.show()
    }

    function show_save_clipboard_to_file_dialog(): void {
        save_clipboard_to_file_dialog.value?.show()
    }

    // ── Event relay objects ──
    // 追加系ダイアログと画面ウィンドウの両方へ同じ束を渡す。
    // ポートは一覧を持たないので、画面の更新に関わるものはホストした各ビューが自分で済ませる。
    // ここの仕事は板ツリー/タグツリーの追随とメッセージ表示だけ（さいはてと同じ）
    const rudbeckiaKyouHandlers = build_kyou_dialog_host_handlers({
        'closed': () => { /* ポート自身は RykvDialogHost を持たない */ },
        // ポートのFABから追加/更新/削除した記録も、開いている画面へ配る。
        // 配らないと「＋から足したのに並べている一覧に出ない」になる。
        // 発生元は RUDBECKIA_PAGE_ORIGIN_ID なので、どのウィンドウも自分の通知とは見なさない
        'deleted_kyou': (kyou: Kyou) => publish_kyou_change({ kind: 'deleted', kyou: kyou }),
        'requested_reload_kyou': (kyou: Kyou) => publish_kyou_change({ kind: 'reload', kyou: kyou }),
        'requested_open_rykv_dialog': () => { /* 記録の編集ダイアログはホストした画面が開く */ },
        'updated_kyou': (kyou: Kyou) => {
            check_mi_board_update(kyou)
            publish_kyou_change({ kind: 'reload', kyou: kyou })
        },
    }, {
        'received_errors': (errors: Array<GkillError>) => write_errors(errors),
        'received_messages': (received_messages: Array<GkillMessage>) => write_messages(received_messages),
        'requested_reload_list': () => publish_kyou_change({ kind: 'reload_list' }),
        'registered_kyou': (kyou: Kyou) => {
            check_mi_board_update(kyou)
            publish_kyou_change({ kind: 'registered', kyou: kyou })
        },
        'registered_tag': (tag: Tag) => check_tag_update(tag),
        'updated_tag': (tag: Tag) => check_tag_update(tag),
    })

    // KFTL/MKFL はタグを registered_tag で上げてこないので、保存完了で両方取り直す
    function onSavedKyouByKftl(): void {
        resync_structs()
    }

    // ── Keyboard shortcuts ──
    // ホストした画面ビューは is_hosted_in_dialog なので登録しない。
    // ポートの中で効くショートカットはここの1組だけ
    const enable_enter_shortcut = ref(true)
    useScopedEnterForKFTL(rudbeckia_root, show_kftl_dialog, enable_enter_shortcut)
    useScopedCtrlVForClipboard(rudbeckia_root, show_save_clipboard_to_file_dialog, enable_enter_shortcut)

    // ── Lifecycle ──
    const onResize = () => {
        resize_content()
    }
    window.addEventListener('resize', onResize)
    onMounted(async () => {
        await reset_dialog_history()
    })
    onUnmounted(() => {
        window.removeEventListener('resize', onResize)
    })

    // ── Init ──
    resize_content()
    load_application_config()

    return {
        // Template refs
        rudbeckia_root,
        page_dialog_host,
        application_config_dialog,
        add_mi_dialog,
        add_nlog_dialog,
        add_lantana_dialog,
        add_timeis_dialog,
        add_urlog_dialog,
        kftl_dialog,
        add_kc_dialog,
        mkfl_dialog,
        upload_file_dialog,
        save_clipboard_to_file_dialog,

        // State
        enable_context_menu,
        enable_dialog,
        actual_height,
        app_title_bar_height,
        gkill_api,
        application_config,
        application_config_load_failed,
        app_content_height,
        app_content_width,
        is_loading,
        messages,

        // Computed
        kyou_change_bus,
        page_list,
        screen_list,

        // Methods
        write_errors,
        write_messages,
        close_message,
        load_application_config,
        open_page_dialog,
        navigate_to_page,
        floating_action_button_style,

        // Dialog show methods
        show_application_config_dialog,
        show_kftl_dialog,
        show_mkfl_dialog,
        show_add_kc_dialog,
        show_urlog_dialog,
        show_timeis_dialog,
        show_mi_dialog,
        show_nlog_dialog,
        show_lantana_dialog,
        show_upload_file_dialog,
        show_save_clipboard_to_file_dialog,


        // Event relay objects
        rudbeckiaKyouHandlers,
        onSavedKyouByKftl,
    }
}
