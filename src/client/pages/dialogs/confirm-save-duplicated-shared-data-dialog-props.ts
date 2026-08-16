'use strict'

import type { ShareLedgerEntry } from '@/classes/share-target-dedup'

export interface ConfirmSaveDuplicatedSharedDataDialogProps {
    /** 既に保存済みだった共有の台帳エントリ。台帳が読めなかったときは null */
    entry: ShareLedgerEntry | null
}
