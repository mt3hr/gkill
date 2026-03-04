<template>
    <v-card>
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("ADD_ACCOUNT_TITLE") }}</span>
                </v-col>
            </v-row>
        </v-card-title>
        <v-text-field v-model="new_user_id" :label="i18n.global.t('USER_ID_TITLE')" />
        <v-checkbox v-model="do_not_initialize" :label="i18n.global.t('DO_INITIALIZE_ADD_ACCOUNT_MESSAGE')" />
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark @click="create_account" color="primary">{{ i18n.global.t("ADD_ACCOUNT_TITLE") }}</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="emits('requested_close_dialog')">{{ i18n.global.t("CANCEL_TITLE")
                        }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import type { CreateAccountViewEmits } from './create-account-view-emits'
import type { CreateAccountViewProps } from './create-account-view-props'
import { AddAccountRequest } from '@/classes/api/req_res/add-account-request';

const props = defineProps<CreateAccountViewProps>()
const emits = defineEmits<CreateAccountViewEmits>()
const new_user_id: Ref<string> = ref("")
const do_not_initialize: Ref<boolean> = ref(false)

async function create_account(): Promise<void> {
    const req = new AddAccountRequest()
    req.account_info.is_enable = true
    req.account_info.is_admin = false
    req.account_info.password_reset_token = null
    req.account_info.user_id = new_user_id.value
    req.do_initialize = !do_not_initialize.value

    const res = await props.gkill_api.add_account(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }

    emits('created_account', res.added_account_info)
    emits('requested_reload_server_config')
    emits('requested_close_dialog')
}
</script>
