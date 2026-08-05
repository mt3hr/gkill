import { computed, nextTick, onMounted, onUnmounted, ref, watch, type Ref } from 'vue'
import { i18n } from '@/i18n'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillMessage } from '@/classes/api/gkill-message'
import { LineLabelData } from '@/classes/kftl/line-label-data'
import { KFTLStatement } from '@/classes/kftl/kftl-statement'
import { KFTL_ASCII_SAVE_CHARACTOR } from '@/classes/kftl/kftl-prefixes'
import { TextAreaInfo } from '@/classes/kftl/text-area-info'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'
import { DiscardTXRequest } from '@/classes/api/req_res/discard-tx-request'
import { CommitTXRequest } from '@/classes/api/req_res/commit-tx-request'
import { tag_exists_in_tag_struct } from '@/classes/tag-struct'
import { useFloatingDialog } from '@/classes/use-floating-dialog'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import type { KFTLProps } from '@/pages/views/kftl-props'
import type { KFTLViewEmits } from '@/pages/views/kftl-view-emits'
import type { KFTLRequest } from '@/classes/kftl/kftl-request'
import type { KFTLTemplateElementData } from '@/classes/datas/kftl-template-element-data'
import type { ComponentRef } from '@/classes/component-ref'

export function useKftlView(options: {
    props: KFTLProps,
    emits: KFTLViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const kftl_template_dialog = ref<ComponentRef | null>(null)

    // ── State refs ──
    const text_area_content: Ref<string> = ref("")
    const text_area_width: Ref<number | 'unset'> = ref(0)
    const text_area_height: Ref<number> = ref(0)
    const line_label_width: Ref<number> = ref(0)
    const line_label_height: Ref<number> = ref(0)

    const title_height = 52
    const action_height = 10
    const kftl_input_height: Ref<number> = ref(0)
    const kftl_input_width: Ref<number | 'unset'> = ref(0)

    const line_label_datas: Ref<Array<LineLabelData>> = ref(new Array<LineLabelData>())
    const line_label_styles: Ref<Array<Record<string, string>>> = ref(new Array<Record<string, string>>())
    const invalid_line_numbers: Ref<Array<number>> = ref(new Array<number>())
    const is_requested_submit: Ref<boolean> = ref(true)
    const show_confirm_unknown_tag_dialog: Ref<boolean> = ref(false)
    const unknown_tags: Ref<string[]> = ref([])

    // ── Dialog UI ──
    useDialogHistoryStack(show_confirm_unknown_tag_dialog)
    const confirm_dialog_ui = useFloatingDialog("kftl-confirm-unknown-tag-dialog", {
        centerMode: "always",
    })

    // ── Computed ──
    const text_area_width_px = computed(() => text_area_width.value.toString().concat("px"))
    const text_area_height_px = computed(() => text_area_height.value.toString().concat("px"))
    const line_label_width_px = computed(() => line_label_width.value.toString().concat("px"))
    const line_label_height_px = computed(() => line_label_height.value.toString().concat("px"))
    const kftl_input_height_px = computed(() => kftl_input_height.value.toString().concat("px"))
    const kftl_input_width_px = computed(() => kftl_input_width.value.toString().concat("px"))

    // ── Watchers ──
    if (props.application_config.is_loaded) {
        is_requested_submit.value = false
    }
    watch(() => props.application_config, () => {
        if (props.application_config.is_loaded) {
            is_requested_submit.value = false
        }
    })
    watch(() => text_area_content.value, (new_value, old_value) => {
        if (new_value === old_value) {
            return
        }
        update_line_labels()
        save_content_to_localstorage()
    })
    watch(line_label_datas, async () => {
        line_label_styles.value.splice(0)
        let prev_target_id = ""
        let background_is_gray = true
        let switch_id = false
        let background_color: string = "white"
        for (let i = 0; i < line_label_datas.value.length; i++) {
            let color: string = "unset"
            switch_id = prev_target_id != line_label_datas.value[i].target_request_id
            if (switch_id) {
                background_is_gray = !background_is_gray
                if (background_is_gray) {
                    if (props.application_config.use_dark_theme) {
                        background_color = '#C0C0C0'
                    } else {
                        background_color = "#f0f0f0"
                    }
                } else {
                    background_color = ""
                }
            }
            if (is_invalid_line(i)) {
                color = "pink"
            }
            line_label_styles.value.push({
                "background-color": background_color,
                "color": color,
            })
            prev_target_id = line_label_datas.value[i].target_request_id
        }
    })

    // ── Lifecycle ──
    nextTick(() => {
        const kftl_text_area_element_id = "kftl_text_area"
        const kftl_text_area_element = document.getElementById(kftl_text_area_element_id)!
        kftl_text_area_element.addEventListener("scroll", update_line_labels)
        update_line_labels()
    })

    restore_content_from_localstorage()

    function onResize() {
        resize()
        update_line_labels()
    }
    window.addEventListener("resize", onResize)
    watch(() => [props.app_content_width, props.app_content_height], () => {
        resize()
        update_line_labels()
    })

    // ── beforeunload guard ──
    // テキストエリアに未保存の内容がある場合、ページ離脱時に警告を表示する
    function onBeforeunload(e: BeforeUnloadEvent) {
        if (text_area_content.value.trim() !== "") {
            e.preventDefault()
        }
    }
    window.addEventListener("beforeunload", onBeforeunload)

    onMounted(() => resize())
    onUnmounted(() => {
        window.removeEventListener("resize", onResize)
        window.removeEventListener("beforeunload", onBeforeunload)
    })

    // ── Internal helpers ──
    async function restore_content_from_localstorage(): Promise<void> {
        const saved_content = localStorage.getItem("kftl_content")
        if (saved_content) {
            text_area_content.value = saved_content
        }
    }

    async function save_content_to_localstorage(): Promise<void> {
        localStorage.setItem("kftl_content", text_area_content.value)
    }

    function sync_line_label_scroll(kftl_text_area_element: HTMLElement): void {
        const kftl_line_label_elements = document.getElementsByClassName("kftl_line_label")!
        for (let i = 0; i < kftl_line_label_elements.length; i++) {
            const kftl_line_label_element = kftl_line_label_elements.item(i)
            if (kftl_line_label_element) {
                kftl_line_label_element.scrollTo(0, kftl_text_area_element.scrollTop)
            }
        }
    }

    async function update_line_labels(): Promise<void> {
        const kftl_text_area_element_id = "kftl_text_area"
        const kftl_text_area_element = document.getElementById(kftl_text_area_element_id)!

        sync_line_label_scroll(kftl_text_area_element)

        const statement = new KFTLStatement(text_area_content.value)
        const textarea_info = new TextAreaInfo()
        textarea_info.text_area_element_id = kftl_text_area_element_id

        line_label_datas.value = statement.generate_line_label_data(textarea_info)

        // ラベル再生成後、DOM更新を待ってスクロール位置を再同期する
        await nextTick()
        sync_line_label_scroll(kftl_text_area_element)

        invalid_line_numbers.value = await statement.get_invalid_line_indexs()

        if ((text_area_content.value.endsWith("\n" + i18n.global.t("KFTL_SAVE_CHARACTOR") + "\n") || text_area_content.value.endsWith("\n" + KFTL_ASCII_SAVE_CHARACTOR + "\n")) && !is_requested_submit.value) {
            is_requested_submit.value = true
            submit()
        }
    }

    function is_invalid_line(line_index: number): boolean {
        for (let i = 0; i < invalid_line_numbers.value.length; i++) {
            if (invalid_line_numbers.value[i] == line_index) {
                return true
            }
        }
        return false
    }

    async function resize(): Promise<void> {
        line_label_width.value = 100
        line_label_height.value = props.app_content_height.valueOf() - title_height - action_height
        text_area_width.value = props.app_content_width.valueOf() - line_label_width.value.valueOf() - 7 // 7はマジックナンバー
        text_area_height.value = props.app_content_height.valueOf() - title_height - action_height
        kftl_input_width.value = line_label_width.value.valueOf() + text_area_width.value.valueOf()
        kftl_input_height.value = props.app_content_height.valueOf() - title_height - action_height
    }

    // ── Business logic ──
    // 全角「！」または半角「!」の保存マーカーを取り除く
    function remove_save_marker(text: string): string {
        const ja_marker = "\n" + i18n.global.t("KFTL_SAVE_CHARACTOR") + "\n"
        if (text.includes(ja_marker)) {
            return text.replace(ja_marker, "\n")
        }
        return text.replace("\n" + KFTL_ASCII_SAVE_CHARACTOR + "\n", "\n")
    }

    // 追加されるタグのうち、TagStructに存在しないものを重複なく集める
    function collect_unknown_tags(kftl_requests: Array<KFTLRequest>): Array<string> {
        const unknown = new Array<string>()
        for (let i = 0; i < kftl_requests.length; i++) {
            const tags = kftl_requests[i].get_tags()
            for (let j = 0; j < tags.length; j++) {
                const tag = tags[j]
                if (tag === "") {
                    continue
                }
                if (tag_exists_in_tag_struct(tag, props.application_config.tag_struct)) {
                    continue
                }
                if (unknown.includes(tag)) {
                    continue
                }
                unknown.push(tag)
            }
        }
        return unknown
    }

    async function submit(): Promise<void> {
        await do_submit(false)
    }

    function cancel_submit(): void {
        close_dialog_via_history(show_confirm_unknown_tag_dialog)
        unknown_tags.value = []
    }

    async function confirm_submit(): Promise<void> {
        close_dialog_via_history(show_confirm_unknown_tag_dialog)
        unknown_tags.value = []
        await do_submit(true)
    }

    async function do_submit(skip_unknown_tag_check: boolean): Promise<void> {
        try {
            if (invalid_line_numbers.value.length != 0) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.kftl_has_invalid_line
                error.error_message = i18n.global.t("KFTL_FOUND_INVALID_LINE_MESSAGE")
                text_area_content.value = remove_save_marker(text_area_content.value)
                emits('received_errors', [error])
                return
            }
            const statement = new KFTLStatement(text_area_content.value)
            const kftl_requests = await statement.generate_requests()

            // TagStructに存在しないタグを検出したら、送信前に確認を取る
            if (!skip_unknown_tag_check) {
                const not_found = collect_unknown_tags(kftl_requests)
                if (not_found.length > 0) {
                    unknown_tags.value = not_found
                    // 保存マーカーを消しておかないと、確認中の入力で再度submitされてしまう
                    text_area_content.value = remove_save_marker(text_area_content.value)
                    show_confirm_unknown_tag_dialog.value = true
                    return
                }
            }

            let last_added_request_time = new Date(Date.now()) // 「、、」でずれた分をPlaingTimeIsにわたすための考慮。リロード時刻より大きかった場合はこの値でTimeIsをリロードする
            let errors = new Array<GkillError>()
            const tx_id = kftl_requests.length > 0 ? kftl_requests[0].get_tx_id() : null
            for (let i = 0; i < kftl_requests.length; i++) {
                const request = kftl_requests[i]
                const request_related_time = request.get_related_time()
                if (request_related_time && request_related_time.getTime() > last_added_request_time.getTime()) {
                    last_added_request_time = request_related_time
                }
                await request.do_request(props.gkill_api, props.application_config).then(request_errors => errors = errors.concat(request_errors))
            }
            if (errors.length != 0) {
                emits('received_errors', errors)
                text_area_content.value = remove_save_marker(text_area_content.value)

                if (tx_id) {
                    const deiscard_req = new DiscardTXRequest()
                    deiscard_req.tx_id = tx_id
                    const discard_res = await props.gkill_api.discard_tx(deiscard_req)
                    if (discard_res.errors && discard_res.errors.length != 0) {
                        emits('received_errors', discard_res.errors)
                    }
                    return
                }
            }
            if (tx_id) {
                const commit_req = new CommitTXRequest()
                commit_req.tx_id = tx_id
                const commit_res = await props.gkill_api.commit_tx(commit_req)
                if (commit_res.errors && commit_res.errors.length != 0) {
                    emits('received_errors', commit_res.errors)
                    text_area_content.value = remove_save_marker(text_area_content.value)
                    return
                }
            }

            clear()
            const message = new GkillMessage()
            message.message_code = GkillMessageCodes.saved_kftls
            message.message = i18n.global.t("SAVED_MESSAGE")
            emits('received_messages', [message])
            emits('saved_kyou_by_kftl', last_added_request_time)
        } finally {
            is_requested_submit.value = false
        }
    }

    async function clear(): Promise<void> {
        text_area_content.value = ""
    }

    function show_kftl_template_dialog(): void {
        kftl_template_dialog.value?.show()
    }

    function paste_template(template: KFTLTemplateElementData): void {
        text_area_content.value = template.template as string
        kftl_template_dialog.value?.hide()
    }

    async function focus_kftl_text_area(): Promise<void> {
        document.getElementById("kftl_text_area")?.focus()
    }

    // ── Event relay objects ──
    const errorMessageRelayHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
    }

    // ── Return ──
    return {
        // Template refs
        kftl_template_dialog,

        // State
        text_area_content,
        line_label_datas,
        line_label_styles,
        is_requested_submit,
        title_height,
        show_confirm_unknown_tag_dialog,
        unknown_tags,

        // Dialog UI
        confirm_dialog_ui,

        // Computed
        text_area_width_px,
        text_area_height_px,
        line_label_width_px,
        line_label_height_px,
        kftl_input_height_px,
        kftl_input_width_px,

        // Business logic
        submit,
        cancel_submit,
        confirm_submit,
        show_kftl_template_dialog,
        paste_template,
        focus_kftl_text_area,

        // Event relay objects
        errorMessageRelayHandlers,
    }
}
