<template>
    <div>
        <miSharedTaskView :app_content_height="app_content_height" :app_content_width="app_content_width"
            :app_title_bar_height="app_title_bar_height" :share_id="share_mi_id"
            :application_config="new ApplicationConfig()" :gkill_api="gkill_api" @received_errors="write_errors"
            @received_messages="write_messages" />
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
import { computed, ref, type Ref } from 'vue'
import { GkillAPI, GkillAPIForSharedMi } from '@/classes/api/gkill-api'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

import miSharedTaskView from './views/mi-shared-task-view.vue'
import { useRoute } from 'vue-router'
import { ApplicationConfig } from '@/classes/datas/config/application-config'

const actual_height: Ref<number> = ref(0)
const element_height: Ref<number> = ref(0)
const browser_url_bar_height: Ref<number> = ref(0)
const app_title_bar_height: Ref<number> = ref(50)
const gkill_api: Ref<GkillAPI> = computed(() => GkillAPIForSharedMi.get_instance_for_share_mi())
const app_content_height: Ref<number> = ref(0)
const app_content_width: Ref<number> = ref(0)
const share_mi_id = computed(() => useRoute().query.share_id!.toString())

GkillAPI.set_gkill_api(GkillAPIForSharedMi.get_instance())

async function resize_content(): Promise<void> {
    const inner_element = document.querySelector('#control-height')
    actual_height.value = window.innerHeight
    element_height.value = inner_element ? inner_element.clientHeight : actual_height.value
    browser_url_bar_height.value = (Number(element_height.value) - Number(actual_height.value)).valueOf()
    app_content_height.value = (Number(element_height.value) - (Number(browser_url_bar_height.value) + Number(app_title_bar_height.value))).valueOf()
    app_content_width.value = window.innerWidth
}

const messages: Ref<Array<{ message: string, id: string, show_snackbar: boolean }>> = ref([])

async function write_errors(errors: Array<GkillError>) {
    const received_messages = new Array<{ message: string, id: string, show_snackbar: boolean }>()
    for (let i = 0; i < errors.length; i++) {
        if (errors[i] && errors[i].error_message) {
            received_messages.push({
                message: errors[i].error_message,
                id: gkill_api.value.generate_uuid(),
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
                id: gkill_api.value.generate_uuid(),
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
