// 編集前に読む: .claude/skills/gkill-client-rudbeckia/SKILL.md（この領域の不変条件の正本）
// src/classes/use-floating-dialog.ts
// Teleport 前提の「壊れにくい」フローティングダイアログ
// - v-overlay 配下の transform の影響を避けるため、Teleport to="body" を想定
// - 位置は transform ではなく left/top を更新（ズレが起きにくい）
// - containerRef が v-if で後から生える前提で ResizeObserver を attach
// - 初回/毎回の中央寄せをオプションで制御
// - ヘッダー内の操作要素（checkbox / btn 等）タップ時はドラッグを開始しない（モバイル対策）
// - 右下コーナーのリサイズハンドルでユーザがダイアログサイズを変更可能

import { computed, getCurrentInstance, inject, onBeforeUnmount, onMounted, provide, ref, watch, type ComputedRef, type InjectionKey, type Ref } from "vue"
import { find_autofocus_target, has_focus_inside } from "@/classes/dialog-autofocus"
import { raise_dialog_history_entries } from "@/classes/use-dialog-history-stack"

type Point = { x: number; y: number }
export type Size = { w: number; h: number }

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

  // ユーザ設定サイズをリセットしてCSS既定サイズに戻す
  resetSize: () => void

  // ユーザがリサイズしたサイズ（null = 未リサイズ）
  userSize: Readonly<Ref<Size | null>>
}

function clamp(v: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, v))
}

function get_pointer_xy(e: MouseEvent | TouchEvent): Point {
  if ("touches" in e) {
    const t = e.touches[0] ?? e.changedTouches[0]
    return { x: t?.clientX ?? 0, y: t?.clientY ?? 0 }
  }
  return { x: e.clientX, y: e.clientY }
}

function is_interactive_target(target: EventTarget | null): boolean {
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
function safe_get(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}
function safe_set(key: string, val: string): void {
  try {
    localStorage.setItem(key, val)
  } catch {
    // noop
  }
}
function safe_remove(key: string): void {
  try {
    localStorage.removeItem(key)
  } catch {
    // noop
  }
}

function load_bool(key: string, default_value: boolean): boolean {
  const raw = safe_get(key)
  if (raw === null) return default_value
  return raw === "1"
}
function save_bool(key: string, v: boolean): void {
  safe_set(key, v ? "1" : "0")
}

function load_point(key: string, default_value: Point): Point {
  try {
    const raw = safe_get(key)
    if (!raw) return default_value
    const p = JSON.parse(raw) as Point
    if (typeof p?.x !== "number" || typeof p?.y !== "number") return default_value
    return p
  } catch {
    return default_value
  }
}
function save_point(key: string, p: Point): void {
  safe_set(key, JSON.stringify(p))
}

function load_size(key: string): Size | null {
  try {
    const raw = safe_get(key)
    if (!raw) return null
    const s = JSON.parse(raw) as Size
    if (typeof s?.w !== "number" || typeof s?.h !== "number") return null
    return s
  } catch {
    return null
  }
}
function save_size(key: string, s: Size): void {
  safe_set(key, JSON.stringify(s))
}

// ── 重なり順 ───────────────────────────────────────────────────────────────
//
// z-index は「開いているダイアログの並び順」から出す。
// **単調増加のカウンタにしてはいけない** —— Vuetify の overlay（メニュー / ツールチップ）が
// 2400 なので、上へ伸ばし続けるとダイアログの中のメニューがダイアログの下へ潜る。
// 並び順から出せば、伸びるのは同時に開いている枚数ぶんだけで済む。
const FLOATING_DIALOG_BASE_Z_INDEX = 1100

/** 開いているダイアログ。末尾が最前面 */
const z_order: Array<object> = []

/** z_order は生の配列なので、変更を computed へ伝えるためのカウンタ */
const z_order_version = ref(0)

/**
 * 入れ子のダイアログの親。
 *
 * 確認ダイアログは `Teleport to="body"` で親ダイアログの**兄弟**になるので、DOM からは
 * 親子が分からない。コンポーネント木は Teleport をまたいでも保たれるので provide/inject で持つ
 */
const floating_dialog_parents = new WeakMap<object, object>()

/**
 * z_token → コンポーネントインスタンス。
 *
 * バックと Escape が閉じる対象（履歴スタックの末尾）を、見た目の最前面に合わせるために使う。
 * `useDialogHistoryStack` も同じコンポーネントの setup で呼ばれるので、
 * インスタンスを鍵にすれば両者が結べる
 */
const floating_dialog_owners = new WeakMap<object, object>()

const FLOATING_DIALOG_PARENT_KEY: InjectionKey<object> = Symbol("gkill_floating_dialog_parent")

function is_descendant_of(token: object, ancestor: object): boolean {
  let current = floating_dialog_parents.get(token)
  while (current !== undefined) {
    if (current === ancestor) return true
    current = floating_dialog_parents.get(current)
  }
  return false
}

function enter_z_order(token: object): void {
  leave_z_order(token)
  z_order.push(token)
  z_order_version.value++
}

function leave_z_order(token: object): void {
  const index = z_order.indexOf(token)
  if (index === -1) return
  z_order.splice(index, 1)
  z_order_version.value++
}

/**
 * 自分と、自分から開いた子孫のダイアログをまとめて最前面へ出す（相対順は保つ）。
 *
 * 子孫を連れていかないと、親をクリックしただけで確認ダイアログが後ろへ隠れる。
 */
function raise_z_order(token: object): Array<object> {
  if (z_order.indexOf(token) === -1) return []

  const raised = z_order.filter((item) => item === token || is_descendant_of(item, token))
  if (raised.length === 0) return []

  // すでに最前面なら触らない（pointerdown のたびに全ダイアログの style を打ち直さない）
  const tail_start = z_order.length - raised.length
  let already_top = true
  for (let i = 0; i < raised.length; i++) {
    if (z_order[tail_start + i] !== raised[i]) {
      already_top = false
      break
    }
  }
  if (already_top) return []

  for (const item of raised) {
    z_order.splice(z_order.indexOf(item), 1)
  }
  z_order.push(...raised)
  z_order_version.value++
  return raised
}

export function useFloatingDialog(
  storage_key: string,
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
    // 中央からずらす量。同じ種類を複数枚開くとき、完全に重ならないようにする
    centerOffset?: Point
    // リサイズ可能にするか（デフォルト true）
    resizable?: boolean
    // 最小サイズ（デフォルト { w: 200, h: 150 }）
    minSize?: Size
    // 高さを保存・復元するか（デフォルト true）
    persistHeight?: boolean
    // Escape キー押下時のコールバック
    onEscape?: () => void
    // 開いたときに最初のテキスト入力欄へフォーカスするか（デフォルト true）
    // 自前でフォーカス先を決めているダイアログだけ false にする
    autofocus?: boolean
  }
): UseFloatingDialogResult {
  const margin = opts?.margin ?? 8
  const center_offset = opts?.centerOffset ?? { x: 0, y: 0 }

  // Teleport to body 前提なので、Vuetify の overlay より前面に出る値にする。
  // 実際の値は開いているダイアログの並び順から出す（z_order の解説を参照）
  const z_token = {}
  const parent_z_token = inject(FLOATING_DIALOG_PARENT_KEY, null)
  if (parent_z_token !== null) {
    floating_dialog_parents.set(z_token, parent_z_token)
  }
  provide(FLOATING_DIALOG_PARENT_KEY, z_token)

  const z_owner = getCurrentInstance() as unknown as object | null
  if (z_owner !== null) {
    floating_dialog_owners.set(z_token, z_owner)
  }

  const z_index = computed<number>(() => {
    if (opts?.zIndex !== undefined) return opts.zIndex
    // z_order は生の配列なので、この参照で変更を購読する
    void z_order_version.value
    return FLOATING_DIALOG_BASE_Z_INDEX + Math.max(0, z_order.indexOf(z_token))
  })

  const center_mode = opts?.centerMode ?? "first"
  const dont_persist_when_always_center = opts?.dontPersistWhenAlwaysCenter ?? false
  const resizable = opts?.resizable ?? true
  const min_w = opts?.minSize?.w ?? 200
  const min_h = opts?.minSize?.h ?? 150
  const persist_height = opts?.persistHeight ?? true
  const autofocus_enabled = opts?.autofocus ?? true

  const pos_key = `${storage_key}:pos`
  const transparent_key = `${storage_key}:transparent`
  const size_key = `${storage_key}:size`

  const container_ref = ref<HTMLElement | null>(null)

  const is_transparent = ref<boolean>(
    load_bool(transparent_key, opts?.defaultTransparent ?? false),
  )

  // 保存があるかどうか（初回中央の判定に使う）
  const has_saved_pos = safe_get(pos_key) != null

  // 位置
  const pos = ref<Point>(
    load_point(pos_key, opts?.defaultPos ?? { x: 16, y: 72 }),
  )

  // ユーザ設定サイズ（null = 未リサイズ、CSS既定サイズを使用）
  const saved_size = resizable ? load_size(size_key) : null
  const user_size = ref<Size | null>(
    saved_size && !persist_height ? { w: saved_size.w, h: 0 } : saved_size,
  )

  // --- Accessibility ---
  const dialog_id = `floating-dialog-${storage_key.replace(/[^a-zA-Z0-9_-]/g, "-")}`
  const label_id = `${dialog_id}__label`
  let escape_handler: ((e: KeyboardEvent) => void) | null = null

  function apply_aria_attributes(el: HTMLElement): void {
    el.setAttribute("role", "dialog")
    el.setAttribute("aria-modal", "true")

    // Find a heading or title/header element for aria-labelledby
    const label_el =
      el.querySelector("h1, h2, h3, h4, h5, h6") ??
      el.querySelector(".gkill-floating-dialog__title")
    if (label_el && label_el.textContent?.trim()) {
      if (!label_el.id) label_el.id = label_id
      el.setAttribute("aria-labelledby", label_el.id)
    } else {
      el.setAttribute("aria-label", storage_key.replace(/-/g, " "))
    }
  }

  function attach_escape_handler(el: HTMLElement): void {
    detach_escape_handler()
    escape_handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.stopPropagation()
        opts?.onEscape?.()
      }
    }
    el.addEventListener("keydown", escape_handler)
  }

  function detach_escape_handler(): void {
    if (escape_handler && container_ref.value) {
      container_ref.value.removeEventListener("keydown", escape_handler)
    }
    escape_handler = null
  }

  // --- Autofocus ---
  // 入力欄はさらに内側の v-if でデータ待ちのことが多い（編集ダイアログは
  // typed data を取ってから本体を描く）ので、生えてくるのを見張って一度だけ当てる
  const autofocus_deadline_ms = 2000
  let autofocus_observer: MutationObserver | null = null
  let autofocus_timer: ReturnType<typeof setTimeout> | null = null

  function detach_autofocus(): void {
    if (autofocus_observer) {
      autofocus_observer.disconnect()
      autofocus_observer = null
    }
    if (autofocus_timer !== null) {
      clearTimeout(autofocus_timer)
      autofocus_timer = null
    }
  }

  function try_autofocus(el: HTMLElement): boolean {
    // 待っている間にユーザーが自分で入力欄を選んでいたら手を出さない
    if (has_focus_inside(el)) {
      detach_autofocus()
      return true
    }
    const target = find_autofocus_target(el)
    if (!target) {
      return false
    }
    // focus は Vuetify のクラス付け替えを誘発するので、先に監視を止める
    detach_autofocus()
    // ダイアログ本文は overflow:auto なので、素の focus だと中身がスクロールする
    target.focus({ preventScroll: true })
    return true
  }

  function attach_autofocus(el: HTMLElement): void {
    detach_autofocus()
    if (!autofocus_enabled) return
    if (try_autofocus(el)) return

    autofocus_observer = new MutationObserver(() => {
      try_autofocus(el)
    })
    autofocus_observer.observe(el, { childList: true, subtree: true })
    autofocus_timer = setTimeout(() => detach_autofocus(), autofocus_deadline_ms)
  }

  // --- End Accessibility ---

  // 内容の変化でサイズが変わるので observer で追従
  const last_rect = ref<{ w: number; h: number }>({ w: 0, h: 0 })
  let ro: ResizeObserver | null = null
  let observed_el: HTMLElement | null = null

  function read_rect(): { w: number; h: number } {
    const el = container_ref.value
    if (!el) return last_rect.value
    const rect = el.getBoundingClientRect()
    const w = rect.width
    const h = rect.height
    if (w > 0 && h > 0) last_rect.value = { w, h }
    return last_rect.value
  }

  function clamp_to_viewport(): void {
    const { w, h } = read_rect()
    if (w <= 0 || h <= 0) return

    const max_x = window.innerWidth - w - margin
    const max_y = window.innerHeight - h - margin

    pos.value = {
      x: clamp(pos.value.x, margin, Math.max(margin, max_x)),
      y: clamp(pos.value.y, margin, Math.max(margin, max_y)),
    }
  }

  function persist_pos(): void {
    if (center_mode === "always" && dont_persist_when_always_center) return
    save_point(pos_key, pos.value)
  }

  // is-user-resized クラスの管理
  function update_resized_class(): void {
    const el = container_ref.value
    if (!el) return
    if (user_size.value) {
      el.classList.add("is-user-resized")
    } else {
      el.classList.remove("is-user-resized")
    }
  }

  const fixed_style = computed<Record<string, string>>(() => {
    const s: Record<string, string> = {
      position: "fixed",
      left: `${Math.round(pos.value.x)}px`,
      top: `${Math.round(pos.value.y)}px`,
      zIndex: String(z_index.value),
      willChange: "left, top",
    }
    if (user_size.value) {
      s.width = `${Math.round(user_size.value.w)}px`
      if (user_size.value.h > 0) {
        s.height = `${Math.round(user_size.value.h)}px`
      }
    }
    return s
  })

  // drag state
  let dragging = false
  let start_pointer: Point = { x: 0, y: 0 }
  let start_pos: Point = { x: 0, y: 0 }

  // resize state
  let resizing = false
  let resize_start_pointer: Point = { x: 0, y: 0 }
  let resize_start_size: Size = { w: 0, h: 0 }

  function onMove(e: MouseEvent | TouchEvent): void {
    if (resizing) {
      if ("touches" in e) e.preventDefault()
      const p = get_pointer_xy(e)
      const max_w = window.innerWidth * 0.95
      const max_h = window.innerHeight * 0.95
      user_size.value = {
        w: clamp(resize_start_size.w + (p.x - resize_start_pointer.x), min_w, max_w),
        h: clamp(resize_start_size.h + (p.y - resize_start_pointer.y), min_h, max_h),
      }
      update_resized_class()
      return
    }

    if (!dragging) return

    // touch でのページスクロールを抑制
    if ("touches" in e) e.preventDefault()

    const p = get_pointer_xy(e)
    const dx = p.x - start_pointer.x
    const dy = p.y - start_pointer.y

    pos.value = { x: start_pos.x + dx, y: start_pos.y + dy }
    clamp_to_viewport()
  }

  function onUp(): void {
    if (resizing) {
      resizing = false
      if (user_size.value) save_size(size_key, user_size.value)
      return
    }
    if (!dragging) return
    dragging = false
    persist_pos()
  }

  function onHeaderPointerDown(e: MouseEvent | TouchEvent): void {
    // ✅ ヘッダー内の操作要素タップではドラッグ開始しない
    if (is_interactive_target(e.target)) return

    // 掴んだ瞬間に rect 更新・clamp（画面外スタート防止）
    read_rect()
    clamp_to_viewport()

    dragging = true
    start_pointer = get_pointer_xy(e)
    start_pos = { ...pos.value }

    // touchstart を抑制しないと「タップ→スクロール」判定が混ざって変な挙動になりがち
    if ("touches" in e) e.preventDefault()
  }

  function onResizePointerDown(e: MouseEvent | TouchEvent): void {
    e.preventDefault()
    e.stopPropagation()

    const rect = container_ref.value?.getBoundingClientRect()
    if (!rect) return

    resizing = true
    resize_start_pointer = get_pointer_xy(e)
    resize_start_size = { w: rect.width, h: rect.height }
  }

  function reset_to_center(): void {
    // サイズが取れない瞬間があるので、まず概算→次フレームで確定
    const r0 = read_rect()
    const estimate_w = r0.w > 0 ? r0.w : Math.min(720, window.innerWidth * 0.85)
    const estimate_h = r0.h > 0 ? r0.h : window.innerHeight * 0.6

    pos.value = {
      x: Math.round((window.innerWidth - estimate_w) / 2) + center_offset.x,
      y: Math.round((window.innerHeight - estimate_h) / 2) + center_offset.y,
    }
    clamp_to_viewport()
    persist_pos()

    requestAnimationFrame(() => {
      const r1 = read_rect()
      if (r1.w > 0 && r1.h > 0) {
        pos.value = {
          x: Math.round((window.innerWidth - r1.w) / 2) + center_offset.x,
          y: Math.round((window.innerHeight - r1.h) / 2) + center_offset.y,
        }
        clamp_to_viewport()
        persist_pos()
      }
    })
  }

  function reset_size(): void {
    user_size.value = null
    safe_remove(size_key)
    update_resized_class()
  }

  // 初回中央寄せの実行フラグ
  let did_auto_center = false

  function auto_center_if_needed(): void {
    if (center_mode === "never") return
    if (center_mode === "always") {
      reset_to_center()
      return
    }

    // center_mode === "first"
    if (did_auto_center) return
    if (!has_saved_pos) {
      reset_to_center()
      did_auto_center = true
    }
  }

  function attach_observer(el: HTMLElement): void {
    if (!ro) return
    if (observed_el) {
      try {
        ro.unobserve(observed_el)
      } catch {
        // noop
      }
    }
    observed_el = el
    ro.observe(el)
  }

  function detach_observer(): void {
    if (!ro || !observed_el) return
    try {
      ro.unobserve(observed_el)
    } catch {
      // noop
    }
    observed_el = null
  }

  function onResize(): void {
    clamp_to_viewport()
    persist_pos()
  }

  // リサイズハンドル要素の管理
  let resize_handle: HTMLElement | null = null

  function create_resize_handle(parent: HTMLElement): void {
    if (!resizable || resize_handle) return
    resize_handle = document.createElement("div")
    resize_handle.className = "gkill-floating-dialog__resize-handle"
    resize_handle.addEventListener("mousedown", onResizePointerDown as EventListener)
    resize_handle.addEventListener("touchstart", onResizePointerDown as EventListener, { passive: false })
    parent.appendChild(resize_handle)
  }

  function remove_resize_handle(): void {
    if (!resize_handle) return
    resize_handle.removeEventListener("mousedown", onResizePointerDown as EventListener)
    resize_handle.removeEventListener("touchstart", onResizePointerDown as EventListener)
    resize_handle.remove()
    resize_handle = null
  }

  // 触ったダイアログを最前面へ。capture で拾うのは、中の要素が
  // stopPropagation していても取りこぼさないため
  let bring_to_front_target: HTMLElement | null = null

  function onBringToFront(): void {
    const raised = raise_z_order(z_token)
    if (raised.length === 0) return
    // バックと Escape が閉じる対象も、見た目の最前面に合わせる
    const owners: Array<object> = []
    for (const token of raised) {
      const owner = floating_dialog_owners.get(token)
      if (owner !== undefined) owners.push(owner)
    }
    raise_dialog_history_entries(owners)
  }

  // iframe の中で起きた pointerdown / focusin は**親のDOMへ一切伝わらない**ので、
  // 本文が iframe のダイアログ（マニュアル・チュートリアル・プラグイン本文）は
  // 上の2つだけだと「本文をクリックしても前面に来ない」。ヘッダを掴んだときしか上がらない。
  //
  // iframe をクリックするとフォーカスが入れ子の閲覧文脈へ移り、
  // 親では window の blur が起きて `document.activeElement` がその iframe になる。
  // 親から観測できる合図はこれだけなので、これを前面化の入口にする。
  //
  // 別アプリへ切り替えたときも window の blur は起きるが、そのとき
  // activeElement が自分の中の iframe だということは直前に本文を触っていたということで、
  // つまりすでに最前面。`raise_z_order` は最前面なら何もしないので実害は無い。
  function onWindowBlurForBringToFront(): void {
    const el = bring_to_front_target
    if (!el) return
    const active = document.activeElement
    if (!(active instanceof HTMLIFrameElement)) return
    if (!el.contains(active)) return
    onBringToFront()
  }

  function attach_bring_to_front(el: HTMLElement): void {
    detach_bring_to_front()
    bring_to_front_target = el
    el.addEventListener("pointerdown", onBringToFront, true)
    el.addEventListener("focusin", onBringToFront, true)
    window.addEventListener("blur", onWindowBlurForBringToFront)
  }

  function detach_bring_to_front(): void {
    if (!bring_to_front_target) return
    bring_to_front_target.removeEventListener("pointerdown", onBringToFront, true)
    bring_to_front_target.removeEventListener("focusin", onBringToFront, true)
    window.removeEventListener("blur", onWindowBlurForBringToFront)
    bring_to_front_target = null
  }

  onMounted(() => {
    ro = new ResizeObserver(() => {
      // リサイズ中はユーザ操作を優先し、clamp を抑制
      if (resizing) return
      // 内容サイズ変化 → 画面外に出ないように補正
      read_rect()
      clamp_to_viewport()
      persist_pos()
    })

    window.addEventListener("resize", onResize, { passive: true })
    window.addEventListener("mousemove", onMove as EventListener, { passive: true })
    window.addEventListener("mouseup", onUp as EventListener, { passive: true })
    window.addEventListener("touchmove", onMove as EventListener, { passive: false })
    window.addEventListener("touchend", onUp as EventListener, { passive: true })
  })

  onBeforeUnmount(() => {
    detach_autofocus()
    detach_escape_handler()
    detach_bring_to_front()
    leave_z_order(z_token)
    remove_resize_handle()
    detach_observer()
    if (ro) ro.disconnect()
    ro = null

    window.removeEventListener("resize", onResize as EventListener)
    window.removeEventListener("mousemove", onMove as EventListener)
    window.removeEventListener("mouseup", onUp as EventListener)
    window.removeEventListener("touchmove", onMove as EventListener)
    window.removeEventListener("touchend", onUp as EventListener)
  })

  // ✅ Teleport の v-if で DOM が生えた瞬間に observer attach & 中央寄せ & リサイズハンドル注入
  watch(
    container_ref,
    (el) => {
      if (!el) {
        detach_autofocus()
        detach_escape_handler()
        detach_bring_to_front()
        leave_z_order(z_token)
        remove_resize_handle()
        detach_observer()
        return
      }

      // 後から開いたものが前に出る（従来の「Teleport の mount 順」と同じ結果）
      enter_z_order(z_token)
      attach_bring_to_front(el)

      if (ro) attach_observer(el)

      // リサイズハンドルを注入
      create_resize_handle(el)

      // is-user-resized クラスを反映
      update_resized_class()

      // Accessibility: ARIA attributes, escape handler, focus management
      apply_aria_attributes(el)
      attach_escape_handler(el)
      attach_autofocus(el)

      // 出現直後は rect が 0 のことがあるので次フレームで処理
      requestAnimationFrame(() => {
        read_rect()

        // 中央寄せが必要なら実行、不要なら画面内に収めるだけ
        auto_center_if_needed()
        clamp_to_viewport()
        persist_pos()
      })

    },
    { flush: "post" },
  )

  watch(is_transparent, (v) => save_bool(transparent_key, v), { immediate: true })

  return {
    containerRef: container_ref,
    fixedStyle: fixed_style,
    onHeaderPointerDown,
    isTransparent: is_transparent,
    resetToCenter: reset_to_center,
    resetSize: reset_size,
    userSize: user_size as Readonly<Ref<Size | null>>,
  }
}
