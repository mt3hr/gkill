'use strict'

import type { TagHistoriesDialogProps } from '@/pages/dialogs/tag-histories-dialog-props'
import type { KyouDialogEmits } from '@/pages/views/kyou-dialog-emits'
import { computed, type Ref, ref } from 'vue'
import type { InfoIdentifier } from '@/classes/datas/info-identifier'
import type { Kyou } from '@/classes/datas/kyou'
import { build_kyou_dialog_relay } from '@/classes/kyou-view-relay'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useTagHistoriesDialog(options: {
    props: TagHistoriesDialogProps
    emits: KyouDialogEmits
}) {
    const { props, emits } = options

    // クリックはフォーカス移動も伴う
    const crudRelayHandlers = build_kyou_dialog_relay(emits, {
        'clicked_kyou': (kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) },
    })
    const tag_highlight_targets = computed<Array<InfoIdentifier>>(() => {
        const info_identifier = props.tag.generate_info_identifier()
        return [info_identifier]
    })
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog, { onClosed: () => emits('closed') })
    const ui = useFloatingDialog("tag-histories-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    const show_kyou: Ref<boolean> = ref(false)
    async function show(): Promise<void> {
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }

    return {
        crudRelayHandlers,
        tag_highlight_targets,
        is_show_dialog,
        ui,
        show_kyou,
        show,
        hide,
    }
}
