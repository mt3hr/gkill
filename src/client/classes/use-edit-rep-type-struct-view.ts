import { nextTick, type Ref, ref, watch } from 'vue'
import type { EditRepTypeStructViewEmits } from '@/pages/views/edit-rep-type-struct-view-emits'
import type { EditRepTypeStructViewProps } from '@/pages/views/edit-rep-type-struct-view-props'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { RepTypeStructElementData } from '@/classes/datas/config/rep-type-struct-element-data'
import type { FolderStructElementData } from '@/classes/datas/config/folder-struct-element-data'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { ComponentRef } from '@/classes/component-ref'

export function useEditRepTypeStructView(options: {
    props: EditRepTypeStructViewProps,
    emits: EditRepTypeStructViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const foldable_struct = ref<ComponentRef | null>(null)
    const edit_rep_type_struct_element_dialog = ref<ComponentRef | null>(null)
    const add_new_folder_dialog = ref<ComponentRef | null>(null)
    const add_new_rep_type_struct_element_dialog = ref<ComponentRef | null>(null)
    const rep_type_struct_context_menu = ref<ComponentRef | null>(null)
    const confirm_delete_rep_type_struct_dialog = ref<ComponentRef | null>(null)

    // ── State refs ──
    const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())

    // ── Watchers ──
    watch(() => props.application_config, () => reload_cloned_application_config())

    // ── Business logic ──
    async function reload_cloned_application_config(): Promise<void> {
        cloned_application_config.value = props.application_config.clone()
        await cloned_application_config.value.append_not_found_rep_types()
    }

    function show_rep_type_contextmenu(e: MouseEvent, id: string | null): void {
        if (id) {
            rep_type_struct_context_menu.value?.show(e, id)
        }
    }

    function show_edit_rep_type_struct_dialog(id: string): void {
        if (!foldable_struct.value) {
            return
        }
        let target_struct_object: RepTypeStructElementData | null = null
        let rep_type_name_walk = (_rep_type: RepTypeStructElementData): void => { }
        rep_type_name_walk = (rep_type: RepTypeStructElementData): void => {
            const rep_type_children = rep_type.children
            if (rep_type.id === id) {
                target_struct_object = rep_type
            } else if (rep_type_children) {
                rep_type_children.forEach(child_rep_type => {
                    if (child_rep_type) {
                        rep_type_name_walk(child_rep_type)
                    }
                })
            }
        }
        rep_type_name_walk(cloned_application_config.value.rep_type_struct)

        if (!target_struct_object) {
            return
        }

        edit_rep_type_struct_element_dialog.value?.show(target_struct_object)
    }

    function update_rep_type_struct(rep_type_struct_obj: RepTypeStructElementData): void {
        let rep_type_name_walk = (_rep_type: RepTypeStructElementData): boolean => false
        rep_type_name_walk = (rep_type: RepTypeStructElementData): boolean => {
            const rep_type_children = rep_type.children
            if (rep_type.id === rep_type_struct_obj.id) {
                return true
            } else if (rep_type_children) {
                for (let i = 0; i < rep_type_children.length; i++) {
                    const child_rep_type = rep_type_children[i]
                    if (rep_type_name_walk(child_rep_type)) {
                        rep_type_children.splice(i, 1, rep_type_struct_obj)
                        return false
                    }
                }
            }
            return false
        }
        rep_type_name_walk(cloned_application_config.value.rep_type_struct)
    }

    async function apply(): Promise<void> {
        emits('requested_apply_rep_type_struct', cloned_application_config.value.rep_type_struct)
        nextTick(() => emits('requested_close_dialog'))
    }

    function show_add_new_rep_type_struct_element_dialog(): void {
        add_new_rep_type_struct_element_dialog.value?.show()
    }

    function show_add_new_folder_dialog(): void {
        add_new_folder_dialog.value?.show()
    }

    async function add_folder_struct_element(folder_struct_element: FolderStructElementData): Promise<void> {
        const rep_type_struct_element = new RepTypeStructElementData()
        rep_type_struct_element.id = folder_struct_element.id
        rep_type_struct_element.is_dir = true
        rep_type_struct_element.check_when_inited = false
        rep_type_struct_element.rep_type_name = folder_struct_element.folder_name
        rep_type_struct_element.children = new Array<RepTypeStructElementData>()
        rep_type_struct_element.key = folder_struct_element.folder_name
        cloned_application_config.value.rep_type_struct.children?.push(rep_type_struct_element)
    }

    async function add_rep_type_struct_element(rep_type_struct_element: RepTypeStructElementData): Promise<void> {
        cloned_application_config.value.rep_type_struct.children?.push(rep_type_struct_element)
    }

    function show_confirm_delete_rep_type_struct_dialog(id: string): void {
        let target_struct_object: RepTypeStructElementData | null = null
        let rep_type_name_walk = (_rep_type: RepTypeStructElementData): void => { }
        rep_type_name_walk = (rep_type: RepTypeStructElementData): void => {
            const rep_type_children = rep_type.children
            if (rep_type.id === id) {
                target_struct_object = rep_type
            } else if (rep_type_children) {
                rep_type_children.forEach(child_rep_type => {
                    if (child_rep_type) {
                        rep_type_name_walk(child_rep_type)
                    }
                })
            }
        }
        rep_type_name_walk(cloned_application_config.value.rep_type_struct)

        if (!target_struct_object) {
            return
        }
        confirm_delete_rep_type_struct_dialog.value?.show(target_struct_object)
    }

    function delete_rep_type_struct(id: string): void {
        let rep_type_name_walk = (_rep_type: RepTypeStructElementData): boolean => false
        rep_type_name_walk = (rep_type: RepTypeStructElementData): boolean => {
            const rep_type_children = rep_type.children
            if (rep_type.id === id) {
                return true
            } else if (rep_type_children) {
                for (let i = 0; i < rep_type_children.length; i++) {
                    const child_rep_type = rep_type_children[i]
                    if (rep_type_name_walk(child_rep_type)) {
                        rep_type_children.splice(i, 1)
                        return false
                    }
                }
            }
            return false
        }
        rep_type_name_walk(cloned_application_config.value.rep_type_struct)
    }

    // ── Template event handlers ──
    function onDblclickedItem(_e: MouseEvent, id: string | null): void {
        if (id) show_edit_rep_type_struct_dialog(id)
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
        edit_rep_type_struct_element_dialog,
        add_new_folder_dialog,
        add_new_rep_type_struct_element_dialog,
        rep_type_struct_context_menu,
        confirm_delete_rep_type_struct_dialog,

        // State
        cloned_application_config,

        // Business logic
        reload_cloned_application_config,
        show_rep_type_contextmenu,
        show_edit_rep_type_struct_dialog,
        update_rep_type_struct,
        apply,
        show_add_new_rep_type_struct_element_dialog,
        show_add_new_folder_dialog,
        add_folder_struct_element,
        add_rep_type_struct_element,
        show_confirm_delete_rep_type_struct_dialog,
        delete_rep_type_struct,

        // Template event handlers
        onDblclickedItem,
        onRequestedCloseDialog,

        // Event relay objects
        errorMessageRelayHandlers,
    }
}
