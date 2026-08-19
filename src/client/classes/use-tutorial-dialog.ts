'use strict'

import { computed, type Ref, ref, watch } from 'vue'
import { useTheme } from 'vuetify'
import type { TutorialDialogProps } from '@/pages/dialogs/tutorial-dialog-props'
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'
import { i18n } from '@/i18n'
import { UpdateApplicationConfigRequest } from '@/classes/api/req_res/update-application-config-request'

export function useTutorialDialog(options: {
    props: TutorialDialogProps
}) {
    const { props } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("tutorial-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    const theme = useTheme()
    const dont_show_again: Ref<boolean> = ref(false)
    const tutorial_url = computed(() => {
        const locale = i18n.global.locale || 'ja'
        const is_dark = theme.global.name.value === 'gkill_dark_theme'
        return `/resources/manual/${locale}/tutorial.html${is_dark ? '?theme=dark' : ''}`
    })
    async function show(): Promise<void> {
        dont_show_again.value = false
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        is_show_dialog.value = false
    }
    watch(dont_show_again, async (checked) => {
        if (!checked) return
        const config = props.application_config.clone()
        config.show_tutorial_on_startup = false
        const req = new UpdateApplicationConfigRequest()
        req.session_id = props.gkill_api.get_session_id()
        req.application_config = config
        await props.gkill_api.update_application_config(req)
        // ブラウザ側キャッシュも更新
        config.show_tutorial_on_startup = false
        props.gkill_api.set_saved_application_config(props.application_config)
    })
    async function close_dialog(): Promise<void> {
        await hide()
    }

    return {
        is_show_dialog,
        ui,
        dont_show_again,
        tutorial_url,
        show,
        hide,
        close_dialog,
    }
}
