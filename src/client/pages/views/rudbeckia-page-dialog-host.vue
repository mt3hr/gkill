<template>
    <!-- 開いている枚数ぶんウィンドウを並べるだけ。開くのは各ダイアログ自身（onMounted で show()） -->
    <RudbeckiaPageDialog v-for="dialog in opened_dialogs" :key="dialog.id" :kind="dialog.kind"
        :slot_index="dialog.slot_index" :cascade_index="dialog.cascade_index"
        :application_config="application_config" :gkill_api="gkill_api"
        :app_content_height="app_content_height" :app_content_width="app_content_width"
        :application_config_load_failed="application_config_load_failed"
        :kyou_change_bus="kyou_change_bus" :origin_id="dialog.id"
        v-on="dialogRelayHandlers" @closed="close(dialog.id)" ref="dialog_refs" />
</template>

<script lang="ts" setup>
import RudbeckiaPageDialog from '../dialogs/rudbeckia-page-dialog.vue'
import type { RudbeckiaPageDialogHostProps } from './rudbeckia-page-dialog-host-props'
import type { RudbeckiaPageDialogHostEmits } from './rudbeckia-page-dialog-host-emits'
import { useRudbeckiaPageDialogHost } from '@/classes/use-rudbeckia-page-dialog-host'

// v-for なのでルートが複数ある。呼び出し側が渡してくる属性のうち
// このホストが受け取らないものは行き場が無いので、素通しせず黙って捨てる
// （kftl-dialog-host.vue と同じ）
defineOptions({ inheritAttrs: false })

defineProps<RudbeckiaPageDialogHostProps>()
const emits = defineEmits<RudbeckiaPageDialogHostEmits>()

const {
    // State
    opened_dialogs,
    dialog_refs,

    // Business logic
    show,
    close,

    // Event relay objects
    dialogRelayHandlers,
} = useRudbeckiaPageDialogHost({ emits })

// 呼び出し側は page_dialog_host.value?.show('rykv') と呼ぶだけでよい
defineExpose({ show })
</script>
