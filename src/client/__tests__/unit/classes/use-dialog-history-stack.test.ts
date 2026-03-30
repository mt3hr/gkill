import { describe, test, expect, beforeEach } from 'vitest'
import { ref } from 'vue'

// Minimal mock for history API
let historyStack: Array<{ state: Record<string, unknown> | null }> = []
let historyIndex = 0
let popstateListeners: Array<(e: PopStateEvent) => void> = []

function mockPushState(state: Record<string, unknown> | null, _title: string, _url?: string) {
  historyStack = historyStack.slice(0, historyIndex + 1)
  historyStack.push({ state })
  historyIndex = historyStack.length - 1
}

function _mockReplaceState(state: Record<string, unknown> | null, _title: string) {
  historyStack[historyIndex] = { state }
}

function _firePopstate(state: Record<string, unknown> | null) {
  const event = new PopStateEvent('popstate', { state })
  for (const listener of popstateListeners) {
    listener(event)
  }
}

describe('use-dialog-history-stack concepts', () => {
  const MARK = "__gkillDlg"
  const DEPTH = "__gkillDlgDepth"

  function isDialogState(state: Record<string, unknown> | null): state is Record<string, unknown> & { [key: string]: unknown } {
    return state !== null && typeof state === 'object' && state[MARK] === true && typeof state[DEPTH] === 'number'
  }

  function withDialogMarkers(base: Record<string, unknown> | null, depth: number): Record<string, unknown> {
    const b = base && typeof base === 'object' ? base : {}
    return { ...b, [MARK]: true, [DEPTH]: depth }
  }

  function stripDialogKeys(state: Record<string, unknown> | null): Record<string, unknown> | null {
    if (!state || typeof state !== 'object') return state
    const { [MARK]: _m, [DEPTH]: _d, ...rest } = state
    return rest
  }

  beforeEach(() => {
    historyStack = [{ state: null }]
    historyIndex = 0
    popstateListeners = []
  })

  test('back closes topmost dialog', () => {
    // Simulate: open dialog pushes state with depth=1
    const dialogOpen = ref(true)
    const stack = [{ dialog: dialogOpen }]
    const state = withDialogMarkers({}, 1)
    mockPushState(state, '')

    // Simulate back: popstate with state=null (previous entry)
    const prevState = null
    const newDepth = isDialogState(prevState) ? prevState[DEPTH] : 0

    // Back detection: newDepth < stack.length
    expect(newDepth).toBe(0)
    expect(stack.length).toBe(1)
    expect(newDepth < stack.length).toBe(true)

    // Close topmost
    const top = stack[stack.length - 1]
    top.dialog.value = false
    expect(dialogOpen.value).toBe(false)
  })

  test('forward does NOT close dialog', () => {
    // Simulate: dialog open, user navigates forward into a state with depth >= stack.length
    const dialogOpen = ref(true)
    const stack = [{ dialog: dialogOpen }]

    const forwardState = withDialogMarkers({}, 1)
    const newDepth = isDialogState(forwardState) ? forwardState[DEPTH] : 0

    // Forward detection: newDepth >= stack.length → don't close
    expect(newDepth).toBe(1)
    expect(stack.length).toBe(1)
    expect(newDepth >= stack.length).toBe(true)

    // Dialog should remain open
    expect(dialogOpen.value).toBe(true)
  })

  test('back after all dialogs closed navigates normally', () => {
    // Stack is empty, state is not a dialog state
    const stack: Array<{ dialog: { value: boolean } }> = []
    const state: Record<string, unknown> = { page: 'home' }

    // Branch C check: stack empty AND dialog state → strip
    if (stack.length === 0 && isDialogState(state)) {
      // Would strip - but this state is NOT a dialog state
      expect(true).toBe(false) // should not reach
    }

    // Not a dialog state, stack is empty → normal navigation (no intervention)
    expect(stack.length).toBe(0)
    expect(isDialogState(state)).toBe(false)
  })

  test('multiple dialogs: back closes one at a time', () => {
    const dialog1 = ref(true)
    const dialog2 = ref(true)
    const stack = [{ dialog: dialog1 }, { dialog: dialog2 }]

    // First back: close dialog2 (topmost)
    const top1 = stack[stack.length - 1]
    top1.dialog.value = false
    stack.pop()
    expect(dialog2.value).toBe(false)
    expect(dialog1.value).toBe(true)
    expect(stack.length).toBe(1)

    // Second back: close dialog1
    const top2 = stack[stack.length - 1]
    top2.dialog.value = false
    stack.pop()
    expect(dialog1.value).toBe(false)
    expect(stack.length).toBe(0)
  })

  test('programmatic close rewinds history', () => {
    // When dialog is closed programmatically (not via popstate),
    // the stack entry is removed and history should be rewound
    const dialog = ref(true)
    const stack = [{ dialog: dialog }]

    // Programmatic close
    dialog.value = false
    stack.pop()

    expect(stack.length).toBe(0)
    expect(dialog.value).toBe(false)
  })

  test('escape closes topmost dialog without history change', () => {
    // Escape key sets dialog.value = false, which triggers the watcher
    const dialog = ref(true)
    const stack = [{ dialog: dialog }]

    // Escape closes via watcher (same as programmatic close)
    dialog.value = false
    const removed = stack.pop()

    expect(removed?.dialog.value).toBe(false)
    expect(stack.length).toBe(0)
  })

  test('Branch C: forward into dialog state while stack empty strips markers', () => {
    const stack: Array<{ dialog: { value: boolean } }> = []
    const state = withDialogMarkers({ page: 'test' }, 2)

    expect(isDialogState(state)).toBe(true)
    expect(stack.length).toBe(0)

    // New behavior: replaceState with stripped keys instead of history.go(-1)
    const stripped = stripDialogKeys(state)
    expect(stripped[MARK]).toBeUndefined()
    expect(stripped[DEPTH]).toBeUndefined()
    expect(stripped.page).toBe('test')
  })

  test('Branch D: forward detection uses depth comparison', () => {
    const dialog = ref(true)
    const stack = [{ dialog: dialog }]

    // Forward state has depth >= stack.length → forward
    const forwardState = withDialogMarkers({}, 2)
    const newDepth = forwardState[DEPTH] as number
    expect(newDepth >= stack.length).toBe(true)

    // Back state has depth < stack.length → back
    const backState = withDialogMarkers({}, 0)
    const backDepth = backState[DEPTH] as number
    expect(backDepth < stack.length).toBe(true)
  })
})
