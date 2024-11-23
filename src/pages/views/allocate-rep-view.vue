<template>
    <v-card>
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>Rep割当管理</span>
                    <span>{{ account.user_id }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn color="primary" @click="show_add_rep_dialog(account)">追加</v-btn>
                </v-col>
            </v-row>
        </v-card-title>
        <v-card>
            <table>
                <tr v-for="repository in repositories" :key="repository.id">
                    <td>
                        <v-checkbox label="有効" v-model="repository.is_enable" />
                    </td>
                    <td>
                        <v-checkbox label="書込" v-model="repository.use_to_write" />
                    </td>
                    <td>
                        <v-checkbox label="ID自動割当" v-model="repository.is_execute_idf_when_reload" />
                    </td>
                    <td>
                        <v-textfield :label="'デバイス名'" v-model="repository.device" />
                    </td>
                    <td>
                        <v-select v-model="repository.type">
                            <option v-for="rep_type, index in rep_types">{{ rep_type }}</option>
                        </v-select>
                    </td>
                    <td>
                        <v-textfield :label="'ファイルPath'" v-model="repository.file" />
                    </td>
                    <td>
                        <v-btn @click="show_confirm_delete_rep_dialog(repository)">削除</v-btn>
                    </td>
                </tr>
            </table>
        </v-card>
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn @click="apply">適用</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn @click="emits('requested_close_dialog')">キャンセル</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
        <AddRepDialog :application_config="application_config" :gkill_api="gkill_api" :server_config="server_config"
            :account="account" @requested_add_rep="add_rep"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" ref="add_rep_dialog" />
        <ConfirmDeleteRepDialog :application_config="application_config" :gkill_api="gkill_api"
            :rep_id="delete_target_rep ? delete_target_rep.id : ''" :server_config="server_config"
            @requested_delete_rep="(rep) => delete_rep(rep)"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" ref="confirm_delete_rep_dialog" />
    </v-card>
</template>
<script setup lang="ts">
import { type Ref, ref, watch } from 'vue'
import type { AllocateRepViewEmits } from './allocate-rep-view-emits'
import type { AllocateRepViewProps } from './allocate-rep-view-props'
import AddRepDialog from '../dialogs/add-rep-dialog.vue'
import ConfirmDeleteRepDialog from '../dialogs/confirm-delete-rep-dialog.vue'
import type { Repository } from '@/classes/datas/config/repository'
import { Account } from '@/classes/datas/config/account'
import { GkillAPI } from '@/classes/api/gkill-api'
import { UpdateUserRepsRequest } from '@/classes/api/req_res/update-user-reps-request'
import { GetServerConfigRequest } from '@/classes/api/req_res/get-server-config-request'
const add_rep_dialog = ref<InstanceType<typeof AddRepDialog> | null>(null);
const confirm_delete_rep_dialog = ref<InstanceType<typeof ConfirmDeleteRepDialog> | null>(null);

const props = defineProps<AllocateRepViewProps>()
const emits = defineEmits<AllocateRepViewEmits>()

const delete_target_rep: Ref<Repository | null> = ref(null)
const repositories: Ref<Array<Repository>> = ref(new Array<Repository>())
const rep_types: Ref<Array<string>> = ref([
    "kmemo",
    "urlog",
    "timeis",
    "mi",
    "nlog",
    "lantana",
    "tag",
    "text",
    "rekyou",
    "directory",
])

watch(() => props.server_config, () => {
    update_repositories()
})
watch(() => props.account, () => {
    update_repositories()
})

function update_repositories(): void {
    const filtered_repository: Array<Repository> = new Array<Repository>()
    props.server_config.repositories.forEach((repository) => {
        if (repository.user_id === props.account.user_id) {
            filtered_repository.push(repository)
        }
    })
    repositories.value = filtered_repository
}

async function add_rep(rep: Repository): Promise<void> {
    repositories.value.push(rep)
}

async function delete_rep(rep: Repository): Promise<void> {
    for (let i = 0; i < repositories.value.length; i++) {
        if (repositories.value[i].id === rep.id) {
            repositories.value.splice(i, 1)
            break
        }
    }
}

function show_confirm_delete_rep_dialog(repository: Repository): void {
    confirm_delete_rep_dialog.value?.show(repository)
}

function show_add_rep_dialog(account: Account): void {
    add_rep_dialog.value?.show(account)
}

async function apply(): Promise<void> {
    const req = new UpdateUserRepsRequest()
    req.session_id = GkillAPI.get_instance().get_session_id()
    req.target_user_id = props.account.user_id
    req.updated_reps = repositories.value
    const res = await GkillAPI.get_instance().update_user_reps(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }

    const server_config_req = new GetServerConfigRequest()
    server_config_req.session_id = GkillAPI.get_instance().get_session_id()
    const server_config_res = await GkillAPI.get_instance().get_server_config(server_config_req)
    if (server_config_res.errors && server_config_res.errors.length !== 0) {
        emits('received_errors', server_config_res.errors)
        return
    }
    if (server_config_res.messages && server_config_res.messages.length !== 0) {
        emits('received_messages', server_config_res.messages)
    }

    emits('requested_reload_server_config', server_config_res.server_config)
    emits('requested_close_dialog')
}
</script>
