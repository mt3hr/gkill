import { ref, type Ref } from 'vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import type { TagStructElementProps } from '@/pages/views/tag-struct-element-props'

export function useTagStructElement(options: {
    props: TagStructElementProps,
}) {
    const { props } = options

    const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())

    async function reload_cloned_application_config(): Promise<void> {
        cloned_application_config.value = props.application_config.clone()
    }

    return {
        cloned_application_config,
        reload_cloned_application_config,
    }
}
