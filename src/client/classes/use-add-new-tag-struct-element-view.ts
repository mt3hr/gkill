import { i18n } from '@/i18n'
import { TagStructElementData } from '@/classes/datas/config/tag-struct-element-data'
import { type Ref, ref } from 'vue'
import type { AddNewTagStructElementViewEmits } from '@/pages/views/add-new-tag-struct-element-view-emits'
import type { AddNewTagStructElementViewProps } from '@/pages/views/add-new-tag-struct-element-view-props'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'

export function useAddNewTagStructElementView(options: {
    props: AddNewTagStructElementViewProps,
    emits: AddNewTagStructElementViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const tag_name: Ref<string> = ref("")
    const check_when_inited: Ref<boolean> = ref(true)
    const is_force_hide: Ref<boolean> = ref(false)

    // ── Business logic ──
    function emits_tag_name(): void {
        if (tag_name.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.tag_struct_title_is_blank
            error.error_message = i18n.global.t("TAG_IS_BLANK_MESSAGE")
            emits('received_errors', [error])
            return
        }

        const tag_struct_element = new TagStructElementData()
        tag_struct_element.id = props.gkill_api.generate_uuid()
        tag_struct_element.is_dir = false
        tag_struct_element.check_when_inited = check_when_inited.value
        tag_struct_element.is_force_hide = is_force_hide.value
        tag_struct_element.children = null
        tag_struct_element.indeterminate = false
        tag_struct_element.key = tag_name.value
        tag_struct_element.tag_name = tag_name.value
        tag_struct_element.name = tag_name.value
        emits('requested_add_tag_struct_element', tag_struct_element)
        emits('requested_close_dialog')
    }

    function reset_tag_name(): void {
        tag_name.value = ""
        check_when_inited.value = true
        is_force_hide.value = false
    }

    // ── Return ──
    return {
        // State
        tag_name,
        check_when_inited,
        is_force_hide,

        // Business logic
        emits_tag_name,
        reset_tag_name,
    }
}
