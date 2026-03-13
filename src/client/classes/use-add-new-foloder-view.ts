import { i18n } from '@/i18n'
import { ref, type Ref } from 'vue'
import { FolderStructElementData } from '@/classes/datas/config/folder-struct-element-data'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import type { AddNewFoloderViewProps } from '@/pages/views/add-new-foloder-view-props'
import type { AddNewFoloderViewEmits } from '@/pages/views/add-new-foloder-view-emits'

export function useAddNewFoloderView(options: {
    props: AddNewFoloderViewProps,
    emits: AddNewFoloderViewEmits,
}) {
    const { props, emits } = options

    const folder_name: Ref<string> = ref("")

    function emits_folder(): void {
        if (folder_name.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.folder_name_is_blank
            error.error_message = i18n.global.t("FOLDER_NAME_IS_BLANK_MESSAGE")
            emits('received_errors', [error])
            return
        }

        const folder_struct_element = new FolderStructElementData()
        folder_struct_element.id = props.gkill_api.generate_uuid()
        folder_struct_element.folder_name = folder_name.value
        emits('requested_add_new_folder', folder_struct_element)
        emits('requested_close_dialog')
    }

    function reset_folder_name(): void {
        folder_name.value = ""
    }

    return {
        folder_name,
        emits_folder,
        reset_folder_name,
    }
}
