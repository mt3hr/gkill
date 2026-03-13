import { ref, type Ref } from 'vue'
import { TagStructElementData } from '@/classes/datas/config/tag-struct-element-data'
import type { EditTagStructElementViewProps } from '@/pages/views/edit-tag-struct-element-view-props'
import type { EditTagStructElementViewEmits } from '@/pages/views/edit-tag-struct-element-view-emits'

export function useEditTagStructElementView(options: {
    props: EditTagStructElementViewProps,
    emits: EditTagStructElementViewEmits,
}) {
    const { props, emits } = options

    const check_when_inited: Ref<boolean> = ref(props.struct_obj.check_when_inited)
    const is_force_hide: Ref<boolean> = ref(props.struct_obj.is_force_hide)

    async function apply(): Promise<void> {
        const tag_struct = new TagStructElementData()
        tag_struct.id = props.struct_obj.id
        tag_struct.check_when_inited = check_when_inited.value
        tag_struct.is_force_hide = is_force_hide.value
        tag_struct.children = props.struct_obj.children
        tag_struct.indeterminate = false
        tag_struct.is_dir = props.struct_obj.is_dir
        tag_struct.key = props.struct_obj.tag_name
        tag_struct.tag_name = props.struct_obj.tag_name
        tag_struct.name = props.struct_obj.tag_name

        emits('requested_update_tag_struct', tag_struct)
        emits('requested_close_dialog')
    }

    return {
        check_when_inited,
        is_force_hide,
        apply,
    }
}
