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
import type { CreateAccountViewEmits } from './create-account-view-emits'
import type { CreateAccountViewProps } from './create-account-view-props'
import { useCreateAccountView } from '@/classes/use-create-account-view'

const props = defineProps<CreateAccountViewProps>()
const emits = defineEmits<CreateAccountViewEmits>()

const {
    new_user_id,
    do_not_initialize,
    create_account,
} = useCreateAccountView({ props, emits })
</script>
