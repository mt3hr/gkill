<template>
    <v-card @contextmenu.prevent="show_context_menu" :width="width" :height="height">
        <v-row v-if="kyou.typed_mi" class="pa-0 ma-0">
            <v-col cols="1" class="pa-0 ma-0">
                <v-checkbox v-model="is_checked_mi" hide-details @click="clicked_mi_check()" />
            </v-col>
            <v-col cols="8" class="pa-0 ma-0">
                <v-card-title>
                    <div class="py-1">{{ kyou.typed_mi.title }}</div>
                </v-card-title>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-card-title>
                    <div class="py-1">{{ kyou.typed_mi.board_name }}</div>
                </v-card-title>
            </v-col>
        </v-row>
        <div :style="{ 'padding-top': '30px' }">
            <div v-if="kyou.typed_mi && kyou.typed_mi.estimate_start_time">
                <span>開始日時：</span>
                <span>{{ format_time(kyou.typed_mi.estimate_start_time) }}</span>
            </div>
            <div v-if="kyou.typed_mi && kyou.typed_mi.estimate_end_time">
                <span>終了日時：</span>
                <span>{{ format_time(kyou.typed_mi.estimate_end_time) }}</span>
            </div>
            <div v-if="kyou.typed_mi && kyou.typed_mi.limit_time">
                <span>期限日時：</span>
                <span>{{ format_time(kyou.typed_mi.limit_time) }}</span>
            </div>
        </div>
        <MiContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => emits('requested_update_check_kyous', kyous, is_checked)"
            ref="context_menu" />
    </v-card>
</template>
<script lang="ts" setup>
import { type Ref, ref, watch } from 'vue'
import MiContextMenu from './mi-context-menu.vue'
import type { Kyou } from '@/classes/datas/kyou'
import type { miKyouViewProps } from './mi-kyou-view-props'
import type { miKyouViewEmits } from './mi-kyou-view-emits'
import moment from 'moment'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import { UpdateMiRequest } from '@/classes/api/req_res/update-mi-request'

const context_menu = ref<InstanceType<typeof MiContextMenu> | null>(null);

const props = defineProps<miKyouViewProps>()
const emits = defineEmits<miKyouViewEmits>()
defineExpose({ show_context_menu })

const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
const is_checked_mi: Ref<boolean> = ref(props.kyou.typed_mi ? props.kyou.typed_mi.is_checked : false)

load_cloned_kyou()

watch(() => props.kyou, async () => {
    await load_cloned_kyou()
    is_checked_mi.value = cloned_kyou.value.typed_mi ? cloned_kyou.value.typed_mi.is_checked : false
})

async function load_cloned_kyou() {
    const kyou = props.kyou.clone()
    await kyou.load_all()
    cloned_kyou.value = kyou
}

function format_time(time: Date): string {
    return moment(time).format("yyyy-MM-DD HH:mm:ss")
}

function show_context_menu(e: PointerEvent): void {
    if (props.enable_context_menu) {
        context_menu.value?.show(e)
    }
}

async function clicked_mi_check(): Promise<void> {
    // 読み取り専用表示だったら何もしない
    if (props.is_readonly_mi_check) {
        return
    }

    is_checked_mi.value = !is_checked_mi.value

    // データがちゃんとあるか確認。なければエラーメッセージを出力する
    const mi = cloned_kyou.value.typed_mi
    if (!mi) {
        const error = new GkillError()
        error.error_code = "//TODO"
        error.error_message = "クライアントのデータが変です"
        const errors = new Array<GkillError>()
        errors.push(error)
        emits('received_errors', errors)
        return
    }

    // 更新がなかったらエラーメッセージを出力する
    if (mi.is_checked === is_checked_mi.value) {
        const error = new GkillError()
        error.error_code = "//TODO"
        error.error_message = "miが更新されていません"
        const errors = new Array<GkillError>()
        errors.push(error)
        emits('received_errors', errors)
        return
    }

    // UserIDやDevice情報を取得する
    const get_gkill_req = new GetGkillInfoRequest()
    const gkill_info_res = await props.gkill_api.get_gkill_info(get_gkill_req)
    if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
        emits('received_errors', gkill_info_res.errors)
        return
    }

    // 更新後mi情報を用意する
    const updated_mi = mi.clone()
    updated_mi.is_checked = is_checked_mi.value
    updated_mi.update_app = "gkill"
    updated_mi.update_device = gkill_info_res.device
    updated_mi.update_time = new Date(Date.now())
    updated_mi.update_user = gkill_info_res.user_id

    // 更新リクエストを飛ばす
    const req = new UpdateMiRequest()
    req.mi = updated_mi
    const res = await props.gkill_api.update_mi(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('requested_reload_kyou', props.kyou)
    return
}
</script>
<style lang="css" scoped>
.mi_title_card {
    border: solid white 0px;
}
</style>