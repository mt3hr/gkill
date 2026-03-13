import { ref, type Ref } from 'vue'
import { RepStructElementData } from '@/classes/datas/config/rep-struct-element-data'
import type { EditRepStructElementViewProps } from '@/pages/views/edit-rep-struct-element-view-props'
import type { EditRepStructElementViewEmits } from '@/pages/views/edit-rep-struct-element-view-emits'

export function useEditRepStructElementView(options: {
    props: EditRepStructElementViewProps,
    emits: EditRepStructElementViewEmits,
}) {
    const { props, emits } = options

    const check_when_inited: Ref<boolean> = ref(props.struct_obj.check_when_inited)
    const ignore_check_rep_rykv: Ref<boolean> = ref(props.struct_obj.ignore_check_rep_rykv)

    async function apply(): Promise<void> {
        const rep_struct = new RepStructElementData()
        rep_struct.id = props.struct_obj.id
        rep_struct.key = props.struct_obj.rep_name
        rep_struct.rep_name = props.struct_obj.rep_name
        rep_struct.name = props.struct_obj.rep_name
        rep_struct.check_when_inited = check_when_inited.value
        rep_struct.ignore_check_rep_rykv = ignore_check_rep_rykv.value
        rep_struct.children = props.struct_obj.children
        rep_struct.indeterminate = false
        rep_struct.is_dir = props.struct_obj.is_dir
        emits('requested_update_rep_struct', rep_struct)
        emits('requested_close_dialog')
    }

    return {
        check_when_inited,
        ignore_check_rep_rykv,
        apply,
    }
}
