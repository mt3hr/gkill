/**
 * save-clipboard-to-file-dialog composable tests.
 * Tests state, helper functions, API calls, and Ctrl+V keyboard hook.
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

import { createApp, defineComponent, ref } from 'vue'
import { createMockGkillAPI } from '../../helpers/mock-api'
import { useSaveClipboardToFileDialog, sanitize_filename } from '@/classes/use-save-clipboard-to-file-dialog'
import { useScopedCtrlVForClipboard } from '@/classes/use-scoped-ctrl-v-for-clipboard'

// ── helpers ──────────────────────────────────────────────────────────────────

function createBaseProps() {
  const api = createMockGkillAPI() as never as ReturnType<typeof createMockGkillAPI> & {
    get_repositories: ReturnType<typeof vi.fn>
  }
  // add get_repositories stub not present in the base mock
  api.get_repositories = vi.fn().mockResolvedValue({
    repositories: [],
    messages: [],
    errors: [],
  })
  return {
    gkill_api: api,
    application_config: {} as never,
    app_content_height: 800,
    app_content_width: 1200,
  }
}

/** Mount a composable so that onMounted / onBeforeUnmount lifecycle hooks fire. */
function withSetup<T>(composable: () => T): { result: T; unmount: () => void } {
  let result: T
  const app = createApp(
    defineComponent({
      setup() {
        result = composable()
        return {}
      },
      template: '<div />',
    }),
  )
  const el = document.createElement('div')
  app.mount(el)
  return { result: result!, unmount: () => app.unmount() }
}

// ── useSaveClipboardToFileDialog ─────────────────────────────────────────────

describe('useSaveClipboardToFileDialog', () => {
  afterEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  test('initial state: dialog is hidden, loading false, no blob', () => {
    const props = createBaseProps()
    const { result, unmount } = withSetup(() =>
      useSaveClipboardToFileDialog({ props: props as never, emits: vi.fn() as never }),
    )
    expect(result.is_show_dialog.value).toBe(false)
    expect(result.is_loading.value).toBe(false)
    expect(result.clipboard_blob.value).toBeNull()
    expect(result.filename.value).toBe('')
    expect(result.error_message_key.value).toBe('')
    unmount()
  })

  test('initial state: conflict_behavior defaults to rename', () => {
    const props = createBaseProps()
    const { result, unmount } = withSetup(() =>
      useSaveClipboardToFileDialog({ props: props as never, emits: vi.fn() as never }),
    )
    // FileUploadConflictBehavior.rename は 'rename' 文字列
    expect(result.conflict_behavior.value).toBeTruthy()
    unmount()
  })

  test('is_image_type() returns true for image MIME', () => {
    const props = createBaseProps()
    const { result, unmount } = withSetup(() =>
      useSaveClipboardToFileDialog({ props: props as never, emits: vi.fn() as never }),
    )
    result.selected_mime_type.value = 'image/png'
    expect(result.is_image_type()).toBe(true)
    expect(result.is_text_type()).toBe(false)
    unmount()
  })

  test('is_text_type() returns true for text MIME', () => {
    const props = createBaseProps()
    const { result, unmount } = withSetup(() =>
      useSaveClipboardToFileDialog({ props: props as never, emits: vi.fn() as never }),
    )
    result.selected_mime_type.value = 'text/plain'
    expect(result.is_text_type()).toBe(true)
    expect(result.is_image_type()).toBe(false)
    unmount()
  })

  test('type_display_name() returns extension in uppercase', () => {
    const props = createBaseProps()
    const { result, unmount } = withSetup(() =>
      useSaveClipboardToFileDialog({ props: props as never, emits: vi.fn() as never }),
    )
    result.selected_mime_type.value = 'image/png'
    expect(result.type_display_name()).toBe('PNG')
    result.selected_mime_type.value = 'text/plain'
    expect(result.type_display_name()).toBe('TXT')
    unmount()
  })

  test('type_display_name() returns empty string when MIME is empty', () => {
    const props = createBaseProps()
    const { result, unmount } = withSetup(() =>
      useSaveClipboardToFileDialog({ props: props as never, emits: vi.fn() as never }),
    )
    result.selected_mime_type.value = ''
    expect(result.type_display_name()).toBe('')
    unmount()
  })

  test('file_size_display() returns empty string when blob is null', () => {
    const props = createBaseProps()
    const { result, unmount } = withSetup(() =>
      useSaveClipboardToFileDialog({ props: props as never, emits: vi.fn() as never }),
    )
    expect(result.file_size_display()).toBe('')
    unmount()
  })

  test('load_clipboard() sets CLIPBOARD_NOT_SUPPORTED_MESSAGE when clipboard API absent', async () => {
    // jsdom does not implement navigator.clipboard.read
    const props = createBaseProps()
    const { result, unmount } = withSetup(() =>
      useSaveClipboardToFileDialog({ props: props as never, emits: vi.fn() as never }),
    )
    await result.load_clipboard()
    expect(result.error_message_key.value).toBe('CLIPBOARD_NOT_SUPPORTED_MESSAGE')
    unmount()
  })

  test('load_clipboard() sets CLIPBOARD_PERMISSION_DENIED_MESSAGE on NotAllowedError', async () => {
    const originalClipboard = Object.getOwnPropertyDescriptor(navigator, 'clipboard')
    Object.defineProperty(navigator, 'clipboard', {
      value: {
        read: vi.fn().mockRejectedValue(Object.assign(new Error('denied'), { name: 'NotAllowedError' })),
      },
      configurable: true,
    })

    const props = createBaseProps()
    const { result, unmount } = withSetup(() =>
      useSaveClipboardToFileDialog({ props: props as never, emits: vi.fn() as never }),
    )
    await result.load_clipboard()
    expect(result.error_message_key.value).toBe('CLIPBOARD_PERMISSION_DENIED_MESSAGE')
    unmount()

    if (originalClipboard) {
      Object.defineProperty(navigator, 'clipboard', originalClipboard)
    } else {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      delete (navigator as any).clipboard
    }
  })

  test('load_clipboard() sets CLIPBOARD_EMPTY_MESSAGE when clipboard has no items', async () => {
    Object.defineProperty(navigator, 'clipboard', {
      value: { read: vi.fn().mockResolvedValue([]) },
      configurable: true,
    })

    const props = createBaseProps()
    const { result, unmount } = withSetup(() =>
      useSaveClipboardToFileDialog({ props: props as never, emits: vi.fn() as never }),
    )
    await result.load_clipboard()
    expect(result.error_message_key.value).toBe('CLIPBOARD_EMPTY_MESSAGE')
    unmount()

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    delete (navigator as any).clipboard
  })

  test('hide() sets is_show_dialog to false', () => {
    const props = createBaseProps()
    const { result, unmount } = withSetup(() =>
      useSaveClipboardToFileDialog({ props: props as never, emits: vi.fn() as never }),
    )
    result.is_show_dialog.value = true
    result.hide()
    expect(result.is_show_dialog.value).toBe(false)
    unmount()
  })

  test('save_or_confirm() calls upload_files with correct rep name', async () => {
    const props = createBaseProps()
    props.gkill_api.get_repositories.mockResolvedValue({
      repositories: [
        { rep_name: 'my-rep', type: 'directory', is_enable: true, use_to_write: true },
      ],
      messages: [],
      errors: [],
    })
    props.gkill_api.upload_files.mockResolvedValue({
      uploaded_kyous: [],
      messages: [],
      errors: [],
    })

    const emits = vi.fn()
    const { result, unmount } = withSetup(() =>
      useSaveClipboardToFileDialog({ props: props as never, emits: emits as never }),
    )

    // set up state that do_save requires
    result.clipboard_blob.value = new Blob(['hello'], { type: 'text/plain' })
    result.selected_mime_type.value = 'text/plain'
    result.filename.value = 'test.txt'
    result.target_rep_names.value = ['my-rep']
    result.target_rep_name.value = 'my-rep'

    await result.save_or_confirm()

    expect(props.gkill_api.upload_files).toHaveBeenCalledTimes(1)
    const req = props.gkill_api.upload_files.mock.calls[0][0]
    expect(req.target_rep_name).toBe('my-rep')
    expect(req.files).toHaveLength(1)
    unmount()
  })

  test('save_or_confirm() emits received_errors when upload_files returns errors', async () => {
    const props = createBaseProps()
    props.gkill_api.upload_files.mockResolvedValue({
      uploaded_kyous: [],
      messages: [],
      errors: [{ error_code: 'E001', error_message: 'upload failed' }],
    })

    const emits = vi.fn()
    const { result, unmount } = withSetup(() =>
      useSaveClipboardToFileDialog({ props: props as never, emits: emits as never }),
    )

    result.clipboard_blob.value = new Blob(['x'], { type: 'text/plain' })
    result.filename.value = 'x.txt'
    result.target_rep_names.value = ['rep']
    result.target_rep_name.value = 'rep'

    await result.save_or_confirm()

    expect(emits).toHaveBeenCalledWith('received_errors', expect.arrayContaining([
      expect.objectContaining({ error_code: 'E001' }),
    ]))
    unmount()
  })

  test('load_target_rep_names filters only directory/enable/write repos', async () => {
    const props = createBaseProps()
    props.gkill_api.get_repositories.mockResolvedValue({
      repositories: [
        { rep_name: 'writable', type: 'directory', is_enable: true, use_to_write: true },
        { rep_name: 'readonly', type: 'directory', is_enable: true, use_to_write: false },
        { rep_name: 'disabled', type: 'directory', is_enable: false, use_to_write: true },
        { rep_name: 'git-rep', type: 'git', is_enable: true, use_to_write: true },
      ],
      messages: [],
      errors: [],
    })
    // prevent clipboard read from throwing in jsdom
    props.gkill_api.upload_files.mockResolvedValue({ uploaded_kyous: [], messages: [], errors: [] })

    const { result, unmount } = withSetup(() =>
      useSaveClipboardToFileDialog({ props: props as never, emits: vi.fn() as never }),
    )

    // show() calls load_target_rep_names internally
    // We cannot call show() directly because it also calls load_clipboard() which may race.
    // Directly test that after show() the target_rep_names are filtered.
    // Use a manual call pathway: override navigator.clipboard so show() can return quickly.
    Object.defineProperty(navigator, 'clipboard', {
      value: { read: vi.fn().mockResolvedValue([]) },
      configurable: true,
    })

    await result.show()
    expect(result.target_rep_names.value).toEqual(['writable'])
    result.hide()

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    delete (navigator as any).clipboard
    unmount()
  })
})

// ── sanitize_filename ────────────────────────────────────────────────────────

describe('sanitize_filename', () => {
  test('removes curly braces from filename', () => {
    expect(sanitize_filename('photo {uuid}.png')).toBe('photo uuid.png')
  })

  test('removes Windows-invalid chars: \\ / : * ? " < > |', () => {
    expect(sanitize_filename('file\\name/test:foo*bar?baz"qux<a>b|c.txt')).toBe('filenametestfoobarbazquxabc.txt')
  })

  test('removes control characters', () => {
    expect(sanitize_filename('file\x00name\x1f.txt')).toBe('filename.txt')
  })

  test('trims leading/trailing whitespace', () => {
    expect(sanitize_filename('  hello.png  ')).toBe('hello.png')
  })

  test('returns "file" when result is empty after sanitization', () => {
    expect(sanitize_filename('{}|<>*?')).toBe('file')
  })

  test('preserves normal filename unchanged', () => {
    expect(sanitize_filename('screenshot_20240101_120000.png')).toBe('screenshot_20240101_120000.png')
  })

  test('preserves Japanese characters', () => {
    expect(sanitize_filename('スクリーンショット 2024-01-01.png')).toBe('スクリーンショット 2024-01-01.png')
  })
})

// ── useScopedCtrlVForClipboard ────────────────────────────────────────────────

describe('useScopedCtrlVForClipboard', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  function fireKeydown(key: string, opts: Partial<KeyboardEventInit> = {}) {
    // Dispatch from document.body so it bubbles to window while e.target is an Element
    // (dispatching directly on window gives e.target = window which lacks .closest())
    document.body.dispatchEvent(
      new KeyboardEvent('keydown', { key, ctrlKey: true, bubbles: true, ...opts }),
    )
  }

  test('Ctrl+V fires openClipboardDialog', () => {
    const openDialog = vi.fn()
    const rootRef = ref<HTMLElement | null>(null)
    const { unmount } = withSetup(() =>
      useScopedCtrlVForClipboard(rootRef, openDialog),
    )

    fireKeydown('v')
    expect(openDialog).toHaveBeenCalledTimes(1)
    unmount()
  })

  test('Meta+V fires openClipboardDialog', () => {
    const openDialog = vi.fn()
    const rootRef = ref<HTMLElement | null>(null)
    const { unmount } = withSetup(() =>
      useScopedCtrlVForClipboard(rootRef, openDialog),
    )

    document.body.dispatchEvent(
      new KeyboardEvent('keydown', { key: 'v', metaKey: true, bubbles: true }),
    )
    expect(openDialog).toHaveBeenCalledTimes(1)
    unmount()
  })

  test('plain V (no modifier) does not fire openClipboardDialog', () => {
    const openDialog = vi.fn()
    const rootRef = ref<HTMLElement | null>(null)
    const { unmount } = withSetup(() =>
      useScopedCtrlVForClipboard(rootRef, openDialog),
    )

    document.body.dispatchEvent(new KeyboardEvent('keydown', { key: 'v', bubbles: true }))
    expect(openDialog).not.toHaveBeenCalled()
    unmount()
  })

  test('Ctrl+Shift+V does not fire openClipboardDialog', () => {
    const openDialog = vi.fn()
    const rootRef = ref<HTMLElement | null>(null)
    const { unmount } = withSetup(() =>
      useScopedCtrlVForClipboard(rootRef, openDialog),
    )

    fireKeydown('v', { shiftKey: true })
    expect(openDialog).not.toHaveBeenCalled()
    unmount()
  })

  test('Ctrl+Alt+V does not fire openClipboardDialog', () => {
    const openDialog = vi.fn()
    const rootRef = ref<HTMLElement | null>(null)
    const { unmount } = withSetup(() =>
      useScopedCtrlVForClipboard(rootRef, openDialog),
    )

    fireKeydown('v', { altKey: true })
    expect(openDialog).not.toHaveBeenCalled()
    unmount()
  })

  test('enabledRef=false suppresses the callback', () => {
    const openDialog = vi.fn()
    const rootRef = ref<HTMLElement | null>(null)
    const enabled = ref(false)
    const { unmount } = withSetup(() =>
      useScopedCtrlVForClipboard(rootRef, openDialog, enabled),
    )

    fireKeydown('v')
    expect(openDialog).not.toHaveBeenCalled()
    unmount()
  })

  test('enabledRef=true allows the callback', () => {
    const openDialog = vi.fn()
    const rootRef = ref<HTMLElement | null>(null)
    const enabled = ref(true)
    const { unmount } = withSetup(() =>
      useScopedCtrlVForClipboard(rootRef, openDialog, enabled),
    )

    fireKeydown('v')
    expect(openDialog).toHaveBeenCalledTimes(1)
    unmount()
  })

  test('listener is removed on unmount', () => {
    const openDialog = vi.fn()
    const rootRef = ref<HTMLElement | null>(null)
    const { unmount } = withSetup(() =>
      useScopedCtrlVForClipboard(rootRef, openDialog),
    )

    unmount()
    fireKeydown('v')
    expect(openDialog).not.toHaveBeenCalled()
  })

  test('key repeat does not fire openClipboardDialog', () => {
    const openDialog = vi.fn()
    const rootRef = ref<HTMLElement | null>(null)
    const { unmount } = withSetup(() =>
      useScopedCtrlVForClipboard(rootRef, openDialog),
    )

    fireKeydown('v', { repeat: true })
    expect(openDialog).not.toHaveBeenCalled()
    unmount()
  })
})
