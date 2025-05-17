<template>
    <v-card @contextmenu.prevent="show_context_menu" :width="width" :height="height">
        <v-row v-if="kyou.typed_mi" class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0" :style="mi_title_style">
                <table class="pa-0 ma-0">
                    <tr>
                        <td class="pa-0 ma-0">
                            <v-checkbox v-model="is_checked_mi" hide-details @click="clicked_mi_check()" />
                        </td>
                        <td class="pa-0 ma-0">
                            <div class="py-1 mi_title">{{ kyou.typed_mi.title }}</div>
                        </td>
                    </tr>
                </table>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-card-title>
                    <div class="py-1 mi_board_name">{{ kyou.typed_mi.board_name }}</div>
                </v-card-title>
            </v-col>
        </v-row>
        <div :style="{ 'padding-top': '30px' }">
            <div v-if="kyou.typed_mi && kyou.typed_mi.estimate_start_time">
                <span>{{ i18n.global.t("MI_START_DATE_TIME_TITLE") }}：</span>
                <span>{{ format_time(kyou.typed_mi.estimate_start_time) }}</span>
            </div>
            <div v-if="kyou.typed_mi && kyou.typed_mi.estimate_end_time">
                <span>{{ i18n.global.t("MI_END_DATE_TIME_TITLE") }}：</span>
                <span>{{ format_time(kyou.typed_mi.estimate_end_time) }}</span>
            </div>
            <div v-if="kyou.typed_mi && kyou.typed_mi.limit_time">
                <span>{{ i18n.global.t("MI_LIMIT_DATE_TIME_TITLE") }}：</span>
                <span>{{ format_time(kyou.typed_mi.limit_time) }}</span>
            </div>
        </div>
        <MiContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            @deleted_kyou="(deleted_kyou) => emits('deleted_kyou', deleted_kyou)"
            @deleted_tag="(deleted_tag) => emits('deleted_tag', deleted_tag)"
            @deleted_text="(deleted_text) => emits('deleted_text', deleted_text)"
            @deleted_notification="(deleted_notification) => emits('deleted_notification', deleted_notification)"
            @registered_kyou="(registered_kyou) => emits('registered_kyou', registered_kyou)"
            @registered_tag="(registered_tag) => emits('registered_tag', registered_tag)"
            @registered_text="(registered_text) => emits('registered_text', registered_text)"
            @registered_notification="(registered_notification) => emits('registered_notification', registered_notification)"
            @updated_kyou="(updated_kyou) => emits('updated_kyou', updated_kyou)"
            @updated_tag="(updated_tag) => emits('updated_tag', updated_tag)"
            @updated_text="(updated_text) => emits('updated_text', updated_text)"
            @updated_notification="(updated_notification) => emits('updated_notification', updated_notification)"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => emits('requested_update_check_kyous', kyous, is_checked)"
            ref="context_menu" />
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { nextTick, type Ref, ref, watch } from 'vue'
import MiContextMenu from './mi-context-menu.vue'
import type { Kyou } from '@/classes/datas/kyou'
import type { miKyouViewProps } from './mi-kyou-view-props'
import moment from 'moment'
import { GkillError } from '@/classes/api/gkill-error'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import { UpdateMiRequest } from '@/classes/api/req_res/update-mi-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import type { KyouViewEmits } from './kyou-view-emits'

const context_menu = ref<InstanceType<typeof MiContextMenu> | null>(null);

const props = defineProps<miKyouViewProps>()
const emits = defineEmits<KyouViewEmits>()
defineExpose({ show_context_menu })

const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
const is_checked_mi: Ref<boolean> = ref(props.kyou.typed_mi ? props.kyou.typed_mi.is_checked : false)

const mi_title_style = ref({
    maxWidth: 'calc(100% - 0px)'
})

load_cloned_kyou()
nextTick(() => update_mi_title_style())

watch(() => props.kyou, async () => {
    await load_cloned_kyou()
    nextTick(() => update_mi_title_style())
    is_checked_mi.value = cloned_kyou.value.typed_mi ? cloned_kyou.value.typed_mi.is_checked : false
})

function update_mi_title_style(): void {
    mi_title_style.value.maxWidth = `calc(100% - ${get_board_name_text_width_px()}px)`
}

async function load_cloned_kyou() {
    const kyou = props.kyou.clone()
    await kyou.load_typed_datas()
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
        error.error_code = GkillErrorCodes.client_mi_is_null
        error.error_message = "クライアントのデータが変です"
        const errors = new Array<GkillError>()
        errors.push(error)
        emits('received_errors', errors)
        return
    }

    // 更新がなかったらエラーメッセージを出力する
    if (mi.is_checked === is_checked_mi.value) {
        const error = new GkillError()
        error.error_code = GkillErrorCodes.mi_is_no_update
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

function get_board_name_text_width_px(): number {
    const mi_board_name_element = document.querySelector(".kyou_".concat(props.kyou.id).concat(" ").concat(".mi_board_name"))
    if (mi_board_name_element == null) {
        return 0
    }
    const text_width = get_text_width(props.kyou.typed_mi?.board_name, get_canvas_font(mi_board_name_element as HTMLElement)).valueOf() + 16 + 16 + 4 // padding + padding + 4px
    return text_width
}

function get_text_width(text: any, font: any): Number {
    const canvas: any = (get_text_width as any).canvas || ((get_text_width as any).canvas = document.createElement("canvas"))
    const context = canvas.getContext("2d")
    context.font = font
    const metrics = context.measureText(text)
    return metrics.width
}

function get_css_style(element: any, prop: any): string {
    return window.getComputedStyle(element, null).getPropertyValue(prop)
}

function get_canvas_font(element = document.body): string {
    const fontWeight = get_css_style(element, 'font-weight') || 'normal'
    const fontSize = get_css_style(element, 'font-size') || '16px'
    const fontFamily = get_css_style(element, 'font-family') || 'Times New Roman'
    return `${fontWeight} ${fontSize} ${fontFamily}`
}
</script>
<style lang="css" scoped>
.mi_title_card {
    border: solid white 0px;
}
</style>