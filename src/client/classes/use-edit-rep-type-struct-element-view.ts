import { type Ref, ref } from 'vue'
import type { EditRepTypeStructElementViewEmits } from '@/pages/views/edit-rep-type-struct-element-view-emits'
import type { EditRepTypeStructElementViewProps } from '@/pages/views/edit-rep-type-struct-element-view-props'
import { RepTypeStructElementData } from '@/classes/datas/config/rep-type-struct-element-data'

export function useEditRepTypeStructElementView(options: {
    props: EditRepTypeStructElementViewProps,
    emits: EditRepTypeStructElementViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const check_when_inited: Ref<boolean> = ref(props.struct_obj.check_when_inited)

    // ── Methods ──
    async function apply(): Promise<void> {
        const rep_type_struct = new RepTypeStructElementData()
        rep_type_struct.id = props.struct_obj.id
        rep_type_struct.check_when_inited = check_when_inited.value
        rep_type_struct.children = props.struct_obj.children
        rep_type_struct.indeterminate = false
        rep_type_struct.is_dir = props.struct_obj.is_dir
        rep_type_struct.key = props.struct_obj.rep_type_name
        rep_type_struct.rep_type_name = props.struct_obj.rep_type_name
        rep_type_struct.name = props.struct_obj.rep_type_name

        emits('requested_update_rep_type_struct', rep_type_struct)
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
