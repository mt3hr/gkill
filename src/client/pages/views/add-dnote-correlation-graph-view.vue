<template>
    <!-- 入力フォームは編集と共通。追加と編集で違うのは「保存したあとどのイベントを出すか」だけなので、
         指標の増減を含む重いフォームは dnote-correlation-graph-editor-view.vue に1つだけ置く -->
    <DnoteCorrelationGraphEditorView :application_config="application_config" :gkill_api="gkill_api"
        :initial_query="initial_query"
        @saved="(query: DnoteCorrelationGraphQuery) => save(query)"
        @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
        @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)" />
</template>

<script setup lang="ts">
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { DnoteCorrelationGraphQuery } from '@/classes/dnote/dnote-correlation'
import DnoteCorrelationGraphEditorView from './dnote-correlation-graph-editor-view.vue'
import type AddDnoteCorrelationGraphViewEmits from './add-dnote-correlation-graph-view-emits'
import type AddDnoteCorrelationGraphViewProps from './add-dnote-correlation-graph-view-props'
import { useAddDnoteCorrelationGraphView } from '@/classes/use-add-dnote-correlation-graph-view'

const props = defineProps<AddDnoteCorrelationGraphViewProps>()
const emits = defineEmits<AddDnoteCorrelationGraphViewEmits>()

const {
    // State
    initial_query,

    // Business logic
    reset,
    save,
} = useAddDnoteCorrelationGraphView({ props, emits })

defineExpose({ reset })
</script>
