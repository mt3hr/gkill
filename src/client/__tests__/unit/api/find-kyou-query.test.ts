import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { MiCheckState } from '@/classes/api/find_query/mi-check-state'
import { MiSortType } from '@/classes/api/find_query/mi-sort-type'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import {
  makeApplicationConfig,
  makeDeviceStructElement,
  makeRepStructElement,
  makeRepTypeStructElement,
  makeTagStructElement,
} from '../../helpers/factory'

/**
 * FindKyouQuery は rykv / mi の検索条件そのもの。
 * ここが壊れると「検索条件が黙って落ちる」形の不具合になり、
 * 画面上は正常に見えたまま結果だけが変わるため気付きにくい。
 */

function asConfig(overrides: Record<string, unknown> = {}): ApplicationConfig {
  return makeApplicationConfig(overrides) as unknown as ApplicationConfig
}

describe('FindKyouQuery', () => {
  describe('parse_words_and_not_words', () => {
    test('スペース区切りの語が words に入る', () => {
      const query = new FindKyouQuery()
      query.keywords = 'alpha beta'
      query.parse_words_and_not_words()

      expect(query.words).toEqual(['alpha', 'beta'])
      expect(query.not_words).toEqual([])
    })

    test('- 前置の語は not_words に入る', () => {
      const query = new FindKyouQuery()
      query.keywords = 'alpha -beta'
      query.parse_words_and_not_words()

      expect(query.words).toEqual(['alpha'])
      expect(query.not_words).toEqual(['beta'])
    })

    test('全角スペースでも分割される', () => {
      const query = new FindKyouQuery()
      query.keywords = 'alpha　beta'
      query.parse_words_and_not_words()

      expect(query.words).toEqual(['alpha', 'beta'])
    })

    test('半角と全角スペースが混ざっても分割される', () => {
      const query = new FindKyouQuery()
      query.keywords = 'alpha　beta gamma　-delta'
      query.parse_words_and_not_words()

      expect(query.words).toEqual(['alpha', 'beta', 'gamma'])
      expect(query.not_words).toEqual(['delta'])
    })

    test('連続スペースは空語を生まない', () => {
      const query = new FindKyouQuery()
      query.keywords = '  alpha   beta  '
      query.parse_words_and_not_words()

      expect(query.words).toEqual(['alpha', 'beta'])
    })

    test('単独の - は次の語を除外語にする', () => {
      const query = new FindKyouQuery()
      query.keywords = 'alpha - beta'
      query.parse_words_and_not_words()

      expect(query.words).toEqual(['alpha'])
      expect(query.not_words).toEqual(['beta'])
    })

    test('除外指定は1語だけに効き、その次の語には持ち越さない', () => {
      const query = new FindKyouQuery()
      query.keywords = '-alpha beta'
      query.parse_words_and_not_words()

      expect(query.not_words).toEqual(['alpha'])
      expect(query.words).toEqual(['beta'])
    })

    test('語中のハイフンは除外扱いにならない', () => {
      const query = new FindKyouQuery()
      query.keywords = 'alpha-beta'
      query.parse_words_and_not_words()

      expect(query.words).toEqual(['alpha-beta'])
      expect(query.not_words).toEqual([])
    })

    // 実装は word.replace("-", "") で先頭1個しか消さないため、
    // "--foo" は "-foo" として除外語に入る。意図的かは不明だが現状の挙動を固定する。
    test('-- 前置は先頭1個だけ剥がされる', () => {
      const query = new FindKyouQuery()
      query.keywords = '--alpha'
      query.parse_words_and_not_words()

      expect(query.not_words).toEqual(['-alpha'])
      expect(query.words).toEqual([])
    })

    test('空文字なら words も not_words も空になる', () => {
      const query = new FindKyouQuery()
      query.keywords = ''
      query.parse_words_and_not_words()

      expect(query.words).toEqual([])
      expect(query.not_words).toEqual([])
    })

    test('timeis_keywords も同じルールで分解される', () => {
      const query = new FindKyouQuery()
      query.timeis_keywords = '作業　-休憩'
      query.parse_words_and_not_words()

      expect(query.timeis_words).toEqual(['作業'])
      expect(query.timeis_not_words).toEqual(['休憩'])
    })

    test('呼び直すと前回の結果が残らない', () => {
      const query = new FindKyouQuery()
      query.keywords = 'alpha'
      query.parse_words_and_not_words()
      query.keywords = 'beta'
      query.parse_words_and_not_words()

      expect(query.words).toEqual(['beta'])
    })
  })

  describe('rep_to_struct', () => {
    test('dvnf形式の rep_name を type/device/time に分解する', () => {
      const query = new FindKyouQuery()
      const rep = makeRepStructElement({ rep_name: 'kmemo_laptop_20240101' })

      const got = query.rep_to_struct(rep as never)

      expect(got).toEqual({ type: 'kmemo', device: 'laptop', time: '20240101' })
    })

    test('dvnf形式でない rep_name は type にそのまま入り device は「なし」になる', () => {
      const query = new FindKyouQuery()
      const rep = makeRepStructElement({ rep_name: 'plugin_rep' })

      const got = query.rep_to_struct(rep as never)

      expect(got).toEqual({ type: 'plugin_rep', device: 'なし', time: '' })
    })
  })

  describe('apply_rep_summary_to_detaul', () => {
    // type と device の両方がチェックされている Rep だけが reps に入る
    function configWithReps(reps: Array<Record<string, unknown>>, checkedType: string, checkedDevice: string) {
      return asConfig({
        rep_struct: makeRepStructElement({
          name: 'root',
          children: reps.map((r) => makeRepStructElement(r)),
        }),
        rep_type_struct: makeRepTypeStructElement({
          name: 'root',
          children: [makeRepTypeStructElement({ key: checkedType, rep_type_name: checkedType, is_checked: true })],
        }),
        device_struct: makeDeviceStructElement({
          name: 'root',
          children: [makeDeviceStructElement({ key: checkedDevice, device_name: checkedDevice, is_checked: true })],
        }),
      })
    }

    test('type と device が両方チェックされている Rep だけが選ばれる', () => {
      const query = new FindKyouQuery()
      const config = configWithReps(
        [
          { rep_name: 'kmemo_laptop_2024' },
          { rep_name: 'kmemo_phone_2024' },
          { rep_name: 'urlog_laptop_2024' },
        ],
        'kmemo',
        'laptop',
      )

      query.apply_rep_summary_to_detaul(config)

      expect(query.reps).toEqual(['kmemo_laptop_2024'])
    })

    test('ignore_check_rep_rykv の Rep は条件に合っても除外される', () => {
      const query = new FindKyouQuery()
      const config = configWithReps(
        [
          { rep_name: 'kmemo_laptop_2024' },
          { rep_name: 'kmemo_laptop_2023', ignore_check_rep_rykv: true },
        ],
        'kmemo',
        'laptop',
      )

      query.apply_rep_summary_to_detaul(config)

      expect(query.reps).toEqual(['kmemo_laptop_2024'])
    })

    test('入れ子の Rep も辿られる', () => {
      const query = new FindKyouQuery()
      const config = asConfig({
        rep_struct: makeRepStructElement({
          name: 'root',
          children: [
            makeRepStructElement({
              rep_name: 'folder',
              is_dir: true,
              children: [makeRepStructElement({ rep_name: 'kmemo_laptop_2024' })],
            }),
          ],
        }),
        rep_type_struct: makeRepTypeStructElement({
          name: 'root',
          children: [makeRepTypeStructElement({ key: 'kmemo', rep_type_name: 'kmemo', is_checked: true })],
        }),
        device_struct: makeDeviceStructElement({
          name: 'root',
          children: [makeDeviceStructElement({ key: 'laptop', device_name: 'laptop', is_checked: true })],
        }),
      })

      query.apply_rep_summary_to_detaul(config)

      expect(query.reps).toContain('kmemo_laptop_2024')
    })

    test('device が未チェックなら何も選ばれない', () => {
      const query = new FindKyouQuery()
      const config = configWithReps([{ rep_name: 'kmemo_laptop_2024' }], 'kmemo', 'phone')

      query.apply_rep_summary_to_detaul(config)

      expect(query.reps).toEqual([])
    })

    test('struct が空なら reps を書き換えない', () => {
      const query = new FindKyouQuery()
      query.reps = ['keep_me']

      query.apply_rep_summary_to_detaul(asConfig())

      expect(query.reps).toEqual(['keep_me'])
    })
  })

  describe('apply_hide_tags', () => {
    test('is_force_hide のタグを木構造から再帰的に集める', () => {
      const query = new FindKyouQuery()
      const config = asConfig({
        tag_struct: makeTagStructElement({
          name: 'root',
          children: [
            makeTagStructElement({ tag_name: 'visible' }),
            makeTagStructElement({ tag_name: 'hidden1', is_force_hide: true }),
            makeTagStructElement({
              tag_name: 'folder',
              children: [makeTagStructElement({ tag_name: 'hidden2', is_force_hide: true })],
            }),
          ],
        }),
      })

      query.apply_hide_tags(config)

      expect(query.hide_tags).toEqual(['hidden1', 'hidden2'])
    })

    test('呼び直しても前回の結果が残らない', () => {
      const query = new FindKyouQuery()
      query.hide_tags = ['stale']
      const config = asConfig({
        tag_struct: makeTagStructElement({
          name: 'root',
          children: [makeTagStructElement({ tag_name: 'hidden', is_force_hide: true })],
        }),
      })

      query.apply_hide_tags(config)

      expect(query.hide_tags).toEqual(['hidden'])
    })

    test('is_force_hide が無ければ空になる', () => {
      const query = new FindKyouQuery()
      query.apply_hide_tags(asConfig())

      expect(query.hide_tags).toEqual([])
    })
  })

  describe('generate_default_query_for_rykv', () => {
    test('check_when_inited のタグ・デバイス・RepTypeが初期条件に入る', () => {
      const config = asConfig({
        tag_struct: makeTagStructElement({
          name: 'root',
          children: [
            makeTagStructElement({ tag_name: 'daily', check_when_inited: true }),
            makeTagStructElement({ tag_name: 'archive' }),
          ],
        }),
        device_struct: makeDeviceStructElement({
          name: 'root',
          children: [
            makeDeviceStructElement({ device_name: 'laptop', check_when_inited: true }),
            makeDeviceStructElement({ device_name: 'phone' }),
          ],
        }),
        rep_type_struct: makeRepTypeStructElement({
          name: 'root',
          children: [
            makeRepTypeStructElement({ rep_type_name: 'kmemo', check_when_inited: true }),
            makeRepTypeStructElement({ rep_type_name: 'urlog' }),
          ],
        }),
      })

      const query = FindKyouQuery.generate_default_query_for_rykv(config)

      expect(query.tags).toEqual(['daily'])
      expect(query.timeis_tags).toEqual(['daily'])
      expect(query.devices_in_sidebar).toEqual(['laptop'])
      expect(query.rep_types_in_sidebar).toEqual(['kmemo'])
    })

    test('rykv_default_period が -1 ならカレンダー条件を付けない', () => {
      const query = FindKyouQuery.generate_default_query_for_rykv(asConfig({ rykv_default_period: -1 }))

      expect(query.use_calendar).toBe(false)
      expect(query.calendar_start_date).toBeNull()
      expect(query.calendar_end_date).toBeNull()
    })

    test('rykv_default_period が指定されていればその日数分のカレンダー条件を付ける', () => {
      const query = FindKyouQuery.generate_default_query_for_rykv(asConfig({ rykv_default_period: 7 }))

      expect(query.use_calendar).toBe(true)
      expect(query.calendar_start_date).toBeInstanceOf(Date)
      expect(query.calendar_end_date).toBeInstanceOf(Date)

      const start = query.calendar_start_date as Date
      const end = query.calendar_end_date as Date
      expect(end.getTime()).toBeGreaterThan(start.getTime())
      // 開始は7日前の0時、終了は今日の終わりなので、差はおよそ8日
      const days = (end.getTime() - start.getTime()) / (24 * 60 * 60 * 1000)
      expect(days).toBeGreaterThan(7.9)
      expect(days).toBeLessThan(8.1)
    })

    test('is_force_hide のタグが hide_tags に反映される', () => {
      const config = asConfig({
        tag_struct: makeTagStructElement({
          name: 'root',
          children: [makeTagStructElement({ tag_name: 'secret', is_force_hide: true })],
        }),
      })

      const query = FindKyouQuery.generate_default_query_for_rykv(config)

      expect(query.hide_tags).toEqual(['secret'])
    })

    test('for_mi は立たない', () => {
      expect(FindKyouQuery.generate_default_query_for_rykv(asConfig()).for_mi).toBe(false)
    })
  })

  describe('generate_default_query_for_mi', () => {
    test('for_mi が立つ', () => {
      expect(FindKyouQuery.generate_default_query_for_mi(asConfig()).for_mi).toBe(true)
    })

    test('Rep は check_when_inited に関係なく全部入る（サーバ側でMiのRepに絞られるため）', () => {
      const config = asConfig({
        rep_struct: makeRepStructElement({
          name: 'root',
          children: [
            makeRepStructElement({ rep_name: 'mi_laptop_2024', check_when_inited: true }),
            makeRepStructElement({ rep_name: 'kmemo_laptop_2024' }),
          ],
        }),
      })

      const query = FindKyouQuery.generate_default_query_for_mi(config)

      expect(query.reps).toEqual(['mi_laptop_2024', 'kmemo_laptop_2024'])
    })

    test('カレンダー条件は付かない（rykv との差分）', () => {
      const query = FindKyouQuery.generate_default_query_for_mi(asConfig({ rykv_default_period: 7 }))

      expect(query.use_calendar).toBe(false)
    })

    test('check_when_inited のタグだけが入る', () => {
      const config = asConfig({
        tag_struct: makeTagStructElement({
          name: 'root',
          children: [
            makeTagStructElement({ tag_name: 'todo', check_when_inited: true }),
            makeTagStructElement({ tag_name: 'done' }),
          ],
        }),
      })

      const query = FindKyouQuery.generate_default_query_for_mi(config)

      expect(query.tags).toEqual(['todo'])
    })
  })

  describe('clone / parse_find_kyou_query のフィールド網羅', () => {
    // 全フィールドに既定値と違う値を入れた個体を作る。
    // 新しいフィールドを FindKyouQuery に足したらここにも足すこと
    // （足し忘れは下の「未設定フィールドが無いこと」テストで落ちる）。
    function makeFullyPopulatedQuery(): FindKyouQuery {
      const query = new FindKyouQuery()
      query.query_id = 'query-1'
      query.use_tags = false
      query.use_reps = false
      query.update_cache = true
      query.use_words = true
      query.keywords = 'alpha -beta'
      query.words_and = true
      query.words = ['alpha']
      query.not_words = ['beta']
      query.use_timeis = true
      query.timeis_words_and = true
      query.timeis_keywords = '作業'
      query.timeis_words = ['作業']
      query.timeis_not_words = ['休憩']
      query.use_timeis_tags = false
      query.timeis_tags = ['tag-timeis']
      query.timeis_tags_and = true
      query.tags = ['tag-1']
      query.hide_tags = ['tag-hidden']
      query.tags_and = true
      query.use_map = true
      query.map_latitude = 35.1
      query.map_longitude = 139.2
      query.map_radius = 300
      query.use_calendar = true
      query.calendar_start_date = new Date('2026-01-01T00:00:00Z')
      query.calendar_end_date = new Date('2026-01-31T23:59:59Z')
      query.use_plaing = true
      query.plaing_time = new Date('2026-01-15T12:00:00Z')
      query.use_update_time = true
      query.update_time = new Date('2026-01-20T09:00:00Z')
      query.use_period_of_time = true
      query.period_of_time_start_time_second = 3600
      query.period_of_time_end_time_second = 7200
      query.period_of_time_week_of_days = [1, 2, 3]
      query.use_rep_types = true
      query.rep_types = ['kmemo']
      query.reps = ['kmemo_laptop_2024']
      query.devices_in_sidebar = ['laptop']
      query.rep_types_in_sidebar = ['kmemo']
      query.is_enable_map_circle_in_sidebar = true
      query.is_image_only = true
      query.is_focus_kyou_in_list_view = true
      query.use_mi_board_name = true
      query.mi_board_name = 'inbox'
      query.use_mi_sort_type = true
      query.mi_sort_type = MiSortType.limit_time
      query.for_mi = true
      query.use_mi_check_state = true
      query.mi_check_state = MiCheckState.checked
      query.include_create_mi = false
      query.include_check_mi = true
      query.include_limit_mi = true
      query.include_start_mi = true
      query.include_end_mi = true
      query.include_end_timeis = false
      query.use_include_id = false
      return query
    }

    // clone / parse_find_kyou_query のどちらもコピーしないフィールド。
    // is_focus_kyou_in_list_view は一覧のフォーカス状態を持つ画面側の一時値で、
    // 検索条件として持ち回らない。
    const NOT_COPIED_FIELDS = ['is_focus_kyou_in_list_view']

    function fieldNames(query: FindKyouQuery): string[] {
      return Object.keys(query)
    }

    test('テスト側の個体が全フィールドを埋めている', () => {
      // このテストが落ちたら makeFullyPopulatedQuery に新フィールドを足す
      const populated = makeFullyPopulatedQuery()
      const fresh = new FindKyouQuery()

      const unchanged = fieldNames(fresh).filter((key) => {
        const a = (populated as unknown as Record<string, unknown>)[key]
        const b = (fresh as unknown as Record<string, unknown>)[key]
        return JSON.stringify(a) === JSON.stringify(b)
      })

      expect(unchanged).toEqual([])
    })

    test('clone が全フィールドをコピーする', () => {
      const original = makeFullyPopulatedQuery()
      const cloned = original.clone()

      const dropped = fieldNames(original).filter((key) => {
        if (NOT_COPIED_FIELDS.includes(key)) return false
        const a = (original as unknown as Record<string, unknown>)[key]
        const b = (cloned as unknown as Record<string, unknown>)[key]
        return JSON.stringify(a) !== JSON.stringify(b)
      })

      expect(dropped).toEqual([])
    })

    test('clone は配列を共有しない', () => {
      const original = makeFullyPopulatedQuery()
      const cloned = original.clone()

      cloned.tags.push('added-later')

      expect(original.tags).toEqual(['tag-1'])
    })

    test('parse_find_kyou_query が全フィールドを復元する', () => {
      const original = makeFullyPopulatedQuery()
      const restored = FindKyouQuery.parse_find_kyou_query(JSON.parse(JSON.stringify(original)))

      const dropped = fieldNames(original).filter((key) => {
        if (NOT_COPIED_FIELDS.includes(key)) return false
        const a = (original as unknown as Record<string, unknown>)[key]
        const b = (restored as unknown as Record<string, unknown>)[key]
        return JSON.stringify(a) !== JSON.stringify(b)
      })

      expect(dropped).toEqual([])
    })

    test('parse_find_kyou_query と clone のコピー対象が一致する', () => {
      // 片方だけにフィールドを足すと、保存はできるのに復元できない
      // （あるいはその逆）という非対称なバグになる
      const original = makeFullyPopulatedQuery()
      const cloned = original.clone()
      const restored = FindKyouQuery.parse_find_kyou_query(JSON.parse(JSON.stringify(original)))

      const mismatched = fieldNames(original).filter((key) => {
        const a = (cloned as unknown as Record<string, unknown>)[key]
        const b = (restored as unknown as Record<string, unknown>)[key]
        return JSON.stringify(a) !== JSON.stringify(b)
      })

      expect(mismatched).toEqual([])
    })
  })
})
