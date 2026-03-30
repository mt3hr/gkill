/**
 * vLongPress directive tests.
 * Tests the Vue 3 directive lifecycle and event behavior.
 */
import { vi } from 'vitest'
import { vLongPress } from '@/classes/long-press'

// jsdom doesn't have PointerEvent; polyfill it
if (typeof globalThis.PointerEvent === 'undefined') {
  (globalThis as unknown as Record<string, unknown>).PointerEvent = class PointerEvent extends MouseEvent {
    constructor(type: string, init?: PointerEventInit) {
      super(type, init)
    }
  }
}

function createEl(): HTMLElement {
  const el = document.createElement('div')
  document.body.appendChild(el)
  return el
}

function mountDirective(el: HTMLElement, value: () => void) {
  vLongPress.mounted!(el, { value } as never, null as never, null as never)
}

function unmountDirective(el: HTMLElement) {
  vLongPress.unmounted!(el, {} as never, null as never, null as never)
}

function firePointerDown(el: HTMLElement, button = 0) {
  el.dispatchEvent(new PointerEvent('pointerdown', { button, bubbles: true }))
}

function firePointerUp(el: HTMLElement) {
  el.dispatchEvent(new PointerEvent('pointerup', { bubbles: true }))
}

describe('vLongPress directive', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
    document.body.innerHTML = ''
  })

  test('fires handler after pressMs timeout on pointerdown', () => {
    const handler = vi.fn()
    const el = createEl()
    mountDirective(el, handler)

    firePointerDown(el)
    vi.advanceTimersByTime(600)
    expect(handler).toHaveBeenCalledTimes(1)
  })

  test('does not fire handler if pointerup before timeout', () => {
    const handler = vi.fn()
    const el = createEl()
    mountDirective(el, handler)

    firePointerDown(el)
    vi.advanceTimersByTime(300)
    firePointerUp(el)
    vi.advanceTimersByTime(600)
    expect(handler).not.toHaveBeenCalled()
  })

  test('suppresses click after long press when suppressClick=true', () => {
    const handler = vi.fn()
    const el = createEl()
    mountDirective(el, { handler, suppressClick: true })

    firePointerDown(el)
    vi.advanceTimersByTime(600)

    const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true })
    const spy = vi.spyOn(clickEvent, 'preventDefault')
    el.dispatchEvent(clickEvent)
    expect(spy).toHaveBeenCalled()
  })

  test('allows click after short press', () => {
    const handler = vi.fn()
    const el = createEl()
    mountDirective(el, { handler, suppressClick: true })

    firePointerDown(el)
    vi.advanceTimersByTime(100)
    firePointerUp(el)

    const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true })
    const spy = vi.spyOn(clickEvent, 'preventDefault')
    el.dispatchEvent(clickEvent)
    expect(spy).not.toHaveBeenCalled()
  })

  test('accepts function binding (shorthand)', () => {
    const handler = vi.fn()
    const el = createEl()
    mountDirective(el, handler)

    firePointerDown(el)
    vi.advanceTimersByTime(600)
    expect(handler).toHaveBeenCalled()
  })

  test('accepts object binding with custom pressMs', () => {
    const handler = vi.fn()
    const el = createEl()
    mountDirective(el, { handler, pressMs: 200 })

    firePointerDown(el)
    vi.advanceTimersByTime(200)
    expect(handler).toHaveBeenCalled()
  })

  test('ignores non-left-button pointerdown', () => {
    const handler = vi.fn()
    const el = createEl()
    mountDirective(el, handler)

    firePointerDown(el, 2) // right button
    vi.advanceTimersByTime(1000)
    expect(handler).not.toHaveBeenCalled()
  })

  test('cleanup removes all event listeners on unmounted', () => {
    const handler = vi.fn()
    const el = createEl()
    mountDirective(el, handler)
    unmountDirective(el)

    firePointerDown(el)
    vi.advanceTimersByTime(1000)
    expect(handler).not.toHaveBeenCalled()
  })
})
