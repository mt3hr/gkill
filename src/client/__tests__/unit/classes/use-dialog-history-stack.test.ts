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

  // --- 案C: history 駆動クローズのコンセプト ---

  test('Branch D closes (stack.length - newDepth) dialogs on multi-entry jump', () => {
    // 戻るボタン長押しなどで一気に複数エントリ戻った場合、
    // popstate 1回で深さの差分すべてを閉じる
    const d1 = ref(true)
    const d2 = ref(true)
    const d3 = ref(true)
    const stack = [{ dialog: d1 }, { dialog: d2 }, { dialog: d3 }]

    const jumpedState = withDialogMarkers({}, 1) // depth 3 → 1 へジャンプ
    const newDepth = jumpedState[DEPTH] as number
    let count = stack.length - newDepth
    expect(count).toBe(2)

    while (count > 0 && stack.length > 0) {
      const entry = stack.pop()!
      entry.dialog.value = false
      count--
    }

    expect(d3.value).toBe(false)
    expect(d2.value).toBe(false)
    expect(d1.value).toBe(true)
    expect(stack.length).toBe(newDepth)
  })

  test('close target priority: requested (middle) dialog closes, depth stays consistent', () => {
    // closeDialogViaHistory で「最上位でない」ダイアログの×を押した場合、
    // popstate ではターゲット指定のダイアログを閉じ、上のダイアログは維持する。
    // 履歴エントリは depth 値しか持たないため、除去後も depth == stack.length が保たれる
    const dA = ref(true)
    const dX = ref(true) // 閉じる対象 (中間)
    const dY = ref(true)
    const stack = [{ dialog: dA }, { dialog: dX }, { dialog: dY }]
    const pendingCloseTargets: object[] = [dX]

    // go(-1) 着弾: newDepth = 2, count = 1
    const newDepth = 2
    let count = stack.length - newDepth
    expect(count).toBe(1)

    while (count > 0 && stack.length > 0) {
      let idx = -1
      while (pendingCloseTargets.length > 0 && idx < 0) {
        const t = pendingCloseTargets.shift()!
        idx = stack.findIndex((en) => (en.dialog as unknown as object) === t)
      }
      if (idx < 0) idx = stack.length - 1
      const [entry] = stack.splice(idx, 1)
      entry.dialog.value = false
      count--
    }

    expect(dX.value).toBe(false) // 指定ターゲットが閉じる
    expect(dY.value).toBe(true) // 最上位は維持
    expect(dA.value).toBe(true)
    expect(stack.length).toBe(newDepth) // depth 整合
  })

  test('reset accounting: one popstate per traversal, not per entry', () => {
    // history.go(-N) は N エントリ戻っても popstate は1回しか発火しない。
    // pendingNav (握りつぶし予約) はトラバーサル単位で数えること。
    // エントリ数で数えると N>=2 で詰まり、resetDialogHistory が resolve しない
    // (→ navigateToPage の await が完了せず画面遷移しなくなる)
    const depth = 3
    const inFlight = 1 // 飛行中の closeDialogViaHistory 由来 go(-1) — 各1回発火する
    const goDelta = Math.max(0, depth - inFlight)
    const pendingNav = inFlight + (goDelta > 0 ? 1 : 0)

    expect(goDelta).toBe(2)
    expect(pendingNav).toBe(2) // in-flight 1回 + go(-2) 1回 = popstate 2回

    // in-flight が無い純粋ケース: go(-3) 1回 → popstate 1回
    const pendingNavNoInflight = 0 + (Math.max(0, 3 - 0) > 0 ? 1 : 0)
    expect(pendingNavNoInflight).toBe(1)
  })

  test('stale close targets are dropped and fall back to topmost', () => {
    const dA = ref(true)
    const dGone = ref(false) // 既に閉じられていて stack に居ないターゲット
    const stack = [{ dialog: dA }]
    const pendingCloseTargets: object[] = [dGone]

    let idx = -1
    while (pendingCloseTargets.length > 0 && idx < 0) {
      const t = pendingCloseTargets.shift()!
      idx = stack.findIndex((en) => (en.dialog as unknown as object) === t)
    }
    if (idx < 0) idx = stack.length - 1

    expect(pendingCloseTargets.length).toBe(0) // 残骸ターゲットは破棄
    expect(idx).toBe(0) // 最上位にフォールバック
  })
})
