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
import { ref, type Ref } from 'vue';
import type { FindKyouQuery } from '../../classes/api/find_query/find-kyou-query';
import type { Kyou } from '../../classes/datas/kyou';
import load_kyous from '../../classes/dnote/kyou-loader';
import DnoteItem from '../../classes/dnote/dnote-item';
import DnoteListQuery from './dnote-list-query';
import AddDnoteListDialog from '../../pages/dialogs/add-dnote-list-dialog.vue';
import AddDnoteItemDialog from '../../pages/dialogs/add-dnote-item-dialog.vue';
import { type DnoteEmits } from '@/pages/views/dnote-emits';
import regist_dictionary from '@/classes/dnote/serialize/regist-dictionary';

const dnote_item_table_view = ref<InstanceType<typeof DnoteItemTableView> | null>(null);
const dnote_list_table_view = ref<InstanceType<typeof DnoteListTableView> | null>(null);
const add_dnote_list_dialog = ref<InstanceType<typeof AddDnoteListDialog> | null>(null);
const add_dnote_item_dialog = ref<InstanceType<typeof AddDnoteItemDialog> | null>(null);

regist_dictionary()
defineProps<DnoteViewProps>()
defineExpose({ reload, abort })
const emits = defineEmits<DnoteEmits>()

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

}

async function load_aggregated_value(abort_controller: AbortController, kyous: Array<Kyou>, query: FindKyouQuery, kyou_is_loaded: boolean) {
    return dnote_item_table_view.value?.load_aggregated_value(abort_controller, kyous, query, kyou_is_loaded)
}

async function load_aggregate_grouping_list(abort_controller: AbortController, kyous: Array<Kyou>, find_kyou_query: FindKyouQuery, kyou_is_loaded: boolean): Promise<void> {
    return dnote_list_table_view.value?.load_aggregate_grouping_list(abort_controller, kyous, find_kyou_query, kyou_is_loaded)
}

function floatingActionButtonStyle() {
    return {
        'bottom': '60px',
        'right': '10px',
        'height': '50px',
        'width': '50px',
    }
}

</script>
<style lang="css" scoped>
.position-fixed {
    position: static !important
}
</style>