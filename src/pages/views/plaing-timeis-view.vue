<template>
    <div class="plaing_timeis_view_wrap">
        <table class="plaing_timeis_view_table">
            <tr>
                <td valign="top">
                    <KyouListView :kyou_height="180" :width="app_content_width" :list_height="kyou_list_view_height"
                        :application_config="application_config" :gkill_api="gkill_api"
                        :matched_kyous="match_kyous_list" :query="query" :last_added_tag="last_added_tag"
                        :is_focused_list="true" :closable="false" :enable_context_menu="enable_context_menu"
                        :enable_dialog="enable_dialog" :is_readonly_mi_check="false" :show_checkbox="true"
                        :show_footer="true" :show_content_only="true" @updated_kyou="reload_list(false)"
                        @registered_kyou="reload_list(false)" @deleted_kyou="reload_list(false)"
                        @received_errors="(errors) => emits('received_errors', errors)"
                        @received_messages="(messages) => emits('received_messages', messages)"
                        @requested_reload_kyou="reload_list(false)" @requested_reload_list="reload_list(false)"
                        @requested_search="search(false)" ref="kyou_list_views" />
                </td>
            </tr>
        </table>
        <AddTimeisDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
            :last_added_tag="last_added_tag" :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
            :enable_dialog="enable_dialog" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="reload_list(false)" @requested_reload_list="reload_list(false)"
            ref="add_timeis_dialog" />
        <AddLantanaDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
            :last_added_tag="last_added_tag" :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
            :enable_dialog="enable_dialog" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="reload_list(false)" @requested_reload_list="reload_list(false)"
            ref="add_lantana_dialog" />
        <AddUrlogDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
            :last_added_tag="last_added_tag" :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
            :enable_dialog="enable_dialog" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="reload_list(false)" @requested_reload_list="reload_list(false)"
            ref="add_urlog_dialog" />
        <AddMiDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
            :last_added_tag="last_added_tag" :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
            :enable_dialog="enable_dialog" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="reload_list(false)" @requested_reload_list="reload_list(false)"
            ref="add_mi_dialog" />
        <AddNlogDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
            :last_added_tag="last_added_tag" :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
            :enable_dialog="enable_dialog" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="reload_list(false)" @requested_reload_list="reload_list(false)"
            ref="add_nlog_dialog" />
        <kftlDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
            :last_added_tag="last_added_tag" :kyou="new Kyou()" :app_content_height="app_content_height"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            :app_content_width="app_content_width" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou: Kyou) => reload_kyou(kyou)" @requested_reload_list="reload_list(false)"
            ref="kftl_dialog" />
        <UploadFileDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
            :application_config="application_config" :gkill_api="gkill_api" :last_added_tag="''"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" ref="upload_file_dialog" />
        <v-avatar :style="floatingActionButtonStyle()" color="indigo" class="position-fixed">
            <v-menu :style="add_kyou_menu_style" transition="slide-x-transition">
                <template v-slot:activator="{ props }">
                    <v-btn color="white" icon="mdi-plus" variant="text" v-bind="props" />
                </template>
                <v-list>
                    <v-list-item @click="show_kftl_dialog()">
                        <v-list-item-title>kftl</v-list-item-title>
                    </v-list-item>
                    <v-list-item @click="show_urlog_dialog()">
                        <v-list-item-title>urlog</v-list-item-title>
                    </v-list-item>
                    <v-list-item @click="show_timeis_dialog()">
                        <v-list-item-title>timeis</v-list-item-title>
                    </v-list-item>
                    <v-list-item @click="show_mi_dialog()">
                        <v-list-item-title>mi</v-list-item-title>
                    </v-list-item>
                    <v-list-item @click="show_nlog_dialog()">
                        <v-list-item-title>nlog</v-list-item-title>
                    </v-list-item>
                    <v-list-item @click="show_lantana_dialog()">
                        <v-list-item-title>lantana</v-list-item-title>
                    </v-list-item>
                    <v-list-item @click="show_upload_file_dialog()">
                        <v-list-item-title>アップロード</v-list-item-title>
                    </v-list-item>
                </v-list>
            </v-menu>
        </v-avatar>
    </div>
</template>
<script setup lang="ts">
import { computed, nextTick, type Ref, ref, watch } from 'vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { Kyou } from '@/classes/datas/kyou'
import AddMiDialog from '../dialogs/add-mi-dialog.vue'
import AddNlogDialog from '../dialogs/add-nlog-dialog.vue'
import KyouListView from './kyou-list-view.vue'
import kftlDialog from '../dialogs/kftl-dialog.vue'
import { GetKyousRequest } from '@/classes/api/req_res/get-kyous-request'
import type KftlDialog from '../dialogs/kftl-dialog.vue'
import AddLantanaDialog from '../dialogs/add-lantana-dialog.vue'
import AddTimeisDialog from '../dialogs/add-timeis-dialog.vue'
import AddUrlogDialog from '../dialogs/add-urlog-dialog.vue'
import UploadFileDialog from '../dialogs/upload-file-dialog.vue'
import moment from 'moment'
import type { PlaingTimeIsViewProps } from './plaing-timeis-view-props'
import type { PlaingTimeIsViewEmits } from './plaing-timeis-emits'

const enable_context_menu = ref(true)
const enable_dialog = ref(true)

const add_mi_dialog = ref<InstanceType<typeof AddMiDialog> | null>(null);
const add_nlog_dialog = ref<InstanceType<typeof AddNlogDialog> | null>(null);
const add_lantana_dialog = ref<InstanceType<typeof AddLantanaDialog> | null>(null);
const add_timeis_dialog = ref<InstanceType<typeof AddTimeisDialog> | null>(null);
const add_urlog_dialog = ref<InstanceType<typeof AddUrlogDialog> | null>(null);
const kftl_dialog = ref<InstanceType<typeof KftlDialog> | null>(null);
const upload_file_dialog = ref<InstanceType<typeof UploadFileDialog> | null>(null);
const kyou_list_views = ref();

const query: Ref<FindKyouQuery> = ref(new FindKyouQuery())

function generate_query(): FindKyouQuery {
    const plaing_timeis_query = new FindKyouQuery()
    plaing_timeis_query.use_plaing = true
    plaing_timeis_query.plaing_time = moment().toDate()
    props.application_config.rep_struct.forEach(rep_struct => {
        plaing_timeis_query.reps.push(rep_struct.rep_name)
    })
    props.application_config.tag_struct.forEach(tag_struct => {
        plaing_timeis_query.tags.push(tag_struct.tag_name)
    })
    return plaing_timeis_query
}

const match_kyous_list: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const focused_column_index: Ref<number> = ref(0)
const focused_kyous_list: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const focused_kyou: Ref<Kyou | null> = ref(null)
const focused_time: Ref<Date> = ref(moment().toDate())
const last_added_tag: Ref<string> = ref("")
const kyou_list_view_height = computed(() => props.app_content_height)

const position_x: Ref<Number> = ref(0)
const position_y: Ref<Number> = ref(0)

const props = defineProps<PlaingTimeIsViewProps>()
const emits = defineEmits<PlaingTimeIsViewEmits>()
defineExpose({ reload_list })


const skip_search_this_tick = ref(false)

watch(() => props.application_config, () => {
    search(false)
})

watch(() => focused_time.value, () => {
    if (!kyou_list_views.value) {
        return
    }
    const kyou_list_view = kyou_list_views.value[focused_column_index.value] as any
    if (!kyou_list_view) {
        return
    }
    kyou_list_view.scroll_to_time(focused_time.value)
})

async function reload_kyou(kyou: Kyou): Promise<void> {
    const kyous_list = match_kyous_list.value
    for (let j = 0; j < kyous_list.length; j++) {
        const kyou_in_list = kyous_list[j]
        if (kyou.id === kyou_in_list.id) {
            const updated_kyou = kyou.clone()
            await updated_kyou.reload()
            await updated_kyou.load_all()
            kyous_list.splice(j, 1, updated_kyou)
        }
    }
    if (focused_kyou.value && focused_kyou.value.id === kyou.id) {
        const updated_kyou = kyou.clone()
        await updated_kyou.reload()
        await updated_kyou.load_all()
        focused_kyou.value = updated_kyou
    }
}

const abort_controller: Ref<AbortController> = ref(new AbortController())
async function search(update_cache: boolean): Promise<void> {
    // 検索する。Tickでまとめる
    query.value = generate_query()
    try {
        if (abort_controller.value) {
            abort_controller.value.abort()
        }

        if (match_kyous_list.value) {
            match_kyous_list.value = []
        }

        const kyou_list_view = kyou_list_views.value as any
        if (kyou_list_view) {
            kyou_list_view.scroll_to(0)
        }
        await nextTick(async () => {
            const kyou_list_view = kyou_list_views.value as any
            if (!kyou_list_view) {
                return
            }
            kyou_list_view.set_loading(true)
            return nextTick(() => { }) // loading表記切り替え待ち
        })

        const req = new GetKyousRequest()
        abort_controller.value = req.abort_controller
        req.query = query.value.clone()
        req.query.parse_words_and_not_words()
        if (update_cache) {
            req.query.update_cache = true
        }
        const res = await props.gkill_api.get_kyous(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
        match_kyous_list.value = res.kyous
        focused_kyous_list.value.splice(0)
        for (let i = 0; i < match_kyous_list.value.length; i++) {
            focused_kyous_list.value.push(match_kyous_list.value[i])
        }
        await nextTick(() => {
            const kyou_list_view = kyou_list_views.value as any
            if (!kyou_list_view) {
                return
            }
            kyou_list_view.scroll_to(0)
            kyou_list_view.set_loading(false)
            skip_search_this_tick.value = false
        })
    } catch (err: any) {
        // abortは握りつぶす
        if (!(err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request"))) {
            // abort以外はエラー出力する
            console.error(err)
        }
    }
}

async function reload_list(update_cache: boolean): Promise<void> {
    return search(update_cache)
}

function floatingActionButtonStyle() {
    return {
        'bottom': '60px',
        'right': '10px',
        'height': '50px',
        'width': '50px'
    }
}

const add_kyou_menu_style = computed(() => `{ position: absolute; left: ${position_x.value}px; top: ${position_y.value}px; }`)

function show_kftl_dialog(): void {
    kftl_dialog.value?.show()
}

function show_timeis_dialog(): void {
    add_timeis_dialog.value?.show()
}
function show_mi_dialog(): void {
    add_mi_dialog.value?.show()
}

function show_nlog_dialog(): void {
    add_nlog_dialog.value?.show()
}

function show_lantana_dialog(): void {
    add_lantana_dialog.value?.show()
}

function show_urlog_dialog(): void {
    add_urlog_dialog.value?.show()
}

function show_upload_file_dialog(): void {
    upload_file_dialog.value?.show()
}
</script>
<style lang="css">
.plaing_timeis_view_table {
    padding-top: 0px;
}

.kyou_detail_view.dummy {
    overflow-x: hidden;
}

.kyou_detail_view {
    width: calc(400px + 8px);
    max-width: calc(400px + 8px);
    min-width: calc(400px + 8px);
}

.kyou_detail_view img.kyou_image {
    width: unset !important;
    height: unset !important;
    max-width: calc(400px - 2px) !important;
    max-height: 85vh !important;
}

.kyou_dialog img.kyou_image {
    width: unset !important;
    height: unset !important;
    max-width: 85vw !important;
    max-height: 85vh !important;
}

.plaing_timeis_view_wrap {
    position: relative;
}
</style>