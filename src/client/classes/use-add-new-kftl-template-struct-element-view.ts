import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import { KFTLTemplateStructElementData } from '@/classes/datas/config/kftl-template-struct-element-data'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import type { AddNewKFTLTemplateStructElementViewProps } from '@/pages/views/add-new-kftl-template-struct-element-view-props'
import type { AddNewKFTLTemplateStructElementViewEmits } from '@/pages/views/add-new-kftl-template-struct-element-view-emits'

export function useAddNewKftlTemplateStructElementView(options: {
    props: AddNewKFTLTemplateStructElementViewProps,
    emits: AddNewKFTLTemplateStructElementViewEmits,
}) {
    const { props, emits } = options

    const title: Ref<string> = ref("")
    const template: Ref<string> = ref("")

    function emits_kftl_template_name(): void {
        if (title.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.kftl_template_title_is_blank
            error.error_message = i18n.global.t("KFTL_TEMPLATE_NAME_IS_BLANK_MESSAGE")
            emits('received_errors', [error])
            return
        }

        if (template.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.kftl_template_content_is_blank
            error.error_message = i18n.global.t("KFTL_TEMPLATE_CONTENT_IS_BLANK_MESSAGE")
            emits('received_errors', [error])
            return
        }

        const kftl_template_struct_element = new KFTLTemplateStructElementData()
        kftl_template_struct_element.id = props.gkill_api.generate_uuid()
        kftl_template_struct_element.is_dir = false
        kftl_template_struct_element.key = title.value
        kftl_template_struct_element.title = title.value
        kftl_template_struct_element.name = title.value
        kftl_template_struct_element.template = template.value
        emits('requested_add_kftl_template_struct_element', kftl_template_struct_element)
        emits('requested_close_dialog')
    }

    function reset_kftl_template_name(): void {
        title.value = ""
        template.value = ""
    }

    return {
        title,
        template,
        emits_kftl_template_name,
        reset_kftl_template_name,
    }
}
