<template>
    <v-row class="pa-0 ma-0">
        <v-col cols="auto" class="pa-0 ma-0">
            <AttachedTag :tag="tag" :application_config="application_config" :gkill_api="gkill_api" :kyou="kyou"
                :highlight_targets="highlight_targets"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" v-on="crudRelayHandlers" />
        </v-col>
        <v-spacer />
        <v-col cols="auto" class="pa-0 ma-0">
            <span class="update_time">
                {{ format_time(tag.update_time) }}
            </span>
        </v-col>
        <v-spacer />
        <v-col cols="auto" class="pa-0 ma-0">
            <span class="update_device">
                {{ tag.update_device }}
            </span>
        </v-col>
    </v-row>
</template>
<script lang="ts" setup>
import type { KyouViewEmits } from './kyou-view-emits'
import type { TagViewProps } from './tag-view-props'
import AttachedTag from './attached-tag.vue';
import { format_time } from '@/classes/format-date-time'
import { useTagView } from '@/classes/use-tag-view'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

const props = defineProps<TagViewProps>()
const emits = defineEmits<KyouViewEmits>()

const crudRelayHandlers = build_kyou_view_relay(emits)

const {
} = useTagView({ props, emits })
</script>
