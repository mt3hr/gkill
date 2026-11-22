import { onMounted, onUnmounted, ref, type Ref } from 'vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { GkillAPI } from '@/classes/api/gkill-api'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { reset_dialog_history } from '@/classes/use-dialog-history-stack'
import { useGkillMessageFeed } from '@/classes/use-gkill-message-feed'
import { apply_baked_locale } from '@/export-i18n'

/** HTML へ焼き込まれたデータの差し込み先の id（export.html と export_html_render.go で対） */
const EXPORT_DATA_ELEMENT_ID = 'gkill-export-data'
/** 焼き込まれたロケールのメッセージの差し込み先の id（ja で出力したときは置かれない） */
const EXPORT_I18N_ELEMENT_ID = 'gkill-export-i18n'

/** 読めた焼き込みデータ。形が合わなければ null になる */
interface ExportedPayload {
    schema_version: number
    title: string
    locale_name: string
}

/**
 * 焼き込まれた JSON を読む。
 *
 * `<script type="application/json">` の textContent を JSON.parse する。
 * `<script>` の中に JS リテラルとして書く形にはしないこと —— 記録の本文に `</script>` が
 * 1つあるだけで、ブラウザがそこから先を HTML として解釈して静かに壊れる。
 */
function read_json_element(element_id: string): unknown {
    const element = document.getElementById(element_id)
    if (!element || !element.textContent) {
        return null
    }
    try {
        return JSON.parse(element.textContent)
    } catch {
        // 壊れた出力でも画面は出す（下の is_ready が false になり、その旨を表示する）
        return null
    }
}

function parse_payload(raw: unknown): ExportedPayload | null {
    if (!raw || typeof raw !== 'object') {
        return null
    }
    const candidate = raw as Partial<ExportedPayload>
    if (typeof candidate.schema_version !== 'number') {
        // ガワだけの状態（差し込み前の template.html）はここに落ちる
        return null
    }
    return {
        schema_version: candidate.schema_version,
        title: typeof candidate.title === 'string' ? candidate.title : '',
        locale_name: typeof candidate.locale_name === 'string' ? candidate.locale_name : 'ja',
    }
}

/**
 * 「HTML で出力」した画面の入口。
 *
 * 画面の寸法の取り方は use-shared-rykv-page.ts と同じ（共有ページと同じく
 * 単独ページとして動く）。違うのは、データをサーバへ取りに行かず、
 * HTML 自身へ焼き込まれた JSON から読むこと。
 *
 * **出力 HTML は通信をしない。** 焼き込みが読めなければ、その旨を出して終わる
 * （サーバへ取りに行くフォールバックを足してはいけない）。
 */
export function useExportedRykvPage() {
    // ── State refs ──
    const actual_height: Ref<number> = ref(0)
    const element_height: Ref<number> = ref(0)
    const browser_url_bar_height: Ref<number> = ref(0)
    const app_title_bar_height: Ref<number> = ref(50)
    const app_content_height: Ref<number> = ref(0)
    const app_content_width: Ref<number> = ref(0)

    const payload: Ref<ExportedPayload | null> = ref(parse_payload(read_json_element(EXPORT_DATA_ELEMENT_ID)))
    const is_ready: Ref<boolean> = ref(payload.value !== null)
    const share_title: Ref<string> = ref(payload.value ? payload.value.title : '')
    const application_config: Ref<ApplicationConfig> = ref(new ApplicationConfig())
    const gkill_api: Ref<GkillAPI> = ref(GkillAPI.get_gkill_api())

    const { push_errors, push_messages } = useGkillMessageFeed()

    function write_errors(errors_: Array<GkillError>): void {
        push_errors(errors_)
    }

    function write_messages(messages_: Array<GkillMessage>): void {
        push_messages(messages_)
    }

    // ── Helpers ──

    function resize_content(): void {
        const inner_element = document.querySelector('#control-height')
        actual_height.value = window.innerHeight
        element_height.value = inner_element ? inner_element.clientHeight : actual_height.value
        browser_url_bar_height.value = Number(element_height.value) - Number(actual_height.value)
        app_content_height.value = Number(element_height.value) - (Number(browser_url_bar_height.value) + Number(app_title_bar_height.value))
        app_content_width.value = window.innerWidth
    }

    /** 焼き込まれたロケールを当てる。ja で出力したときは差し込みが無いので何もしない */
    function apply_locale(): void {
        if (!payload.value) {
            return
        }
        apply_baked_locale(payload.value.locale_name, read_json_element(EXPORT_I18N_ELEMENT_ID))
    }

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
    apply_locale()
    resize_content()

    return {
        // State
        actual_height,
        app_title_bar_height,
        app_content_height,
        app_content_width,
        is_ready,
        share_title,
        application_config,
        gkill_api,

        // Event handlers
        write_errors,
        write_messages,
    }
}
