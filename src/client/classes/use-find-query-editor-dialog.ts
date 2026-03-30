'use strict'

import { nextTick, ref, watch, type Ref } from 'vue'
import type FindQueryEditorDialogProps from '@/pages/dialogs/find-query-editor-dialog-props'
import type FindQueryEditorDialogEmits from '@/pages/dialogs/find-query-editor-dialog-emits'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'

export function useFindQueryEditorDialog(options: {
    props: FindQueryEditorDialogProps
    emits: FindQueryEditorDialogEmits
    model_value: Ref<FindKyouQuery | undefined>
}) {
    const { props, emits: _emits, model_value } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("find-query-editor-dialog", {
        centerMode: "always",
    })

    const inited = ref(false)
    const cloned_find_kyou_query = ref<FindKyouQuery | null>(null)
    const received_application_config = ref(new ApplicationConfig())

    watch(() => inited.value, () => {
        if (inited.value) {
            return nextTick(async () => {
                model_value.value = cloned_find_kyou_query.value!
            })
        }
    })

    async function show(find_kyou_query: FindKyouQuery): Promise<void> {
        return nextTick(async () => {
            cloned_find_kyou_query.value = find_kyou_query
            cloned_find_kyou_query.value.query_id = props.gkill_api.generate_uuid()
            is_show_dialog.value = true
            received_application_config.value = new ApplicationConfig()
            await nextTick(() => received_application_config.value = props.application_config) // TODO なんかApplicationConfigが切り替わったタイミングでQueryEditorが読み込まれるっぽい・・・
        })
    }
    async function hide(): Promise<void> {
        is_show_dialog.value = false
    }

    return {
        is_show_dialog,
        ui,
        inited,
        cloned_find_kyou_query,
        received_application_config,
        show,
        hide,
    }
}
