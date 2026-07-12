import { ref, computed, watch, nextTick } from 'vue'
import type { RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import type { IDFKyouProps } from '@/pages/views/idf-kyou-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { ComponentRef } from '@/classes/component-ref'
import { detectAndDecodeText } from '@/classes/decode-text'
import { is_markdown_file_name, markdown_to_safe_html, truncate_markdown, MD_LINK_DATA_ATTRIBUTE } from '@/classes/markdown-to-html'
import { render_mermaid_diagrams } from '@/classes/mermaid-render'
import { GetIDFKyouByRelativePathRequest } from '@/classes/api/req_res/get-idf-kyou-by-relative-path-request'
import { GetKyouRequest } from '@/classes/api/req_res/get-kyou-request'

const TEXT_EXTENSIONS = new Set(['txt'])

const MAX_TEXT_LENGTH = 10000

// リスト表示では高さが固定でごく一部しか見えないため、
// 全文をパース・サニタイズせず短く切り詰める。
const MAX_MARKDOWN_LENGTH_IN_LIST = 2000

function get_extension(filename: string): string {
    const dot = filename.lastIndexOf('.')
    return dot >= 0 ? filename.slice(dot + 1).toLowerCase() : ''
}

export function useIDFKyouView(options: {
    props: IDFKyouProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const context_menu = ref<ComponentRef | null>(null)
    const markdown_content = ref<HTMLElement | null>(null)

    // ── Text file preview ──
    const text_content = ref<string | null>(null)
    const markdown_html = ref<string>('')
    const text_loading = ref(false)

    // v-virtual-scrollがコンポーネントを使い回すため、
    // await中に対象Kyouが差し替わったら古いレスポンスを捨てる。
    let load_seq = 0

    // インライン表示対象外のファイル種別
    function is_previewable_file(): boolean {
        const idf = props.kyou.typed_idf_kyou
        return !!idf && !idf.is_image && !idf.is_video && !idf.is_audio && !idf.is_zip
    }

    const is_text = computed((): boolean => {
        if (!is_previewable_file()) return false
        const fname = props.kyou.typed_idf_kyou?.file_name ?? ''
        return TEXT_EXTENSIONS.has(get_extension(fname))
    })

    const is_markdown = computed((): boolean => {
        if (!is_previewable_file()) return false
        const fname = props.kyou.typed_idf_kyou?.file_name ?? ''
        return is_markdown_file_name(fname)
    })

    async function load_text_content(): Promise<void> {
        const url = props.kyou.typed_idf_kyou?.file_url
        if (!url || (!is_text.value && !is_markdown.value)) return

        const seq = ++load_seq
        const render_as_markdown = is_markdown.value
        text_loading.value = true
        text_content.value = null
        markdown_html.value = ''
        try {
            const res = await fetch(url)
            if (!res.ok) return
            const bytes = new Uint8Array(await res.arrayBuffer())
            // 文字コードを判定してデコードする (Shift_JIS等の文字化け対策)
            const raw = detectAndDecodeText(bytes)

            if (render_as_markdown) {
                const max_length = props.is_image_request_to_thumb_size ? MAX_MARKDOWN_LENGTH_IN_LIST : MAX_TEXT_LENGTH
                const html = await markdown_to_safe_html(truncate_markdown(raw, max_length), url)
                if (seq !== load_seq) return
                markdown_html.value = html

                // Mermaidの描画はDOMを要求するため、v-htmlが反映されてから行う
                await nextTick()
                if (seq !== load_seq || !markdown_content.value) return
                await render_mermaid_diagrams(markdown_content.value)
                return
            }

            if (seq !== load_seq) return
            text_content.value = raw.length > MAX_TEXT_LENGTH
                ? raw.slice(0, MAX_TEXT_LENGTH) + '\n…'
                : raw
        } catch {
            // 取得失敗時はリンク表示にフォールバックするため無視
        } finally {
            if (seq === load_seq) {
                text_loading.value = false
            }
        }
    }

    watch(
        () => props.kyou.typed_idf_kyou?.file_url,
        () => { if (is_text.value || is_markdown.value) load_text_content() },
        { immediate: true },
    )

    // ── Business logic ──
    function show_context_menu(e: PointerEvent): void {
        if (props.enable_context_menu) {
            context_menu.value?.show(e)
        }
    }

    function open_link(): void {
        const url = props.kyou.typed_idf_kyou?.file_url
        if (url) {
            window.open(url, "_blank")
        }
    }

    // ── Markdown内の相対MDリンク → KyouDialog ──

    function find_md_link_anchor(e: MouseEvent): HTMLAnchorElement | null {
        const target = e.target
        if (!(target instanceof Element)) return null
        return target.closest(`a[${MD_LINK_DATA_ATTRIBUTE}]`)
    }

    function on_markdown_content_click(e: MouseEvent): void {
        const anchor = find_md_link_anchor(e)
        if (!anchor) return
        // 修飾キー付きクリックはブラウザ既定の動作 (新規タブで開く等) に任せる
        if (e.ctrlKey || e.metaKey || e.shiftKey) return
        // ダブルクリック1回目のクリックで新規タブが開かないよう、シングルクリックの遷移は無効化する
        e.preventDefault()
    }

    async function on_markdown_content_dblclick(e: MouseEvent): Promise<void> {
        const anchor = find_md_link_anchor(e)
        if (!anchor) return
        e.preventDefault()
        // 親KyouViewのダブルクリック (現在のKyouのダイアログ表示) を抑止する
        e.stopPropagation()

        const fallback_url = anchor.getAttribute('href')
        const open_fallback = () => {
            if (fallback_url) {
                window.open(fallback_url, '_blank')
            }
        }

        // enable_dialog は内側KyouViewのdblclick抑止にも使われている (ryuu-item-view.vue) ため、
        // enable_dialog=false でも enable_md_link_dialog=true ならMarkDownリンクのダイアログを開く。
        // 未指定のBoolean propはVueが undefined ではなく false に解決するため ?? では上書きできない。
        if (!props.enable_md_link_dialog && !props.enable_dialog) {
            open_fallback()
            return
        }

        try {
            const req = new GetIDFKyouByRelativePathRequest()
            req.target_id = props.kyou.id
            // フラグメント・クエリを除いた相対パスをサーバへ渡す
            req.relative_path = (anchor.getAttribute(MD_LINK_DATA_ATTRIBUTE) ?? '').split(/[#?]/)[0]
            const res = await props.gkill_api.get_idf_kyou_by_relative_path(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.kyou_id) {
                const kyou_req = new GetKyouRequest()
                kyou_req.id = res.kyou_id
                const kyou_res = await props.gkill_api.get_kyou(kyou_req)
                if (kyou_res.errors && kyou_res.errors.length !== 0) {
                    emits('received_errors', kyou_res.errors)
                    return
                }
                if (kyou_res.kyou_histories.length !== 0) {
                    emits('requested_open_rykv_dialog', 'kyou', kyou_res.kyou_histories[0])
                    return
                }
            }
            // 対象がKyouとして見つからない場合は従来どおり生ファイルを新規タブで開く
            open_fallback()
        } catch {
            open_fallback()
        }
    }

    function buildMediaUrl(fileUrl: string, isVideoThumb: boolean): string {
        let is_added_query = false
        if (isVideoThumb) {
            if (!is_added_query) {
                fileUrl += "?"
            } else {
                fileUrl += "&"
            }
            fileUrl += "is_video=true"
            is_added_query = true

            if (!is_added_query) {
                fileUrl += "?"
            } else {
                fileUrl += "&"
            }
            fileUrl += "thumb=400x400"
            is_added_query = true
            return fileUrl
        }
        if (props.is_image_request_to_thumb_size) {
            if (!is_added_query) {
                fileUrl += "?"
            } else {
                fileUrl += "&"
            }
            fileUrl += "thumb=400x400"
            is_added_query = true
        }
        return fileUrl
    }

    // ── Event relay objects ──
    const crudRelayHandlers = {
        'deleted_kyou': (kyou: Kyou) => emits('deleted_kyou', kyou),
        'deleted_tag': (tag: Tag) => emits('deleted_tag', tag),
        'deleted_text': (text: Text) => emits('deleted_text', text),
        'deleted_notification': (notification: Notification) => emits('deleted_notification', notification),
        'registered_kyou': (kyou: Kyou) => emits('registered_kyou', kyou),
        'registered_tag': (tag: Tag) => emits('registered_tag', tag),
        'registered_text': (text: Text) => emits('registered_text', text),
        'registered_notification': (notification: Notification) => emits('registered_notification', notification),
        'updated_kyou': (kyou: Kyou) => emits('updated_kyou', kyou),
        'updated_tag': (tag: Tag) => emits('updated_tag', tag),
        'updated_text': (text: Text) => emits('updated_text', text),
        'updated_notification': (notification: Notification) => emits('updated_notification', notification),
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'requested_reload_kyou': (kyou: Kyou) => emits('requested_reload_kyou', kyou),
        'requested_reload_list': () => emits('requested_reload_list'),
        'requested_update_check_kyous': (kyous: Array<Kyou>, checked: boolean) => emits('requested_update_check_kyous', kyous, checked),
        'requested_open_rykv_dialog': (kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => emits('requested_open_rykv_dialog', kind, kyou, payload),
    }

    // ── Return ──
    return {
        // Template refs
        context_menu,
        markdown_content,

        // Text preview
        is_text,
        is_markdown,
        text_content,
        markdown_html,
        text_loading,

        // Business logic
        show_context_menu,
        open_link,
        on_markdown_content_click,
        on_markdown_content_dblclick,
        buildMediaUrl,

        // Event relay objects
        crudRelayHandlers,
    }
}
