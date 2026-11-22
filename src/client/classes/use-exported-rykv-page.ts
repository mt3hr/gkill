import { onMounted, onUnmounted, ref, shallowRef, type Ref, type ShallowRef } from 'vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { GkillAPI } from '@/classes/api/gkill-api'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { reset_dialog_history } from '@/classes/use-dialog-history-stack'
import { useGkillMessageFeed } from '@/classes/use-gkill-message-feed'
import { apply_baked_locale } from '@/export-i18n'
import {
    build_exported_api, parse_exported_payload, type ExportedKyouPayload,
} from '@/classes/exported-kyou-payload'

/** HTML へ焼き込まれたデータの差し込み先の id（export.html と export_html_render.go で対） */
const export_data_element_id = 'gkill-export-data'
/** 焼き込まれたロケールのメッセージの差し込み先の id（ja で出力したときは置かれない） */
const export_i18n_element_id = 'gkill-export-i18n'

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
        // 壊れた出力でも画面は出す（is_ready が false になり、その旨を表示する）
        return null
    }
}

/**
 * 「HTML で出力」した画面の入口。
 *
 * 画面の寸法の取り方は use-shared-rykv-page.ts と同じ（共有ページと同じく単独ページ）。
 * 違うのは、データをサーバへ取りに行かず HTML 自身へ焼き込まれた JSON から読むこと。
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

    const raw_payload = read_json_element(export_data_element_id)
    const payload: Ref<ExportedKyouPayload | null> = ref(parse_exported_payload(raw_payload))
    const is_ready: Ref<boolean> = ref(payload.value !== null)
    const share_title: Ref<string> = ref(payload.value ? payload.value.title : '')
    const is_exported_with_dnote: Ref<boolean> = ref(payload.value ? payload.value.with_dnote : false)
    const is_exported_with_map: Ref<boolean> = ref(payload.value ? payload.value.with_map : false)
    const is_exported_with_sidebar: Ref<boolean> = ref(payload.value ? payload.value.with_sidebar : false)
    const application_config: Ref<ApplicationConfig> = ref(new ApplicationConfig())
    // GkillAPI は private / protected メンバを持つので ref() には入れられない
    // （Vue の UnwrapRef が非 public メンバを落とし、unwrap 後の型をクラスへ戻せなくなる）。
    // 差し替えを検知できれば十分で、深い reactive は要らない
    const gkill_api: ShallowRef<GkillAPI> = shallowRef(GkillAPI.get_gkill_api())

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
        apply_baked_locale(payload.value.locale_name, read_json_element(export_i18n_element_id))
    }

    /**
     * 焼き込みを通信をしない API へ流し込み、画面が使う API を差し替える。
     *
     * `GkillAPI.set_gkill_api()` までやるのは共有ページと同じ（データクラスの
     * `load_*` は `GkillAPI.get_gkill_api()` を自分で引くので、差し替えないと
     * 本体の fetch に落ちる＝通信する）。
     */
    function apply_payload(): void {
        if (!payload.value) {
            return
        }
        const exported_api = build_exported_api(raw_payload)
        GkillAPI.set_gkill_api(exported_api)
        gkill_api.value = exported_api
        application_config.value = exported_api.exported_application_config
        application_config.value.for_export_kyou = true
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
    apply_payload()
    resize_content()

    return {
        // State
        actual_height,
        app_title_bar_height,
        app_content_height,
        app_content_width,
        is_ready,
        share_title,
        is_exported_with_dnote,
        is_exported_with_map,
        is_exported_with_sidebar,
        application_config,
        gkill_api,

        // Event handlers
        write_errors,
        write_messages,
    }
}
