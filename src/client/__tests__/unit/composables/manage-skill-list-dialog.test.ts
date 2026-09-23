/**
 * スキル管理ダイアログ（use-manage-skill-list-dialog / use-browse-skill-files-dialog）の検証。ADR-0634。
 *
 * アップロードは2段階: 1段目は dry_run で計画（追加・削除・変更）を確認ダイアログへ出すだけで、
 * 2段目の「適用」で**同じ zip** を dry_run=false で送る。1段目で書き込んでしまう・2段目で別の中身を
 * 送る・確認前に置き換わる、のどれも利用者のスキルを黙って壊すので、送った要求の形まで固定する。
 * 画面の削除はスキル丸ごと（path を空）だけ。
 */
import { describe, expect, test, vi } from 'vitest'

vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))
vi.mock('@/classes/use-dialog-history-stack', () => ({
    useDialogHistoryStack: vi.fn(),
    close_dialog_via_history: vi.fn(),
}))
vi.mock('@/classes/use-floating-dialog', () => ({
    useFloatingDialog: vi.fn(() => ({})),
}))
const save_as_mock = vi.fn()
vi.mock('@/classes/save-as', () => ({
    save_as: (...args: Array<unknown>) => save_as_mock(...args),
}))
vi.mock('@/classes/file-base64', async (import_original) => ({
    ...(await import_original<typeof import('@/classes/file-base64')>()),
    read_file_as_data_url: vi.fn(async () => 'data:application/zip;base64,UEsDBA=='),
}))

// req_res は GkillAPIRequest を継承する。循環importがあるため gkill-api を先に評価させる
import '@/classes/api/gkill-api'

import { useManageSkillListDialog } from '@/classes/use-manage-skill-list-dialog'
import { useBrowseSkillFilesDialog } from '@/classes/use-browse-skill-files-dialog'
import { base64_to_blob } from '@/classes/file-base64'
import type { ManageSkillListDialogProps } from '@/pages/dialogs/manage-skill-list-dialog-props'
import type { ManageSkillListDialogEmits } from '@/pages/dialogs/manage-skill-list-dialog-emits'
import type { BrowseSkillFilesDialogProps } from '@/pages/dialogs/browse-skill-files-dialog-props'
import type { BrowseSkillFilesDialogEmits } from '@/pages/dialogs/browse-skill-files-dialog-emits'
import type { UploadSkillRequest } from '@/classes/api/req_res/upload-skill-request'
import type { DeleteSkillRequest } from '@/classes/api/req_res/delete-skill-request'
import type { GetSkillRequest } from '@/classes/api/req_res/get-skill-request'
import type { SkillInfo } from '@/classes/api/req_res/get-skill-list-response'

const plan = { name: 'weekly', is_new: false, added: ['a.md'], removed: ['old.md'], changed: ['SKILL.md'], ignored: [] }

function ok<T extends object>(extra: T) {
    return { errors: [], messages: [], ...extra }
}

function create_manage_dialog(api_overrides: Record<string, unknown> = {}) {
    const api = {
        get_skill_list: vi.fn(async () => ok({ skills: [{ name: 'weekly', description: 'd', updated_time: '', file_count: 1, invalid_reason: '' }] })),
        upload_skill: vi.fn(async (req: UploadSkillRequest) => ok({ plan: plan, applied: !req.dry_run, messages: req.dry_run ? [] : [{ message_code: 'MSG000091', message: 'uploaded' }] })),
        delete_skill: vi.fn(async () => ok({ messages: [{ message_code: 'MSG000093', message: 'deleted' }] })),
        download_skill: vi.fn(async () => ok({ file_name: 'weekly.zip', zip_base64: 'UEsDBA==' })),
        ...api_overrides,
    }
    const props = { application_config: {}, gkill_api: api } as unknown as ManageSkillListDialogProps
    const emitted: Array<{ event: string, payload: unknown }> = []
    const emits = ((event: string, payload: unknown) => { emitted.push({ event, payload }) }) as unknown as ManageSkillListDialogEmits
    const dialog = useManageSkillListDialog({ props, emits })
    const confirm_upload_show = vi.fn()
    dialog.confirm_upload_skill_dialog.value = { show: confirm_upload_show } as unknown as typeof dialog.confirm_upload_skill_dialog.value
    return { dialog, api, emitted, confirm_upload_show }
}

function file_change_event(): { event: Event, input: HTMLInputElement } {
    const input = document.createElement('input')
    input.type = 'file'
    const file = new File([new Uint8Array([0x50, 0x4b, 0x03, 0x04])], 'weekly.zip', { type: 'application/zip' })
    Object.defineProperty(input, 'files', { value: [file] })
    const event = new Event('change')
    Object.defineProperty(event, 'target', { value: input })
    return { event, input }
}

describe('useManageSkillListDialog', () => {
    test('開くと一覧を読み込む', async () => {
        const { dialog, api } = create_manage_dialog()
        await dialog.show()
        expect(api.get_skill_list).toHaveBeenCalledTimes(1)
        expect(dialog.skills.value.map(s => s.name)).toEqual(['weekly'])
        expect(dialog.is_loading.value).toBe(false)
    })

    test('アップロードの1段目は dry_run だけを送り、計画を確認ダイアログへ出す（書き込まない）', async () => {
        const { dialog, api, confirm_upload_show } = create_manage_dialog()
        const { event } = file_change_event()
        await dialog.onSelectedUploadFile(event)

        expect(api.upload_skill).toHaveBeenCalledTimes(1)
        const req = api.upload_skill.mock.calls[0][0] as UploadSkillRequest
        expect(req.dry_run).toBe(true)
        expect(req.zip_base64).toBe('data:application/zip;base64,UEsDBA==')
        expect(confirm_upload_show).toHaveBeenCalledWith(plan)
        // 同じファイルを選び直しても change が起きるよう、選択を空へ戻す
        expect((event.target as HTMLInputElement).value).toBe('')
        expect(dialog.is_uploading.value).toBe(false)
    })

    test('2段目の「適用」は1段目と同じ zip を dry_run=false で送り、一覧を読み直す。二度目は何もしない', async () => {
        const { dialog, api, emitted } = create_manage_dialog()
        await dialog.onSelectedUploadFile(file_change_event().event)
        await dialog.apply_upload_skill()

        expect(api.upload_skill).toHaveBeenCalledTimes(2)
        const applied = api.upload_skill.mock.calls[1][0] as UploadSkillRequest
        expect(applied.dry_run).toBe(false)
        expect(applied.zip_base64).toBe('data:application/zip;base64,UEsDBA==')
        expect(emitted.some(e => e.event === 'received_messages')).toBe(true)
        expect(api.get_skill_list).toHaveBeenCalledTimes(1)

        await dialog.apply_upload_skill()
        expect(api.upload_skill).toHaveBeenCalledTimes(2)
    })

    test('1段目でサーバが断った zip は確認に進まず、適用もできない', async () => {
        const { dialog, api, emitted, confirm_upload_show } = create_manage_dialog({
            upload_skill: vi.fn(async () => ({ errors: [{ error_code: 'ERR000435', error_message: 'bad zip' }], messages: [], plan: null, applied: false })),
        })
        await dialog.onSelectedUploadFile(file_change_event().event)
        expect(confirm_upload_show).not.toHaveBeenCalled()
        expect(emitted.filter(e => e.event === 'received_errors')).toHaveLength(1)

        await dialog.apply_upload_skill()
        expect(api.upload_skill).toHaveBeenCalledTimes(1)
    })

    test('画面の削除はスキル丸ごと（path を空）で、消したら一覧を読み直す', async () => {
        const { dialog, api } = create_manage_dialog()
        await dialog.delete_skill({ name: 'weekly' } as SkillInfo)
        const req = api.delete_skill.mock.calls[0][0] as DeleteSkillRequest
        expect(req.name).toBe('weekly')
        expect(req.path).toBe('')
        expect(api.get_skill_list).toHaveBeenCalledTimes(1)
    })

    test('ダウンロードは base64 の zip を Blob にして保存する', async () => {
        save_as_mock.mockClear()
        const { dialog } = create_manage_dialog()
        await dialog.download_skill({ name: 'weekly' } as SkillInfo)
        expect(save_as_mock).toHaveBeenCalledTimes(1)
        const [blob, file_name] = save_as_mock.mock.calls[0] as [Blob, string]
        expect(file_name).toBe('weekly.zip')
        expect(blob.type).toBe('application/zip')
        expect(new Uint8Array(await blob.arrayBuffer())).toEqual(new Uint8Array([0x50, 0x4b, 0x03, 0x04]))
    })
})

describe('base64_to_blob', () => {
    test('バイト列をそのまま戻す（0x80 以上も化けない）', async () => {
        const blob = base64_to_blob(btoa(String.fromCharCode(0x00, 0x7f, 0x80, 0xff)), 'application/octet-stream')
        expect(new Uint8Array(await blob.arrayBuffer())).toEqual(new Uint8Array([0x00, 0x7f, 0x80, 0xff]))
    })
})

describe('useBrowseSkillFilesDialog', () => {
    const skill_detail = {
        name: 'weekly', description: 'd', invalid_reason: '', updated_time: '', content: '---\nname: weekly\n---\n', revision: 'r0',
        files: [
            { path: 'SKILL.md', size: 10, is_text: true, revision: 'r0', updated_time: '' },
            { path: 'assets/logo.png', size: 8, is_text: false, revision: 'r1', updated_time: '' },
            { path: 'references/a.md', size: 3, is_text: true, revision: 'r2', updated_time: '' },
            { path: 'references/b.md', size: 3, is_text: true, revision: 'r3', updated_time: '' },
        ],
    }

    function create_browse_dialog(get_skill: (req: GetSkillRequest) => Promise<unknown>) {
        const api = { get_skill: vi.fn(get_skill) }
        const props = { application_config: {}, gkill_api: api } as unknown as BrowseSkillFilesDialogProps
        const emits = vi.fn() as unknown as BrowseSkillFilesDialogEmits
        return { dialog: useBrowseSkillFilesDialog({ props, emits }), api }
    }

    test('開くと SKILL.md を出す。バイナリは読みに行かずに「表示できません」', async () => {
        const { dialog, api } = create_browse_dialog(async () => ok({ skill: skill_detail, file: null }))
        await dialog.show('weekly')
        expect(dialog.text.value).toBe(skill_detail.content)
        expect(dialog.selected_path.value).toBe('SKILL.md')

        await dialog.select_file(skill_detail.files[1])
        expect(dialog.is_binary.value).toBe(true)
        expect(api.get_skill).toHaveBeenCalledTimes(1)
    })

    test('テキストは path 付きで読み、応答が追い越しても最後に選んだファイルを出す', async () => {
        let release_a: () => void = () => { }
        const { dialog } = create_browse_dialog(async (req: GetSkillRequest) => {
            if (req.path === '') {
                return ok({ skill: skill_detail, file: null })
            }
            if (req.path === 'references/a.md') {
                await new Promise<void>(resolve => { release_a = resolve })
            }
            return ok({ skill: null, file: { path: req.path, size: 3, is_text: true, revision: 'r', updated_time: '', content: 'content of ' + req.path, content_base64: '', content_omitted: false } })
        })
        await dialog.show('weekly')
        const slow = dialog.select_file(skill_detail.files[2])
        await dialog.select_file(skill_detail.files[3])
        release_a()
        await slow
        expect(dialog.selected_path.value).toBe('references/b.md')
        expect(dialog.text.value).toBe('content of references/b.md')
        expect(dialog.is_loading.value).toBe(false)
    })
})
