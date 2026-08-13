<template>
  <v-card class="pa-2" variant="flat">
    <v-text-field v-model="title" :label="i18n.global.t('DNOTE_TITLE_TITLE')" />
    <v-select v-model="granularity" :items="granularities" item-title="label" item-value="value"
      :label="i18n.global.t('DNOTE_TREND_GRANULARITY')" />
    <v-select v-model="method" :items="methods" item-title="label" item-value="value"
      :label="i18n.global.t('DNOTE_CORRELATION_METHOD_TITLE')" />
    <v-text-field v-model.number="lag" type="number" step="1"
      :label="i18n.global.t('DNOTE_CORRELATION_LAG_TITLE')"
      :hint="i18n.global.t('DNOTE_CORRELATION_LAG_HINT')" persistent-hint />

    <v-card v-for="(metric, index) in metrics" :key="metric.id" variant="outlined" class="pa-2 my-2">
      <v-row class="pa-0 ma-0" align="center">
        <v-col class="pa-0 ma-0">
          <v-text-field v-model="metric.title" :label="i18n.global.t('DNOTE_CORRELATION_METRIC_TITLE')" />
        </v-col>
        <v-col cols="auto" class="pa-0 ma-0">
          <v-btn icon="mdi-arrow-up" size="small" variant="text" :disabled="index === 0"
            @click="move_metric(index, -1)" />
          <v-btn icon="mdi-arrow-down" size="small" variant="text" :disabled="index === metrics.length - 1"
            @click="move_metric(index, 1)" />
          <!-- 相関は2指標そろって初めて意味を持つので、2件目以降しか消せない -->
          <v-btn icon="mdi-delete" size="small" variant="text" color="secondary" :disabled="metrics.length <= 2"
            @click="delete_metric(index)" />
        </v-col>
      </v-row>
      <v-select v-model="metric.aggregate_target" :items="aggregate_targets" item-title="label" item-value="value"
        :label="i18n.global.t('DNOTE_AGGREGATE_TARGET')" />
      <PredicateGroup v-model="metric.root_predicate" :is_root="true" />
    </v-card>

    <v-btn color="primary" variant="text" prepend-icon="mdi-plus" :disabled="metrics.length >= 10"
      @click="add_metric">{{ i18n.global.t('DNOTE_CORRELATION_ADD_METRIC_TITLE') }}</v-btn>
    <v-alert v-if="validation_message" type="error" variant="tonal" class="my-2">{{ validation_message }}</v-alert>
    <v-row class="pa-0 ma-0 flex-row-reverse gkill-dialog-actions">
      <v-col cols="auto" class="pa-0 ma-0">
        <v-btn color="primary" @click="save">{{ i18n.global.t('SAVE_TITLE') }}</v-btn>
      </v-col>
      <v-spacer />
      <v-col cols="auto" class="pa-0 ma-0">
        <v-btn color="secondary" @click="load_query">{{ i18n.global.t('RESET_TITLE') }}</v-btn>
      </v-col>
    </v-row>
  </v-card>
</template>

<script setup lang="ts">
import { toRef } from "vue"
import { i18n } from "@/i18n"
import type DnoteCorrelationGraphEditorViewEmits from "./dnote-correlation-graph-editor-view-emits"
import type DnoteCorrelationGraphEditorViewProps from "./dnote-correlation-graph-editor-view-props"
import PredicateGroup from "./edit-dnote-predicate-group.vue"
import { useDnoteCorrelationGraphEditorView } from "@/classes/use-dnote-correlation-graph-editor-view"

const props = defineProps<DnoteCorrelationGraphEditorViewProps>()
const emits = defineEmits<DnoteCorrelationGraphEditorViewEmits>()

const {
  // State
  title,
  granularity,
  method,
  lag,
  metrics,
  aggregate_targets,
  granularities,
  methods,
  validation_message,

  // Business logic
  load_query,
  add_metric,
  delete_metric,
  move_metric,
  save,
} = useDnoteCorrelationGraphEditorView({ props, emits, initial_query: toRef(props, "initial_query") })
</script>
