import { nextTick, type Ref, ref, watch } from 'vue'
import type { EditTagStructViewEmits } from '@/pages/views/edit-tag-struct-view-emits'
import type { EditTagStructViewProps } from '@/pages/views/edit-tag-struct-view-props'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { TagStructElementData } from '@/classes/datas/config/tag-struct-element-data'
import type { FolderStructElementData } from '@/classes/datas/config/folder-struct-element-data'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

export function useEditTagStructView(options: {
    props: EditTagStructViewProps,
    emits: EditTagStructViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const foldable_struct = ref<any>(null)
    const edit_tag_struct_element_dialog = ref<any>(null)
    const add_new_folder_dialog = ref<any>(null)
    const add_new_tag_struct_element_dialog = ref<any>(null)
    const tag_struct_context_menu = ref<any>(null)
    const confirm_delete_tag_struct_dialog = ref<any>(null)

    // ── State refs ──
    const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())

    // ── Watchers ──
    watch(() => props.application_config, () => reload_cloned_application_config())

    // ── Business logic ──
    async function reload_cloned_application_config(): Promise<void> {
        cloned_application_config.value = props.application_config.clone()
        await cloned_application_config.value.append_not_found_tags()
    }

    function show_tag_contextmenu(e: MouseEvent, id: string | null): void {
        if (id) {
            tag_struct_context_menu.value?.show(e, id)
        }
    }

    function show_edit_tag_struct_dialog(id: string): void {
        if (!foldable_struct.value) {
            return
        }
        let target_struct_object: TagStructElementData | null = null
        let tag_name_walk = (_tag: TagStructElementData): void => { }
        tag_name_walk = (tag: TagStructElementData): void => {
            const tag_children = tag.children
            if (tag.id === id) {
                target_struct_object = tag
            } else if (tag_children) {
                tag_children.forEach(child_tag => {
                    if (child_tag) {
                        tag_name_walk(child_tag)
                    }
                })
            }
        }
        tag_name_walk(cloned_application_config.value.tag_struct)

        if (!target_struct_object) {
            return
        }

        edit_tag_struct_element_dialog.value?.show(target_struct_object)
    }

    function update_tag_struct(tag_struct_obj: TagStructElementData): void {
        let tag_name_walk = (_tag: TagStructElementData): boolean => false
        tag_name_walk = (tag: TagStructElementData): boolean => {
            const tag_children = tag.children
            if (tag.id === tag_struct_obj.id) {
                return true
            } else if (tag_children) {
                for (let i = 0; i < tag_children.length; i++) {
                    const child_tag = tag_children[i]
                    if (tag_name_walk(child_tag)) {
                        tag_children.splice(i, 1, tag_struct_obj)
                        return false
                    }
                }
            }
            return false
        }
        tag_name_walk(cloned_application_config.value.tag_struct)
    }

    async function apply(): Promise<void> {
        emits('requested_apply_tag_struct', cloned_application_config.value.tag_struct)
        nextTick(() => emits('requested_close_dialog'))
    }

    function show_add_new_tag_struct_element_dialog(): void {
        add_new_tag_struct_element_dialog.value?.show()
    }

    function show_add_new_folder_dialog(): void {
        add_new_folder_dialog.value?.show()
    }

    async function add_folder_struct_element(folder_struct_element: FolderStructElementData): Promise<void> {
        const tag_struct_element = new TagStructElementData()
        tag_struct_element.id = folder_struct_element.id
        tag_struct_element.is_dir = true
        tag_struct_element.check_when_inited = false
        tag_struct_element.tag_name = folder_struct_element.folder_name
        tag_struct_element.children = new Array<TagStructElementData>()
        tag_struct_element.key = folder_struct_element.folder_name
        cloned_application_config.value.tag_struct.children?.push(tag_struct_element)
    }

    async function add_tag_struct_element(tag_struct_element: TagStructElementData): Promise<void> {
        cloned_application_config.value.tag_struct.children?.push(tag_struct_element)
    }

    function show_confirm_delete_tag_struct_dialog(id: string): void {
        let target_struct_object: TagStructElementData | null = null
        let tag_name_walk = (_tag: TagStructElementData): void => { }
        tag_name_walk = (tag: TagStructElementData): void => {
            const tag_children = tag.children
            if (tag.id === id) {
                target_struct_object = tag
            } else if (tag_children) {
                tag_children.forEach(child_tag => {
                    if (child_tag) {
                        tag_name_walk(child_tag)
                    }
                })
            }
        }
        tag_name_walk(cloned_application_config.value.tag_struct)

        if (!target_struct_object) {
            return
        }
        confirm_delete_tag_struct_dialog.value?.show(target_struct_object)
    }

    function delete_tag_struct(id: string): void {
        let tag_name_walk = (_tag: TagStructElementData): boolean => false
        tag_name_walk = (tag: TagStructElementData): boolean => {
            const tag_children = tag.children
            if (tag.id === id) {
                return true
            } else if (tag_children) {
                for (let i = 0; i < tag_children.length; i++) {
                    const child_tag = tag_children[i]
                    if (tag_name_walk(child_tag)) {
                        tag_children.splice(i, 1)
                        return false
                    }
                }
            }
            return false
        }
        tag_name_walk(cloned_application_config.value.tag_struct)
    }

    // ── Template event handlers ──
    function onDblclickedItem(_e: MouseEvent, id: string | null): void {
        if (id) show_edit_tag_struct_dialog(id)
    }

    function onRequestedCloseDialog(): void {
        emits('requested_close_dialog')
    }

    // ── Event relay objects ──
    const errorMessageRelayHandlers = {
        'received_errors': (...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>),
        'received_messages': (...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>),
    }

    // ── Return ──
    return {
        // Template refs
        foldable_struct,
        edit_tag_struct_element_dialog,
        add_new_folder_dialog,
        add_new_tag_struct_element_dialog,
        tag_struct_context_menu,
        confirm_delete_tag_struct_dialog,

        // State
        cloned_application_config,

        // Business logic
        reload_cloned_application_config,
        show_tag_contextmenu,
        show_edit_tag_struct_dialog,
        update_tag_struct,
        apply,
        show_add_new_tag_struct_element_dialog,
        show_add_new_folder_dialog,
        add_folder_struct_element,
        add_tag_struct_element,
        show_confirm_delete_tag_struct_dialog,
        delete_tag_struct,

        // Template event handlers
        onDblclickedItem,
        onRequestedCloseDialog,

        // Event relay objects
        errorMessageRelayHandlers,
    }
}
