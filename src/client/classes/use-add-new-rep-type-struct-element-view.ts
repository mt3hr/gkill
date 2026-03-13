import { i18n } from '@/i18n'
import { RepTypeStructElementData } from '@/classes/datas/config/rep-type-struct-element-data'
import { type Ref, ref } from 'vue'
import type { AddNewRepTypeStructElementViewEmits } from '@/pages/views/add-new-rep-type-struct-element-view-emits'
import type { AddNewRepTypeStructElementViewProps } from '@/pages/views/add-new-rep-type-struct-element-view-props'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'

export function useAddNewRepTypeStructElementView(options: {
    props: AddNewRepTypeStructElementViewProps,
    emits: AddNewRepTypeStructElementViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const rep_type_name: Ref<string> = ref("")
    const check_when_inited: Ref<boolean> = ref(true)

    // ── Business logic ──
    function emits_rep_type_name(): void {
        if (rep_type_name.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.rep_type_struct_title_is_blank
            error.error_message = i18n.global.t("REP_TYPE_IS_BLANK_MESSAGE")
            emits('received_errors', [error])
            return
        }

        const rep_type_struct_element = new RepTypeStructElementData()
        rep_type_struct_element.id = props.gkill_api.generate_uuid()
        rep_type_struct_element.is_dir = false
        rep_type_struct_element.check_when_inited = check_when_inited.value
        rep_type_struct_element.children = null
        rep_type_struct_element.indeterminate = false
        rep_type_struct_element.key = rep_type_name.value
        rep_type_struct_element.rep_type_name = rep_type_name.value
        rep_type_struct_element.name = rep_type_name.value
        emits('requested_add_rep_type_struct_element', rep_type_struct_element)
        emits('requested_close_dialog')
    }

    function reset_rep_type_name(): void {
        rep_type_name.value = ""
        check_when_inited.value = true
    }

    // ── Return ──
    return {
        // State
        rep_type_name,
        check_when_inited,

        // Business logic
        emits_rep_type_name,
        reset_rep_type_name,
    }
}
