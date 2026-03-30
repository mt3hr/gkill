import { nextTick, type Ref, ref, watch } from 'vue'
import type { EditKFTLTemplateStructViewEmits } from '@/pages/views/edit-kftl-template-struct-view-emits'
import type { EditKFTLTemplateStructViewProps } from '@/pages/views/edit-kftl-template-struct-view-props'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { KFTLTemplateStructElementData } from '@/classes/datas/config/kftl-template-struct-element-data'
import type { FolderStructElementData } from '@/classes/datas/config/folder-struct-element-data'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { ComponentRef } from '@/classes/component-ref'

export function useEditKftlTemplateStructView(options: {
    props: EditKFTLTemplateStructViewProps,
    emits: EditKFTLTemplateStructViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const foldable_struct = ref<ComponentRef | null>(null)
    const edit_kftl_template_struct_element_dialog = ref<ComponentRef | null>(null)
    const add_new_folder_dialog = ref<ComponentRef | null>(null)
    const add_new_kftl_template_struct_element_dialog = ref<ComponentRef | null>(null)
    const kftl_template_struct_context_menu = ref<ComponentRef | null>(null)
    const confirm_delete_kftl_template_struct_dialog = ref<ComponentRef | null>(null)

    // ── State refs ──
    const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())

    // ── Watchers ──
    watch(() => props.application_config, () => reload_cloned_application_config())

    // ── Business logic ──
    async function reload_cloned_application_config(): Promise<void> {
        cloned_application_config.value = props.application_config.clone()
    }

    function show_kftl_template_contextmenu(e: MouseEvent, id: string | null): void {
        if (id) {
            kftl_template_struct_context_menu.value?.show(e, id)
        }
    }

    function show_edit_kftl_template_struct_dialog(id: string): void {
        if (!foldable_struct.value) {
            return
        }
        let target_struct_object: KFTLTemplateStructElementData | null = null
        let kftl_template_walk = (_kftl_template: KFTLTemplateStructElementData): void => { }
        kftl_template_walk = (kftl_template: KFTLTemplateStructElementData): void => {
            const kftl_template_children = kftl_template.children
            if (kftl_template.id === id) {
                target_struct_object = kftl_template
            } else if (kftl_template_children) {
                kftl_template_children.forEach(child_kftl_template => {
                    if (child_kftl_template) {
                        kftl_template_walk(child_kftl_template)
                    }
                })
            }
        }
        kftl_template_walk(cloned_application_config.value.kftl_template_struct)

        if (!target_struct_object) {
            return
        }

        edit_kftl_template_struct_element_dialog.value?.show(target_struct_object)
    }

    function update_kftl_template_struct(kftl_template_struct_obj: KFTLTemplateStructElementData): void {
        let kftl_template_walk = (_kftl_template: KFTLTemplateStructElementData): boolean => false
        kftl_template_walk = (kftl_template: KFTLTemplateStructElementData): boolean => {
            const kftl_template_children = kftl_template.children
            if (kftl_template.id === kftl_template_struct_obj.id) {
                return true
            } else if (kftl_template_children) {
                for (let i = 0; i < kftl_template_children.length; i++) {
                    const child_kftl_template = kftl_template_children[i]
                    if (kftl_template_walk(child_kftl_template)) {
                        kftl_template_children.splice(i, 1, kftl_template_struct_obj)
                        return false
                    }
                }
            }
            return false
        }
        kftl_template_walk(cloned_application_config.value.kftl_template_struct)
    }

    async function apply(): Promise<void> {
        emits('requested_apply_kftl_template_struct', cloned_application_config.value.kftl_template_struct)
        nextTick(() => emits('requested_close_dialog'))
    }

    function show_add_new_kftl_template_struct_element_dialog_fn(): void {
        add_new_kftl_template_struct_element_dialog.value?.show()
    }

    function show_add_new_folder_dialog_fn(): void {
        add_new_folder_dialog.value?.show()
    }

    async function add_folder_struct_element(folder_struct_element: FolderStructElementData): Promise<void> {
        const kftl_template_struct_element = new KFTLTemplateStructElementData()
        kftl_template_struct_element.id = folder_struct_element.id
        kftl_template_struct_element.is_dir = true
        kftl_template_struct_element.title = folder_struct_element.folder_name
        kftl_template_struct_element.children = new Array<KFTLTemplateStructElementData>()
        kftl_template_struct_element.key = folder_struct_element.folder_name
        kftl_template_struct_element.name = folder_struct_element.folder_name
        cloned_application_config.value.kftl_template_struct.children?.push(kftl_template_struct_element)
    }

    async function add_kftl_template_struct_element(kftl_template_struct_element: KFTLTemplateStructElementData): Promise<void> {
        cloned_application_config.value.kftl_template_struct.children?.push(kftl_template_struct_element)
    }

    function show_confirm_delete_kftl_template_struct_dialog(id: string): void {
        let target_struct_object: KFTLTemplateStructElementData | null = null
        let kftl_template_walk = (_kftl_template_struct: KFTLTemplateStructElementData): void => { }
        kftl_template_walk = (kftl_template_struct: KFTLTemplateStructElementData): void => {
            const kftl_template_children = kftl_template_struct.children
            if (kftl_template_struct.id === id) {
                target_struct_object = kftl_template_struct
            } else if (kftl_template_children) {
                kftl_template_children.forEach(child_kftl_template => {
                    if (child_kftl_template) {
                        kftl_template_walk(child_kftl_template)
                    }
                })
            }
        }
        kftl_template_walk(cloned_application_config.value.kftl_template_struct)

        if (!target_struct_object) {
            return
        }
        confirm_delete_kftl_template_struct_dialog.value?.show(target_struct_object)
    }

    function delete_kftl_template_struct(id: string): void {
        let kftl_template_walk = (_kftl_template_struct: KFTLTemplateStructElementData): boolean => false
        kftl_template_walk = (kftl_template_struct: KFTLTemplateStructElementData): boolean => {
            const kftl_template_children = kftl_template_struct.children
            if (kftl_template_struct.id === id) {
                return true
            } else if (kftl_template_children) {
                for (let i = 0; i < kftl_template_children.length; i++) {
                    const child_kftl_template = kftl_template_children[i]
                    if (kftl_template_walk(child_kftl_template)) {
                        kftl_template_children.splice(i, 1)
                        return false
                    }
                }
            }
            return false
        }
        kftl_template_walk(cloned_application_config.value.kftl_template_struct)
    }

    // ── Template event handlers ──
    function onDblclickedItem(_e: MouseEvent, id: string | null): void {
        if (id) show_edit_kftl_template_struct_dialog(id)
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
        edit_kftl_template_struct_element_dialog,
        add_new_folder_dialog,
        add_new_kftl_template_struct_element_dialog,
        kftl_template_struct_context_menu,
        confirm_delete_kftl_template_struct_dialog,

        // State
        cloned_application_config,

        // Business logic
        reload_cloned_application_config,
        show_kftl_template_contextmenu,
        show_edit_kftl_template_struct_dialog,
        update_kftl_template_struct,
        apply,
        show_add_new_kftl_template_struct_element_dialog: show_add_new_kftl_template_struct_element_dialog_fn,
        show_add_new_folder_dialog: show_add_new_folder_dialog_fn,
        add_folder_struct_element,
        add_kftl_template_struct_element,
        show_confirm_delete_kftl_template_struct_dialog,
        delete_kftl_template_struct,

        // Template event handlers
        onDblclickedItem,
        onRequestedCloseDialog,

        // Event relay objects
        errorMessageRelayHandlers,
    }
}
