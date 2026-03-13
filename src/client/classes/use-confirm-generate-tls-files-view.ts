import { GenerateTLSFileRequest } from '@/classes/api/req_res/generate-tls-file-request'
import type { GenerateTLSFileResponse } from '@/classes/api/req_res/generate-tls-file-response'
import type { ConfirmGenerateTLSFilesViewProps } from '@/pages/views/confirm-generate-tls-files-view-props'
import type { ConfirmGenerateTLSFilesViewEmits } from '@/pages/views/confirm-generate-tls-files-view-emits'

export function useConfirmGenerateTlsFilesView(options: {
    props: ConfirmGenerateTLSFilesViewProps,
    emits: ConfirmGenerateTLSFilesViewEmits,
}) {
    const { props, emits } = options

    async function generate_tls_files(): Promise<void> {
        const req = new GenerateTLSFileRequest()
        const res: GenerateTLSFileResponse = await props.gkill_api.generate_tls_file(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
        emits('generated_tls_files')
        emits('requested_close_dialog')
    }

    return {
        generate_tls_files,
    }
}
