import { nextTick, type Ref, ref, watch } from 'vue'
import type { EditDeviceStructViewEmits } from '@/pages/views/edit-device-struct-view-emits'
import type { EditDeviceStructViewProps } from '@/pages/views/edit-device-struct-view-props'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { DeviceStructElementData } from '@/classes/datas/config/device-struct-element-data'
import type { FolderStructElementData } from '@/classes/datas/config/folder-struct-element-data'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { ComponentRef } from '@/classes/component-ref'

export function useEditDeviceStructView(options: {
    props: EditDeviceStructViewProps,
    emits: EditDeviceStructViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const foldable_struct = ref<ComponentRef | null>(null)
    const edit_device_struct_element_dialog = ref<ComponentRef | null>(null)
    const add_new_folder_dialog = ref<ComponentRef | null>(null)
    const add_new_device_struct_element_dialog = ref<ComponentRef | null>(null)
    const device_struct_context_menu = ref<ComponentRef | null>(null)
    const confirm_delete_device_struct_dialog = ref<ComponentRef | null>(null)

    // ── State refs ──
    const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())

    // ── Watchers ──
    watch(() => props.application_config, () => reload_cloned_application_config())

    // ── Business logic ──
    async function reload_cloned_application_config(): Promise<void> {
        cloned_application_config.value = props.application_config.clone()
    }

    function show_device_contextmenu(e: MouseEvent, id: string | null): void {
        if (id) {
            device_struct_context_menu.value?.show(e, id)
        }
    }

    function show_edit_device_struct_dialog(id: string): void {
        if (!foldable_struct.value) {
            return
        }
        let target_struct_object: DeviceStructElementData | null = null
        let device_name_walk = (_device: DeviceStructElementData): void => { }
        device_name_walk = (device: DeviceStructElementData): void => {
            const device_children = device.children
            if (device.id === id) {
                target_struct_object = device
            } else if (device_children) {
                device_children.forEach(child_device => {
                    if (child_device) {
                        device_name_walk(child_device)
                    }
                })
            }
        }
        device_name_walk(cloned_application_config.value.device_struct)

        if (!target_struct_object) {
            return
        }

        edit_device_struct_element_dialog.value?.show(target_struct_object)
    }

    function update_device_struct(device_struct_obj: DeviceStructElementData): void {
        let device_name_walk = (_device: DeviceStructElementData): boolean => false
        device_name_walk = (device: DeviceStructElementData): boolean => {
            const device_children = device.children
            if (device.id === device_struct_obj.id) {
                return true
            } else if (device_children) {
                for (let i = 0; i < device_children.length; i++) {
                    const child_device = device_children[i]
                    if (device_name_walk(child_device)) {
                        device_children.splice(i, 1, device_struct_obj)
                        return false
                    }
                }
            }
            return false
        }
        device_name_walk(cloned_application_config.value.device_struct)
    }

    async function apply(): Promise<void> {
        emits('requested_apply_device_struct', cloned_application_config.value.device_struct)
        nextTick(() => emits('requested_close_dialog'))
    }

    function show_add_new_device_struct_element_dialog(): void {
        add_new_device_struct_element_dialog.value?.show()
    }

    function show_add_new_folder_dialog(): void {
        add_new_folder_dialog.value?.show()
    }

    async function add_folder_struct_element(folder_struct_element: FolderStructElementData): Promise<void> {
        const device_struct_element = new DeviceStructElementData()
        device_struct_element.id = folder_struct_element.id
        device_struct_element.is_dir = true
        device_struct_element.check_when_inited = false
        device_struct_element.device_name = folder_struct_element.folder_name
        device_struct_element.children = new Array<DeviceStructElementData>()
        device_struct_element.key = folder_struct_element.folder_name
        cloned_application_config.value.device_struct.children?.push(device_struct_element)
    }

    async function add_device_struct_element(device_struct_element: DeviceStructElementData): Promise<void> {
        cloned_application_config.value.device_struct.children?.push(device_struct_element)
    }

    function show_confirm_delete_device_struct_dialog(id: string): void {
        let target_struct_object: DeviceStructElementData | null = null
        let device_name_walk = (_device: DeviceStructElementData): void => { }
        device_name_walk = (device: DeviceStructElementData): void => {
            const device_children = device.children
            if (device.id === id) {
                target_struct_object = device
            } else if (device_children) {
                device_children.forEach(child_device => {
                    if (child_device) {
                        device_name_walk(child_device)
                    }
                })
            }
        }
        device_name_walk(cloned_application_config.value.device_struct)

        if (!target_struct_object) {
            return
        }
        confirm_delete_device_struct_dialog.value?.show(target_struct_object)
    }

    function delete_device_struct(id: string): void {
        let device_name_walk = (_device: DeviceStructElementData): boolean => false
        device_name_walk = (device: DeviceStructElementData): boolean => {
            const device_children = device.children
            if (device.id === id) {
                return true
            } else if (device_children) {
                for (let i = 0; i < device_children.length; i++) {
                    const child_device = device_children[i]
                    if (device_name_walk(child_device)) {
                        device_children.splice(i, 1)
                        return false
                    }
                }
            }
            return false
        }
        device_name_walk(cloned_application_config.value.device_struct)
    }

    // ── Template event handlers ──
    function onDblclickedItem(_e: MouseEvent, id: string | null): void {
        if (id) show_edit_device_struct_dialog(id)
    }

    function onRequestedCloseDialog(): void {
        emits('requested_close_dialog')
    }

    // ── Event relay objects ──
    const errorMessageRelayHandlers = {
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
    }

    // ── Return ──
    return {
        // Template refs
        foldable_struct,
        edit_device_struct_element_dialog,
        add_new_folder_dialog,
        add_new_device_struct_element_dialog,
        device_struct_context_menu,
        confirm_delete_device_struct_dialog,

        // State
        cloned_application_config,

        // Business logic
        reload_cloned_application_config,
        show_device_contextmenu,
        show_edit_device_struct_dialog,
        update_device_struct,
        apply,
        show_add_new_device_struct_element_dialog,
        show_add_new_folder_dialog,
        add_folder_struct_element,
        add_device_struct_element,
        show_confirm_delete_device_struct_dialog,
        delete_device_struct,

        // Template event handlers
        onDblclickedItem,
        onRequestedCloseDialog,

        // Event relay objects
        errorMessageRelayHandlers,
    }
}
