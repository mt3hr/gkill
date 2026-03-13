import { type Ref, ref } from 'vue'
import type { EditDeviceStructElementViewEmits } from '@/pages/views/edit-device-struct-element-view-emits'
import type { EditDeviceStructElementViewProps } from '@/pages/views/edit-device-struct-element-view-props'
import { DeviceStructElementData } from '@/classes/datas/config/device-struct-element-data'

export function useEditDeviceStructElementView(options: {
    props: EditDeviceStructElementViewProps,
    emits: EditDeviceStructElementViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const check_when_inited: Ref<boolean> = ref(props.struct_obj.check_when_inited)

    // ── Methods ──
    async function apply(): Promise<void> {
        const device_struct = new DeviceStructElementData()
        device_struct.id = props.struct_obj.id
        device_struct.check_when_inited = check_when_inited.value
        device_struct.children = props.struct_obj.children
        device_struct.indeterminate = false
        device_struct.is_dir = props.struct_obj.is_dir
        device_struct.key = props.struct_obj.device_name
        device_struct.device_name = props.struct_obj.device_name
        device_struct.name = props.struct_obj.device_name
        emits('requested_update_device_struct', device_struct)
        emits('requested_close_dialog')
    }

    // ── Return ──
    return {
        // State
        check_when_inited,

        // Methods
        apply,
    }
}
