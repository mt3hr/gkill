<template>
    <!-- メモ帳ウィンドウは複数枚開ける。開いている枚数ぶんダイアログを並べるだけで、
         開くのは各ダイアログ自身（onMounted で show()） -->
    <KFTLDialog v-for="dialog in opened_dialogs" :key="dialog.id" :slot_index="dialog.slot_index"
        :application_config="application_config" :gkill_api="gkill_api"
        :app_content_height="app_content_height" :app_content_width="app_content_width"
        v-on="dialogRelayHandlers" @closed="close(dialog.id)" ref="dialog_refs" />
</template>

<script lang="ts" setup>
import KFTLDialog from '../dialogs/kftl-dialog.vue'
import type { KFTLDialogHostProps } from './kftl-dialog-host-props'
import type { KFTLDialogHostEmits } from './kftl-dialog-host-emits'
import { useKftlDialogHost } from '@/classes/use-kftl-dialog-host'

// v-for なのでルートが複数ある。呼び出し側は Kyou 系の中継束（crudRelayHandlers）を
// まるごと渡してくるが、メモ帳ダイアログが出すのは KFTLDialogEmits の6つだけ。
// 残りは行き場が無いので、素通しせず黙って捨てる（従来と同じ挙動）
defineOptions({ inheritAttrs: false })

defineProps<KFTLDialogHostProps>()
const emits = defineEmits<KFTLDialogHostEmits>()

const {
    // State
    opened_dialogs,
    dialog_refs,

    // Business logic
    show,
    close,

    // Event relay objects
    dialogRelayHandlers,
} = useKftlDialogHost({ emits })

// 呼び出し側（rykv / mi / dashboard / saihate / plaing）は
// これまでどおり `kftl_dialog.value?.show()` を呼ぶだけでよい
defineExpose({ show })
</script>
