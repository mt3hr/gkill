import { i18n } from '@/i18n'
import { type Ref, ref, watch } from 'vue'
import type { AllocateRepViewEmits } from '@/pages/views/allocate-rep-view-emits'
import type { AllocateRepViewProps } from '@/pages/views/allocate-rep-view-props'
import type { Repository } from '@/classes/datas/config/repository'
import { Account } from '@/classes/datas/config/account'
import { UpdateUserRepsRequest } from '@/classes/api/req_res/update-user-reps-request'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

export function useAllocateRepView(options: {
    props: AllocateRepViewProps,
    emits: AllocateRepViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const add_rep_dialog = ref<any>(null)
    const confirm_delete_rep_dialog = ref<any>(null)

    // ── State refs ──
    const delete_target_rep: Ref<Repository | null> = ref(null)
    const repositories: Ref<Array<Repository>> = ref(new Array<Repository>())
    const rep_types: Ref<Array<string>> = ref([
        "kmemo",
        "kc",
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

    // ── Watchers ──
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

    // ── Init ──
    update_repositories()

    // ── Business logic ──
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

    // ── Template event handlers ──
    function onClickAddRep(): void {
        show_add_rep_dialog(props.account)
    }

    function onRequestedCloseDialog(): void {
        emits('requested_close_dialog')
    }

    // ── Event relay objects ──
    const addRepHandlers = {
        'requested_add_rep': (...args: any[]) => add_rep(args[0] as Repository),
        'received_errors': (...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>),
        'received_messages': (...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>),
    }

    const confirmDeleteRepHandlers = {
        'requested_delete_rep': (...rep: any) => delete_rep(rep as Repository),
        'received_errors': (...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>),
        'received_messages': (...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>),
    }

    return {
        // Template refs
        add_rep_dialog,
        confirm_delete_rep_dialog,

        // State
        delete_target_rep,
        repositories,
        rep_types,
        devices,

        // Business logic
        update_repositories,
        add_rep,
        delete_rep,
        show_confirm_delete_rep_dialog,
        show_add_rep_dialog,
        apply,

        // Template event handlers
        onClickAddRep,
        onRequestedCloseDialog,

        // Event relay objects
        addRepHandlers,
        confirmDeleteRepHandlers,
    }
}
