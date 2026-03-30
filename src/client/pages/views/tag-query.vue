<template>
    <div>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-checkbox readonly v-model="use_tag" :label="i18n.global.t('TAG_QUERY_TITLE')" hide-details />
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pb-0 mb-0 pr-0 pt-2">
                <v-btn dark color="secondary" @click="emits('request_clear_tag_query')" hide-details>{{
                    i18n.global.t("CLEAR_TITLE") }}</v-btn>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn v-if="is_and_search" icon="mdi-set-center"
                    @click="is_and_search = !is_and_search; emits('request_update_and_search_tags', is_and_search)" />
                <v-btn v-if="!is_and_search" icon="mdi-set-all"
                    @click="is_and_search = !is_and_search; emits('request_update_and_search_tags', is_and_search)" />
            </v-col>
        </v-row>
        <table v-show="use_tag" class="taglist">
            <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
                :is_open="true" :struct_obj="tag_struct" :is_editable="false" :is_root="true" :is_show_checkbox="true"
                @clicked_items="clicked_items" @requested_update_check_state="update_check_state"
                @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
                @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
                ref="foldable_struct" />
        </table>
    </div>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { TagQueryEmits } from './tag-query-emits'
import type { TagQueryProps } from './tag-query-props'
import FoldableStruct from './foldable-struct.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { useTagQuery } from '@/classes/use-tag-query'

const props = defineProps<TagQueryProps>()
const emits = defineEmits<TagQueryEmits>()

const {
    // Template refs
    foldable_struct,

    // State
    use_tag,
    is_and_search,
    tag_struct,

    // Methods used in template
    clicked_items,
    update_check_state,

    // Exposed methods
    get_use_tag,
    get_tags,
    get_is_and_search,
    update_check,
} = useTagQuery({ props, emits })

defineExpose({ get_use_tag, get_tags, get_is_and_search, update_check })
</script>
