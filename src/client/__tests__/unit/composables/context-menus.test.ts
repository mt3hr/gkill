/**
 * Context Menu Composable tests.
 * Tests the representative useKmemoContextMenu deeply.
 */
import { vi } from 'vitest'

vi.mock('@/i18n', () => ({
  default: { global: { t: (key: string) => key, locale: 'ja' } },
  i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

vi.mock('@/classes/api/gkill-api', () => ({
  GkillAPI: {
    get_instance: vi.fn(() => ({
      get_session_id: vi.fn(() => 'mock-session'),
      generate_uuid: vi.fn(() => 'mock-uuid'),
    })),
    get_gkill_api: vi.fn(() => ({
      get_session_id: vi.fn(() => 'mock-session'),
    })),
  },
}))

vi.mock('@/classes/delete-gkill-cache', () => ({
  default: vi.fn().mockResolvedValue(undefined),
  delete_gkill_config_cache: vi.fn().mockResolvedValue(undefined),
}))

import { createMockGkillAPI } from '../../helpers/mock-api'
import { makeKyouWithKmemo } from '../../helpers/factory'
import { useKmemoContextMenu } from '@/classes/use-kmemo-context-menu'
import { useMiContextMenu } from '@/classes/use-mi-context-menu'
import { useNlogContextMenu } from '@/classes/use-nlog-context-menu'
import { useKCContextMenu } from '@/classes/use-kc-context-menu'
import { useURLogContextMenu } from '@/classes/use-ur-log-context-menu'
import { useTimeIsContextMenu } from '@/classes/use-time-is-context-menu'
import { useLantanaContextMenu } from '@/classes/use-lantana-context-menu'
import { useIDFKyouContextMenu } from '@/classes/use-idf-kyou-context-menu'
import { useReKyouContextMenu } from '@/classes/use-re-kyou-context-menu'
import { useGitCommitLogContextMenu } from '@/classes/use-git-commit-log-context-menu'

// Mock clipboard
Object.assign(navigator, {
  clipboard: { writeText: vi.fn().mockResolvedValue(undefined) },
})

function createMockProps() {
  return {
    gkill_api: createMockGkillAPI() as any,
    application_config: {
      device: 'test-device',
      user_id: 'admin',
      mi_default_board: 'Inbox',
      session_is_local: false,
    } as any,
    kyou: {
      ...makeKyouWithKmemo('テストメモ'),
      clone: () => ({ ...makeKyouWithKmemo('テストメモ') }),
    } as any,
    highlight_targets: [],
    enable_context_menu: true,
    enable_dialog: true,
  }
}

describe('useKmemoContextMenu', () => {
  let props: ReturnType<typeof createMockProps>
  let emits: ReturnType<typeof vi.fn>
  let menu: ReturnType<typeof useKmemoContextMenu>

  beforeEach(() => {
    vi.clearAllMocks()
    props = createMockProps()
    emits = vi.fn()
    menu = useKmemoContextMenu({ props, emits })
  })

  test('initial state: is_show is false', () => {
    expect(menu.is_show.value).toBe(false)
  })

  test('show() sets is_show=true', async () => {
    const event = new MouseEvent('contextmenu', { clientX: 100, clientY: 200 }) as any
    await menu.show(event)
    expect(menu.is_show.value).toBe(true)
  })

  test('show() loads tag history from gkill_api', async () => {
    props.gkill_api.get_saved_tag_history.mockReturnValue(['tag1', 'tag2'])
    await menu.show(new MouseEvent('contextmenu') as any)
    expect(menu.tag_history.value).toEqual(['tag1', 'tag2'])
  })

  test('copy_id() calls clipboard.writeText with kyou.id', async () => {
    await menu.copy_id()
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(props.kyou.id)
  })

  test('copy_id() emits received_messages', async () => {
    await menu.copy_id()
    const calls = emits.mock.calls.filter((c: any[]) => c[0] === 'received_messages')
    expect(calls.length).toBe(1)
  })

  test('show_edit_kmemo_dialog() emits requested_open_rykv_dialog with edit_kmemo', async () => {
    await menu.show_edit_kmemo_dialog()
    expect(emits).toHaveBeenCalledWith('requested_open_rykv_dialog', 'edit_kmemo', props.kyou)
  })

  test('show_add_tag_dialog() emits with add_tag kind', async () => {
    await menu.show_add_tag_dialog()
    expect(emits).toHaveBeenCalledWith('requested_open_rykv_dialog', 'add_tag', props.kyou)
  })

  test('show_add_text_dialog() emits with add_text kind', async () => {
    await menu.show_add_text_dialog()
    expect(emits).toHaveBeenCalledWith('requested_open_rykv_dialog', 'add_text', props.kyou)
  })

  test('show_confirm_delete_kyou_dialog() emits with confirm_delete_kyou', async () => {
    await menu.show_confirm_delete_kyou_dialog()
    expect(emits).toHaveBeenCalledWith('requested_open_rykv_dialog', 'confirm_delete_kyou', props.kyou)
  })

  test('show_confirm_rekyou_dialog() emits with confirm_re_kyou', async () => {
    await menu.show_confirm_rekyou_dialog()
    expect(emits).toHaveBeenCalledWith('requested_open_rykv_dialog', 'confirm_re_kyou', props.kyou)
  })

  test('add_tag_from_history() splits on \\u3001 and calls add_tag for each', async () => {
    props.gkill_api.add_tag.mockResolvedValue({
      messages: [{ message_code: 'MSG_OK' }],
      errors: [],
      added_tag: { tag: 'タグA' },
    })
    await menu.add_tag_from_history('タグA\u3001タグB')
    expect(props.gkill_api.add_tag).toHaveBeenCalledTimes(2)
  })

  test('add_tag_from_history() emits registered_tag per tag', async () => {
    props.gkill_api.add_tag.mockResolvedValue({
      messages: [],
      errors: [],
      added_tag: { tag: 'タグ' },
    })
    await menu.add_tag_from_history('タグ')
    const calls = emits.mock.calls.filter((c: any[]) => c[0] === 'registered_tag')
    expect(calls.length).toBe(1)
  })

  test('add_tag_from_history() emits received_errors on API failure', async () => {
    props.gkill_api.add_tag.mockResolvedValue({
      messages: [],
      errors: [{ error_code: 'ERR001', error_message: 'failed' }],
    })
    await menu.add_tag_from_history('テスト')
    const errorCalls = emits.mock.calls.filter((c: any[]) => c[0] === 'received_errors')
    expect(errorCalls.length).toBe(1)
  })

  test('add_tag_from_history() calls push_tag_to_history', async () => {
    props.gkill_api.add_tag.mockResolvedValue({ messages: [], errors: [], added_tag: {} })
    await menu.add_tag_from_history('テスト')
    expect(props.gkill_api.push_tag_to_history).toHaveBeenCalledWith('テスト')
  })

  test('open_folder() calls gkill_api.open_directory', async () => {
    await menu.open_folder()
    expect(props.gkill_api.open_directory).toHaveBeenCalled()
  })

  test('open_file() calls gkill_api.open_file', async () => {
    await menu.open_file()
    expect(props.gkill_api.open_file).toHaveBeenCalled()
  })

  test('context_menu_style computed is a string', () => {
    expect(typeof menu.context_menu_style.value).toBe('string')
  })

  test('tag_history ref is initialized as array', () => {
    expect(Array.isArray(menu.tag_history.value)).toBe(true)
  })

  test('returns all expected methods', () => {
    expect(typeof menu.show).toBe('function')
    expect(typeof menu.copy_id).toBe('function')
    expect(typeof menu.show_edit_kmemo_dialog).toBe('function')
    expect(typeof menu.show_add_tag_dialog).toBe('function')
    expect(typeof menu.show_add_text_dialog).toBe('function')
    expect(typeof menu.show_confirm_delete_kyou_dialog).toBe('function')
    expect(typeof menu.show_confirm_rekyou_dialog).toBe('function')
    expect(typeof menu.show_kyou_histories_dialog).toBe('function')
    expect(typeof menu.open_folder).toBe('function')
    expect(typeof menu.open_file).toBe('function')
    expect(typeof menu.add_tag_from_history).toBe('function')
  })
})

// ========== Other entity context menus ==========
// Each follows the same pattern as useKmemoContextMenu.
// Test basic functionality: initialization, show, copy_id.

const entityMenus = [
  { name: 'useMiContextMenu', factory: useMiContextMenu },
  { name: 'useNlogContextMenu', factory: useNlogContextMenu },
  { name: 'useKCContextMenu', factory: useKCContextMenu },
  { name: 'useURLogContextMenu', factory: useURLogContextMenu },
  { name: 'useTimeIsContextMenu', factory: useTimeIsContextMenu },
  { name: 'useLantanaContextMenu', factory: useLantanaContextMenu },
  { name: 'useIDFKyouContextMenu', factory: useIDFKyouContextMenu },
  { name: 'useReKyouContextMenu', factory: useReKyouContextMenu },
  { name: 'useGitCommitLogContextMenu', factory: useGitCommitLogContextMenu },
] as const

describe.each(entityMenus)('$name', ({ factory }) => {
  test('is_show starts as false', () => {
    const props = createMockProps()
    const emits = vi.fn()
    const menu = factory({ props, emits } as any)
    expect(menu.is_show.value).toBe(false)
  })

  test('show() sets is_show=true', async () => {
    const props = createMockProps()
    const emits = vi.fn()
    const menu = factory({ props, emits } as any)
    await menu.show(new MouseEvent('contextmenu', { clientX: 50, clientY: 50 }) as any)
    expect(menu.is_show.value).toBe(true)
  })

  test('copy_id() emits received_messages', async () => {
    const props = createMockProps()
    const emits = vi.fn()
    const menu = factory({ props, emits } as any)
    await menu.copy_id()
    const msgCalls = emits.mock.calls.filter((c: any[]) => c[0] === 'received_messages')
    expect(msgCalls.length).toBe(1)
  })

  test('has add_tag_from_history method', () => {
    const props = createMockProps()
    const emits = vi.fn()
    const menu = factory({ props, emits } as any)
    expect(typeof menu.add_tag_from_history).toBe('function')
  })
})
