<template>
    <v-card @contextmenu.prevent="show_context_menu">
        <v-row v-if="kyou.typed_mi" class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-checkbox v-model="is_checked_mi" hide-details @click="clicked_mi_check()" />
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <v-card-title>
                    {{ kyou.typed_mi.title }}
                </v-card-title>
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <v-card-title>
                    {{ kyou.typed_mi.board_name }}
                </v-card-title>
            </v-col>
        </v-row>
        <div v-if="kyou.typed_mi && kyou.typed_mi.estimate_start_time">
            開始日時：<span>{{ format_time(kyou.typed_mi.estimate_start_time) }}</span>
        </div>
        <div v-if="kyou.typed_mi && kyou.typed_mi.estimate_end_time">
            終了日時： <span>{{ format_time(kyou.typed_mi.estimate_end_time) }}</span>
        </div>
        <div v-if="kyou.typed_mi && kyou.typed_mi.limit_time">
            期限日時： <span>{{ format_time(kyou.typed_mi.limit_time) }}</span>
        </div>
    </v-card>
    <MiContextMenu :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[kyou.generate_info_identifer()]" :kyou="kyou"
        :last_added_tag="last_added_tag" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => emits('requested_update_check_kyous', kyous, is_checked)"
        ref="context_menu" />
</template>
<script lang="ts" setup>
import type { Mi } from '@/classes/datas/mi'
import { nextTick, type Ref, ref } from 'vue'
import MiContextMenu from './mi-context-menu.vue'
import type { Kyou } from '@/classes/datas/kyou'
import type { miKyouViewProps } from './mi-kyou-view-props'
import type { miKyouViewEmits } from './mi-kyou-view-emits'
import moment from 'moment'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import { UpdateMiRequest } from '@/classes/api/req_res/update-mi-request'
import router from '@/router'

const context_menu = ref<InstanceType<typeof MiContextMenu> | null>(null);

const props = defineProps<miKyouViewProps>()
const emits = defineEmits<miKyouViewEmits>()

const is_checked_mi: Ref<boolean> = ref(props.kyou.typed_mi ? props.kyou.typed_mi.is_checked : false)

function format_time(time: Date): string {
    return moment(time).format("yyyy-MM-DD HH:mm:ss")
}

function show_context_menu(e: PointerEvent): void {
    context_menu.value?.show(e)
}

async function clicked_mi_check(): Promise<void> {
    // データがちゃんとあるか確認。なければエラーメッセージを出力する
    const mi = props.kyou.typed_mi
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

    // セッションIDを取得する
    const session_id = window.localStorage.getItem("gkill_session_id")
    if (!session_id) {
        window.localStorage.removeItem("gkill_session_id")
        router.replace('/login')
        return
    }
    // UserIDやDevice情報を取得する
    const get_gkill_req = new GetGkillInfoRequest()
    get_gkill_req.session_id = session_id
    const gkill_info_res = await props.gkill_api.get_gkill_info(get_gkill_req)
    if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
        emits('received_errors', gkill_info_res.errors)
        return
    }

    // 更新後mi情報を用意する
    const updated_mi = await mi.clone()
    updated_mi.is_checked = !is_checked_mi.value
    updated_mi.update_app = "gkill"
    updated_mi.update_device = gkill_info_res.device
    updated_mi.update_time = new Date(Date.now())
    updated_mi.update_user = gkill_info_res.user_id

    // 更新リクエストを飛ばす
    const req = new UpdateMiRequest()
    req.session_id = session_id
    req.mi = updated_mi
    const res = await props.gkill_api.update_mi(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('updated_kyou', res.updated_mi_kyou)
    return
}
</script>
<style lang="css" scoped>
.mi_title_card {
    border: solid white 0px;
}
</style>