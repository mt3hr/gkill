import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import { RepStructElementData } from '@/classes/datas/config/rep-struct-element-data'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import type { AddNewRepStructElementViewProps } from '@/pages/views/add-new-rep-struct-element-view-props'
import type { AddNewRepStructElementViewEmits } from '@/pages/views/add-new-rep-struct-element-view-emits'

export function useAddNewRepStructElementView(options: {
    props: AddNewRepStructElementViewProps,
    emits: AddNewRepStructElementViewEmits,
}) {
    const { props, emits } = options

    const rep_name: Ref<string> = ref("")
    const check_when_inited: Ref<boolean> = ref(true)
    const ignore_check_rep_rykv: Ref<boolean> = ref(false)

    function emits_rep_name(): void {
        if (rep_name.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.rep_struct_title_is_blank
            error.error_message = i18n.global.t("REP_IS_BLANK_MESSAGE")
            emits('received_errors', [error])
            return
        }

        const rep_struct_element = new RepStructElementData()
        rep_struct_element.id = props.gkill_api.generate_uuid()
        rep_struct_element.is_dir = false
        rep_struct_element.check_when_inited = check_when_inited.value
        rep_struct_element.ignore_check_rep_rykv = ignore_check_rep_rykv.value
        rep_struct_element.children = null
        rep_struct_element.indeterminate = false
        rep_struct_element.key = rep_name.value
        rep_struct_element.rep_name = rep_name.value
        rep_struct_element.name = rep_name.value
        emits('requested_add_rep_struct_element', rep_struct_element)
        emits('requested_close_dialog')
    }

    function reset_rep_name(): void {
        rep_name.value = ""
        check_when_inited.value = true
        ignore_check_rep_rykv.value = false
    }

    return {
        rep_name,
        check_when_inited,
        ignore_check_rep_rykv,
        emits_rep_name,
        reset_rep_name,
    }
}
