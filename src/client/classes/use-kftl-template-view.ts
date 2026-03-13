import { ref, type Ref } from 'vue'
import { KFTLTemplateElementData } from '@/classes/datas/kftl-template-element-data'
import type { KFTLTemplateViewProps } from '@/pages/views/kftl-template-view-props'
import type { KFTLTemplateViewEmits } from '@/pages/views/kftl-template-view-emits'

export function useKFTLTemplateView(options: {
    props: KFTLTemplateViewProps,
    emits: KFTLTemplateViewEmits,
}) {
    const { props, emits } = options

    const child_template_dialogs: Ref<Array<any>> = ref(new Array<any>())

    function clicked_template_button(template: KFTLTemplateElementData, index: number): void {
        if (!template.children) {
            emits('clicked_template_element_leaf', template)
            emits('requested_close_dialog')
            return
        }
        child_template_dialogs.value[index].show()
    }

    return {
        child_template_dialogs,
        clicked_template_button,
    }
}
