'use strict'

import { ref, type Ref } from 'vue'
import type { ServerConfigDialogProps } from '@/pages/dialogs/server-config-dialog-props'
import type { ServerConfigDialogEmits } from '@/pages/dialogs/server-config-dialog-emits'
import { ServerConfig } from '@/classes/datas/config/server-config'
import { GetServerConfigsRequest } from '@/classes/api/req_res/get-server-configs-request'
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'

export function useServerConfigDialog(options: {
    props: ServerConfigDialogProps
    emits: ServerConfigDialogEmits
}) {
    const { props, emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("server-config-dialog", {
        centerMode: "always",
    })

    const server_configs: Ref<Array<ServerConfig>> = ref(new Array<ServerConfig>())

    async function show(): Promise<void> {
        load_server_configs()
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        is_show_dialog.value = false
    }
    async function load_server_configs(): Promise<void> {
        server_configs.value.splice(0)
        const req = new GetServerConfigsRequest()
        const res = await props.gkill_api.get_server_configs(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
        server_configs.value = res.server_configs
    }

    return {
        is_show_dialog,
        ui,
        server_configs,
        show,
        hide,
        load_server_configs,
    }
}
