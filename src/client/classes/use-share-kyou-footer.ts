import { ref } from 'vue'
import type { ShareKyouFooterProps } from '@/pages/views/share-kyou-footer-props'
import type { ShareKyouFooterEmits } from '@/pages/views/share-kyou-footer-emits'
import type { ShareKyousInfo } from '@/classes/datas/share-kyous-info'
import type ShareKyousListDialog from '@/pages/dialogs/share-kyou-list-dialog.vue'
import type ShareKyousListLinkDialog from '@/pages/dialogs/share-kyou-list-link-dialog.vue'
import type ManageShareKyousListDialog from '@/pages/dialogs/manage-share-task-list-dialog.vue'

export function useShareKyouFooter(options: {
    props: ShareKyouFooterProps,
    emits: ShareKyouFooterEmits,
}) {
    const { props: _props, emits: _emits } = options

    const share_kyou_list_dialog = ref<InstanceType<typeof ShareKyousListDialog> | null>(null)
    const share_kyou_list_link_dialog = ref<InstanceType<typeof ShareKyousListLinkDialog> | null>(null)
    const manage_share_kyou_list_dialog = ref<InstanceType<typeof ManageShareKyousListDialog> | null>(null)

    function show_share_kyou_list_dialog() {
        const dialog = share_kyou_list_dialog.value
        if (dialog) {
            dialog.show()
        }
    }

    function show_share_kyou_list_link_dialog(share_kyou_list_info: ShareKyousInfo) {
        const dialog = share_kyou_list_link_dialog.value
        if (dialog) {
            dialog.show(share_kyou_list_info)
        }
    }

    function show_manage_share_kyou_dialog() {
        const dialog = manage_share_kyou_list_dialog.value
        if (dialog) {
            dialog.show()
        }
    }

    return {
        share_kyou_list_dialog,
        share_kyou_list_link_dialog,
        manage_share_kyou_list_dialog,
        show_share_kyou_list_dialog,
        show_share_kyou_list_link_dialog,
        show_manage_share_kyou_dialog,
    }
}
