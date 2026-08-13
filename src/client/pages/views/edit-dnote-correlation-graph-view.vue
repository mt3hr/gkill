<template>
    <!-- 入力フォームは追加と共通。add-dnote-correlation-graph-view.vue のコメントを参照 -->
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
import type EditDnoteCorrelationGraphViewEmits from './edit-dnote-correlation-graph-view-emits'
import type EditDnoteCorrelationGraphViewProps from './edit-dnote-correlation-graph-view-props'
import { useEditDnoteCorrelationGraphView } from '@/classes/use-edit-dnote-correlation-graph-view'

const props = defineProps<EditDnoteCorrelationGraphViewProps>()
const emits = defineEmits<EditDnoteCorrelationGraphViewEmits>()

const {
    // Computed
    initial_query,

    // Business logic
    save,
} = useEditDnoteCorrelationGraphView({ props, emits })
</script>
