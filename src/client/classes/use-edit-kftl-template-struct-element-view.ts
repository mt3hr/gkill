import { type Ref, ref } from 'vue'
import type { EditKFTLTemplateStructElementViewEmits } from '@/pages/views/edit-kftl-template-struct-element-view-emits'
import type { EditKFTLTemplateStructElementViewProps } from '@/pages/views/edit-kftl-template-struct-element-view-props'
import { KFTLTemplateElementData } from '@/classes/datas/kftl-template-element-data'

export function useEditKFTLTemplateStructElementView(options: {
    props: EditKFTLTemplateStructElementViewProps,
    emits: EditKFTLTemplateStructElementViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const title: Ref<string> = ref(props.struct_obj.title)
    const template: Ref<string | null> = ref(props.struct_obj.template)
    // 説明（運用メモ）。古い保存データには欄が無いので undefined を空文字に倒す
    const description: Ref<string> = ref(props.struct_obj.description ?? "")

    // ── Methods ──
    async function apply(): Promise<void> {
        const kftl_template_struct = new KFTLTemplateElementData()
        kftl_template_struct.id = props.struct_obj.id
        kftl_template_struct.title = title.value
        kftl_template_struct.template = template.value ? template.value : ""
        kftl_template_struct.description = description.value
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
        description,

        // Methods
        apply,
    }
}
