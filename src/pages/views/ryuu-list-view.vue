<template>
    <v-card>
        <div v-if="related_time" class="ryuu_views">
            <RyuuListItemView v-for="related_kyou_query, index in related_kyou_queries"
                v-model="related_kyou_queries[index]" :key="related_kyou_query.id" :gkill_api="gkill_api"
                :application_config="application_config" :enable_dialog="true" :enable_context_menu="true"
                :related_time="related_time" :abort_controller="abort_controler"
                :find_kyou_query_default="find_kyou_query_default" :editable="editable"
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
                ref="related_kyou_list_item_views" />
        </div>
        <AddRyuuItemDialog :gkill_api="gkill_api" :application_config="application_config"
            @requested_add_related_kyou_query="(...related_kyou_query: any[]) => add_related_kyou_query(related_kyou_query[0] as RelatedKyouQuery)"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            ref="add_ryuu_item_dialog" />
        <v-avatar v-if="editable" :style="floatingActionButtonStyle()" color="primary" class="position-fixed">
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
                <v-btn dark color="secondary" @click="emits('requested_close_dialog')">{{ i18n.global.t("CANCEL_TITLE")
                }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { ref, defineEmits, defineProps, type Ref, watch, nextTick } from 'vue';
import AddRyuuItemDialog from '../dialogs/add-ryuu-item-dialog.vue';
import RyuuListItemView from './ryuu-list-item-view.vue';
import RelatedKyouQuery from '../../classes/dnote/related-kyou-query';
import type RyuuListViewProps from './ryuu-list-view-props';
import type RyuuListViewEmits from './ryuu-list-view-emits';
import { build_dnote_predicate_from_json } from '@/classes/dnote/serialize/regist-dictionary';
import { ApplicationConfig } from '@/classes/datas/config/application-config';
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query';
import { onUnmounted } from 'vue';
import type { Kyou } from '@/classes/datas/kyou';
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';

const add_ryuu_item_dialog = ref<InstanceType<typeof AddRyuuItemDialog> | null>(null);

const related_kyou_list_item_views = ref()

const model_value = defineModel<ApplicationConfig>()
const props = defineProps<RyuuListViewProps>()
const emits = defineEmits<RyuuListViewEmits>()

const related_kyou_queries: Ref<Array<RelatedKyouQuery>> = ref(new Array<RelatedKyouQuery>())
const abort_controler: Ref<AbortController> = ref(new AbortController())

nextTick(async () => {
    await load_from_application_config()
    if (props.editable) {
        return
    }
    abort_controler.value.abort()
    abort_controler.value = new AbortController()

    nextTick(() => load_related_kyou())
})

watch(() => props.related_time, () => {
    if (props.editable) {
        return
    }
    abort_controler.value.abort()
    abort_controler.value = new AbortController()
    nextTick(() => { load_related_kyou() })
})

onUnmounted(() => {
    abort_controler.value.abort()
    abort_controler.value = new AbortController()
})

async function load_related_kyou(): Promise<void> {
    if (!related_kyou_list_item_views.value) {
        return
    }
    const wait_promises = []
    for (let i = 0; i < related_kyou_list_item_views.value.length; i++) {
        wait_promises.push(related_kyou_list_item_views.value[i].load_related_kyou())
    }
    await Promise.all(wait_promises)
}

async function load_from_application_config(): Promise<void> {
    nextTick(() => {
        related_kyou_queries.value.splice(0)
        related_kyou_queries.value.push(...load_from_json(props.application_config.ryuu_json_data))
    })
}

function load_from_json(json: any): Array<RelatedKyouQuery> {
    const related_kyou_queries = new Array<RelatedKyouQuery>()
    if (!json) {
        return related_kyou_queries
    }
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
        related_kyou_queries.push(related_kyou_query)
    }
    return related_kyou_queries
}

function to_json(related_kyou_queries: Array<RelatedKyouQuery>): any {
    const json = []
    for (let i = 0; i < related_kyou_queries.length; i++) {
        const related_kyou_query = related_kyou_queries[i]
        const json_record = {
            id: related_kyou_query.id,
            title: related_kyou_query.title,
            prefix: related_kyou_query.prefix,
            suffix: related_kyou_query.suffix,
            predicate: related_kyou_query.predicate.predicate_struct_to_json(),
            related_time_match_type: related_kyou_query.related_time_match_type,
            find_kyou_query: related_kyou_query.find_kyou_query,
            find_duration_hour: related_kyou_query.find_duration_hour,
        }
        json.push(json_record)
    }
    return json
}

function add_related_kyou_query(related_kyou_query: RelatedKyouQuery): void {
    related_kyou_queries.value.push(related_kyou_query)
}

async function apply(): Promise<void> {
    if (model_value.value) {
        model_value.value.ryuu_json_data = to_json(related_kyou_queries.value)
        emits('requested_close_dialog')
    }
}

function floatingActionButtonStyle() {
    return {
        'bottom': '60px',
        'right': '10px',
        'height': '50px',
        'width': '50px',
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

</script>
<style lang="css" scoped>
.ryuu_views {
    position: relative;
    width: -webkit-fill-available;
    min-width: 400px;
    min-height: 20vh;
}
</style>
