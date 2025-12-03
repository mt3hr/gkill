<template>
    <div>
        <rykvView :app_content_height="app_content_height" :app_content_width="app_content_width"
            :app_title_bar_height="app_title_bar_height" :application_config="application_config" :gkill_api="gkill_api"
            :is_shared_rykv_view="true" :share_title="share_title" @received_errors="(...errors :any[]) => write_errors(errors)"
            @received_messages="(...messages :any[]) => write_messages(messages)" />
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
import { i18n } from '@/i18n'
'use strict'
import { ref, type Ref } from 'vue'
import { GkillAPI } from '@/classes/api/gkill-api'
import { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

import rykvView from './views/rykv-view.vue'
import type { KyouViewEmits } from './views/kyou-view-emits'
import type { SharedRYKVPageProps } from './shared-rykv-page-props'

defineProps<SharedRYKVPageProps>()
defineEmits<KyouViewEmits>()

const actual_height: Ref<Number> = ref(0)
const element_height: Ref<Number> = ref(0)
const browser_url_bar_height: Ref<Number> = ref(0)
const app_title_bar_height: Ref<Number> = ref(50)
const app_content_height: Ref<Number> = ref(0)
const app_content_width: Ref<Number> = ref(0)

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

window.addEventListener('resize', () => {
    resize_content()
})

resize_content()

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
.tag_struct_root::-webkit-scrollbar,
.rep_struct_root::-webkit-scrollbar,
.rep_type_struct_root::-webkit-scrollbar,
.device_struct_root::-webkit-scrollbar,
.kftl_template_struct_root::-webkit-scrollbar,
.v-navigation-drawer__content::-webkit-scrollbar,
.kyou_detail_view::-webkit-scrollbar,
.kyou_list_view::-webkit-scrollbar,
.kyou_list_view_image::-webkit-scrollbar,
.dnote_list_view::-webkit-scrollbar,
.kftl_text_area::-webkit-scrollbar,
.v-dialog .v-card::-webkit-scrollbar {
    margin-left: 1px;
    width: 8px;
    height: 8px;
}

.tag_struct_root::-webkit-scrollbar-thumb,
.rep_struct_root::-webkit-scrollbar-thumb,
.rep_type_struct_root::-webkit-scrollbar-thumb,
.device_struct_root::-webkit-scrollbar-thumb,
.kftl_template_struct_root::-webkit-scrollbar-thumb,
.v-navigation-drawer__content::-webkit-scrollbar-thumb,
.kyou_detail_view::-webkit-scrollbar-thumb,
.ryuu_view::-webkit-scrollbar-thumb,
.kyou_list_view::-webkit-scrollbar-thumb,
.kyou_list_view_image::-webkit-scrollbar-thumb,
.dnote_list_view::-webkit-scrollbar-thumb,
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

.gkill_context_menu_list {
    max-height: 70vh;
    overflow-y: scroll;
}
</style>