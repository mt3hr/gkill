<template>
    <div>
        <DnoteItemTableView :application_config="application_config" :gkill_api="gkill_api" :editable="editable"
            v-model="dnote_item_table_view_data" ref="dnote_item_table_view" />
        <DnoteListTableView :application_config="application_config" :gkill_api="gkill_api" :editable="editable"
            v-model="dnote_list_item_table_view_data" ref="dnote_list_table_view" />
        <v-avatar v-if="editable" :style="floatingActionButtonStyle()" color="primary" class="position-fixed">
            <v-menu transition="slide-x-transition">
                <template v-slot:activator="{ props }">
                    <v-btn color="white" icon="mdi-plus" variant="text" v-bind="props" />
                </template>
                <v-list>
                    <v-list-item @click="add_dnote_item_dialog?.show()">
                        <v-list-item-title>{{ $t("ADD_DNOTE_ITEM_MENU_TITLE") }}</v-list-item-title>
                    </v-list-item>
                    <v-list-item @click="add_dnote_list_dialog?.show()">
                        <v-list-item-title>{{ $t("ADD_DNOTE_LIST_MENU_TITLE") }}</v-list-item-title>
                    </v-list-item>
                </v-list>
            </v-menu>
        </v-avatar>
        <v-row v-if="editable" class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark @click="apply" color="primary">{{ $t("APPLY_TITLE") }}</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="secondary" @click="emits('requested_close_dialog')">{{ $t("CANCEL_TITLE")
                }}</v-btn>
            </v-col>
        </v-row>
        <AddDnoteListDialog :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_add_dnote_list_query="(dnote_list_query) => dnote_list_item_table_view_data.push(dnote_list_query)"
            ref="add_dnote_list_dialog" />
        <AddDnoteItemDialog :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_add_dnote_item="(dnote_item) => dnote_item_table_view_data[0].push(dnote_item)"
            ref="add_dnote_item_dialog" />
    </div>
</template>
<script lang="ts" setup>
import { type DnoteViewProps } from '@/pages/views/dnote-view-props';
import DnoteItemTableView from './dnote-item-table-view.vue';
import DnoteListTableView from './dnote-list-table-view.vue';
import { nextTick, ref, watch, type Ref } from 'vue';
import type { FindKyouQuery } from '../../classes/api/find_query/find-kyou-query';
import type { Kyou } from '../../classes/datas/kyou';
import load_kyous from '../../classes/dnote/kyou-loader';
import DnoteItem from '../../classes/dnote/dnote-item';
import DnoteListQuery from './dnote-list-query';
import AddDnoteListDialog from '../../pages/dialogs/add-dnote-list-dialog.vue';
import AddDnoteItemDialog from '../../pages/dialogs/add-dnote-item-dialog.vue';
import { type DnoteEmits } from '@/pages/views/dnote-emits';
import regist_dictionary from '@/classes/dnote/serialize/regist-dictionary';
import { UpdateDnoteJSONDataRequest } from '@/classes/api/req_res/update-dnote-json-data-request';

const dnote_item_table_view = ref<InstanceType<typeof DnoteItemTableView> | null>(null);
const dnote_list_table_view = ref<InstanceType<typeof DnoteListTableView> | null>(null);
const add_dnote_list_dialog = ref<InstanceType<typeof AddDnoteListDialog> | null>(null);
const add_dnote_item_dialog = ref<InstanceType<typeof AddDnoteItemDialog> | null>(null);

regist_dictionary()
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

async function reload(kyous: Array<Kyou>, query: FindKyouQuery): Promise<void> {
    if (dnote_item_table_view_data.value.length === 0) {
        dnote_item_table_view_data.value.push(new Array<DnoteItem>())
    }

    abort_controller.value.abort()
    abort_controller.value = new AbortController()
    const cloned_kyou = await load_kyous(abort_controller.value, kyous, true)
    const kyou_is_loaded = true
    await load_aggregated_value(abort_controller.value, cloned_kyou, query, kyou_is_loaded)
    await load_aggregate_grouping_list(abort_controller.value, cloned_kyou, query, kyou_is_loaded)
}

function abort(): void {
    abort_controller.value.abort()
}

async function load_aggregated_value(abort_controller: AbortController, kyous: Array<Kyou>, query: FindKyouQuery, kyou_is_loaded: boolean) {
    return dnote_item_table_view.value?.load_aggregated_value(abort_controller, kyous, query, kyou_is_loaded)
}

async function load_aggregate_grouping_list(abort_controller: AbortController, kyous: Array<Kyou>, find_kyou_query: FindKyouQuery, kyou_is_loaded: boolean): Promise<void> {
    return dnote_list_table_view.value?.load_aggregate_grouping_list(abort_controller, kyous, find_kyou_query, kyou_is_loaded)
}

function to_json(): any {
    return {
        dnote_item_table_view_data: dnote_item_table_view_data.value,
        dnote_list_item_tabel_view_data: dnote_list_item_table_view_data.value,
    }
}

function from_json(json: any): void {
    dnote_item_table_view_data.value = json.dnote_item_table_view_data
    dnote_list_item_table_view_data.value = json.dnote_list_item_table_view_data
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
    emits('requested_reload_application_config', res.application_config)
    emits('requested_close_dialog')
}

</script>
<style lang="css" scoped>
.position-fixed {
    position: static !important
}
</style>