'use strict'

import { type Ref, ref } from 'vue'
import type { AddRepViewProps } from '@/pages/views/add-rep-view-props'
import type { AddRepViewEmits } from '@/pages/views/add-rep-view-emits'
import { Repository } from '@/classes/datas/config/repository'

export function useAddRepView(options: { props: AddRepViewProps, emits: AddRepViewEmits }) {
    const { props, emits } = options

    const device: Ref<string> = ref("")
    const type: Ref<string> = ref("")
    const file: Ref<string> = ref("")
    const use_to_write: Ref<boolean> = ref(false)
    const is_execute_idf_when_reload: Ref<boolean> = ref(false)
    const is_enable: Ref<boolean> = ref(true)

    const devices: Ref<Array<string>> = ref((() => {
        const devices = Array<string>()
        for (let i = 0; i < props.server_configs.length; i++) {
            devices.push(props.server_configs[i].device)
        }
        return devices
    })())

    const rep_types: Ref<Array<string>> = ref([
        "kmemo",
        "kc",
        "urlog",
        "timeis",
        "mi",
        "nlog",
        "lantana",
        "tag",
        "text",
        "rekyou",
        "directory",
        "gpslog",
        "git_commit_log",
        "notification",
    ])

    async function add_rep(): Promise<void> {
        const repository = new Repository()
        repository.id = props.gkill_api.generate_uuid()
        repository.device = device.value
        repository.user_id = props.account.user_id
        repository.type = type.value
        repository.file = file.value
        repository.use_to_write = use_to_write.value
        repository.is_execute_idf_when_reload = is_execute_idf_when_reload.value
        repository.is_enable = is_enable.value
        emits('requested_add_rep', repository)
        emits('requested_close_dialog')
    }

    return {
        device,
        type,
        file,
        use_to_write,
        is_execute_idf_when_reload,
        is_enable,
        devices,
        rep_types,
        add_rep,
    }
}
