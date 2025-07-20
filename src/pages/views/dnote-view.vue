<template>
    <v-card class="dnote_view">
        <v-overlay v-model="is_loading" :content-class="'dnote_progress_overlay'" class="align-center justify-center"
            contained persistent>
            <v-progress-circular indeterminate color="primary" class="align-center justify-center" />
            <div v-if="getted_kyous_count !== target_kyous_count" class="align-center justify-center">
                <div class="align-center justify-center overlay_message">
                    {{ i18n.global.t('DNOTE_GETTING_DATA') }}
                </div>
                <div class="align-center justify-center overlay_message">
                    {{ getted_kyous_count }}/{{ target_kyous_count }}
                </div>
            </div>
            <div v-if="getted_kyous_count === target_kyous_count" class="align-center justify-center">
                <div class="align-center justify-center overlay_message">
                    {{ i18n.global.t('DNOTE_CALCURATING') }}
                </div>
                <div class="align-center justify-center overlay_message">
                    {{ finished_aggregate_task }}/{{ estimate_aggregate_task }}
                </div>
                <div class="align-center justify-center overlay_message">{{ i18n.global.t('DNOTE_PLEASE_WAIT_MESSAGE')
                    }}</div>
            </div>
        </v-overlay>
        <h1>
            <span>{{ start_date_str }}</span>
            <span v-if="end_date_str !== '' && start_date_str != end_date_str">～</span>
            <span v-if="end_date_str !== '' && start_date_str != end_date_str">{{ end_date_str }}</span>
            <span v-if="start_date_str === '' && !(end_date_str !== '' && start_date_str != end_date_str)">{{
                i18n.global.t("DNOTE_WHOLE_PERIOD_TITLE") }}</span>
        </h1>
        <DnoteItemTableView :application_config="application_config" :gkill_api="gkill_api" :editable="editable"
            v-model="dnote_item_table_view_data" @deleted_kyou="(kyou) => emits('deleted_kyou', kyou)"
            @deleted_tag="(tag) => emits('deleted_tag', tag)" @deleted_text="(text) => emits('deleted_text', text)"
            @deleted_notification="(notification) => emits('deleted_notification', notification)"
            @registered_kyou="(kyou) => emits('registered_kyou', kyou)"
            @registered_tag="(tag) => emits('registered_tag', tag)"
            @registered_text="(text) => emits('registered_text', text)"
            @registered_notification="(notification) => emits('registered_notification', notification)"
            @updated_kyou="(kyou) => emits('updated_kyou', kyou)" @updated_tag="(tag) => emits('updated_tag', tag)"
            @updated_text="(text) => emits('updated_text', text)"
            @updated_notification="(notification) => emits('updated_notification', notification)"
            @finish_a_aggregate_task="finished_aggregate_task++" ref="dnote_item_table_view" />
        <DnoteListTableView :application_config="application_config" :gkill_api="gkill_api" :editable="editable"
            v-model="dnote_list_item_table_view_data" @deleted_kyou="(kyou) => emits('deleted_kyou', kyou)"
            @deleted_tag="(tag) => emits('deleted_tag', tag)" @deleted_text="(text) => emits('deleted_text', text)"
            @deleted_notification="(notification) => emits('deleted_notification', notification)"
            @registered_kyou="(kyou) => emits('registered_kyou', kyou)"
            @registered_tag="(tag) => emits('registered_tag', tag)"
            @registered_text="(text) => emits('registered_text', text)"
            @registered_notification="(notification) => emits('registered_notification', notification)"
            @updated_kyou="(kyou) => emits('updated_kyou', kyou)" @updated_tag="(tag) => emits('updated_tag', tag)"
            @updated_text="(text) => emits('updated_text', text)"
            @updated_notification="(notification) => emits('updated_notification', notification)"
            @finish_a_aggregate_task="finished_aggregate_task++" ref="dnote_list_table_view" />
        <v-avatar v-if="editable" :style="floatingActionButtonStyle()" color="primary" class="position-fixed">
            <v-menu transition="slide-x-transition">
                <template v-slot:activator="{ props }">
                    <v-btn color="white" icon="mdi-plus" variant="text" v-bind="props" />
                </template>
                <v-list>
                    <v-list-item @click="add_dnote_item_dialog?.show()">
                        <v-list-item-title>{{ i18n.global.t("ADD_DNOTE_ITEM_MENU_TITLE") }}</v-list-item-title>
                    </v-list-item>
                    <v-list-item @click="add_dnote_list_dialog?.show()">
                        <v-list-item-title>{{ i18n.global.t("ADD_DNOTE_LIST_MENU_TITLE") }}</v-list-item-title>
                    </v-list-item>
                </v-list>
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
        <AddDnoteListDialog :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" @requested_add_dnote_list_query="(dnote_list_query) => {
                dnote_list_item_table_view_data.push(dnote_list_query)
                load_aggregated_value(abort_controller, [], new FindKyouQuery(), true);
                load_aggregate_grouping_list(abort_controller, [], new FindKyouQuery(), true)
            }" ref="add_dnote_list_dialog" />
        <AddDnoteItemDialog :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" @requested_add_dnote_item="(dnote_item) => {
                dnote_item_table_view_data[0].push(dnote_item);
                load_aggregated_value(abort_controller, [], new FindKyouQuery(), true);
                load_aggregate_grouping_list(abort_controller, [], new FindKyouQuery(), true)
            }" ref="add_dnote_item_dialog" />
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type DnoteViewProps } from '@/pages/views/dnote-view-props';
import DnoteItemTableView from './dnote-item-table-view.vue';
import DnoteListTableView from './dnote-list-table-view.vue';
import { computed, nextTick, ref, watch, type Ref } from 'vue';
import { FindKyouQuery } from '../../classes/api/find_query/find-kyou-query';
import type { Kyou } from '../../classes/datas/kyou';
import DnoteItem from '../../classes/dnote/dnote-item';
import DnoteListQuery from './dnote-list-query';
import AddDnoteListDialog from '../../pages/dialogs/add-dnote-list-dialog.vue';
import AddDnoteItemDialog from '../../pages/dialogs/add-dnote-item-dialog.vue';
import { type DnoteEmits } from '@/pages/views/dnote-emits';
import regist_dictionary, { build_dnote_aggregate_target_from_json, build_dnote_key_getter_from_json, build_dnote_predicate_from_json } from '@/classes/dnote/serialize/regist-dictionary';
import { UpdateDnoteJSONDataRequest } from '@/classes/api/req_res/update-dnote-json-data-request';
import moment from 'moment';

const dnote_item_table_view = ref<InstanceType<typeof DnoteItemTableView> | null>(null);
const dnote_list_table_view = ref<InstanceType<typeof DnoteListTableView> | null>(null);
const add_dnote_list_dialog = ref<InstanceType<typeof AddDnoteListDialog> | null>(null);
const add_dnote_item_dialog = ref<InstanceType<typeof AddDnoteItemDialog> | null>(null);

const props = defineProps<DnoteViewProps>()
defineExpose({ reload, abort })
const emits = defineEmits<DnoteEmits>()

watch(() => props.application_config, () => {
    load_from_application_config()
})

nextTick(() => {
    load_from_application_config()
})

const dnote_item_table_view_data: Ref<Array<Array<DnoteItem>>> = ref(new Array<Array<DnoteItem>>())
const dnote_list_item_table_view_data: Ref<Array<DnoteListQuery>> = ref(new Array<DnoteListQuery>())
const abort_controller = ref(new AbortController())
const is_loading = ref(true)

const target_kyous_count = ref(0)
const getted_kyous_count = ref(0)
const estimate_aggregate_task = ref(0)
const finished_aggregate_task = ref(0)

const first_kyou_date_str = ref("")
const last_kyou_date_str = ref("")
const start_date_str: Ref<string> = computed(() => props.query.use_calendar ? (moment(props.query.calendar_start_date ? props.query.calendar_start_date : moment().toDate()).format("YYYY-MM-DD")) : first_kyou_date_str.value)
const end_date_str: Ref<string> = computed(() => props.query.use_calendar ? (moment(props.query.calendar_end_date ? props.query.calendar_end_date : moment().toDate()).format("YYYY-MM-DD")) : last_kyou_date_str.value)

async function reload(kyous: Array<Kyou>, query: FindKyouQuery): Promise<void> {
    is_loading.value = true
    first_kyou_date_str.value = kyous && kyous.length > 0 ? moment(kyous[kyous.length - 1].related_time).format("YYYY-MM-DD") : ""
    last_kyou_date_str.value = kyous && kyous.length > 0 ? moment(kyous[0].related_time).format("YYYY-MM-DD") : ""

    reset_view()
    if (dnote_item_table_view_data.value.length === 0) {
        dnote_item_table_view_data.value.push(new Array<DnoteItem>())
    }
    await abort()

    const trimed_kyous_map = new Map<string, Kyou>()
    for (let i = 0; i < kyous.length; i++) {
        trimed_kyous_map.set(kyous[i].id, kyous[i])
    }
    const trimed_kyous = new Array<Kyou>()
    trimed_kyous_map.forEach((kyou) => trimed_kyous.push(kyou))

    target_kyous_count.value = trimed_kyous.length
    getted_kyous_count.value = 0
    finished_aggregate_task.value = 0
    estimate_aggregate_task.value = 0
    for (let i = 0; i < dnote_item_table_view_data.value.length; i++) {
        estimate_aggregate_task.value += dnote_item_table_view_data.value[i].length
    }
    estimate_aggregate_task.value += dnote_list_item_table_view_data.value.length

    const cloned_kyou = await load_kyous(abort_controller.value, trimed_kyous, true, true)
    const kyou_is_loaded = true
    const waitPromises = new Array<Promise<any>>()
    waitPromises.push(load_aggregated_value(abort_controller.value, cloned_kyou, query, kyou_is_loaded))
    waitPromises.push(load_aggregate_grouping_list(abort_controller.value, cloned_kyou, query, kyou_is_loaded))
    await Promise.all(waitPromises)
    is_loading.value = false
}

async function reset_view(): Promise<void> {
    return nextTick(async () => {
        dnote_item_table_view_data.value = new Array<Array<DnoteItem>>()
        dnote_list_item_table_view_data.value = new Array<DnoteListQuery>()
        load_from_application_config()
        await dnote_item_table_view.value?.reset()
        await dnote_list_table_view.value?.reset()
    })
}

async function abort(): Promise<any> {
    abort_controller.value.abort()
    abort_controller.value = new AbortController()
    return reset_view()
}

async function load_aggregated_value(abort_controller: AbortController, kyous: Array<Kyou>, query: FindKyouQuery, kyou_is_loaded: boolean) {
    return dnote_item_table_view.value?.load_aggregated_value(abort_controller, kyous, query, kyou_is_loaded)
}

async function load_aggregate_grouping_list(abort_controller: AbortController, kyous: Array<Kyou>, find_kyou_query: FindKyouQuery, kyou_is_loaded: boolean): Promise<void> {
    return dnote_list_table_view.value?.load_aggregate_grouping_list(abort_controller, kyous, find_kyou_query, kyou_is_loaded)
}

function to_json(): any {
    const dnote_item_table_view_data_seliarized = []
    for (let i = 0; i < dnote_item_table_view_data.value.length; i++) {
        const list = []
        const dnote_item_col = dnote_item_table_view_data.value[i]
        for (let j = 0; j < dnote_item_col.length; j++) {
            const dnote_item = dnote_item_col[j]
            const record = {
                id: dnote_item.id,
                prefix: dnote_item.prefix,
                suffix: dnote_item.suffix,
                title: dnote_item.title,
                aggregate_target: dnote_item.agregate_target.to_json(),
                predicate: dnote_item.predicate.predicate_struct_to_json(),
            }
            list.push(record)
        }
        dnote_item_table_view_data_seliarized.push(list)
    }

    const dnote_list_item_table_view_data_seliarized = []
    for (let i = 0; i < dnote_list_item_table_view_data.value.length; i++) {
        const list_find_query = dnote_list_item_table_view_data.value[i]
        const record = {
            id: list_find_query.id,
            prefix: list_find_query.prefix,
            suffix: list_find_query.suffix,
            title: list_find_query.title,
            aggregate_target: list_find_query.aggregate_target.to_json(),
            predicate: list_find_query.predicate.predicate_struct_to_json(),
            key_getter: list_find_query.key_getter.to_json(),
        }
        dnote_list_item_table_view_data_seliarized.push(record)
    }

    return {
        dnote_item_table_view_data: dnote_item_table_view_data_seliarized,
        dnote_list_item_table_view_data: dnote_list_item_table_view_data_seliarized,
    }
}

function from_json(json: any): void {
    regist_dictionary()
    const items: Array<Array<DnoteItem>> = ((json && json.dnote_item_table_view_data ? json.dnote_item_table_view_data : []) || []).map((col: any[]) =>
        col.map((itemJson: any) => {
            const item = new DnoteItem()
            item.id = itemJson.id
            item.prefix = itemJson.prefix
            item.suffix = itemJson.suffix
            item.title = itemJson.title
            item.agregate_target = build_dnote_aggregate_target_from_json(itemJson.aggregate_target)
            item.predicate = build_dnote_predicate_from_json(itemJson.predicate)
            return item
        })
    )
    dnote_item_table_view_data.value = items

    const queries: Array<DnoteListQuery> = ((json && json.dnote_list_item_table_view_data ? json.dnote_list_item_table_view_data : []) || []).map((queryJson: any) => {
        const query = new DnoteListQuery()
        query.id = queryJson.id
        query.prefix = queryJson.prefix
        query.suffix = queryJson.suffix
        query.title = queryJson.title
        query.aggregate_target = build_dnote_aggregate_target_from_json(queryJson.aggregate_target)
        query.predicate = build_dnote_predicate_from_json(queryJson.predicate)
        query.key_getter = build_dnote_key_getter_from_json(queryJson.key_getter)
        return query
    })
    dnote_list_item_table_view_data.value = queries
    if (dnote_item_table_view_data.value.length === 0) {
        dnote_item_table_view_data.value.push(new Array<DnoteItem>())
    }
}

function load_from_application_config(): void {
    from_json(props.application_config.dnote_json_data)
}

function floatingActionButtonStyle() {
    return {
        'bottom': '60px',
        'right': '10px',
        'height': '50px',
        'width': '50px',
    }
}

async function apply(): Promise<void> {
    const req = new UpdateDnoteJSONDataRequest()
    req.dnote_json_data = to_json()
    const res = await props.gkill_api.update_dnote_json_data(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('requested_close_dialog')
}

// 進捗表示のためかか共通からコピー
async function load_kyous(abort_controller: AbortController, kyous: Array<Kyou>, get_latest_data: boolean, clone: boolean): Promise<Array<Kyou>> {
    const cloned_kyous = new Array<Kyou>()
    for (let i = 0; i < kyous.length; i++) {
        let kyou: Kyou = kyous[i]
        const waitPromises = []
        if (clone) {
            kyou = kyous[i].clone()
            kyou.abort_controller = abort_controller
        }
        if (get_latest_data) {
            await kyou.reload(false, true)
        }
        if (clone || get_latest_data) {
            waitPromises.push(kyou.load_typed_datas())
            waitPromises.push(kyou.load_attached_tags())
            waitPromises.push(kyou.load_attached_texts())
        }
        await Promise.all(waitPromises)
        cloned_kyous.push(kyou)
        getted_kyous_count.value++
    }
    return cloned_kyous
}
</script>
<style lang="css" scoped>
.position-fixed {
    position: relative;
}
</style>
<style lang="css">
.git_commit_log_message {
    white-space: pre-line;
}

.plus_value {
    color: limegreen;
}

.minus_value {
    color: crimson;
}

.dnote_view {
    position: relative;
    width: 625px;
    min-width: 625px;
}

.dnote_view .lantana_icon {
    position: relative;
    width: 20px !important;
    height: 20px !important;
    max-width: 20px !important;
    min-width: 20px !important;
    max-height: 20px !important;
    min-height: 20px !important;
    display: inline-block;
}

.dnote_view .lantana_icon_fill {
    width: 20px !important;
    height: 20px !important;
    max-width: 20px !important;
    min-width: 20px !important;
    max-height: 20px !important;
    min-height: 20px !important;
    display: inline-block;
    z-index: 10;
}

.dnote_view .lantana_icon_harf_left {
    position: absolute;
    left: 0px;
    width: 10px !important;
    height: 20px !important;
    max-width: 10px !important;
    min-width: 10px !important;
    max-height: 20px !important;
    min-height: 20px !important;
    object-fit: cover;
    object-position: 0 0;
    display: inline-block;
    z-index: 10;
}

.dnote_view .lantana_icon_harf_right {
    position: absolute;
    left: 0px;
    width: 20px !important;
    height: 20px !important;
    max-width: 20px !important;
    min-width: 20px !important;
    max-height: 20px !important;
    min-height: 20px !important;
    display: inline-block;
    z-index: 9;
}

.dnote_view .lantana_icon_none {
    width: 20px !important;
    height: 20px !important;
    max-width: 20px !important;
    min-width: 20px !important;
    max-height: 20px !important;
    min-height: 20px !important;
    display: inline-block;
    z-index: 10;
}

.dnote_view .gray {
    filter: grayscale(100%);
}

.dnote_view .lantana_icon_tr {
    width: calc(20px * 5);
    max-width: calc(20px * 5);
    min-width: calc(20px * 5);
}

.dnote_view .lantana_icon_td {
    width: 20px !important;
    height: 20px !important;
    max-width: 20px !important;
    min-width: 20px !important;
    max-height: 20px !important;
    min-height: 20px !important;
    display: inline-block;
}
</style>
<style lang="css" scoped>
.overlay_target {
    z-index: -10000;
    position: absolute;
    min-height: calc(v-bind('app_content_height.toString().concat("px")'));
    min-width: calc(100vw);
    display: flex;
    flex-direction: column;
    align-items: center;
}

.overlay_message {
    text-align: center;
}
</style>
<style lang="css">
.dnote_progress_overlay {
    display: flex;
    flex-direction: column;
    align-items: center;
}
</style>