<template>
    <v-card class="kftl_view">
        <v-card-title :height="title_height">
            <v-row>
                <v-col cols="auto">
                    {{ i18n.global.t("KFTL_ADD_KYOU_TITLE") }}
                </v-col>
                <v-spacer />
                <v-col cols="auto">
                    <v-btn dark color="primary" @click="show_kftl_template_dialog" :disabled="is_requested_submit">{{
                        i18n.global.t("KFTL_TEMPLATE_TITLE") }}</v-btn>
                </v-col>
                <v-col cols="auto">
                    <v-btn dark color="primary" @click="submit" :disabled="is_requested_submit">{{
                        i18n.global.t("SAVE_TITLE")
                        }}</v-btn>
                </v-col>
            </v-row>
        </v-card-title>
        <table class="kftl_input">
            <tr>
                <td>
                    <div class="kftl_line_label line_label_wrap">
                        <KFTLLineLabel v-for="(line_label_data, index) in line_label_datas"
                            :key="line_label_data.target_request_id" :application_config="application_config"
                            :gkill_api="gkill_api" :line_label_data="line_label_data"
                            :style="line_label_styles[index]" />
                    </div>
                </td>
                <td>
                    <div class="kftl_text_area_wrap">
                        <textarea id="kftl_text_area" class="kftl_text_area" v-model="text_area_content"
                            :readonly="is_requested_submit" autofocus></textarea>
                    </div>
                </td>
            </tr>
        </table>
        <KFTLTemplateDialog :application_config="application_config" :gkill_api="gkill_api"
            :template="application_config.parsed_kftl_template"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @clicked_template_element_leaf="paste_template" ref="kftl_template_dialog" />
    </v-card>
</template>

<script setup lang="ts">
import { i18n } from '@/i18n'
import { computed, nextTick, onMounted, ref, watch, type Ref } from 'vue'
import { GkillError } from '@/classes/api/gkill-error'
import { LineLabelData } from '@/classes/kftl/line-label-data'

import type { KFTLProps } from './kftl-props'
import type { KFTLViewEmits } from './kftl-view-emits'

import KFTLLineLabel from './kftl-line-label.vue'
import KFTLTemplateDialog from '../dialogs/kftl-template-dialog.vue'
import { KFTLStatement } from '@/classes/kftl/kftl-statement'
import { TextAreaInfo } from '@/classes/kftl/text-area-info'
import { GkillMessage } from '@/classes/api/gkill-message'
import type { KFTLTemplateElementData } from '@/classes/datas/kftl-template-element-data'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'
import { DiscardTXRequest } from '@/classes/api/req_res/discard-tx-request'
import { CommitTXRequest } from '@/classes/api/req_res/commit-tx-request'

const kftl_template_dialog = ref<InstanceType<typeof KFTLTemplateDialog> | null>(null);

const text_area_content: Ref<string> = ref("")
const text_area_width: Ref<Number> = ref(0)
const text_area_width_px = computed(() => text_area_width.value.toString().concat("px"))
const text_area_height: Ref<Number> = ref(0)
const text_area_height_px = computed(() => text_area_height.value.toString().concat("px"))
const line_label_width: Ref<Number> = ref(0)
const line_label_width_px = computed(() => line_label_width.value.toString().concat("px"))
const line_label_height: Ref<Number> = ref(0)
const line_label_height_px = computed(() => line_label_height.value.toString().concat("px"))

const title_height = 52
const action_height = 10
const kftl_input_height: Ref<number> = ref(0)
const kftl_input_height_px = computed(() => kftl_input_height.value.toString().concat("px"))
const kftl_input_width: Ref<number> = ref(0)
const kftl_input_width_px = computed(() => kftl_input_width.value.toString().concat("px"))

const line_label_datas: Ref<Array<LineLabelData>> = ref(new Array<LineLabelData>())
const line_label_styles: Ref<Array<any>> = ref(new Array<any>())
const invalid_line_numbers: Ref<Array<Number>> = ref(new Array<Number>())
const is_requested_submit: Ref<boolean> = ref(true)

const props = defineProps<KFTLProps>()
const emits = defineEmits<KFTLViewEmits>()

if (props.application_config.is_loaded) {
    is_requested_submit.value = false
}
watch(() => props.application_config, () => {
    if (props.application_config.is_loaded) {
        is_requested_submit.value = false
    }
})
watch(() => text_area_content.value, () => {
    update_line_labels()
    save_content_to_localstorage()
})
watch(line_label_datas, async () => {
    line_label_styles.value.splice(0)
    let prev_target_id = ""
    let background_is_gray = true
    let switch_id = false
    let background_color: string = "white"
    for (let i = 0; i < line_label_datas.value.length; i++) {
        let color: string = "unset"
        switch_id = prev_target_id != line_label_datas.value[i].target_request_id
        if (switch_id) {
            background_is_gray = !background_is_gray
            if (background_is_gray) {
                if (props.application_config.use_dark_theme) {
                    background_color = '#C0C0C0'
                } else {
                    background_color = "#f0f0f0"
                }
            } else {
                background_color = ""
            }
        }
        if (is_invalid_line(i)) {
            color = "pink"
        }
        line_label_styles.value.push({
            "background-color": background_color,
            "color": color,
        })
        prev_target_id = line_label_datas.value[i].target_request_id
    }
})

nextTick(() => {
    const kftl_text_area_element_id = "kftl_text_area"
    const kftl_text_area_element = document.getElementById(kftl_text_area_element_id)!!
    kftl_text_area_element.addEventListener("scroll", update_line_labels)
    update_line_labels()
})

restore_content_from_localstorage()

async function restore_content_from_localstorage(): Promise<void> {
    const saved_content = localStorage.getItem("kftl_content")
    if (saved_content) {
        text_area_content.value = saved_content
    }
}

async function save_content_to_localstorage(): Promise<void> {
    localStorage.setItem("kftl_content", text_area_content.value)
}

async function update_line_labels(): Promise<void> {
    const kftl_text_area_element_id = "kftl_text_area"
    const kftl_text_area_element = document.getElementById(kftl_text_area_element_id)!!
    const kftl_line_label_elements = document.getElementsByClassName("kftl_line_label")!!
    for (let i = 0; i < kftl_line_label_elements.length; i++) {
        const kftl_line_label_element = kftl_line_label_elements.item(i)
        if (kftl_line_label_element) {
            kftl_line_label_element.scrollTo(0, kftl_text_area_element.scrollTop)
        }
    }

    const statement = new KFTLStatement(text_area_content.value)
    const textarea_info = new TextAreaInfo()
    textarea_info.text_area_element_id = kftl_text_area_element_id

    line_label_datas.value = statement.generate_line_label_data(textarea_info)

    invalid_line_numbers.value = await statement.get_invalid_line_indexs()

    if (text_area_content.value.endsWith("\n！\n") && !is_requested_submit.value) {
        is_requested_submit.value = true
        submit()
    }
}

function is_invalid_line(line_index: Number): boolean {
    for (let i = 0; i < invalid_line_numbers.value.length; i++) {
        if (invalid_line_numbers.value[i] == line_index) {
            return true
        }
    }
    return false
}

async function submit(): Promise<void> {
    try {
        if (invalid_line_numbers.value.length != 0) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.kftl_has_invalid_line
            error.error_message = i18n.global.t("KFTL_FOUND_INVALID_LINE_MESSAGE")
            emits('received_errors', [error])
            return
        }
        const statement = new KFTLStatement(text_area_content.value)
        const kftl_requests = await statement.generate_requests()
        let last_added_request_time = new Date(Date.now()) // 「、、」でずれた分をPlaingTimeIsにわたすための考慮。リロード時刻より大きかった場合はこの値でTimeIsをリロードする
        let errors = new Array<GkillError>()
        const tx_id = kftl_requests.length > 0 ? kftl_requests[0].get_tx_id() : null
        for (let i = 0; i < kftl_requests.length; i++) {
            const request = kftl_requests[i]
            const request_related_time = request.get_related_time()
            if (request_related_time && request_related_time.getTime() > last_added_request_time.getTime()) {
                last_added_request_time = request_related_time
            }
            await request.do_request().then(request_errors => errors = errors.concat(request_errors))
        }
        if (errors.length != 0) {
            emits('received_errors', errors)
            if (tx_id) {
                const deiscard_req = new DiscardTXRequest()
                deiscard_req.tx_id = tx_id
                const discard_res = await props.gkill_api.discard_tx(deiscard_req)
                if (discard_res.errors && discard_res.errors.length != 0) {
                    emits('received_errors', discard_res.errors)
                }
                return
            }
        }
        if (tx_id) {
            const commit_req = new CommitTXRequest()
            commit_req.tx_id = tx_id
            const commit_res = await props.gkill_api.commit_tx(commit_req)
            if (commit_res.errors && commit_res.errors.length != 0) {
                emits('received_errors', commit_res.errors)
                return
            }
        }

        clear()
        const message = new GkillMessage()
        message.message_code = GkillMessageCodes.saved_kftls
        message.message = i18n.global.t("SAVED_MESSAGE")
        emits('received_messages', [message])
        emits('saved_kyou_by_kftl', last_added_request_time)
    } finally {
        is_requested_submit.value = false
    }
}

async function clear(): Promise<void> {
    text_area_content.value = ""
}

async function show_kftl_template_dialog(): Promise<void> {
    kftl_template_dialog.value?.show()
}

async function resize(): Promise<void> {
    line_label_width.value = 100
    line_label_height.value = props.app_content_height.valueOf() - title_height - action_height
    text_area_width.value = props.app_content_width.valueOf() - line_label_width.value.valueOf() - 7 // 7はマジックナンバー
    text_area_height.value = props.app_content_height.valueOf() - title_height - action_height
    kftl_input_width.value = line_label_width.value.valueOf() + text_area_width.value.valueOf()
    kftl_input_height.value = props.app_content_height.valueOf() - title_height - action_height
}

function paste_template(template: KFTLTemplateElementData): void {
    text_area_content.value = template.template as string
    kftl_template_dialog.value?.hide()
}


window.addEventListener("resize", () => {
    resize()
    update_line_labels()
})
onMounted(() => resize())

</script>

<style lang="css" scoped>
.kftl_text_area_wrap {
    height: 100%;
    width: calc(v-bind(text_area_width_px));
}

.kftl_text_area {
    height: calc(v-bind(text_area_height_px));
    width: calc(v-bind(text_area_width_px));
    resize: none;
}

.line_label_wrap {
    color: rgb(var(--v-theme-background-focused));
    padding-right: 16px;
    height: calc(v-bind(line_label_height_px));
    width: calc(v-bind(line_label_width_px));
    text-align: right;
}

textarea {
    border: solid 1px silver;
}

.kftl_input {
    height: calc(v-bind(kftl_input_height_px));
    width: calc(v-bind(kftl_input_width_px));
    overflow-y: scroll;
}

.kftl_line_label {
    overflow-y: hidden;
}

.kftl_line_label::-webkit-scrollbar {
    display: none;
}

.kftl_view {
    overflow-y: hidden;
}
</style>