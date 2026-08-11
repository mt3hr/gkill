'use strict'

import { shallowRef, type ShallowRef } from 'vue'
import { GkillAPI } from '@/classes/api/gkill-api'
import { GetPluginListRequest } from '@/classes/api/req_res/get-plugin-list-request'

/**
 * プラグインが提供しているリポジトリ名の集合。
 *
 * プラグインが返した記録は読み取り専用で、gkill側から書き換えられない
 * （マニュアルにもそう書いてある）。ところがプラグインが型別データを
 * 提供すると、その記録はネイティブと同じビューで描かれるため、
 * 編集・削除メニューが付いてしまう。それを抑止するための判定に使う。
 *
 * 状態はモジュールレベルのシングルトンで持つ。
 * コンテキストメニューは行数ぶんインスタンス化されるので、
 * コンポーネントごとに持つと同じ一覧を何度も取りに行くことになる。
 */
const plugin_rep_names: ShallowRef<Set<string>> = shallowRef(new Set<string>())

/** 取得中・取得済みの状態。二重取得を防ぐ。 */
let load_promise: Promise<void> | null = null

async function load_plugin_rep_names(): Promise<void> {
    const req = new GetPluginListRequest()
    const res = await GkillAPI.get_gkill_api().get_plugin_list(req)
    // 成功時 errors は null で返ってくるので、素のスプレッドはしない
    const errors = res.errors ?? []
    if (errors.length !== 0) {
        return
    }
    const names = new Set<string>()
    for (const plugin of res.plugins ?? []) {
        if (plugin.rep_name !== '') {
            names.add(plugin.rep_name)
        }
    }
    plugin_rep_names.value = names
}

/**
 * プラグイン由来のリポジトリ名を扱うコンポーザブル。
 *
 * 初回呼び出しで1度だけ `/api/get_plugin_list` を引き、以降は使い回す。
 * 取得が終わるまで `is_plugin_rep` は false を返すので、
 * 一瞬だけ編集メニューが出ることはありうる。編集されても破綻はしない
 * （追記式なので、gkill側に同一IDの新しい版が積まれて上書きされるだけ）ので、
 * 取得を待ってメニューの表示自体を遅らせることはしない。
 */
export function usePluginRepNames(): {
    plugin_rep_names: ShallowRef<Set<string>>
    is_plugin_rep: (rep_name: string | null | undefined) => boolean
} {
    if (load_promise === null) {
        load_promise = load_plugin_rep_names().catch(() => {
            // 取れなくても致命的ではない。次のマウントで引き直せるように戻す
            load_promise = null
        })
    }

    const is_plugin_rep = (rep_name: string | null | undefined): boolean => {
        if (!rep_name) {
            return false
        }
        return plugin_rep_names.value.has(rep_name)
    }

    return { plugin_rep_names, is_plugin_rep }
}

/**
 * 取得済みの一覧を捨てる。テスト用。
 */
export function reset_plugin_rep_names_for_test(): void {
    plugin_rep_names.value = new Set<string>()
    load_promise = null
}
