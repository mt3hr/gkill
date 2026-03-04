// src/classes/use-floating-dialog.ts
// Teleport 前提の「壊れにくい」フローティングダイアログ
// - v-overlay 配下の transform の影響を避けるため、Teleport to="body" を想定
// - 位置は transform ではなく left/top を更新（ズレが起きにくい）
// - containerRef が v-if で後から生える前提で ResizeObserver を attach
// - 初回/毎回の中央寄せをオプションで制御
// - ヘッダー内の操作要素（checkbox / btn 等）タップ時はドラッグを開始しない（モバイル対策）

import { computed, onBeforeUnmount, onMounted, ref, watch, type ComputedRef, type Ref } from "vue"

type Point = { x: number; y: number }

export type UseFloatingDialogResult = {
  // template: :ref="ui.containerRef"
  containerRef: Ref<HTMLElement | null>

  // template: :style="ui.fixedStyle.value"
  fixedStyle: ComputedRef<Record<string, string>>

  // header: @mousedown / @touchstart
  onHeaderPointerDown: (e: MouseEvent | TouchEvent) => void

  // checkbox/v-switch etc: v-model
  isTransparent: Ref<boolean>

  // 「中央へ戻す」ボタンなどから呼ぶ
  resetToCenter: () => void
}

function clamp(v: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, v))
}

function getPointerXY(e: MouseEvent | TouchEvent): Point {
  if ("touches" in e) {
    const t = e.touches[0] ?? e.changedTouches[0]
    return { x: t?.clientX ?? 0, y: t?.clientY ?? 0 }
  }
  return { x: e.clientX, y: e.clientY }
}

function isInteractiveTarget(target: EventTarget | null): boolean {
  const el = target as HTMLElement | null
  if (!el) return false

  // v-checkbox/v-switch/v-btn などの Vuetify 構造も拾う
  const selector = [
    "button",
    "a",
    "input",
    "textarea",
    "select",
    "label",
    "[role=button]",
    "[role=checkbox]",
    "[data-no-drag]",
    ".v-btn",
    ".v-btn__content",
    ".v-selection-control",
    ".v-selection-control__input",
    ".v-switch",
    ".v-checkbox",
    ".gkill-floating-dialog__btn",
    ".gkill-floating-dialog__toggle",
  ].join(",")

  return !!el.closest(selector)
}

// localStorage が使えない環境でも落ちないようにする
function safeGet(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}
function safeSet(key: string, val: string): void {
  try {
    localStorage.setItem(key, val)
  } catch {
    // noop
  }
}

function loadBool(key: string, defaultValue: boolean): boolean {
  const raw = safeGet(key)
  if (raw === null) return defaultValue
  return raw === "1"
}
function saveBool(key: string, v: boolean): void {
  safeSet(key, v ? "1" : "0")
}

function loadPoint(key: string, defaultValue: Point): Point {
  try {
    const raw = safeGet(key)
    if (!raw) return defaultValue
    const p = JSON.parse(raw) as Point
    if (typeof p?.x !== "number" || typeof p?.y !== "number") return defaultValue
    return p
  } catch {
    return defaultValue
  }
}
function savePoint(key: string, p: Point): void {
  safeSet(key, JSON.stringify(p))
}

export function useFloatingDialog(
  storageKey: string,
  opts?: {
    defaultTransparent?: boolean
    margin?: number
    zIndex?: number
    // 保存が無い場合の初期位置（centerMode="never"のとき等）
    defaultPos?: Point
    // "first": 初回だけ中央（保存が無いとき）
    // "always": 毎回中央
    // "never": 中央寄せしない
    centerMode?: "first" | "always" | "never"
    // centerMode="always" で「中央に出しても保存しない」方が良い場合 true
    dontPersistWhenAlwaysCenter?: boolean
  }
): UseFloatingDialogResult {
  const margin = opts?.margin ?? 8
  // Teleport to body 前提なので、Vuetify の overlay より前面に出る値にする
  const zIndex = opts?.zIndex ?? 1100
  const centerMode = opts?.centerMode ?? "first"
  const dontPersistWhenAlwaysCenter = opts?.dontPersistWhenAlwaysCenter ?? false

  const posKey = `${storageKey}:pos`
  const transparentKey = `${storageKey}:transparent`

  const containerRef = ref<HTMLElement | null>(null)

  const isTransparent = ref<boolean>(
    loadBool(transparentKey, opts?.defaultTransparent ?? false),
  )

  // 保存があるかどうか（初回中央の判定に使う）
  const hasSavedPos = safeGet(posKey) != null

  // 位置
  const pos = ref<Point>(
    loadPoint(posKey, opts?.defaultPos ?? { x: 16, y: 72 }),
  )

  // 内容の変化でサイズが変わるので observer で追従
  const lastRect = ref<{ w: number; h: number }>({ w: 0, h: 0 })
  let ro: ResizeObserver | null = null
  let observedEl: HTMLElement | null = null

  function readRect(): { w: number; h: number } {
    const el = containerRef.value
    if (!el) return lastRect.value
    const rect = el.getBoundingClientRect()
    const w = rect.width
    const h = rect.height
    if (w > 0 && h > 0) lastRect.value = { w, h }
    return lastRect.value
  }

  function clampToViewport(): void {
    const { w, h } = readRect()
    if (w <= 0 || h <= 0) return

    const maxX = window.innerWidth - w - margin
    const maxY = window.innerHeight - h - margin

    pos.value = {
      x: clamp(pos.value.x, margin, Math.max(margin, maxX)),
      y: clamp(pos.value.y, margin, Math.max(margin, maxY)),
    }
  }

  function persistPos(): void {
    if (centerMode === "always" && dontPersistWhenAlwaysCenter) return
    savePoint(posKey, pos.value)
  }

  const fixedStyle = computed<Record<string, string>>(() => {
    return {
      position: "fixed",
      left: `${Math.round(pos.value.x)}px`,
      top: `${Math.round(pos.value.y)}px`,
      zIndex: String(zIndex),
      willChange: "left, top",
    }
  })

  // drag state
  let dragging = false
  let startPointer: Point = { x: 0, y: 0 }
  let startPos: Point = { x: 0, y: 0 }

  function onMove(e: MouseEvent | TouchEvent): void {
    if (!dragging) return

    // touch でのページスクロールを抑制
    if ("touches" in e) e.preventDefault()

    const p = getPointerXY(e)
    const dx = p.x - startPointer.x
    const dy = p.y - startPointer.y

    pos.value = { x: startPos.x + dx, y: startPos.y + dy }
    clampToViewport()
  }

  function onUp(): void {
    if (!dragging) return
    dragging = false
    persistPos()
  }

  function onHeaderPointerDown(e: MouseEvent | TouchEvent): void {
    // ✅ ヘッダー内の操作要素タップではドラッグ開始しない
    if (isInteractiveTarget(e.target)) return

    // 掴んだ瞬間に rect 更新・clamp（画面外スタート防止）
    readRect()
    clampToViewport()

    dragging = true
    startPointer = getPointerXY(e)
    startPos = { ...pos.value }

    // touchstart を抑制しないと「タップ→スクロール」判定が混ざって変な挙動になりがち
    if ("touches" in e) e.preventDefault()
  }

  function resetToCenter(): void {
    // サイズが取れない瞬間があるので、まず概算→次フレームで確定
    const r0 = readRect()
    const estimateW = r0.w > 0 ? r0.w : Math.min(720, window.innerWidth * 0.85)
    const estimateH = r0.h > 0 ? r0.h : window.innerHeight * 0.6

    pos.value = {
      x: Math.round((window.innerWidth - estimateW) / 2),
      y: Math.round((window.innerHeight - estimateH) / 2),
    }
    clampToViewport()
    persistPos()

    requestAnimationFrame(() => {
      const r1 = readRect()
      if (r1.w > 0 && r1.h > 0) {
        pos.value = {
          x: Math.round((window.innerWidth - r1.w) / 2),
          y: Math.round((window.innerHeight - r1.h) / 2),
        }
        clampToViewport()
        persistPos()
      }
    })
  }

  // 初回中央寄せの実行フラグ
  let didAutoCenter = false

  function autoCenterIfNeeded(): void {
    if (centerMode === "never") return
    if (centerMode === "always") {
      resetToCenter()
      return
    }

    // centerMode === "first"
    if (didAutoCenter) return
    if (!hasSavedPos) {
      resetToCenter()
      didAutoCenter = true
    }
  }

  function attachObserver(el: HTMLElement): void {
    if (!ro) return
    if (observedEl) {
      try {
        ro.unobserve(observedEl)
      } catch {
        // noop
      }
    }
    observedEl = el
    ro.observe(el)
  }

  function detachObserver(): void {
    if (!ro || !observedEl) return
    try {
      ro.unobserve(observedEl)
    } catch {
      // noop
    }
    observedEl = null
  }

  function onResize(): void {
    clampToViewport()
    persistPos()
  }

  onMounted(() => {
    ro = new ResizeObserver(() => {
      // 内容サイズ変化 → 画面外に出ないように補正
      readRect()
      clampToViewport()
      persistPos()
    })

    window.addEventListener("resize", onResize, { passive: true })
    window.addEventListener("mousemove", onMove as any, { passive: true })
    window.addEventListener("mouseup", onUp as any, { passive: true })
    window.addEventListener("touchmove", onMove as any, { passive: false })
    window.addEventListener("touchend", onUp as any, { passive: true })
  })

  onBeforeUnmount(() => {
    detachObserver()
    if (ro) ro.disconnect()
    ro = null

    window.removeEventListener("resize", onResize as any)
    window.removeEventListener("mousemove", onMove as any)
    window.removeEventListener("mouseup", onUp as any)
    window.removeEventListener("touchmove", onMove as any)
    window.removeEventListener("touchend", onUp as any)
  })

  // ✅ Teleport の v-if で DOM が生えた瞬間に observer attach & 中央寄せ
  watch(
    containerRef,
    (el) => {
      if (!el) {
        detachObserver()
        return
      }

      if (ro) attachObserver(el)

      // 出現直後は rect が 0 のことがあるので次フレームで処理
      requestAnimationFrame(() => {
        readRect()

        // 中央寄せが必要なら実行、不要なら画面内に収めるだけ
        autoCenterIfNeeded()
        clampToViewport()
        persistPos()
      })
    },
    { flush: "post" },
  )

  watch(isTransparent, (v) => saveBool(transparentKey, v), { immediate: true })

  return {
    containerRef,
    fixedStyle,
    onHeaderPointerDown,
    isTransparent,
    resetToCenter,
  }
}
