'use strict'

import { nextTick, ref, watch, type Ref } from 'vue'
import type { MiFindQueryEditorDialogProps } from '@/pages/dialogs/mi-find-query-editor-dialog-props'
import type { MiFindQueryEditorDialogEmits } from '@/pages/dialogs/mi-find-query-editor-dialog-emits'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'

export function useMiFindQueryEditorDialog(options: {
    props: MiFindQueryEditorDialogProps
    emits: MiFindQueryEditorDialogEmits
    model_value: Ref<FindKyouQuery | undefined>
}) {
    const { props, emits: _emits, model_value } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("mi-find-query-editor-dialog", {
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
            cloned_find_kyou_query.value = find_kyou_query.clone()
            // query_idが空=「値が未セット」の印で、エディタ側がApplicationConfig既定を
            // 適用する判定に使う。ここで無条件に採番すると空の印が潰れて既定が効かない。
            // セット済みのクエリだけ、呼び出し元のIDと衝突しないよう新IDへ振り直す
            if (cloned_find_kyou_query.value.query_id !== "") {
                cloned_find_kyou_query.value.query_id = props.gkill_api.generate_uuid()
            }
            is_show_dialog.value = true
            received_application_config.value = new ApplicationConfig()
            await nextTick(() => received_application_config.value = props.application_config)
        })
    }

    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
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
