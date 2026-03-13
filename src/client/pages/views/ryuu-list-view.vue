<template>
    <v-card>
        <div v-if="target_kyou" class="ryuu_views">
            <v-tabs v-if="ryuu_definitions.length > 1 || editable" v-model="current_definition_index" show-arrows>
                <v-tab v-for="(def, i) in ryuu_definitions" :key="i" :value="i">
                    {{ def.name }}
                </v-tab>
                <v-btn v-if="editable" icon="mdi-plus" size="small" variant="text" class="align-self-center ml-1"
                    @click="add_definition" :title="i18n.global.t('ADD_RYUU_DEFINITION_TITLE')" />
            </v-tabs>
            <v-row v-if="editable && ryuu_definitions.length > 0" class="pa-2 ma-0" align="center">
                <v-col class="pa-0 ma-0">
                    <v-text-field v-model="ryuu_definitions[current_definition_index].name"
                        :label="i18n.global.t('RYUU_DEFINITION_NAME_LABEL')" density="compact" hide-details />
                </v-col>
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn v-if="ryuu_definitions.length > 1" icon="mdi-delete" size="small" variant="text"
                        @click="delete_current_definition"
                        :title="i18n.global.t('DELETE_RYUU_DEFINITION_TITLE')" />
                </v-col>
            </v-row>

            <v-window v-model="current_definition_index">
                <v-window-item v-for="(def, i) in ryuu_definitions" :key="i" :value="i">
                    <RyuuListItemView v-for="(query, qIdx) in def.queries" :key="query.id"
                        v-model="def.queries[qIdx]" :gkill_api="gkill_api" :application_config="application_config"
                        :enable_dialog="true" :enable_context_menu="true" :target_kyou="target_kyou"
                        :abort_controller="abort_controler" :find_kyou_query_default="find_kyou_query_default"
                        :editable="editable"
                        @requested_move_related_kyou_query="(...id: any[]) => handle_move_related_kyou_query(id[0] as string, id[1] as string, id[2] as 'up' | 'down')"
                        @requested_delete_related_kyou_list_query="(...id: any[]) => delete_related_kyou_query(id[0] as string)"
                        @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0])"
                        @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                        @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0])"
                        @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                        @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
                        @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                        @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                        @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                        @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
                        @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                        @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                        @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                        @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                        @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                        @requested_reload_kyou="(...cloned_kyou: any[]) => emits('requested_reload_kyou', cloned_kyou[0] as Kyou)"
                        @requested_reload_list="emits('requested_reload_list')"
                        @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)"
                        @focused_kyou="(...kyou: any[]) => emits('focused_kyou', kyou[0] as Kyou)"
                        @clicked_kyou="(...kyou: any[]) => { emits('focused_kyou', kyou[0] as Kyou); emits('clicked_kyou', kyou[0] as Kyou) }"
                        @requested_open_rykv_dialog="(...params: any[]) => open_rykv_dialog(params[0], params[1], params[2])"
                        ref="related_kyou_list_item_views" />
                </v-window-item>
            </v-window>

            <AddRyuuItemDialog :gkill_api="gkill_api" :application_config="application_config"
                @requested_add_related_kyou_query="(...related_kyou_query: any[]) => add_related_kyou_query(related_kyou_query[0] as RelatedKyouQuery)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                ref="add_ryuu_item_dialog" />
            <RykvDialogHost :application_config="application_config" :gkill_api="gkill_api" :dialogs="opened_dialogs"
                :enable_context_menu="true" :enable_dialog="true"
                @closed="(...id: any[]) => close_rykv_dialog(id[0] as string)"
                @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0] as Kyou)"
                @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
                @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
                @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
                @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
                @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
                @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
                @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
                @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
                @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
                @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
                @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @focused_kyou="(...kyou: any[]) => emits('focused_kyou', kyou[0] as Kyou)"
                @clicked_kyou="(...kyou: any[]) => { emits('focused_kyou', kyou[0] as Kyou); emits('clicked_kyou', kyou[0] as Kyou) }"
                @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)"
                @requested_open_rykv_dialog="(...params: any[]) => open_rykv_dialog(params[0], params[1], params[2])" />

            <v-avatar v-if="editable" :style="floatingActionButtonStyle()" color="primary" class="position-fixed-ryuu">
                <v-menu transition="slide-x-transition">
                    <template v-slot:activator="{ props }">
                        <v-btn color="white" icon="mdi-plus" variant="text" v-bind="props"
                            @click="add_ryuu_item_dialog?.show()" />
                    </template>
                </v-menu>
            </v-avatar>

            <v-row v-if="editable" class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark @click="apply" color="primary">{{ i18n.global.t("APPLY_TITLE") }}</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="emits('requested_close_dialog')">
                        {{ i18n.global.t("CANCEL_TITLE") }}
                    </v-btn>
                </v-col>
            </v-row>
        </div>
    </v-card>
</template>

<script lang="ts" setup>
import { i18n } from '@/i18n'
import { ref, computed, type Ref, watch, nextTick, onUnmounted } from 'vue';
import AddRyuuItemDialog from '../dialogs/add-ryuu-item-dialog.vue';
import RyuuListItemView from './ryuu-list-item-view.vue';
import RelatedKyouQuery from '../../classes/dnote/related-kyou-query';
import type RyuuListViewProps from './ryuu-list-view-props';
import type RyuuListViewEmits from './ryuu-list-view-emits';
import { build_dnote_predicate_from_json } from '@/classes/dnote/serialize/regist-dictionary';
import { ApplicationConfig } from '@/classes/datas/config/application-config';
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query';
import type { Kyou } from '@/classes/datas/kyou';
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';
import RykvDialogHost from './rykv-dialog-host.vue';
import type { OpenedRykvDialog, RykvDialogKind, RykvDialogPayload } from './rykv-dialog-kind';

interface RyuuDefinition {
    name: string
    queries: Array<RelatedKyouQuery>
}

const add_ryuu_item_dialog = ref<InstanceType<typeof AddRyuuItemDialog> | null>(null);
const related_kyou_list_item_views = ref()

const model_value = defineModel<ApplicationConfig>()
const props = defineProps<RyuuListViewProps>()
const emits = defineEmits<RyuuListViewEmits>()

const ryuu_definitions: Ref<Array<RyuuDefinition>> = ref([])
const current_definition_index = ref(0)
watch(current_definition_index, (newIdx, oldIdx) => {
    if (newIdx === oldIdx) return
    if (!props.editable) {
        abort_controler.value.abort()
        abort_controler.value = new AbortController()
        nextTick(() => load_related_kyou())
    }
})
const related_kyou_queries = computed({
    get: () => {
        if (ryuu_definitions.value.length === 0) return [] as Array<RelatedKyouQuery>
        const idx = current_definition_index.value
        const safeIdx = (idx >= 0 && idx < ryuu_definitions.value.length) ? idx : 0
        return ryuu_definitions.value[safeIdx].queries
    },
    set: (val: Array<RelatedKyouQuery>) => {
        if (ryuu_definitions.value.length === 0) return
        const idx = current_definition_index.value
        if (idx >= 0 && idx < ryuu_definitions.value.length) {
            ryuu_definitions.value[idx].queries = val
        }
    }
})
const opened_dialogs: Ref<Array<OpenedRykvDialog>> = ref([])
const abort_controler: Ref<AbortController> = ref(new AbortController())

nextTick(async () => {
    await load_from_application_config()
    if (props.editable) return

    abort_controler.value.abort()
    abort_controler.value = new AbortController()
    nextTick(() => load_related_kyou())
})

watch(() => props.target_kyou, () => {
    if (props.editable && !props.target_kyou) return
    abort_controler.value.abort()
    abort_controler.value = new AbortController()
    nextTick(() => { load_related_kyou() })
})

onUnmounted(() => {
    abort_controler.value.abort()
    abort_controler.value = new AbortController()
})

async function load_related_kyou(): Promise<void> {
    if (!related_kyou_list_item_views.value) return
    const wait_promises = []
    for (let i = 0; i < related_kyou_list_item_views.value.length; i++) {
        wait_promises.push(related_kyou_list_item_views.value[i].load_related_kyou())
    }
    await Promise.all(wait_promises)
}

async function load_from_application_config(): Promise<void> {
    nextTick(() => {
        from_json(props.application_config.ryuu_json_data)
    })
}

function parse_single_definition_queries(json: any): Array<RelatedKyouQuery> {
    const queries = new Array<RelatedKyouQuery>()
    if (!json) return queries
    for (let i = 0; i < json.length; i++) {
        const related_kyou_query = new RelatedKyouQuery()
        related_kyou_query.id = json[i].id
        related_kyou_query.title = json[i].title
        related_kyou_query.prefix = json[i].prefix
        related_kyou_query.suffix = json[i].suffix
        related_kyou_query.predicate = build_dnote_predicate_from_json(json[i].predicate)
        related_kyou_query.related_time_match_type = json[i].related_time_match_type
        related_kyou_query.find_kyou_query = json[i].find_kyou_query ? FindKyouQuery.parse_find_kyou_query(json[i].find_kyou_query) : null
        related_kyou_query.find_duration_hour = json[i].find_duration_hour
        queries.push(related_kyou_query)
    }
    return queries
}

function from_json(json: any): void {
    let definitions_json: any[]
    if (Array.isArray(json) && json.length > 0 && json[0] !== null && typeof json[0] === 'object' && 'name' in json[0] && 'queries' in json[0]) {
        definitions_json = json
    } else if (Array.isArray(json)) {
        definitions_json = [{ name: i18n.global.t('RYUU_DEFINITION_DEFAULT_NAME'), queries: json }]
    } else {
        definitions_json = [{ name: i18n.global.t('RYUU_DEFINITION_DEFAULT_NAME'), queries: [] }]
    }
    ryuu_definitions.value = definitions_json.map((def_json: any) => ({
        name: def_json.name || i18n.global.t('RYUU_DEFINITION_DEFAULT_NAME'),
        queries: parse_single_definition_queries(def_json.queries),
    }))
    if (current_definition_index.value >= ryuu_definitions.value.length) {
        current_definition_index.value = 0
    }
}

function serialize_single_definition(def: RyuuDefinition): any {
    const json = []
    for (let i = 0; i < def.queries.length; i++) {
        const related_kyou_query = def.queries[i]
        json.push({
            id: related_kyou_query.id,
            title: related_kyou_query.title,
            prefix: related_kyou_query.prefix,
            suffix: related_kyou_query.suffix,
            predicate: related_kyou_query.predicate.predicate_struct_to_json(),
            related_time_match_type: related_kyou_query.related_time_match_type,
            find_kyou_query: related_kyou_query.find_kyou_query,
            find_duration_hour: related_kyou_query.find_duration_hour,
        })
    }
    return { name: def.name, queries: json }
}

function to_json(): any {
    return ryuu_definitions.value.map(serialize_single_definition)
}

function add_definition(): void {
    const new_def: RyuuDefinition = {
        name: i18n.global.t('RYUU_DEFINITION_DEFAULT_NAME') + " " + (ryuu_definitions.value.length + 1),
        queries: new Array<RelatedKyouQuery>(),
    }
    ryuu_definitions.value.push(new_def)
    current_definition_index.value = ryuu_definitions.value.length - 1
}

function delete_current_definition(): void {
    if (ryuu_definitions.value.length <= 1) return
    ryuu_definitions.value.splice(current_definition_index.value, 1)
    if (current_definition_index.value >= ryuu_definitions.value.length) {
        current_definition_index.value = ryuu_definitions.value.length - 1
    }
    if (!props.editable) {
        abort_controler.value.abort()
        abort_controler.value = new AbortController()
        nextTick(() => load_related_kyou())
    }
}

function add_related_kyou_query(related_kyou_query: RelatedKyouQuery): void {
    related_kyou_queries.value.push(related_kyou_query)
}

async function apply(): Promise<void> {
    if (!model_value.value) return
    const ryuu_json_data = to_json()
    model_value.value.ryuu_json_data = ryuu_json_data
    emits('requested_apply_ryuu_struct', ryuu_json_data)
    nextTick(() => emits('requested_close_dialog'))
}

function floatingActionButtonStyle() {
    return {
        bottom: '60px',
        right: '10px',
        height: '50px',
        width: '50px',
    }
}

function delete_related_kyou_query(id: string): void {
    let delete_target_index: number | null = null
    for (let i = 0; i < related_kyou_queries.value.length; i++) {
        if (related_kyou_queries.value[i].id === id) {
            delete_target_index = i
            break
        }
    }
    if (delete_target_index !== null) {
        related_kyou_queries.value.splice(delete_target_index, 1)
    }
}

/**
 * FoldableStruct式：上/下挿入で並び替え
 */
function handle_move_related_kyou_query(srcId: string, targetId: string, dropType: 'up' | 'down'): void {
    if (!props.editable) return

    const from = related_kyou_queries.value.findIndex(v => v.id === srcId)
    const target = related_kyou_queries.value.findIndex(v => v.id === targetId)
    if (from < 0 || target < 0) return
    if (from === target) return

    const [item] = related_kyou_queries.value.splice(from, 1)

    // remove後のtarget補正
    let t = target
    if (from < target) t = target - 1

    const insertIndex = (dropType === 'up') ? t : (t + 1)
    related_kyou_queries.value.splice(insertIndex, 0, item)

    nextTick(() => load_related_kyou())
}

function open_rykv_dialog(kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload): void {
    opened_dialogs.value.push({
        id: props.gkill_api.generate_uuid(),
        kind,
        kyou: kyou.clone(),
        payload: payload ?? null,
        opened_at: Date.now(),
    })
}

function close_rykv_dialog(dialog_id: string): void {
    for (let i = 0; i < opened_dialogs.value.length; i++) {
        if (opened_dialogs.value[i].id === dialog_id) {
            opened_dialogs.value.splice(i, 1)
            break
        }
    }
}
</script>

<style lang="css" scoped>
.ryuu_views {
    position: relative;
    width: -webkit-fill-available;
    min-width: 400px;
    min-height: 20vh;
}
</style>
