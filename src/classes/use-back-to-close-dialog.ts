// useBackToCloseDialog.ts
import type { Ref } from 'vue'
import { watch, onBeforeUnmount } from 'vue'

type Entry = { id: string; dialog: Ref<boolean> }

const stack: Entry[] = []
const backClosingIds = new Set<string>()

let listening = false
let ignoreNextPopstate = 0

function onPopState() {
    if (ignoreNextPopstate > 0) {
        ignoreNextPopstate--
        return
    }

    const top = stack[stack.length - 1]
    if (!top) return

    // 「Backで閉じた」ことを印付けして、watch側が掃除(back)を二重にしないようにする
    backClosingIds.add(top.id)
    top.dialog.value = false
}

function ensureListener() {
    if (listening) return
    window.addEventListener('popstate', onPopState, { passive: true })
    listening = true
}

function maybeRemoveListener() {
    if (!listening) return
    if (stack.length > 0) return
    window.removeEventListener('popstate', onPopState)
    listening = false
}

function makeId() {
    // randomUUIDが無い環境向けにフォールバック
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const c = (globalThis as any).crypto
    return c?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

export function useBackToCloseDialog(dialog: Ref<boolean>) {
    const id = makeId()
    let pushed = false

    watch(
        dialog,
        (open) => {
            if (open) {
                // スタックに登録（入れ子は積み上がる）
                if (!stack.some((e) => e.id === id)) stack.push({ id, dialog })

                // このダイアログの分の「戻る1段」を作る（URLは変えない）
                if (!pushed) {
                    history.pushState({ __dlg: true, id }, '')
                    pushed = true
                }

                ensureListener()
                return
            }

            // close
            const idx = stack.findIndex((e) => e.id === id)
            if (idx !== -1) stack.splice(idx, 1)

            if (pushed) {
                if (backClosingIds.has(id)) {
                    // ユーザーがBackした結果閉じた：履歴はすでに戻ってるので何もしない
                    backClosingIds.delete(id)
                } else {
                    // ×ボタン/外側クリック等で閉じた：積んだ履歴を掃除する
                    ignoreNextPopstate++
                    history.back()
                }
                pushed = false
            }

            maybeRemoveListener()
        },
        { immediate: false }
    )

    onBeforeUnmount(() => {
        // コンポーネント破棄時にスタックから除去
        const idx = stack.findIndex((e) => e.id === id)
        if (idx !== -1) stack.splice(idx, 1)
        maybeRemoveListener()
    })
}
