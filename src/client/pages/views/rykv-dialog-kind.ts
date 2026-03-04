'use strict'

import type { Notification } from '@/classes/datas/notification'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Kyou } from '@/classes/datas/kyou'

export type RykvDialogKind =
  | 'kyou'
  | 'edit_kmemo'
  | 'edit_kc'
  | 'edit_mi'
  | 'edit_nlog'
  | 'edit_lantana'
  | 'edit_timeis'
  | 'edit_urlog'
  | 'edit_idf_kyou'
  | 'edit_re_kyou'
  | 'add_tag'
  | 'add_text'
  | 'add_notification'
  | 'confirm_delete_kyou'
  | 'confirm_re_kyou'
  | 'kyou_histories'
  | 'edit_tag'
  | 'confirm_delete_tag'
  | 'tag_histories'
  | 'edit_text'
  | 'confirm_delete_text'
  | 'text_histories'
  | 'edit_notification'
  | 'confirm_delete_notification'
  | 'notification_histories'

export type RykvDialogPayload = Tag | Text | Notification | null

export interface OpenedRykvDialog {
  id: string
  kind: RykvDialogKind
  kyou: Kyou
  payload: RykvDialogPayload
  opened_at: number
}
