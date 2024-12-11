<template>
    <v-dialog v-model="is_show_dialog">
        <ConfirmDeleteTagView :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="tag_highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag" :tag="tag"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" 
            @requested_close_dialog="hide()"
            />
    </v-dialog>
</template>
<script setup lang="ts">
import { computed, type Ref, ref, watch } from 'vue'
import type { ConfirmDeleteTagDialogProps } from './confirm-delete-tag-dialog-props'
import type { KyouDialogEmits } from '../views/kyou-dialog-emits'
import ConfirmDeleteTagView from '../views/confirm-delete-tag-view.vue'
import type { InfoIdentifier } from '@/classes/datas/info-identifier';

const props = defineProps<ConfirmDeleteTagDialogProps>()
const emits = defineEmits<KyouDialogEmits>()
defineExpose({ show, hide })

const tag_highlight_targets = computed<Array<InfoIdentifier>>(() => {
    const info_identifer = props.tag.generate_info_identifer()
    return [info_identifer]
})

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
