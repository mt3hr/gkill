import { vi } from 'vitest'

/**
 * Creates a mock GkillAPI object with vi.fn() stubs for common operations.
 * Use this when you need to inject a mock API without importing the real class
 * (which has heavy dependencies and side effects).
 */
export function createMockGkillAPI() {
  let _session_id = ''

  return {
    // Auth
    login: vi.fn().mockResolvedValue({ session_id: 'mock-session', messages: [], errors: [] }),
    logout: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    reset_password: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    set_new_password: vi.fn().mockResolvedValue({ messages: [], errors: [] }),

    // Session
    get_session_id: vi.fn(() => _session_id),
    set_session_id: vi.fn((id: string) => { _session_id = id }),
    get_session_id_from_cookie_store: vi.fn().mockResolvedValue(''),
    check_auth: vi.fn(),

    // UUID
    generate_uuid: vi.fn(() => {
      // deterministic UUIDs for testing
      const hex = Array.from({ length: 32 }, (_, i) => (i % 16).toString(16)).join('')
      return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`
    }),

    // Data read operations
    get_kyous: vi.fn().mockResolvedValue({ kyous: [], messages: [], errors: [] }),
    get_kyou: vi.fn().mockResolvedValue({ kyou: null, messages: [], errors: [] }),
    get_kmemo: vi.fn().mockResolvedValue({ kmemo: null, messages: [], errors: [] }),
    get_mi: vi.fn().mockResolvedValue({ mi: null, messages: [], errors: [] }),
    get_tags_by_target_id: vi.fn().mockResolvedValue({ tags: [], messages: [], errors: [] }),
    get_texts_by_target_id: vi.fn().mockResolvedValue({ texts: [], messages: [], errors: [] }),
    get_mi_board_list: vi.fn().mockResolvedValue({ mi_board_names: [], messages: [], errors: [] }),
    get_all_tag_names: vi.fn().mockResolvedValue({ tag_names: [], messages: [], errors: [] }),
    get_all_rep_names: vi.fn().mockResolvedValue({ rep_names: [], messages: [], errors: [] }),
    get_application_config: vi.fn().mockResolvedValue({ application_config: null, messages: [], errors: [] }),

    // Data write operations
    add_kmemo: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    add_tag: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    add_text: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    add_mi: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    add_timeis: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    add_lantana: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    add_nlog: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    add_urlog: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    add_kc: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    add_rekyou: vi.fn().mockResolvedValue({ messages: [], errors: [] }),

    // Update operations
    update_kmemo: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    update_tag: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    update_text: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    update_mi: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    update_timeis: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    update_lantana: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    update_nlog: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    update_urlog: vi.fn().mockResolvedValue({ messages: [], errors: [] }),

    // Delete operations
    delete_kmemo: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    delete_tag: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    delete_text: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    delete_mi: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    delete_timeis: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    delete_lantana: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    delete_nlog: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    delete_urlog: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    delete_kc: vi.fn().mockResolvedValue({ messages: [], errors: [] }),

    // Notification
    add_notification: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    update_notification: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    delete_notification: vi.fn().mockResolvedValue({ messages: [], errors: [] }),

    // Context menu helpers
    get_saved_tag_history: vi.fn(() => []),
    push_tag_to_history: vi.fn(),
    open_directory: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    open_file: vi.fn().mockResolvedValue({ messages: [], errors: [] }),

    // Sharing
    add_share_kyou_list_info: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    get_shared_kyous: vi.fn().mockResolvedValue({ kyous: [], messages: [], errors: [] }),
    delete_share_kyou_list_infos: vi.fn().mockResolvedValue({ messages: [], errors: [] }),

    // Upload
    upload_files: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    upload_gps_log_files: vi.fn().mockResolvedValue({ messages: [], errors: [] }),

    // Endpoint addresses (for verification)
    login_address: '/api/login',
    get_kyous_address: '/api/get_kyous',
    add_kmemo_address: '/api/add_kmemo',
  }
}

export type MockGkillAPI = ReturnType<typeof createMockGkillAPI>
