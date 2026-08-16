/**
 * Kyouに紐づくタグの追加・論理削除の共有処理。
 *
 * この関数群は add-tag-view / KFTL / 12本のコンテキストメニュー /
 * Kyouの追加・編集18画面から使われるので、意味論をここで固定する。
 */
import { describe, test, expect, vi, beforeEach } from 'vitest'

vi.mock('@/i18n', () => ({
  default: { global: { t: (key: string) => key, locale: 'ja' } },
  i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

vi.mock('@/classes/api/gkill-api', () => ({
  GkillAPI: {
    get_instance: vi.fn(() => ({ get_session_id: vi.fn(() => 'mock-session') })),
    get_gkill_api: vi.fn(() => ({ get_session_id: vi.fn(() => 'mock-session') })),
  },
}))

vi.mock('@/classes/delete-gkill-cache', () => ({
  default: vi.fn().mockResolvedValue(undefined),
  delete_gkill_config_cache: vi.fn().mockResolvedValue(undefined),
}))

import { createMockGkillAPI } from '../../helpers/mock-api'
import { Tag } from '@/classes/datas/tag'
import {
  parse_tag_names,
  add_tags_to_target,
  remove_attached_tags,
  apply_kyou_tag_changes,
} from '@/classes/kyou-tags'

function create_application_config() {
  return { device: 'test-device', user_id: 'admin' } as never
}

function create_tag(id: string, tag_name: string): Tag {
  const tag = new Tag()
  tag.id = id
  tag.tag = tag_name
  tag.target_id = 'target-kyou-id'
  return tag
}

describe('parse_tag_names', () => {
  test('「、」で分割する', () => {
    expect(parse_tag_names('仕事、読み物')).toEqual(['仕事', '読み物'])
  })

  test('前後の空白を落とす', () => {
    expect(parse_tag_names(' 仕事 、  読み物 ')).toEqual(['仕事', '読み物'])
  })

  test('空の要素は落とす', () => {
    expect(parse_tag_names('仕事、、読み物、')).toEqual(['仕事', '読み物'])
    expect(parse_tag_names('')).toEqual([])
    expect(parse_tag_names('、、')).toEqual([])
  })

  // サーバの重複チェックはタグIDだけを見る(usecase/tag.go)ので、
  // 同じ名前を2回書けば2件できてしまう。落とすのはクライアントの責任
  test('重複は大小を無視して落とし、最初の表記を残す', () => {
    expect(parse_tag_names('仕事、仕事')).toEqual(['仕事'])
    expect(parse_tag_names('Work、work、WORK')).toEqual(['Work'])
  })
})

describe('add_tags_to_target', () => {
  let gkill_api: ReturnType<typeof createMockGkillAPI>

  beforeEach(() => {
    vi.clearAllMocks()
    gkill_api = createMockGkillAPI()
    let counter = 0
    gkill_api.generate_uuid.mockImplementation(() => `generated-uuid-${counter++}`)
  })

  test('タグ名の順にadd_tagを呼び、target_idを入れる', async () => {
    gkill_api.add_tag.mockImplementation((req: { tag: Tag }) => Promise.resolve({
      added_tag: req.tag, messages: [], errors: [],
    }))

    const result = await add_tags_to_target(gkill_api as never, create_application_config(), 'target-kyou-id', ['仕事', '読み物'])

    expect(gkill_api.add_tag).toHaveBeenCalledTimes(2)
    expect(result.added_tags.map(tag => tag.tag)).toEqual(['仕事', '読み物'])
    expect(result.added_tags.every(tag => tag.target_id === 'target-kyou-id')).toBe(true)
    expect(result.errors).toHaveLength(0)
  })

  test('タグごとに別のidを振る', async () => {
    gkill_api.add_tag.mockImplementation((req: { tag: Tag }) => Promise.resolve({
      added_tag: req.tag, messages: [], errors: [],
    }))

    const result = await add_tags_to_target(gkill_api as never, create_application_config(), 'target-kyou-id', ['仕事', '読み物'])

    expect(new Set(result.added_tags.map(tag => tag.id)).size).toBe(2)
  })

  test('タグ名が空配列ならAPIを呼ばない', async () => {
    const result = await add_tags_to_target(gkill_api as never, create_application_config(), 'target-kyou-id', [])

    expect(gkill_api.add_tag).not.toHaveBeenCalled()
    expect(gkill_api.push_tag_to_history).not.toHaveBeenCalled()
    expect(result.added_tags).toHaveLength(0)
  })

  // 途中で打ち切ると「3個書いたのに1個だけ付いた」理由が利用者に見えない
  test('1件失敗しても残りは投げ、エラーは集約して返す', async () => {
    gkill_api.add_tag
      .mockResolvedValueOnce({ added_tag: null, messages: [], errors: [{ error_code: 'ERR000056', error_message: 'exist' }] })
      .mockImplementationOnce((req: { tag: Tag }) => Promise.resolve({ added_tag: req.tag, messages: [], errors: [] }))

    const result = await add_tags_to_target(gkill_api as never, create_application_config(), 'target-kyou-id', ['仕事', '読み物'])

    expect(gkill_api.add_tag).toHaveBeenCalledTimes(2)
    expect(result.errors).toHaveLength(1)
    expect(result.added_tags.map(tag => tag.tag)).toEqual(['読み物'])
  })

  // 1つも付かなかったのに履歴が動くと、次回の履歴チップが
  // 「付けられなかったタグ」で埋まる
  test('1件も付かなかったら履歴を更新しない', async () => {
    gkill_api.add_tag.mockResolvedValue({ added_tag: null, messages: [], errors: [{ error_code: 'ERR000056', error_message: 'exist' }] })

    await add_tags_to_target(gkill_api as never, create_application_config(), 'target-kyou-id', ['仕事'])

    expect(gkill_api.push_tag_to_history).not.toHaveBeenCalled()
    expect(gkill_api.set_saved_last_added_tag).not.toHaveBeenCalled()
  })

  test('1件でも付いたら履歴へ積む', async () => {
    gkill_api.add_tag.mockImplementation((req: { tag: Tag }) => Promise.resolve({
      added_tag: req.tag, messages: [], errors: [],
    }))

    await add_tags_to_target(gkill_api as never, create_application_config(), 'target-kyou-id', ['仕事', '読み物'])

    expect(gkill_api.push_tag_to_history).toHaveBeenCalledWith('仕事、読み物')
    expect(gkill_api.set_saved_last_added_tag).toHaveBeenCalledWith('仕事、読み物')
  })
})

describe('remove_attached_tags', () => {
  let gkill_api: ReturnType<typeof createMockGkillAPI>

  beforeEach(() => {
    vi.clearAllMocks()
    gkill_api = createMockGkillAPI()
  })

  // gkillのリポジトリは追記のみなので、削除は is_deleted=true の版を足す
  test('is_deletedを立てた版をupdate_tagで送る', async () => {
    gkill_api.update_tag.mockImplementation((req: { tag: Tag }) => Promise.resolve({
      updated_tag: req.tag, messages: [], errors: [],
    }))

    const result = await remove_attached_tags(gkill_api as never, create_application_config(), [create_tag('tag-1', '買い物')])

    expect(gkill_api.update_tag).toHaveBeenCalledTimes(1)
    expect(gkill_api.update_tag.mock.calls[0][0].tag.is_deleted).toBe(true)
    expect(gkill_api.update_tag.mock.calls[0][0].tag.id).toBe('tag-1')
    expect(result.removed_tags).toHaveLength(1)
  })

  test('元のTagは書き換えない', async () => {
    gkill_api.update_tag.mockImplementation((req: { tag: Tag }) => Promise.resolve({
      updated_tag: req.tag, messages: [], errors: [],
    }))
    const original = create_tag('tag-1', '買い物')

    await remove_attached_tags(gkill_api as never, create_application_config(), [original])

    expect(original.is_deleted).toBe(false)
  })

  test('空配列ならAPIを呼ばない', async () => {
    const result = await remove_attached_tags(gkill_api as never, create_application_config(), [])

    expect(gkill_api.update_tag).not.toHaveBeenCalled()
    expect(result.removed_tags).toHaveLength(0)
  })
})

describe('apply_kyou_tag_changes', () => {
  let gkill_api: ReturnType<typeof createMockGkillAPI>

  beforeEach(() => {
    vi.clearAllMocks()
    gkill_api = createMockGkillAPI()
    gkill_api.add_tag.mockImplementation((req: { tag: Tag }) => Promise.resolve({
      added_tag: req.tag, messages: [], errors: [],
    }))
    gkill_api.update_tag.mockImplementation((req: { tag: Tag }) => Promise.resolve({
      updated_tag: req.tag, messages: [], errors: [],
    }))
  })

  test('追加と削除の両方を反映する', async () => {
    const result = await apply_kyou_tag_changes(gkill_api as never, create_application_config(), 'target-kyou-id',
      ['急ぎ'], [create_tag('tag-1', '買い物')])

    expect(result.added_tags.map(tag => tag.tag)).toEqual(['急ぎ'])
    expect(result.removed_tags.map(tag => tag.tag)).toEqual(['買い物'])
    expect(result.errors).toHaveLength(0)
  })

  // 追加が失敗したときに削除まで進めない。元の状態が残るほうが安全
  test('追加が失敗したら削除は行わない', async () => {
    gkill_api.add_tag.mockResolvedValue({ added_tag: null, messages: [], errors: [{ error_code: 'ERR000001', error_message: 'failed' }] })

    const result = await apply_kyou_tag_changes(gkill_api as never, create_application_config(), 'target-kyou-id',
      ['急ぎ'], [create_tag('tag-1', '買い物')])

    expect(gkill_api.update_tag).not.toHaveBeenCalled()
    expect(result.removed_tags).toHaveLength(0)
    expect(result.errors).toHaveLength(1)
  })

  test('どちらも空ならAPIを呼ばない', async () => {
    const result = await apply_kyou_tag_changes(gkill_api as never, create_application_config(), 'target-kyou-id', [], [])

    expect(gkill_api.add_tag).not.toHaveBeenCalled()
    expect(gkill_api.update_tag).not.toHaveBeenCalled()
    expect(result.added_tags).toHaveLength(0)
    expect(result.removed_tags).toHaveLength(0)
  })
})
