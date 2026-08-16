/**
 * Kyouの追加/編集画面に埋め込むタグ欄。
 *
 * このコンポーザブルは値を集めるだけで、実際の登録は親の save() が行う。
 * 親が読む get_tag_names / get_removed_tags / has_pending_changes の意味論を固定する。
 */
import { describe, test, expect, vi, beforeEach } from 'vitest'
import { nextTick } from 'vue'

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
import { useEditKyouTagsView } from '@/classes/use-edit-kyou-tags-view'

function create_tag(id: string, tag_name: string): Tag {
  const tag = new Tag()
  tag.id = id
  tag.tag = tag_name
  tag.target_id = 'target-kyou-id'
  return tag
}

function create_props(overrides: Record<string, unknown> = {}) {
  return {
    gkill_api: createMockGkillAPI() as never,
    application_config: { device: 'test-device', user_id: 'admin', tag_struct: { children: [] } } as never,
    kyou: null,
    is_readonly: false,
    ...overrides,
  }
}

describe('useEditKyouTagsView 追加画面(kyou=null)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  test('既存タグを読みに行かない', () => {
    const props = create_props()
    useEditKyouTagsView({ props: props as never, emits: vi.fn() as never })

    expect(props.gkill_api.get_tags_by_target_id).not.toHaveBeenCalled()
  })

  test('入力欄を「、」で割って返す', () => {
    const view = useEditKyouTagsView({ props: create_props() as never, emits: vi.fn() as never })
    view.tag_names_text.value = '仕事、読み物'

    expect(view.get_tag_names()).toEqual(['仕事', '読み物'])
  })

  test('外すタグは常に空', () => {
    const view = useEditKyouTagsView({ props: create_props() as never, emits: vi.fn() as never })
    view.tag_names_text.value = '仕事'

    expect(view.get_removed_tags()).toEqual([])
  })

  test('has_pending_changes は入力があるときだけ真', () => {
    const view = useEditKyouTagsView({ props: create_props() as never, emits: vi.fn() as never })
    expect(view.has_pending_changes()).toBe(false)

    view.tag_names_text.value = '仕事'
    expect(view.has_pending_changes()).toBe(true)

    // 空白だけの入力は「タグを書いた」ことにならない。
    // JSの trim() は全角スペース(U+3000)も落とすので、全角だけでも空になる
    view.tag_names_text.value = '　'
    expect(view.get_tag_names()).toEqual([])
    expect(view.has_pending_changes()).toBe(false)
  })

  test('reset() で入力欄が空になる', () => {
    const view = useEditKyouTagsView({ props: create_props() as never, emits: vi.fn() as never })
    view.tag_names_text.value = '仕事'

    view.reset()

    expect(view.tag_names_text.value).toBe('')
    expect(view.has_pending_changes()).toBe(false)
  })
})

describe('useEditKyouTagsView 履歴チップ', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  test('押すと入力欄の末尾へ「、」付きで足す', () => {
    const props = create_props()
    props.gkill_api.get_saved_tag_history.mockReturnValue(['仕事', '買い物'])
    const view = useEditKyouTagsView({ props: props as never, emits: vi.fn() as never })

    view.append_history_tag('仕事')
    expect(view.tag_names_text.value).toBe('仕事')

    view.append_history_tag('買い物')
    expect(view.tag_names_text.value).toBe('仕事、買い物')
  })

  test('既に書かれているタグは足さない', () => {
    const props = create_props()
    const view = useEditKyouTagsView({ props: props as never, emits: vi.fn() as never })
    view.tag_names_text.value = '仕事'

    view.append_history_tag('仕事')

    expect(view.tag_names_text.value).toBe('仕事')
  })

  test('読み取り専用なら足さない', () => {
    const props = create_props({ is_readonly: true })
    const view = useEditKyouTagsView({ props: props as never, emits: vi.fn() as never })

    view.append_history_tag('仕事')

    expect(view.tag_names_text.value).toBe('')
  })

  // 「タグ無し」は絞り込み用の番兵で実在のタグではない
  test('「no tags」番兵は履歴チップに出さない', () => {
    const props = create_props()
    props.gkill_api.get_saved_tag_history.mockReturnValue(['no tags', '仕事'])
    const view = useEditKyouTagsView({ props: props as never, emits: vi.fn() as never })

    expect(view.tag_history.value).toEqual(['仕事'])
  })
})

describe('useEditKyouTagsView 編集画面(kyouあり)', () => {
  let props: ReturnType<typeof create_props>

  beforeEach(async () => {
    vi.clearAllMocks()
    props = create_props({ kyou: { id: 'target-kyou-id' } })
    props.gkill_api.get_tags_by_target_id.mockResolvedValue({
      tags: [create_tag('tag-1', '買い物'), create_tag('tag-2', '日用品')],
      messages: [], errors: [],
    })
  })

  // props.kyou.attached_tags は当てにできない。編集ビューの load() が呼ぶのは
  // load_typed_datas() だけで、添付タグは読まれていない
  test('既存タグは get_tags_by_target_id で自分で引く', async () => {
    const view = useEditKyouTagsView({ props: props as never, emits: vi.fn() as never })
    await nextTick()
    await nextTick()

    expect(props.gkill_api.get_tags_by_target_id).toHaveBeenCalledTimes(1)
    expect(props.gkill_api.get_tags_by_target_id.mock.calls[0][0].target_id).toBe('target-kyou-id')
    expect(view.existing_tags.value.map(tag => tag.tag)).toEqual(['買い物', '日用品'])
  })

  test('⊗ を押しても保存するまでサーバへ行かない', async () => {
    const view = useEditKyouTagsView({ props: props as never, emits: vi.fn() as never })
    await nextTick()
    await nextTick()

    view.toggle_remove(view.existing_tags.value[0])

    expect(props.gkill_api.update_tag).not.toHaveBeenCalled()
    expect(view.is_removed(view.existing_tags.value[0])).toBe(true)
    expect(view.get_removed_tags().map(tag => tag.tag)).toEqual(['買い物'])
  })

  test('もう一度押すと削除マークが外れる', async () => {
    const view = useEditKyouTagsView({ props: props as never, emits: vi.fn() as never })
    await nextTick()
    await nextTick()

    view.toggle_remove(view.existing_tags.value[0])
    view.toggle_remove(view.existing_tags.value[0])

    expect(view.is_removed(view.existing_tags.value[0])).toBe(false)
    expect(view.get_removed_tags()).toEqual([])
  })

  test('読み取り専用なら削除マークを付けられない', async () => {
    props.is_readonly = true
    const view = useEditKyouTagsView({ props: props as never, emits: vi.fn() as never })
    await nextTick()
    await nextTick()

    view.toggle_remove(view.existing_tags.value[0])

    expect(view.get_removed_tags()).toEqual([])
  })

  // サーバの重複チェックはタグIDだけを見るので、落とさないと同じ名前が2件付く
  test('既存タグと同名の入力は落とす（大小無視）', async () => {
    const view = useEditKyouTagsView({ props: props as never, emits: vi.fn() as never })
    await nextTick()
    await nextTick()

    view.tag_names_text.value = '買い物、急ぎ、日用品'

    expect(view.get_tag_names()).toEqual(['急ぎ'])
  })

  test('削除マークを付けた既存タグと同名なら入力を残す', async () => {
    const view = useEditKyouTagsView({ props: props as never, emits: vi.fn() as never })
    await nextTick()
    await nextTick()

    view.toggle_remove(view.existing_tags.value[0])
    view.tag_names_text.value = '買い物'

    expect(view.get_tag_names()).toEqual(['買い物'])
  })

  test('has_pending_changes は削除マークだけでも真', async () => {
    const view = useEditKyouTagsView({ props: props as never, emits: vi.fn() as never })
    await nextTick()
    await nextTick()

    expect(view.has_pending_changes()).toBe(false)

    view.toggle_remove(view.existing_tags.value[0])

    expect(view.has_pending_changes()).toBe(true)
  })

  test('reset() で削除マークも入力欄も戻る', async () => {
    const view = useEditKyouTagsView({ props: props as never, emits: vi.fn() as never })
    await nextTick()
    await nextTick()

    view.toggle_remove(view.existing_tags.value[0])
    view.tag_names_text.value = '急ぎ'

    view.reset()

    expect(view.tag_names_text.value).toBe('')
    expect(view.get_removed_tags()).toEqual([])
    expect(view.has_pending_changes()).toBe(false)
  })

  test('読み込みに失敗したら received_errors を上げる', async () => {
    const emits = vi.fn()
    props.gkill_api.get_tags_by_target_id.mockResolvedValue({
      tags: null, messages: [], errors: [{ error_code: 'ERR000001', error_message: 'failed' }],
    })
    useEditKyouTagsView({ props: props as never, emits: emits as never })
    await nextTick()
    await nextTick()

    expect(emits.mock.calls.filter(call => call[0] === 'received_errors')).toHaveLength(1)
  })
})
