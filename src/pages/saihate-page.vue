<template>
    <div class="saihate_view_wrap">
        <v-app-bar :height="app_title_bar_height.valueOf()" class="app_bar" color="primary" app flat>
            <v-btn icon="mdi-menu" :ripple="false" link="false" :style="{ opacity: 0, cursor: 'unset', }" />
            <v-toolbar-title>さいはて</v-toolbar-title>
        </v-app-bar>
        <v-main class="main">
            <div class="overlay_target">
                <v-overlay v-model="is_loading" class="align-center justify-center" persistent contained>
                    <v-progress-circular indeterminate color="primary" />
                </v-overlay>
            </div>
            <v-avatar :style="floatingActionButtonStyle()" color="primary" class="position-fixed">
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
            <AddTimeisDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :last_added_tag="''" :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" @received_errors="write_errors" @received_messages="write_messages"
                ref="add_timeis_dialog" />
            <AddLantanaDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :last_added_tag="''" :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" @received_errors="write_errors" @received_messages="write_messages"
                ref="add_lantana_dialog" />
            <AddUrlogDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :last_added_tag="''" :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" @received_errors="write_errors" @received_messages="write_messages"
                ref="add_urlog_dialog" />
            <AddMiDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :last_added_tag="''" :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" @received_errors="write_errors" @received_messages="write_messages"
                ref="add_mi_dialog" />
            <AddNlogDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :last_added_tag="''" :kyou="new Kyou()" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" @received_errors="write_errors" @received_messages="write_messages"
                ref="add_nlog_dialog" />
            <kftlDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
                :last_added_tag="''" :kyou="new Kyou()" :app_content_height="app_content_height"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                :app_content_width="app_content_width" @received_errors="write_errors"
                @received_messages="write_messages" ref="kftl_dialog" />
            <UploadFileDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
                :application_config="application_config" :gkill_api="gkill_api" :last_added_tag="''"
                @received_errors="write_errors" @received_messages="write_messages" ref="upload_file_dialog" />
        </v-main>
        <div class="alert_container">
            <v-slide-y-transition group>
                <v-alert v-for="message in messages" theme="dark" :key="message.id">
                    {{ message.message }}
                </v-alert>
            </v-slide-y-transition>
        </div>
    </div>
</template>

<script lang="ts" setup>
'use strict'
import { computed, ref, watch, type Ref } from 'vue'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { GkillAPI } from '@/classes/api/gkill-api'
import { GetApplicationConfigRequest } from '@/classes/api/req_res/get-application-config-request'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

import { Kyou } from '@/classes/datas/kyou'
import AddTimeisDialog from './dialogs/add-timeis-dialog.vue'
import AddLantanaDialog from './dialogs/add-lantana-dialog.vue'
import AddUrlogDialog from './dialogs/add-urlog-dialog.vue'
import AddMiDialog from './dialogs/add-mi-dialog.vue'
import AddNlogDialog from './dialogs/add-nlog-dialog.vue'
import kftlDialog from './dialogs/kftl-dialog.vue'
import UploadFileDialog from './dialogs/upload-file-dialog.vue'

const add_mi_dialog = ref<InstanceType<typeof AddMiDialog> | null>(null);
const add_nlog_dialog = ref<InstanceType<typeof AddNlogDialog> | null>(null);
const add_lantana_dialog = ref<InstanceType<typeof AddLantanaDialog> | null>(null);
const add_timeis_dialog = ref<InstanceType<typeof AddTimeisDialog> | null>(null);
const add_urlog_dialog = ref<InstanceType<typeof AddUrlogDialog> | null>(null);
const kftl_dialog = ref<InstanceType<typeof kftlDialog> | null>(null);
const upload_file_dialog = ref<InstanceType<typeof UploadFileDialog> | null>(null);

const enable_context_menu = ref(true)
const enable_dialog = ref(false)

const actual_height: Ref<Number> = ref(0)
const element_height: Ref<Number> = ref(0)
const browser_url_bar_height: Ref<Number> = ref(0)
const app_title_bar_height: Ref<Number> = ref(50)
const gkill_api = computed(() => GkillAPI.get_instance())
const application_config: Ref<ApplicationConfig> = ref(new ApplicationConfig())
const app_content_height: Ref<Number> = ref(0)
const app_content_width: Ref<Number> = ref(0)

async function show_dialog(): Promise<void> {
    const dialog = new URL(location.href).searchParams.get('dialog')
    switch (dialog) {
        case 'timeis':
            show_timeis_dialog()
            break
        case 'mi':
            show_mi_dialog()
            break
        case 'nlog':
            show_nlog_dialog()
            break
        case 'lantana':
            show_lantana_dialog()
            break
        case 'urlog':
            show_urlog_dialog()
            break
        case 'kftl':
            show_kftl_dialog()
            break
        case 'file':
            show_upload_file_dialog()
            break
        default:
            break
    }
}

async function load_application_config(): Promise<void> {
    const req = new GetApplicationConfigRequest()

    return gkill_api.value.get_application_config(req)
        .then(res => {
            if (res.errors && res.errors.length != 0) {
                write_errors(res.errors)
                return
            }

            application_config.value = res.application_config
            GkillAPI.get_instance().set_saved_application_config(res.application_config)

            if (res.messages && res.messages.length != 0) {
                write_messages(res.messages)
                return
            }
        })
}

async function resize_content(): Promise<void> {
    const inner_element = document.querySelector('#control-height')
    actual_height.value = window.innerHeight
    element_height.value = inner_element ? inner_element.clientHeight : actual_height.value
    browser_url_bar_height.value = Number(element_height.value) - Number(actual_height.value)
    app_content_height.value = Number(element_height.value) - (Number(browser_url_bar_height.value) + Number(app_title_bar_height.value))
    app_content_width.value = window.innerWidth
}

const messages: Ref<Array<{ message: string, id: string, show_snackbar: boolean }>> = ref([])

async function write_errors(errors: Array<GkillError>) {
    const received_messages = new Array<{ message: string, id: string, show_snackbar: boolean }>()
    for (let i = 0; i < errors.length; i++) {
        if (errors[i] && errors[i].error_message) {
            received_messages.push({
                message: errors[i].error_message,
                id: GkillAPI.get_instance().generate_uuid(),
                show_snackbar: true,
            })
        }
    }
    messages.value.push(...received_messages)
    sleep(2500).then(() => {
        for (let i = 0; i < received_messages.length; i++) {
            messages.value.splice(0, 1)
        }
    })
}

async function write_messages(messages_: Array<GkillMessage>) {
    const received_messages = new Array<{ message: string, id: string, show_snackbar: boolean }>()
    for (let i = 0; i < messages_.length; i++) {
        if (messages_[i] && messages_[i].message) {
            received_messages.push({
                message: messages_[i].message,
                id: GkillAPI.get_instance().generate_uuid(),
                show_snackbar: true,
            })
        }
    }
    messages.value.push(...received_messages)
    sleep(2500).then(() => {
        for (let i = 0; i < received_messages.length; i++) {
            messages.value.splice(0, 1)
        }
    })
}

const sleep = (time: number) => new Promise<void>((r) => setTimeout(r, time))

const is_loading = ref(true)
watch(() => application_config.value, () => {
    is_loading.value = false
})
watch(() => is_loading.value, (new_value: boolean, old_value: boolean) => {
    if (old_value !== new_value && !new_value) {
        show_dialog()
    }
})

window.addEventListener('resize', () => {
    resize_content()
})

resize_content()
load_application_config()

function floatingActionButtonStyle() {
    return {
        'bottom': '60px',
        'right': '10px',
        'height': '50px',
        'width': '50px'
    }
}

const position_x: Ref<Number> = ref(0)
const position_y: Ref<Number> = ref(0)
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
/* 不要なスクロールバーを消す */
body,
.v-application--wrap,
.v-navigation-drawer--open {
    overflow-y: scroll !important;
    overflow-x: auto !important;
    height: calc(actual_height) !important;
    min-height: calc(actual_height) !important;
    max-height: calc(actual_height) !important;
}

body {
    overflow-y: hidden !important;
}

body::-webkit-scrollbar {
    display: none;
}

/* メッセージ、エラーメッセージ */
.alert_container {
    position: fixed;
    top: 60px;
    right: 10px;
    display: grid;
    grid-gap: .5em;
    z-index: 100000000;
}

/* ダイアログ */
.kyou_detail_view,
.kyou_list_view,
.v-dialog .v-card {
    overflow-y: scroll;
}

/* スクロールバー */
.v-navigation-drawer__content::-webkit-scrollbar,
.kyou_detail_view::-webkit-scrollbar,
.kyou_list_view::-webkit-scrollbar,
.kyou_list_view_image::-webkit-scrollbar,
.aggregate_amount_list::-webkit-scrollbar,
.aggregate_location_list::-webkit-scrollbar,
.aggregate_people_list::-webkit-scrollbar,
.kftl_text_area::-webkit-scrollbar,
.v-dialog .v-card::-webkit-scrollbar {
    margin-left: 1px;
    width: 8px;
}

.v-navigation-drawer__content::-webkit-scrollbar-thumb,
.kyou_detail_view::-webkit-scrollbar-thumb,
.kyou_list_view::-webkit-scrollbar-thumb,
.kyou_list_view_image::-webkit-scrollbar-thumb,
.aggregate_amount_list::-webkit-scrollbar-thumb,
.aggregate_location_list::-webkit-scrollbar-thumb,
.aggregate_people_list::-webkit-scrollbar-thumb,
.kftl_text_area::-webkit-scrollbar-thumb,
.v-dialog .v-card::-webkit-scrollbar-thumb {
    background: rgb(var(--v-theme-primary));
    width: 6px;
    border-radius: 5px;
}

/* テーブルの隙間埋め */
table,
tr,
td {
    border-spacing: 0 !important;
}
</style>
<style lang="css" scoped>
.overlay_target {
    z-index: -10000;
    position: absolute;
    min-height: calc(v-bind('app_content_height.toString().concat("px")'));
    min-width: calc(100vw);
}
</style>