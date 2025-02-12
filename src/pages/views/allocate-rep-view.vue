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
                        <v-select :label="'デバイス名'" v-model="repository.device" :items="devices" />
                    </td>
                    <td>
                        <v-select v-model="repository.type" readonly :items="rep_types" label="RepType" />
                    </td>
                    <td>
                        <v-text-field :width="600" :label="'ファイルPath'" v-model="repository.file" />
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
        <AddRepDialog :application_config="application_config" :gkill_api="gkill_api" :server_configs="server_configs"
            :account="account" @requested_add_rep="add_rep"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" ref="add_rep_dialog" />
        <ConfirmDeleteRepDialog :application_config="application_config" :gkill_api="gkill_api"
            :rep_id="delete_target_rep ? delete_target_rep.id : ''" :server_configs="server_configs"
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
import { UpdateUserRepsRequest } from '@/classes/api/req_res/update-user-reps-request'
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
    "gpslog",
    "git_commit_log",
    "notification",
])

const devices: Ref<Array<string>> = ref((() => {
    const devices = Array<string>()
    for (let i = 0; i < props.server_configs.length; i++) {
        devices.push(props.server_configs[i].device)
    }
    return devices
})())

watch(() => props.server_configs, () => {
    update_repositories()

    const new_devices = Array<string>()
    for (let i = 0; i < props.server_configs.length; i++) {
        new_devices.push(props.server_configs[i].device)
    }
    devices.value = new_devices
})
watch(() => props.account, () => {
    update_repositories()
})
update_repositories()

function update_repositories(): void {
    const filtered_repository: Array<Repository> = new Array<Repository>()
    props.server_configs[0].repositories.forEach((repository) => {
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
    req.target_user_id = props.account.user_id
    req.updated_reps = repositories.value
    const res = await props.gkill_api.update_user_reps(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }

    emits('requested_reload_server_config')
    emits('requested_close_dialog')
}
</script>
