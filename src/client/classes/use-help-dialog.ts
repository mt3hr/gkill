'use strict'

import { computed, type Ref, ref } from 'vue'
import { useTheme } from 'vuetify'
import type { HelpDialogProps } from '@/pages/dialogs/help-dialog-props'
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'
import { i18n } from '@/i18n'

export function useHelpDialog(options: {
    props: HelpDialogProps
}) {
    const { props } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("help-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    const theme = useTheme()
    const help_url = computed(() => {
        const locale = i18n.global.locale || 'ja'
        const is_dark = theme.global.name.value === 'gkill_dark_theme'
        return `/resources/manual/${locale}/${props.screen_name}.html${is_dark ? '?theme=dark' : ''}`
    })
    async function show(): Promise<void> {
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        is_show_dialog.value = false
    }

    return {
        is_show_dialog,
        ui,
        help_url,
        show,
        hide,
    }
}
