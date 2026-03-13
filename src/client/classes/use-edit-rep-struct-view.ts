import { i18n } from '@/i18n'
import { nextTick, type Ref, ref, watch } from 'vue'
import type { EditRepStructViewEmits } from '@/pages/views/edit-rep-struct-view-emits'
import type { EditRepStructViewProps } from '@/pages/views/edit-rep-struct-view-props'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { RepStructElementData } from '@/classes/datas/config/rep-struct-element-data'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

export function useEditRepStructView(options: {
    props: EditRepStructViewProps,
    emits: EditRepStructViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const foldable_struct = ref<any>(null)
    const edit_rep_struct_element_dialog = ref<any>(null)
    const add_new_rep_struct_element_dialog = ref<any>(null)
    const rep_struct_context_menu = ref<any>(null)
    const confirm_delete_rep_struct_dialog = ref<any>(null)

    // ── State refs ──
    const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())

    // ── Watchers ──
    watch(() => props.application_config, () => reload_cloned_application_config())

    // ── Business logic ──
    async function reload_cloned_application_config(): Promise<void> {
        cloned_application_config.value = props.application_config.clone()
        await cloned_application_config.value.append_not_found_reps()
    }

    function show_rep_contextmenu(e: MouseEvent, id: string | null): void {
        if (id) {
            rep_struct_context_menu.value?.show(e, id)
        }
    }

    function show_edit_rep_struct_dialog(id: string): void {
        if (!foldable_struct.value) {
            return
        }
        let target_struct_object: RepStructElementData | null = null
        let rep_name_walk = (_rep: RepStructElementData): void => { }
        rep_name_walk = (rep: RepStructElementData): void => {
            const rep_children = rep.children
            if (rep.id === id) {
                target_struct_object = rep
            } else if (rep_children) {
                rep_children.forEach(child_rep => {
                    if (child_rep) {
                        rep_name_walk(child_rep)
                    }
                })
            }
        }
        rep_name_walk(cloned_application_config.value.rep_struct)

        if (!target_struct_object) {
            return
        }

        edit_rep_struct_element_dialog.value?.show(target_struct_object)
    }

    function update_rep_struct(rep_struct_obj: RepStructElementData): void {
        let rep_name_walk = (_rep: RepStructElementData): boolean => false
        rep_name_walk = (rep: RepStructElementData): boolean => {
            const rep_children = rep.children
            if (rep.id === rep_struct_obj.id) {
                return true
            } else if (rep_children) {
                for (let i = 0; i < rep_children.length; i++) {
                    const child_rep = rep_children[i]
                    if (rep_name_walk(child_rep)) {
                        rep_children.splice(i, 1, rep_struct_obj)
                        return false
                    }
                }
            }
            return false
        }
        rep_name_walk(cloned_application_config.value.rep_struct)
    }

    async function apply(): Promise<void> {
        emits('requested_apply_rep_struct', cloned_application_config.value.rep_struct)
        nextTick(() => emits('requested_close_dialog'))
    }

    function show_add_new_rep_struct_element_dialog(): void {
        add_new_rep_struct_element_dialog.value?.show()
    }

    async function add_rep_struct_element(rep_struct_element: RepStructElementData): Promise<void> {
        cloned_application_config.value.rep_struct.children?.push(rep_struct_element)
    }

    function show_confirm_delete_rep_struct_dialog(id: string): void {
        let target_struct_object: RepStructElementData | null = null
        let rep_name_walk = (_rep: RepStructElementData): void => { }
        rep_name_walk = (rep: RepStructElementData): void => {
            const rep_children = rep.children
            if (rep.id === id) {
                target_struct_object = rep
            } else if (rep_children) {
                rep_children.forEach(child_rep => {
                    if (child_rep) {
                        rep_name_walk(child_rep)
                    }
                })
            }
        }
        rep_name_walk(cloned_application_config.value.rep_struct)

        if (!target_struct_object) {
            return
        }
        confirm_delete_rep_struct_dialog.value?.show(target_struct_object)
    }

    function delete_rep_struct(id: string): void {
        let rep_name_walk = (_rep: RepStructElementData): boolean => false
        rep_name_walk = (rep: RepStructElementData): boolean => {
            const rep_children = rep.children
            if (rep.id === id) {
                return true
            } else if (rep_children) {
                for (let i = 0; i < rep_children.length; i++) {
                    const child_rep = rep_children[i]
                    if (rep_name_walk(child_rep)) {
                        rep_children.splice(i, 1)
                        return false
                    }
                }
            }
            return false
        }
        rep_name_walk(cloned_application_config.value.rep_struct)
    }

    // ── Template event handlers ──
    function onDblclickedItem(e: MouseEvent, id: string | null): void {
        if (id) show_edit_rep_struct_dialog(id)
    }

    function onRequestedCloseDialog(): void {
        emits('requested_close_dialog')
    }

    // ── Event relay objects ──
    const errorMessageHandlers = {
        'received_errors': (...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>),
        'received_messages': (...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>),
    }

    const addNewRepHandlers = {
        ...errorMessageHandlers,
        'requested_add_rep_struct_element': (...args: any[]) => add_rep_struct_element(args[0] as RepStructElementData),
    }

    const editRepHandlers = {
        ...errorMessageHandlers,
        'requested_update_rep_struct': (...args: any[]) => update_rep_struct(args[0] as RepStructElementData),
    }

    const repContextMenuHandlers = {
        ...errorMessageHandlers,
        'requested_edit_rep': (...id: any[]) => show_edit_rep_struct_dialog(id[0] as string),
        'requested_delete_rep': (...id: any[]) => show_confirm_delete_rep_struct_dialog(id[0] as string),
    }

    const confirmDeleteHandlers = {
        ...errorMessageHandlers,
        'requested_delete_rep': (...id: any[]) => delete_rep_struct(id[0] as string),
    }

    return {
        // Template refs
        foldable_struct,
        edit_rep_struct_element_dialog,
        add_new_rep_struct_element_dialog,
        rep_struct_context_menu,
        confirm_delete_rep_struct_dialog,

        // State
        cloned_application_config,

        // Business logic
        reload_cloned_application_config,
        show_rep_contextmenu,
        show_edit_rep_struct_dialog,
        update_rep_struct,
        apply,
        show_add_new_rep_struct_element_dialog,
        add_rep_struct_element,
        show_confirm_delete_rep_struct_dialog,
        delete_rep_struct,

        // Template event handlers
        onDblclickedItem,
        onRequestedCloseDialog,

        // Event relay objects
        errorMessageHandlers,
        addNewRepHandlers,
        editRepHandlers,
        repContextMenuHandlers,
        confirmDeleteHandlers,
    }
}
