import { type Ref, ref } from 'vue'
import type { EditKFTLTemplateStructElementViewEmits } from '@/pages/views/edit-kftl-template-struct-element-view-emits'
import type { EditKFTLTemplateStructElementViewProps } from '@/pages/views/edit-kftl-template-struct-element-view-props'
import { KFTLTemplateStructElementData } from '@/classes/datas/config/kftl-template-struct-element-data'

export function useEditKFTLTemplateStructElementView(options: {
    props: EditKFTLTemplateStructElementViewProps,
    emits: EditKFTLTemplateStructElementViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const title: Ref<string> = ref(props.struct_obj.title)
    const template: Ref<string | null> = ref(props.struct_obj.template)

    // ── Methods ──
    async function apply(): Promise<void> {
        const kftl_template_struct = new KFTLTemplateStructElementData()
        kftl_template_struct.id = props.struct_obj.id
        kftl_template_struct.title = title.value
        kftl_template_struct.template = template.value ? template.value : ""
        kftl_template_struct.key = title.value
        kftl_template_struct.name = title.value
        kftl_template_struct.is_dir = props.struct_obj.is_dir
        kftl_template_struct.children = props.struct_obj.children
        emits('requested_update_kftl_template_struct', kftl_template_struct)
        emits('requested_close_dialog')
    }

    // ── Return ──
    return {
        // State
        title,
        template,

        // Methods
        apply,
    }
}
