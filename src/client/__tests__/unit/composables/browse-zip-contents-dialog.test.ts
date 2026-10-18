/**
 * ZIPの中を辿るダイアログ（`useBrowseZipContentsDialog`）の純ロジック。
 *
 * このコンポーザブルは365行あってテストが1本も無く、しかも
 * `src/client/ABOUT_TEST.md` が「ZIPファイルブラウズダイアログをE2Eでカバー」と
 * 書いていたのに **E2Eに zip という語が1つも無かった**（虚偽のカバレッジ申告）。
 *
 * 中身のうちサーバに触らない部分——フォルダ階層の導出とビューワーの巡回——は
 * 全部ここで固定できる。壊れたときの出方が「フォルダが1つ多い/少ない」
 * 「隣の画像へ進めない」のように、エラーを出さず表示だけおかしくなる種類なので、
 * 機械で見張る価値が高い。
 */
import { describe, expect, test, vi, beforeEach, afterEach } from 'vitest'
import { createApp, defineComponent, h, nextTick } from 'vue'

vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

// **この副作用インポートを消さないこと（順番も動かさないこと）。**
// gkill-api-request.ts が gkill-api.ts を import し、gkill-api.ts は
// application-config.ts 経由で各 Request クラスへ戻ってくる循環がある。
// アプリでは gkill-api.ts が入口なので解けるが、テストが Request 側から入ると
// 基底クラスが undefined のまま extends され
// `Class extends value undefined is not a constructor` で テストファイルごと落ちる。
import '@/classes/api/gkill-api'
import { useBrowseZipContentsDialog } from '@/classes/use-browse-zip-contents-dialog'
import type { ZipEntry } from '@/classes/api/req_res/browse-zip-contents-response'

// ダイアログの UI（useFloatingDialog）はコンポーザブル側に入っている。
// jsdom には ResizeObserver が無いので最小の実装を差し込む
(globalThis as unknown as { ResizeObserver: unknown }).ResizeObserver = class {
    observe(): void { }
    unobserve(): void { }
    disconnect(): void { }
}

type Dialog = ReturnType<typeof useBrowseZipContentsDialog>

function entry(path: string, overrides: Partial<ZipEntry> = {}): ZipEntry {
    return {
        path: path,
        is_dir: false,
        size: 100,
        is_image: false,
        is_text: false,
        is_video: false,
        is_audio: false,
        is_pdf: false,
        file_url: 'https://example.invalid/' + path,
        ...overrides,
    }
}

let mounted_apps = new Array<ReturnType<typeof createApp>>()

/** onMounted / useDialogHistoryStack が動くよう、ホストコンポーネントの中で呼ぶ */
function mount_dialog(entries: Array<ZipEntry>): Dialog {
    let dialog: Dialog | null = null
    const Host = defineComponent({
        setup() {
            dialog = useBrowseZipContentsDialog({
                props: {
                    kyou: { id: 'kyou-1' },
                    gkill_api: {
                        browse_zip_contents: vi.fn().mockResolvedValue({
                            entries: entries, errors: null, messages: null,
                        }),
                    },
                } as unknown as Parameters<typeof useBrowseZipContentsDialog>[0]['props'],
                emits: vi.fn() as unknown as Parameters<typeof useBrowseZipContentsDialog>[0]['emits'],
            })
            return () => h('div')
        },
    })
    const app = createApp(Host)
    app.mount(document.createElement('div'))
    mounted_apps.push(app)
    const mounted_dialog = dialog!
    mounted_dialog.all_entries.value = entries
    return mounted_dialog
}

beforeEach(() => {
    mounted_apps = []
})
afterEach(() => {
    for (const app of mounted_apps) {
        app.unmount()
    }
    vi.restoreAllMocks()
})

// ZIP は「フォルダのエントリ」を持たないことがある（圧縮ソフト次第）。
// なのでフォルダ一覧はファイルのパスから導出するしかない
const SAMPLE = [
    entry('readme.txt', { is_text: true }),
    entry('img/a.png', { is_image: true }),
    entry('img/b.png', { is_image: true }),
    entry('img/sub/c.png', { is_image: true }),
    entry('doc/', { is_dir: true }),
    entry('doc/note.md', { is_text: true }),
    entry('movie.mp4', { is_video: true }),
]

describe('フォルダ階層の導出', () => {
    test('ルートの直下フォルダは、明示エントリが無くてもパスから導出する', () => {
        const dialog = mount_dialog(SAMPLE)
        expect(dialog.current_subdirs.value.map(subdir => subdir.name)).toEqual(['doc', 'img'])
        expect(dialog.current_subdirs.value.map(subdir => subdir.path)).toEqual(['doc', 'img'])
    })

    test('ルートのファイルは直下のものだけ（孫は含めない）', () => {
        const dialog = mount_dialog(SAMPLE)
        expect(dialog.current_files.value.map(file => file.path)).toEqual(['readme.txt', 'movie.mp4'])
    })

    test('フォルダのエントリ自体はファイル一覧に出さない', () => {
        const dialog = mount_dialog(SAMPLE)
        dialog.navigate_to('doc')
        expect(dialog.current_files.value.map(file => file.path)).toEqual(['doc/note.md'])
    })

    test('潜ると、その階層の直下だけが見える', () => {
        const dialog = mount_dialog(SAMPLE)
        dialog.navigate_to('img')
        expect(dialog.current_subdirs.value.map(subdir => subdir.path)).toEqual(['img/sub'])
        expect(dialog.current_files.value.map(file => file.path)).toEqual(['img/a.png', 'img/b.png'])
    })

    test('パンくずは階層ごとの累積パスになる', () => {
        const dialog = mount_dialog(SAMPLE)
        expect(dialog.breadcrumbs.value).toEqual([]) // ルートでは出さない
        dialog.navigate_to('img/sub')
        expect(dialog.breadcrumbs.value).toEqual([
            { name: 'img', path: 'img' },
            { name: 'sub', path: 'img/sub' },
        ])
    })

    test('navigate_up は1階層ずつ戻り、ルートで止まる', () => {
        const dialog = mount_dialog(SAMPLE)
        dialog.navigate_to('img/sub')
        dialog.navigate_up()
        expect(dialog.current_dir.value).toBe('img')
        dialog.navigate_up()
        expect(dialog.current_dir.value).toBe('')
        dialog.navigate_up()
        expect(dialog.current_dir.value).toBe('')
    })

    // 圧縮ソフトによっては、フォルダのエントリが**末尾スラッシュ無し**で入る。
    // そのときは「配下のファイルから導出する」経路にも「自分自身のエントリ」にも
    // 引っかからないので、is_dir を見る枝が要る。
    // SAMPLE の 'doc/' は末尾スラッシュがあり最初の枝で拾えてしまうため、
    // この枝は別のフィクスチャでしか固定できない
    test('末尾スラッシュの無い空フォルダのエントリも一覧に出す', () => {
        const dialog = mount_dialog([
            entry('empty_dir', { is_dir: true }),
            entry('readme.txt', { is_text: true }),
        ])
        expect(dialog.current_subdirs.value.map(subdir => subdir.name)).toEqual(['empty_dir'])
        expect(dialog.current_files.value.map(file => file.path)).toEqual(['readme.txt'])
    })

    test('file_name はパスの末尾を返す', () => {
        const dialog = mount_dialog(SAMPLE)
        expect(dialog.file_name('img/sub/c.png')).toBe('c.png')
        expect(dialog.file_name('readme.txt')).toBe('readme.txt')
    })
})

describe('画像の拡大表示', () => {
    test('その階層の画像だけを並びとして扱う', () => {
        const dialog = mount_dialog(SAMPLE)
        dialog.navigate_to('img')
        expect(dialog.current_image_entries.value.map(image => image.path))
            .toEqual(['img/a.png', 'img/b.png'])
    })

    test('開いた画像の位置から前後へ動き、両端で止まる', () => {
        const dialog = mount_dialog(SAMPLE)
        dialog.navigate_to('img')
        dialog.open_enlarged_by_entry(entry('img/b.png', { is_image: true }))
        expect(dialog.enlarged_image_index.value).toBe(1)

        dialog.show_next_image() // 末尾なので動かない
        expect(dialog.enlarged_image_index.value).toBe(1)
        dialog.show_prev_image()
        expect(dialog.enlarged_image_index.value).toBe(0)
        dialog.show_prev_image() // 先頭なので動かない
        expect(dialog.enlarged_image_index.value).toBe(0)
    })

    test('その階層に無い画像では開かない', () => {
        const dialog = mount_dialog(SAMPLE)
        dialog.open_enlarged_by_entry(entry('img/a.png', { is_image: true })) // いまルートにいる
        expect(dialog.enlarged_image_index.value).toBe(-1)
    })

    test('フォルダを移ったら拡大表示を閉じる', async () => {
        const dialog = mount_dialog(SAMPLE)
        dialog.navigate_to('img')
        dialog.open_enlarged_by_entry(entry('img/a.png', { is_image: true }))
        expect(dialog.enlarged_image_index.value).toBe(0)

        dialog.navigate_to('doc')
        await nextTick()
        expect(dialog.enlarged_image_index.value).toBe(-1)
    })
})

describe('メディアビューワー', () => {
    // テンプレートの分岐順（image・text優先）と揃える必要がある。
    // 揃っていないと「一覧ではテキストとして開くのに、前後移動ではメディア扱い」になる
    test('画像・テキストにも該当するものはメディアの並びに入れない', () => {
        const dialog = mount_dialog([
            entry('a.mp4', { is_video: true }),
            entry('b.ts', { is_video: true, is_text: true }), // 拡張子が衝突する実例
            entry('c.mp3', { is_audio: true }),
        ])
        expect(dialog.current_media_entries.value.map(media => media.path)).toEqual(['a.mp4', 'c.mp3'])
    })

    test('前後へ動き、両端で止まる', () => {
        const media = [entry('a.mp4', { is_video: true }), entry('c.mp3', { is_audio: true })]
        const dialog = mount_dialog(media)
        dialog.open_media_viewer(media[0])
        expect(dialog.media_viewer_index.value).toBe(0)

        dialog.show_prev_media() // 先頭
        expect(dialog.media_viewer_index.value).toBe(0)
        dialog.show_next_media()
        expect(dialog.media_viewer_index.value).toBe(1)
        dialog.show_next_media() // 末尾
        expect(dialog.media_viewer_index.value).toBe(1)
    })

    test('隣へ移ると再生エラー表示を消す', () => {
        const media = [entry('a.mp4', { is_video: true }), entry('c.mp3', { is_audio: true })]
        const dialog = mount_dialog(media)
        dialog.open_media_viewer(media[0])
        dialog.onMediaError()
        expect(dialog.media_error.value).toBe(true)

        dialog.show_next_media()
        expect(dialog.media_error.value).toBe(false)
    })

    test('閉じると対象もエラー表示も落ちる', () => {
        const media = [entry('a.mp4', { is_video: true })]
        const dialog = mount_dialog(media)
        dialog.open_media_viewer(media[0])
        dialog.onMediaError()

        dialog.close_media_viewer()
        expect(dialog.media_viewer_entry.value).toBeNull()
        expect(dialog.media_error.value).toBe(false)
        expect(dialog.media_viewer_index.value).toBe(-1)
    })
})

describe('サイズ表示', () => {
    test.each([
        [0, '0 B'],
        [1023, '1023 B'],
        [1024, '1.0 KB'],
        [1024 * 1024 - 1, '1024.0 KB'],
        [1024 * 1024, '1.0 MB'],
        [1536 * 1024, '1.5 MB'],
    ])('%d バイト → %s', (bytes, want) => {
        const dialog = mount_dialog([])
        expect(dialog.format_size(bytes)).toBe(want)
    })
})

// テキストビューワーは 512KiB（use-browse-zip-contents-dialog.ts の TEXT_VIEWER_MAX_BYTES）で読むのを止める。
// 止めた位置は文字の境目とは限らないので、末尾に切れた多バイト文字が残る。
// これをそのまま文字コード判定に渡すと、UTF-8 の厳密デコードが末尾の切れ端で失敗して
// Shift_JIS / EUC-JP の試行へ落ち、全体が文字化けする（または置換文字だらけになる）。
// 応答に Content-Length があると読む前に「大きすぎる」で止めるので、ここを通るのは Content-Length が無いときだけ。
describe('テキストビューワーの切り詰め', () => {
    const TEXT_VIEWER_MAX_BYTES = 512 * 1024
    const TRUNCATED_SUFFIX = '\n\nZIP_TEXT_TRUNCATED_MESSAGE' // i18n はキーをそのまま返すようモックしてある

    /** Content-Length を返さず、本文をチャンクに分けて返す応答 */
    function stub_streaming_fetch(bytes: Uint8Array, chunk_size = 64 * 1024): void {
        vi.stubGlobal('fetch', vi.fn().mockImplementation(async () => {
            let offset = 0
            return {
                ok: true,
                status: 200,
                headers: { get: (): string | null => null },
                body: {
                    getReader: () => ({
                        read: async (): Promise<{ done: boolean, value?: Uint8Array }> => {
                            if (offset >= bytes.length) {
                                return { done: true }
                            }
                            const value = bytes.slice(offset, offset + chunk_size)
                            offset += chunk_size
                            return { done: false, value }
                        },
                        cancel: async (): Promise<void> => { },
                    }),
                },
            }
        }))
    }

    async function open_text(bytes: Uint8Array): Promise<string> {
        stub_streaming_fetch(bytes)
        const text_entry = entry('log.txt', { is_text: true })
        const dialog = mount_dialog([text_entry])
        await dialog.open_text_viewer(text_entry)
        return dialog.text_viewer_content.value
    }

    afterEach(() => {
        vi.unstubAllGlobals()
    })

    test('UTF-8 で切れ目が文字の途中でも、切れ端だけ捨てて UTF-8 として読む', async () => {
        // 「あ」は3バイト。524288 = 3 × 174762 + 2 なので、最後の1文字が2バイトで切れる
        const text = await open_text(new TextEncoder().encode('あ'.repeat(200000)))

        expect(text.endsWith(TRUNCATED_SUFFIX)).toBe(true)
        const body = text.slice(0, -TRUNCATED_SUFFIX.length)
        expect(body.includes('�'), '置換文字が混じった（UTF-8 として読めていない）').toBe(false)
        expect(/^あ+$/.test(body), '「あ」以外の文字が混じった（別の文字コードとして読まれた）').toBe(true)
        expect(body.length).toBe(Math.floor(TEXT_VIEWER_MAX_BYTES / 3))
    })

    test('Shift_JIS で切れ目が2バイト文字の先行バイトの直後でも、Shift_JIS として読む', async () => {
        // 「日」は Shift_JIS で 0x93 0xFA。先頭の ASCII 1バイトで1つずらし、
        // 512KiB 目が次の「日」の先行バイト 0x93 になるようにする
        const char_count = 300000
        const bytes = new Uint8Array(1 + char_count * 2)
        bytes[0] = 0x61 // 'a'
        for (let i = 0; i < char_count; i++) {
            bytes[1 + i * 2] = 0x93
            bytes[2 + i * 2] = 0xFA
        }
        const text = await open_text(bytes)

        expect(text.endsWith(TRUNCATED_SUFFIX)).toBe(true)
        const body = text.slice(0, -TRUNCATED_SUFFIX.length)
        expect(body.includes('�'), '置換文字が混じった（Shift_JIS として読めていない）').toBe(false)
        expect(/^a日+$/.test(body), '「日」以外の文字が混じった').toBe(true)
        expect(body.length).toBe(1 + (TEXT_VIEWER_MAX_BYTES - 1 - 1) / 2)
    })

    test('上限に届かないファイルは従来どおり全文をそのまま出す', async () => {
        const text = await open_text(new TextEncoder().encode('1行目\n2行目\nあいう'))
        expect(text).toBe('1行目\n2行目\nあいう')
    })
})
