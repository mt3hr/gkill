import { i18n } from '@/i18n'
import { DeviceStructElementData } from '@/classes/datas/config/device-struct-element-data'
import { type Ref, ref } from 'vue'
import type { AddNewDeviceStructElementViewEmits } from '@/pages/views/add-new-device-struct-element-view-emits'
import type { AddNewDeviceStructElementViewProps } from '@/pages/views/add-new-device-struct-element-view-props'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'

export function useAddNewDeviceStructElementView(options: {
    props: AddNewDeviceStructElementViewProps,
    emits: AddNewDeviceStructElementViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const device_name: Ref<string> = ref("")
    const check_when_inited: Ref<boolean> = ref(true)
    const is_force_hide: Ref<boolean> = ref(false)

    // ── Business logic ──
    function emits_device_name(): void {
        if (device_name.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.device_name_is_blank
            error.error_message = i18n.global.t("DEVICE_NAME_IS_BLANK_MESSAGE")
            emits('received_errors', [error])
            return
        }

        const device_struct_element = new DeviceStructElementData()
        device_struct_element.id = props.gkill_api.generate_uuid()
        device_struct_element.is_dir = false
        device_struct_element.check_when_inited = check_when_inited.value
        device_struct_element.children = null
        device_struct_element.indeterminate = false
        device_struct_element.key = device_name.value
        device_struct_element.device_name = device_name.value
        device_struct_element.name = device_name.value
        emits('requested_add_device_struct_element', device_struct_element)
        emits('requested_close_dialog')
    }

    function reset_device_name(): void {
        device_name.value = ""
        check_when_inited.value = true
        is_force_hide.value = false
    }

    // ── Return ──
    return {
        // State
        device_name,
        check_when_inited,
        is_force_hide,

        // Business logic
        emits_device_name,
        reset_device_name,
    }
}
