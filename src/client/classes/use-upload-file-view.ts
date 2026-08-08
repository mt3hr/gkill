import { nextTick, type Ref, ref } from 'vue'
import type { UploadFileViewProps } from '@/pages/views/upload-file-view-props'
import type { UploadFileViewEmits } from '@/pages/views/upload-file-view-emits'
import type { Kyou } from '@/classes/datas/kyou'
import type { Repository } from '@/classes/datas/config/repository'
import { FileUploadConflictBehavior } from '@/classes/api/req_res/file-upload-conflict-behavior'
import { UploadFilesRequest } from '@/classes/api/req_res/upload-files-request'
import { UploadGPSLogFilesRequest } from '@/classes/api/req_res/upload-gps-log-files-request'
import { FileData } from '@/classes/api/file-data'
import { GetRepositoriesRequest } from '@/classes/api/req_res/get-repositories-request'
import type DecideRelatedTimeUploadedFileDialog from '@/pages/dialogs/decide-related-time-uploaded-file-dialog.vue'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

export function useUploadFileView(options: {
    props: UploadFileViewProps,
    emits: UploadFileViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const decide_related_time_uploaded_file_dialog = ref<InstanceType<typeof DecideRelatedTimeUploadedFileDialog> | null>(null)
    const file_input = ref<HTMLInputElement | null>(null)
    const gps_file_input = ref<HTMLInputElement | null>(null)

    // ── State refs ──
    const is_requested_submit = ref(false)
    const tab = ref(2)
    const conflict_behavior_file: Ref<FileUploadConflictBehavior> = ref(FileUploadConflictBehavior.rename)
    const conflict_behavior_gps_file: Ref<FileUploadConflictBehavior> = ref(FileUploadConflictBehavior.merge)
    const target_rep_names_for_file: Ref<Array<string>> = ref(new Array<string>())
    const target_rep_name_for_file: Ref<string> = ref("")
    const target_rep_names_for_gps_file: Ref<Array<string>> = ref(new Array<string>())
    const target_rep_name_for_gps_file: Ref<string> = ref("")
    const gps_log_files: Ref<File | File[] | null> = ref(null)
    const files: Ref<File | File[] | null> = ref(null)
    const uploaded_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
    const is_dragging_over_file: Ref<boolean> = ref(false)
    const is_dragging_over_gps_file: Ref<boolean> = ref(false)

    // ── Init ──
    nextTick(() => load_target_rep_names())

    // ── Business logic ──
    async function load_target_rep_names(): Promise<void> {
        target_rep_names_for_file.value.splice(0)
        target_rep_names_for_gps_file.value.splice(0)
        const req = new GetRepositoriesRequest()
        const res = await props.gkill_api.get_repositories(req)
        if (res.errors && res.errors.length != 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length != 0) {
            // emits('received_messages', res.messages)
        }
        res.repositories?.forEach((rep: Repository) => {
            if (rep.type == "directory" && rep.is_enable && rep.use_to_write) {
                if (target_rep_name_for_file.value === "") {
                    target_rep_name_for_file.value = rep.rep_name
                }
                target_rep_names_for_file.value.push(rep.rep_name)
            }
            if (rep.type == "gpslog" && rep.is_enable && rep.use_to_write) {
                if (target_rep_name_for_gps_file.value === "") {
                    target_rep_name_for_gps_file.value = rep.rep_name
                }
                target_rep_names_for_gps_file.value.push(rep.rep_name)
            }
        })
    }

    // アップロードはbase64化に時間がかかるぶん、待っている間に次を落とされやすい。
    // 二重に走らせると同じファイルが2つ登録されるので直列化する
    async function upload_files(): Promise<void> {
        if (is_requested_submit.value) {
            return
        }
        is_requested_submit.value = true
        try {
            await execute_upload_files()
        } finally {
            is_requested_submit.value = false
        }
    }

    async function execute_upload_files(): Promise<void> {
        if (!files.value) {
            return
        }
        let files_value = files.value
        if (files_value instanceof File) {
            files_value = [files_value]
        }
        const req = new UploadFilesRequest()
        req.conflict_behavior = conflict_behavior_file.value
        req.target_rep_name = target_rep_name_for_file.value
        req.files = new Array<FileData>()
        for (let i = 0; i < files_value.length; i++) {
            const file = files_value[i]
            const filedata = new FileData()
            filedata.data_base64 = await to_base64(file)
            filedata.file_name = file.name
            filedata.last_modified = new Date(file.lastModified)
            req.files.push(filedata)
        }

        const res = await props.gkill_api.upload_files(req)
        if (res.errors && res.errors.length != 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length != 0) {
            emits('received_messages', res.messages)
        }

        uploaded_kyous.value.splice(0)
        for (let i = 0; i < res.uploaded_kyous.length; i++) {
            const uploaded_kyou = res.uploaded_kyous[i]
            await uploaded_kyou.reload(true)
            uploaded_kyous.value.push(uploaded_kyou)
        }
        files.value = null
        decide_related_time_uploaded_file_dialog.value?.show()
    }

    async function upload_gps_log_files(): Promise<void> {
        if (is_requested_submit.value) {
            return
        }
        is_requested_submit.value = true
        try {
            await execute_upload_gps_log_files()
        } finally {
            is_requested_submit.value = false
        }
    }

    async function execute_upload_gps_log_files(): Promise<void> {
        if (!gps_log_files.value) {
            return
        }
        let gps_log_files_value = gps_log_files.value
        if (gps_log_files_value instanceof File) {
            gps_log_files_value = [gps_log_files_value]
        }
        const req = new UploadGPSLogFilesRequest()
        req.conflict_behavior = conflict_behavior_gps_file.value
        req.target_rep_name = target_rep_name_for_gps_file.value
        req.gps_log_files = new Array<FileData>()
        for (let i = 0; i < gps_log_files_value.length; i++) {
            const gps_log_file = gps_log_files_value[i]
            const filedata = new FileData()
            filedata.data_base64 = await to_base64(gps_log_file)
            filedata.file_name = gps_log_file.name
            req.gps_log_files.push(filedata)
        }

        const res = await props.gkill_api.upload_gpslog_files(req)
        if (res.errors && res.errors.length != 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length != 0) {
            emits('received_messages', res.messages)
        }
        gps_log_files.value = null
    }

    async function to_base64(file: File): Promise<string> {
        return new Promise((resolve, reject) => {
            const reader = new FileReader();
            reader.readAsDataURL(file);
            reader.onload = () => resolve(reader.result as string);
            reader.onerror = (error) => reject(error);
        })
    }

    function onDragenterFile(e: DragEvent): void {
        e.preventDefault()
        e.stopPropagation()
        if (!e.dataTransfer?.types.includes('Files')) return
        is_dragging_over_file.value = true
    }

    function onDragleaveFile(e: DragEvent): void {
        e.preventDefault()
        e.stopPropagation()
        is_dragging_over_file.value = false
    }

    function onDragoverFile(e: DragEvent): void {
        e.preventDefault()
        e.stopPropagation()
        if (e.dataTransfer) {
            e.dataTransfer.dropEffect = "copy"
        }
    }

    async function onDropFile(e: DragEvent): Promise<void> {
        e.preventDefault()
        e.stopPropagation()
        is_dragging_over_file.value = false
        if (!e.dataTransfer) return
        const dropped_files = Array.from(e.dataTransfer.files)
        if (dropped_files.length === 0) return
        files.value = dropped_files
        await upload_files()
    }

    function onDragenterGpsFile(e: DragEvent): void {
        e.preventDefault()
        e.stopPropagation()
        if (!e.dataTransfer?.types.includes('Files')) return
        is_dragging_over_gps_file.value = true
    }

    function onDragleaveGpsFile(e: DragEvent): void {
        e.preventDefault()
        e.stopPropagation()
        is_dragging_over_gps_file.value = false
    }

    function onDragoverGpsFile(e: DragEvent): void {
        e.preventDefault()
        e.stopPropagation()
        if (e.dataTransfer) {
            e.dataTransfer.dropEffect = "copy"
        }
    }

    async function onDropGpsFile(e: DragEvent): Promise<void> {
        e.preventDefault()
        e.stopPropagation()
        is_dragging_over_gps_file.value = false
        if (!e.dataTransfer) return
        const dropped_files = Array.from(e.dataTransfer.files).filter(
            (f: File) => f.name.toLowerCase().endsWith('.gpx')
        )
        if (dropped_files.length === 0) return
        gps_log_files.value = dropped_files
        await upload_gps_log_files()
    }

    function trigger_file_input(): void {
        file_input.value?.click()
    }

    async function onFileInputChange(event: Event): Promise<void> {
        const input = event.target as HTMLInputElement
        if (!input.files?.length) return
        files.value = Array.from(input.files)
        await upload_files()
        input.value = ''
    }

    function trigger_gps_file_input(): void {
        gps_file_input.value?.click()
    }

    async function onGpsFileInputChange(event: Event): Promise<void> {
        const input = event.target as HTMLInputElement
        if (!input.files?.length) return
        gps_log_files.value = Array.from(input.files)
        await upload_gps_log_files()
        input.value = ''
    }

    async function reload_kyou(kyou: Kyou): Promise<void> {
        for (let i = 0; i < uploaded_kyous.value.length; i++) {
            const uploaded_kyou = uploaded_kyous.value[i]
            if (kyou.id === uploaded_kyou.id) {
                const updated_kyou = kyou.clone()
                await updated_kyou.reload(true)
                await updated_kyou.load_all()
                uploaded_kyous.value.splice(i, 1, updated_kyou)
            }
        }
    }

    function remove_uploaded_kyou(deleted_kyou: Kyou): void {
        for (let i = uploaded_kyous.value.length - 1; i >= 0; i--) {
            if (uploaded_kyous.value[i].id === deleted_kyou.id) {
                uploaded_kyous.value.splice(i, 1)
            }
        }
        emits('deleted_kyou', deleted_kyou)
    }

    // ── CRUD relay handlers ──
    const crudRelayHandlers = build_kyou_view_relay(emits, {
        // アップロード直後のKyouはこの画面が抱えているので、再読込・削除は自前で処理する
        'requested_reload_kyou': (kyou: Kyou) => reload_kyou(kyou),
        'deleted_kyou': (kyou: Kyou) => remove_uploaded_kyou(kyou),
    })

    // ── Return ──
    return {
        // Template refs
        decide_related_time_uploaded_file_dialog,
        file_input,
        gps_file_input,

        // State
        is_requested_submit,
        tab,
        conflict_behavior_file,
        conflict_behavior_gps_file,
        target_rep_names_for_file,
        target_rep_name_for_file,
        target_rep_names_for_gps_file,
        target_rep_name_for_gps_file,
        uploaded_kyous,

        // Business logic / template handlers
        load_target_rep_names,
        reload_kyou,
        remove_uploaded_kyou,

        // Drag state
        is_dragging_over_file,
        is_dragging_over_gps_file,

        // Drag handlers
        onDragenterFile,
        onDragleaveFile,
        onDragoverFile,
        onDropFile,
        onDragenterGpsFile,
        onDragleaveGpsFile,
        onDragoverGpsFile,
        onDropGpsFile,

        // Click handlers
        trigger_file_input,
        onFileInputChange,
        trigger_gps_file_input,
        onGpsFileInputChange,

        // Event relay objects
        crudRelayHandlers,
    }
}
