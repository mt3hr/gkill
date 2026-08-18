/**
 * 起動時チュートリアルは「起動時に1回だけ」。
 *
 * `application_config` は `use-config-struct-sync.ts` が未知のタグ / リポジトリ /
 * 端末を見つけるたびに `clone()` を代入して **ref ごと差し替える**（深い変更だけでは
 * `watch(() => props.application_config, ...)` が発火しないため意図的）。
 * `clone()` は `is_loaded` も `show_tutorial_on_startup` も引き継ぐので、
 * 素朴な watch のままだと**タグを1つ足すたびにチュートリアルが開き直す**。
 * 追加/編集ダイアログからタグを打てるようになって以降、日常的に踏む経路。
 */
import { describe, expect, test, vi } from 'vitest'
import { nextTick, ref } from 'vue'

import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { useTutorialOnStartup, type TutorialDialogRef } from '@/classes/use-tutorial-on-startup'

function make_config(is_loaded: boolean, show_tutorial: boolean): ApplicationConfig {
    const config = new ApplicationConfig()
    config.is_loaded = is_loaded
    config.show_tutorial_on_startup = show_tutorial
    return config
}

function make_harness(show_tutorial = true) {
    const show = vi.fn()
    const application_config = ref<ApplicationConfig>(make_config(false, show_tutorial))
    const tutorial_dialog = ref<TutorialDialogRef | null>({ show: show })
    useTutorialOnStartup(application_config, tutorial_dialog)
    return { show: show, application_config: application_config }
}

/** 設定の到着（未ロード → ロード済み）を再現する */
async function load(harness: ReturnType<typeof make_harness>, show_tutorial = true): Promise<void> {
    harness.application_config.value = make_config(true, show_tutorial)
    await nextTick()
    await nextTick()
}

describe('useTutorialOnStartup', () => {
    test('設定が届いたらチュートリアルを出す', async () => {
        const harness = make_harness()

        await load(harness)

        expect(harness.show).toHaveBeenCalledTimes(1)
    })

    // これが本題。clone() の代入は「タグを1つ足した」だけで何度も起きる
    test('設定の ref が差し替わっても二度目は出さない', async () => {
        const harness = make_harness()
        await load(harness)

        for (let i = 0; i < 3; i++) {
            harness.application_config.value = harness.application_config.value.clone()
            await nextTick()
            await nextTick()
        }

        expect(harness.show, '設定のref差し替えでチュートリアルが開き直している').toHaveBeenCalledTimes(1)
    })

    test('未ロードの設定では出さない', async () => {
        const harness = make_harness()

        harness.application_config.value = make_config(false, true)
        await nextTick()
        await nextTick()

        expect(harness.show).not.toHaveBeenCalled()
    })

    test('show_tutorial_on_startup が false なら出さない', async () => {
        const harness = make_harness(false)

        await load(harness, false)

        expect(harness.show).not.toHaveBeenCalled()
    })

    // 一度出さずに済んだあと設定で ON にされたら、次の起動から出せばよい。
    // ここで出すと設定画面を閉じた瞬間にチュートリアルが被さる
    test('あとから ON にされても、その場では出さない', async () => {
        const harness = make_harness(false)
        await load(harness, false)

        harness.application_config.value = make_config(true, true)
        await nextTick()
        await nextTick()

        expect(harness.show, '設定を変えた直後にチュートリアルが被さっている').not.toHaveBeenCalled()
    })
})
