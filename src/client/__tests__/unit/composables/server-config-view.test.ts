/**
 * useServerConfigView（サーバ設定画面）の検証。
 *
 * この画面は props の ServerConfig を編集用に複製してから v-model で直接書き換える。
 * 複製が `concat()`（配列だけの浅いコピー）だと要素は props と同一参照のままなので、
 * 「適用」を押す前から props 側 ―― つまり設定ダイアログを開いた側が持っている
 * サーバ設定 ―― が書き換わってしまい、キャンセルしても戻らなくなる。
 * 要素ごとに clone() していることをここで固定する。
 */
import { describe, expect, test, vi } from 'vitest'

// req_res は GkillAPIRequest を継承する。GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の
// 循環importがあるため、本番同様に gkill-api を先に評価させる
import '@/classes/api/gkill-api'

vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

import { nextTick, reactive } from 'vue'
import { ServerConfig } from '@/classes/datas/config/server-config'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { useServerConfigView } from '@/classes/use-server-config-view'
import type { ServerConfigViewProps } from '@/pages/views/server-config-view-props'
import type { ServerConfigViewEmits } from '@/pages/views/server-config-view-emits'
import type { GkillError } from '@/classes/api/gkill-error'

/**
 * 本番と同じ経路で props を組む。
 * サーバから来るのはメソッドを持たない生JSONで、GkillAPI.get_server_configs が
 * Object.assign で ServerConfig に詰め直している。ここで直接 new ServerConfig() を
 * 組み立ててしまうと、詰め直しを外したときにこのテストだけ通ってしまう
 */
function make_server_config(device: string, options: { enable_this_device: boolean, address: string }): ServerConfig {
    const json = {
        device: device,
        enable_this_device: options.enable_this_device,
        address: options.address,
        urlog_timeout: 60_000_000_000,
        gkill_notification_private_key: 'private-key',
    }
    return Object.assign(new ServerConfig(), json)
}

function make_server_configs(): Array<ServerConfig> {
    return [
        make_server_config('desktop', { enable_this_device: true, address: ':9999' }),
        make_server_config('laptop', { enable_this_device: false, address: ':8888' }),
    ]
}

interface EmittedEvent {
    event: string
    args: Array<unknown>
}

function create_view(server_configs: Array<ServerConfig> = make_server_configs()) {
    const emitted = new Array<EmittedEvent>()
    const emits = ((event: string, ...args: Array<unknown>) => {
        emitted.push({ event: event, args: args })
    }) as unknown as ServerConfigViewEmits
    const props = reactive({
        server_configs: server_configs,
        application_config: {},
        gkill_api: { update_server_config: vi.fn().mockResolvedValue({ messages: null, errors: null }) },
    }) as unknown as ServerConfigViewProps
    const view = useServerConfigView({ props: props, emits: emits })
    return { view: view, props: props, emitted: emitted, server_configs: server_configs }
}

// 初期化は nextTick コールバックの中で clone() を await するので、数tick回す
async function flush(): Promise<void> {
    for (let i = 0; i < 5; i++) {
        await nextTick()
    }
}

function errors_of(emitted: Array<EmittedEvent>): Array<GkillError> {
    const errors = new Array<GkillError>()
    for (const entry of emitted) {
        if (entry.event === 'received_errors') {
            errors.push(...(entry.args[0] as Array<GkillError>))
        }
    }
    return errors
}

describe('props の複製', () => {
    test('要素ごとに clone するので、props の要素と同一参照にならない', async () => {
        const { view, server_configs } = create_view()
        await flush()

        expect(view.cloned_server_configs.value).toHaveLength(2)
        for (let i = 0; i < server_configs.length; i++) {
            expect(
                view.cloned_server_configs.value[i],
                'concat() の浅いコピーになっている（要素が props と同じインスタンス）',
            ).not.toBe(server_configs[i])
        }
    })

    test('編集中の値は適用まで props 側へ漏れない', async () => {
        const { view, server_configs } = create_view()
        await flush()

        // 入力欄は v-model="server_config.address" のようにオブジェクトを直接書き換える
        view.server_config.value.address = '19999'
        view.cloned_server_configs.value[1].address = '18888'
        await flush()

        expect(server_configs[0].address, '適用前に props 側が書き換わっている').toBe(':9999')
        expect(server_configs[1].address, '適用前に props 側が書き換わっている').toBe(':8888')
    })

    test('現在のデバイスが選択され、その設定が編集対象になる', async () => {
        const { view } = create_view()
        await flush()

        expect(view.device.value).toBe('desktop')
        expect(view.server_config.value.device).toBe('desktop')
        expect(view.devices.value).toEqual(['desktop', 'laptop'])
    })

    // 画面に出ていない項目も複製に載っていないと、「適用」が全レコード
    // DELETE→再INSERTなので保存時に消える。とくに通知の秘密鍵が空になると
    // サーバがVAPID鍵を作り直し、既存のプッシュ通知購読が全部切れる
    test('画面に出ていない項目も複製に引き継がれる', async () => {
        const { view } = create_view()
        await flush()

        expect(view.server_config.value.gkill_notification_private_key).toBe('private-key')
        expect(view.server_config.value.urlog_timeout).toBe(60_000_000_000)
    })
})

// 有効な端末が1つも無いDBでも、初期化で落ちずにエラーを通知して止まること。
// 以前は filter(...)[0].device と書いていたので TypeError になり、
// nextTick の中で握り潰されて画面が空のまま無言で固まっていた
describe('有効な端末が見つからない場合', () => {
    test('例外にならず not_found_enable_device を通知する', async () => {
        const { view, emitted } = create_view([
            make_server_config('desktop', { enable_this_device: false, address: ':9999' }),
            make_server_config('laptop', { enable_this_device: false, address: ':8888' }),
        ])
        await flush()

        expect(view.devices.value, '端末一覧の読み込みまで巻き添えで止まっている').toEqual(['desktop', 'laptop'])
        const errors = errors_of(emitted)
        expect(errors).toHaveLength(1)
        expect(errors[0].error_code).toBe(GkillErrorCodes.not_found_enable_device)
    })
})

// 設定ダイアログは get_server_configs の応答を待たずに開き、このviewを
// v-show で隠すだけ（v-if ではない）ので、空の props でも初期化が走る。
// それを「有効なプロファイルが見つかりません」として通知していたため、
// ダイアログを開くたびに毎回エラーが出ていた
describe('サーバ設定がまだ届いていない場合', () => {
    test('0件では通知しない', async () => {
        const { view, emitted } = create_view([])
        await flush()

        expect(errors_of(emitted), '未ロードをエラーとして通知している').toHaveLength(0)
        expect(view.devices.value).toEqual([])
    })

    test('応答が届いたら改めて初期化され、エラーも出ない', async () => {
        const { view, props, emitted } = create_view([])
        await flush()

        props.server_configs = make_server_configs()
        await flush()

        expect(view.device.value).toBe('desktop')
        expect(view.server_config.value.address).toBe(':9999')
        expect(view.devices.value).toEqual(['desktop', 'laptop'])
        expect(errors_of(emitted)).toHaveLength(0)
    })
})

// Go/DBはナノ秒の time.Duration、入力欄は秒。
// 変換を挟まないと入力欄に 60000000000 が出て、30と入れると30ナノ秒が保存される
describe('urlog_timeout_sec', () => {
    test('ナノ秒を秒にして見せる', async () => {
        const { view } = create_view()
        await flush()

        expect(view.urlog_timeout_sec.value).toBe(60)
    })

    test('秒で書き込むとナノ秒で保持される', async () => {
        const { view } = create_view()
        await flush()

        view.urlog_timeout_sec.value = 30
        await flush()

        expect(view.server_config.value.urlog_timeout).toBe(30_000_000_000)
        expect(view.urlog_timeout_sec.value).toBe(30)
    })
})

describe('add_device', () => {
    test('既存と同じ名前ならエラーを出して追加しない', async () => {
        const { view, emitted } = create_view()
        await flush()

        view.onSettedNewDeviceName('laptop')
        await flush()

        expect(view.cloned_server_configs.value, '重複した名前で追加している').toHaveLength(2)
        expect(view.device.value, '重複時にデバイスを切り替えてはいけない').toBe('desktop')
        const errors = errors_of(emitted)
        expect(errors).toHaveLength(1)
        expect(errors[0].error_code).toBe(GkillErrorCodes.device_is_already_exist)
    })

    test('新しい名前なら現在値のコピーを足して、そのデバイスへ切り替える', async () => {
        const { view, emitted, server_configs } = create_view()
        await flush()
        view.server_config.value.address = '19999'
        view.server_config.value.use_gkill_notification = true

        view.onSettedNewDeviceName('tablet')
        await flush()

        expect(view.cloned_server_configs.value).toHaveLength(3)
        expect(view.devices.value).toEqual(['desktop', 'laptop', 'tablet'])
        expect(view.device.value, '追加したデバイスへ切り替わっていない').toBe('tablet')
        expect(view.server_config.value.device).toBe('tablet')
        // 現在値のコピーであること（既定値の新規 ServerConfig ではない）
        expect(view.server_config.value.address).toBe('19999')
        expect(view.server_config.value.use_gkill_notification).toBe(true)
        expect(errors_of(emitted)).toHaveLength(0)
        // 追加もまた「適用」までは props に触らない
        expect(server_configs).toHaveLength(2)
    })

    test('切り替えると enable_this_device が新しいデバイスだけ true になる', async () => {
        const { view } = create_view()
        await flush()

        view.onSettedNewDeviceName('tablet')
        await flush()

        const enabled = view.cloned_server_configs.value.filter((config) => config.enable_this_device)
        expect(enabled).toHaveLength(1)
        expect(enabled[0].device).toBe('tablet')
    })
})

describe('delete_current_server_config', () => {
    test('削除後は先頭の設定が選択される', async () => {
        const { view } = create_view()
        await flush()
        view.device.value = 'laptop'
        await flush()
        expect(view.server_config.value.device).toBe('laptop')

        view.delete_current_server_config()
        await flush()

        expect(view.cloned_server_configs.value).toHaveLength(1)
        expect(view.devices.value).toEqual(['desktop'])
        expect(view.device.value, '削除後に選択が残骸のままになっている').toBe('desktop')
        expect(view.server_config.value.device).toBe('desktop')
        expect(
            view.cloned_server_configs.value[0].enable_this_device,
            '選択し直したデバイスが有効になっていない',
        ).toBe(true)
    })

    // 全件消すとサーバ設定が0件になり、「有効なプロファイルがちょうど1件」を
    // 前提にしている起動処理が通らなくなる。
    // 以前は splice 後の cloned_server_configs[0] を無条件に読んでいたので、
    // 最後の1件では undefined.device で TypeError になっていた
    test('最後の1件は削除しない', async () => {
        const { view, emitted } = create_view([
            make_server_config('desktop', { enable_this_device: true, address: ':9999' }),
        ])
        await flush()

        view.delete_current_server_config()
        await flush()

        expect(view.cloned_server_configs.value).toHaveLength(1)
        expect(view.devices.value).toEqual(['desktop'])
        expect(view.device.value).toBe('desktop')
        expect(view.server_config.value.device).toBe('desktop')
        expect(errors_of(emitted)).toHaveLength(0)
    })

    test('削除しても props 側の配列は減らない（適用まで確定しない）', async () => {
        const { view, server_configs } = create_view()
        await flush()

        view.delete_current_server_config()
        await flush()

        expect(server_configs).toHaveLength(2)
    })
})
