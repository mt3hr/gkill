/**
 * フローティングダイアログの重なり順。
 *
 * z-index は「開いているダイアログの並び順」から出す。単調増加のカウンタにすると
 * Vuetify の overlay（メニュー / ツールチップ、2400）を追い越して、ダイアログの中の
 * メニューがダイアログの下へ潜る。ここではその上限と、
 * 「親をクリックしても、その親から開いた確認ダイアログは前に残る」ことを固定する。
 */
import { afterEach, beforeEach, describe, test, expect } from 'vitest'
import { createApp, defineComponent, h, nextTick, ref, type App, type Ref } from 'vue'

import { useFloatingDialog } from '@/classes/use-floating-dialog'

const BASE_Z_INDEX = 1100

beforeEach(() => {
    // jsdom に ResizeObserver は無い。useFloatingDialog は onMounted で必ず作る
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (globalThis as any).ResizeObserver = class {
        observe(): void { }
        unobserve(): void { }
        disconnect(): void { }
    }
})

interface MountedDialog {
    /** 親ダイアログの中から開く確認ダイアログ相当 */
    show_child: Ref<boolean>
    app: App<Element>
    z_index(): number
    child_z_index(): number
    click(): void
}

const mounted: Array<MountedDialog> = []

afterEach(() => {
    while (mounted.length > 0) {
        mounted.pop()?.app.unmount()
    }
    document.body.innerHTML = ''
})

/**
 * ダイアログを1枚マウントする。
 *
 * 子ダイアログは Teleport を使わず入れ子で描くが、z-order の親子は provide/inject で
 * 決まる（コンポーネント木は Teleport をまたいでも保たれる）ので、判定はこれで足りる。
 */
async function mount_dialog(): Promise<MountedDialog> {
    const show_child: Ref<boolean> = ref(false)

    const ChildDialog = defineComponent({
        setup() {
            const ui = useFloatingDialog('test-child-dialog', { centerMode: 'never' })
            return () => h('div', { class: 'child', ref: ui.containerRef, style: ui.fixedStyle.value })
        },
    })

    const ParentDialog = defineComponent({
        setup() {
            const ui = useFloatingDialog('test-parent-dialog', { centerMode: 'never' })
            return () => h(
                'div',
                { class: 'parent', ref: ui.containerRef, style: ui.fixedStyle.value },
                show_child.value ? [h(ChildDialog)] : [],
            )
        },
    })

    const app = createApp(ParentDialog)
    const host = document.createElement('div')
    document.body.appendChild(host)
    app.mount(host)
    // container_ref の watch は flush: 'post'。そこで並び順へ入るので、
    // style へ反映されるのはさらに次の描画
    await nextTick()
    await nextTick()

    const parent_element = host.querySelector('.parent') as HTMLElement

    const dialog: MountedDialog = {
        show_child: show_child,
        app: app,
        z_index: () => Number(parent_element.style.zIndex),
        child_z_index: () => Number((host.querySelector('.child') as HTMLElement).style.zIndex),
        click: () => parent_element.dispatchEvent(new Event('pointerdown', { bubbles: true })),
    }
    mounted.push(dialog)
    return dialog
}

describe('フローティングダイアログの重なり順', () => {
    test('後から開いたダイアログのほうが前に出る', async () => {
        const first = await mount_dialog()
        const second = await mount_dialog()

        expect(first.z_index()).toBe(BASE_Z_INDEX)
        expect(second.z_index()).toBeGreaterThan(first.z_index())
    })

    test('z-index は開いている枚数ぶんしか伸びない（Vuetify の overlay を追い越さない）', async () => {
        const dialogs: Array<MountedDialog> = []
        for (let i = 0; i < 5; i++) {
            dialogs.push(await mount_dialog())
        }
        // 何度前面化しても増えないこと
        for (let i = 0; i < 20; i++) {
            dialogs[i % dialogs.length].click()
        }
        await nextTick()

        const max_z_index = Math.max(...dialogs.map(dialog => dialog.z_index()))
        expect(max_z_index).toBe(BASE_Z_INDEX + dialogs.length - 1)
        expect(max_z_index, 'Vuetify の overlay(2400) を追い越している').toBeLessThan(2400)
    })

    test('クリックしたダイアログが最前面へ出る', async () => {
        const first = await mount_dialog()
        const second = await mount_dialog()
        expect(first.z_index()).toBeLessThan(second.z_index())

        first.click()
        await nextTick()

        expect(first.z_index()).toBeGreaterThan(second.z_index())
    })

    // 確認ダイアログは Teleport で親の兄弟になるので、素朴に前面化すると
    // 「親をクリックしただけで確認が後ろへ隠れる」
    test('親を前面へ出しても、その親から開いた確認ダイアログは前に残る', async () => {
        const parent = await mount_dialog()
        parent.show_child.value = true
        await nextTick()

        const other = await mount_dialog()
        expect(other.z_index()).toBeGreaterThan(parent.child_z_index())

        parent.click()
        await nextTick()

        expect(parent.z_index()).toBeGreaterThan(other.z_index())
        expect(parent.child_z_index(), '確認ダイアログが親の後ろへ隠れた').toBeGreaterThan(parent.z_index())
    })

    test('閉じたダイアログは並びから抜ける', async () => {
        const first = await mount_dialog()
        const second = await mount_dialog()
        expect(second.z_index()).toBe(BASE_Z_INDEX + 1)

        second.app.unmount()
        mounted.splice(mounted.indexOf(second), 1)
        await nextTick()

        const third = await mount_dialog()
        expect(first.z_index()).toBe(BASE_Z_INDEX)
        expect(third.z_index()).toBe(BASE_Z_INDEX + 1)
    })
})
