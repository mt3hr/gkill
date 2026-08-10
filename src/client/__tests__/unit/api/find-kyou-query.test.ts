import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { MiCheckState } from '@/classes/api/find_query/mi-check-state'
import { MiSortType } from '@/classes/api/find_query/mi-sort-type'
import { deep_equals } from '@/classes/deep-equals'
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
    // キーワードグループの活性は words / not_words の非null が担う。
    // null（未使用）のまま呼んでも何もしないため、パース検証では有効化してから使う
    function makeKeywordEnabledQuery(): FindKyouQuery {
      const query = new FindKyouQuery()
      query.words = []
      query.not_words = []
      return query
    }

    test('グループ未使用(null)なら keywords があっても何もしない', () => {
      const query = new FindKyouQuery()
      query.keywords = 'alpha beta'
      query.timeis_keywords = '作業'
      query.parse_words_and_not_words()

      expect(query.words).toBeNull()
      expect(query.not_words).toBeNull()
      expect(query.timeis_words).toBeNull()
      expect(query.timeis_not_words).toBeNull()
    })

    test('words / not_words の片方でも非nullならパースされる', () => {
      const query = new FindKyouQuery()
      query.words = ['stale']
      query.keywords = 'alpha'
      query.parse_words_and_not_words()

      expect(query.words).toEqual(['alpha'])
      expect(query.not_words).toEqual([])
    })

    test('スペース区切りの語が words に入る', () => {
      const query = makeKeywordEnabledQuery()
      query.keywords = 'alpha beta'
      query.parse_words_and_not_words()

      expect(query.words).toEqual(['alpha', 'beta'])
      expect(query.not_words).toEqual([])
    })

    test('- 前置の語は not_words に入る', () => {
      const query = makeKeywordEnabledQuery()
      query.keywords = 'alpha -beta'
      query.parse_words_and_not_words()

      expect(query.words).toEqual(['alpha'])
      expect(query.not_words).toEqual(['beta'])
    })

    test('全角スペースでも分割される', () => {
      const query = makeKeywordEnabledQuery()
      query.keywords = 'alpha　beta'
      query.parse_words_and_not_words()

      expect(query.words).toEqual(['alpha', 'beta'])
    })

    test('半角と全角スペースが混ざっても分割される', () => {
      const query = makeKeywordEnabledQuery()
      query.keywords = 'alpha　beta gamma　-delta'
      query.parse_words_and_not_words()

      expect(query.words).toEqual(['alpha', 'beta', 'gamma'])
      expect(query.not_words).toEqual(['delta'])
    })

    test('連続スペースは空語を生まない', () => {
      const query = makeKeywordEnabledQuery()
      query.keywords = '  alpha   beta  '
      query.parse_words_and_not_words()

      expect(query.words).toEqual(['alpha', 'beta'])
    })

    test('単独の - は次の語を除外語にする', () => {
      const query = makeKeywordEnabledQuery()
      query.keywords = 'alpha - beta'
      query.parse_words_and_not_words()

      expect(query.words).toEqual(['alpha'])
      expect(query.not_words).toEqual(['beta'])
    })

    test('除外指定は1語だけに効き、その次の語には持ち越さない', () => {
      const query = makeKeywordEnabledQuery()
      query.keywords = '-alpha beta'
      query.parse_words_and_not_words()

      expect(query.not_words).toEqual(['alpha'])
      expect(query.words).toEqual(['beta'])
    })

    test('語中のハイフンは除外扱いにならない', () => {
      const query = makeKeywordEnabledQuery()
      query.keywords = 'alpha-beta'
      query.parse_words_and_not_words()

      expect(query.words).toEqual(['alpha-beta'])
      expect(query.not_words).toEqual([])
    })

    // 実装は word.replace("-", "") で先頭1個しか消さないため、
    // "--foo" は "-foo" として除外語に入る。意図的かは不明だが現状の挙動を固定する。
    test('-- 前置は先頭1個だけ剥がされる', () => {
      const query = makeKeywordEnabledQuery()
      query.keywords = '--alpha'
      query.parse_words_and_not_words()

      expect(query.not_words).toEqual(['-alpha'])
      expect(query.words).toEqual([])
    })

    test('空文字なら words も not_words も空になる', () => {
      const query = makeKeywordEnabledQuery()
      query.keywords = ''
      query.parse_words_and_not_words()

      expect(query.words).toEqual([])
      expect(query.not_words).toEqual([])
    })

    test('timeis_keywords も同じルールで分解される', () => {
      const query = new FindKyouQuery()
      query.timeis_words = []
      query.timeis_keywords = '作業　-休憩'
      query.parse_words_and_not_words()

      expect(query.timeis_words).toEqual(['作業'])
      expect(query.timeis_not_words).toEqual(['休憩'])
    })

    test('呼び直すと前回の結果が残らない', () => {
      const query = makeKeywordEnabledQuery()
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

    test('rep_name を持たないノードでも throw せず空名として扱う', () => {
      // 実DBで REP_TYPE_STRUCT の内容が REP_STRUCT キーへ保存されていた実例がある。
      // ここで throw すると既定検索条件の生成とサマリ→記録先詳細の算出が丸ごと死ぬ
      const query = new FindKyouQuery()
      const rep = makeRepStructElement({ rep_name: undefined, name: 'Box' })

      const got = query.rep_to_struct(rep as never)

      expect(got).toEqual({ type: '', device: 'なし', time: '' })
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

    test('rep_name を持たない混入ノードがあっても算出は生き残る', () => {
      // 実環境の「プロファイル×記録分類→記録先詳細の算出が死ぬ」の正体。
      // 混入ノードで throw すると reps が二度と計算されなくなる
      const query = new FindKyouQuery()
      const config = configWithReps(
        [
          { rep_name: undefined, name: 'Box', rep_type_name: 'Box' },
          { rep_name: 'kmemo_laptop_2024' },
        ],
        'kmemo',
        'laptop',
      )

      expect(() => query.apply_rep_summary_to_detaul(config)).not.toThrow()
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
      // TimeIsタグはグループ未使用（null）のまま。サイドバーのTimeIsタグツリーは
      // null のとき collect_inited_tag_names でフォールバック表示する
      expect(query.timeis_tags).toBeNull()
      expect(query.devices_in_sidebar).toEqual(['laptop'])
      expect(query.rep_types_in_sidebar).toEqual(['kmemo'])
    })

    test('rykv_default_period が -1 ならカレンダー条件を付けない(null=未使用)', () => {
      const query = FindKyouQuery.generate_default_query_for_rykv(asConfig({ rykv_default_period: -1 }))

      expect(query.calendar_start_date).toBeNull()
      expect(query.calendar_end_date).toBeNull()
    })

    test('rykv_default_period が指定されていればその日数分のカレンダー条件を付ける', () => {
      const query = FindKyouQuery.generate_default_query_for_rykv(asConfig({ rykv_default_period: 7 }))

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

    test('rep_name を持たない混入ノードは reps に入らない', () => {
      const config = asConfig({
        rep_struct: makeRepStructElement({
          name: 'root',
          children: [
            makeRepStructElement({ rep_name: undefined, name: 'Box', rep_type_name: 'Box' }),
            makeRepStructElement({ rep_name: 'mi_laptop_2024' }),
          ],
        }),
      })

      const query = FindKyouQuery.generate_default_query_for_mi(config)

      expect(query.reps).toEqual(['mi_laptop_2024'])
    })

    test('カレンダー条件は付かない（rykv との差分）', () => {
      const query = FindKyouQuery.generate_default_query_for_mi(asConfig({ rykv_default_period: 7 }))

      expect(query.calendar_start_date).toBeNull()
      expect(query.calendar_end_date).toBeNull()
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

  describe('generate_default_query_for_plaing_timeis', () => {
    test('タグフィルタは未使用(null)で、Rep は check_when_inited に関係なく全部入る', () => {
      const config = asConfig({
        rep_struct: makeRepStructElement({
          name: 'root',
          children: [
            makeRepStructElement({ rep_name: 'timeis_laptop_2024', check_when_inited: true }),
            makeRepStructElement({ rep_name: 'kmemo_laptop_2024' }),
          ],
        }),
      })

      const query = FindKyouQuery.generate_default_query_for_plaing_timeis(config)

      expect(query.tags).toBeNull()
      expect(query.reps).toEqual(['timeis_laptop_2024', 'kmemo_laptop_2024'])
    })

    test('rep_name を持たない混入ノードは reps に入らない', () => {
      const config = asConfig({
        rep_struct: makeRepStructElement({
          name: 'root',
          children: [
            makeRepStructElement({ rep_name: undefined, name: 'Box', rep_type_name: 'Box' }),
            makeRepStructElement({ rep_name: 'timeis_laptop_2024' }),
          ],
        }),
      })

      const query = FindKyouQuery.generate_default_query_for_plaing_timeis(config)

      expect(query.reps).toEqual(['timeis_laptop_2024'])
    })

    test('初期チェックタグがあってもタグフィルタは null のまま（旧 use_tags=false と等価）', () => {
      const config = asConfig({
        tag_struct: makeTagStructElement({
          name: 'root',
          children: [
            makeTagStructElement({ tag_name: 'work', check_when_inited: true }),
            makeTagStructElement({ tag_name: 'archive' }),
          ],
        }),
      })

      const query = FindKyouQuery.generate_default_query_for_plaing_timeis(config)

      expect(query.tags).toBeNull()
    })

    test('apply_hide_tags は呼ばれない（従来のplaing検索の既定動作を変えない）', () => {
      const config = asConfig({
        tag_struct: makeTagStructElement({
          name: 'root',
          children: [makeTagStructElement({ tag_name: 'secret', is_force_hide: true })],
        }),
      })

      const query = FindKyouQuery.generate_default_query_for_plaing_timeis(config)

      expect(query.hide_tags).toEqual([])
    })

    test('plaing_time は設定しない（呼び出し側が強制する）', () => {
      const query = FindKyouQuery.generate_default_query_for_plaing_timeis(asConfig())

      expect(query.plaing_time).toBeNull()
    })
  })

  describe('clone / parse_find_kyou_query のフィールド網羅', () => {
    // 全41フィールドに既定値と違う値を入れた個体を作る
    // （nullable フィールドには非null のプローブ値を入れる）。
    // 新しいフィールドを FindKyouQuery に足したらここにも足すこと
    // （足し忘れは下の「未設定フィールドが無いこと」テストで落ちる）。
    function makeFullyPopulatedQuery(): FindKyouQuery {
      const query = new FindKyouQuery()
      query.query_id = 'query-1'
      query.update_cache = true
      query.keywords = 'alpha -beta'
      query.words_and = true
      query.words = ['alpha']
      query.not_words = ['beta']
      query.timeis_keywords = '作業'
      query.timeis_words_and = true
      query.timeis_words = ['作業']
      query.timeis_not_words = ['休憩']
      query.timeis_tags = ['tag-timeis']
      query.timeis_tags_and = true
      query.tags = ['tag-1']
      query.hide_tags = ['tag-hidden']
      query.tags_and = true
      query.reps = ['kmemo_laptop_2024']
      query.rep_types = ['kmemo']
      query.map_latitude = 35.1
      query.map_longitude = 139.2
      query.map_radius = 300
      query.calendar_start_date = new Date('2026-01-01T00:00:00Z')
      query.calendar_end_date = new Date('2026-01-31T23:59:59Z')
      query.plaing_time = new Date('2026-01-15T12:00:00Z')
      query.period_of_time_start_time_second = 3600
      query.period_of_time_end_time_second = 7200
      query.period_of_time_week_of_days = [1, 2, 3]
      query.devices_in_sidebar = ['laptop']
      query.rep_types_in_sidebar = ['kmemo']
      query.is_enable_map_circle_in_sidebar = true
      query.is_image_only = true
      query.is_focus_kyou_in_list_view = true
      query.mi_board_name = 'inbox'
      query.mi_sort_type = MiSortType.limit_time
      query.mi_check_state = MiCheckState.checked
      query.for_mi = true
      query.include_create_mi = false
      query.include_check_mi = true
      query.include_limit_mi = true
      query.include_start_mi = true
      query.include_end_mi = true
      query.include_end_timeis = false
      return query
    }

    // clone / parse_find_kyou_query のどちらもコピーしないフィールド
    const NOT_COPIED_FIELDS: string[] = []

    function fieldNames(query: FindKyouQuery): string[] {
      return Object.keys(query)
    }

    test('フィールド集合は新形式の41個で、use_* / update_time が存在しない', () => {
      const keys = fieldNames(new FindKyouQuery())

      expect(keys).toHaveLength(41)
      expect(keys.filter((key) => key.startsWith('use_'))).toEqual([])
      expect(keys).not.toContain('update_time')
    })

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

      cloned.tags!.push('added-later')

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

    test('parse_find_kyou_query は日付フィールドをDateへ復元し、元クエリとdeep_equalsになる', () => {
      // JSON化された日付はISO文字列になる。文字列のまま持つと、サイドバーの
      // generate_query(Dateを返す)とのdeep_equals比較が恒久的に不一致になり、
      // 機械的なupdated_queryを値比較で吸収する安全網が復元列に対して一度も効かない
      // (JSON.stringify同士の比較ではDateと文字列の差が見えないので、ここで固定する)
      const query = new FindKyouQuery()
      query.query_id = 'saved-col'
      query.calendar_start_date = new Date('2025-03-10T00:00:00+09:00')
      query.calendar_end_date = new Date('2025-03-12T23:59:59.999+09:00')
      query.plaing_time = new Date('2025-03-11T12:00:00+09:00')
      query.period_of_time_week_of_days = [1, 3]

      const json = JSON.parse(JSON.stringify(query))
      const restored = FindKyouQuery.parse_find_kyou_query(json)

      expect(restored.calendar_start_date).toBeInstanceOf(Date)
      expect(restored.calendar_end_date).toBeInstanceOf(Date)
      expect(restored.plaing_time).toBeInstanceOf(Date)
      expect(deep_equals(restored, query)).toBe(true)

      // 配列は元JSONと参照を共有しない
      restored.period_of_time_week_of_days!.push(6)
      expect(json.period_of_time_week_of_days).toEqual([1, 3])
    })

    test('parse_find_kyou_query は未設定(null)の日付をnullのまま復元する', () => {
      const empty = new FindKyouQuery()
      empty.query_id = 'empty-col'

      const restored = FindKyouQuery.parse_find_kyou_query(JSON.parse(JSON.stringify(empty)))

      expect(restored.calendar_start_date).toBeNull()
      expect(restored.calendar_end_date).toBeNull()
      expect(restored.plaing_time).toBeNull()
      expect(deep_equals(restored, empty)).toBe(true)
    })

    test('旧形式JSON(use_*入り)を読ませても旧キーがインスタンスへ生えず、キー集合は新形式と一致する', () => {
      // 旧キーが1つでも混入すると deep_equals のキー数比較が崩れ、
      // サイドバーの「機械的な再emitを同値比較で捨てる」ガードが永久に効かなくなる
      // （検索中の列をクリックすると飛行中の検索がabortされる不具合が再発する）
      const legacy_json = {
        query_id: 'legacy-col',
        keywords: 'foo',
        use_words: false,
        words: ['foo'],
        not_words: [],
        use_tags: true,
        tags: null,
        use_reps: false,
        reps: ['Kmemo'],
        use_timeis: false,
        use_timeis_tags: true,
        timeis_tags: ['tt'],
        use_mi_board_name: false,
        mi_board_name: 'inbox',
        use_update_time: true,
        update_time: '2026-01-20T09:00:00.000Z',
        use_include_id: false,
      }

      const restored = FindKyouQuery.parse_find_kyou_query(legacy_json)

      // キー集合は新形式のインスタンスと完全一致（旧キーは生えない）
      expect(fieldNames(restored).sort()).toEqual(fieldNames(new FindKyouQuery()).sort())

      // use_X=false の値は null 化、use_X=true の null 値は物質化される
      expect(restored.words).toBeNull()
      expect(restored.not_words).toBeNull()
      expect(restored.tags).toEqual([])
      expect(restored.reps).toBeNull()
      expect(restored.timeis_tags).toBeNull() // use_timeis=false が道連れにする
      expect(restored.mi_board_name).toBeNull()
      // クライアント専用フィールドは保全される
      expect(restored.keywords).toBe('foo')
      expect(restored.query_id).toBe('legacy-col')
    })

    test('古い世代のJSON(フィールド欠落)でもthrowせず、セット済みの値を優先し欠落は既定値になる', () => {
      // 実環境のryuu/dashboard/dnote等の設定には数世代前のビルドが保存したJSONが残っている。
      // かつては欠落フィールドへのundefined代入がJSON.stringifyでキーごと落ち、
      // 「一度欠けたフィールドは再保存しても戻らない」固定化が起きていた(欠落JSONの出所)。
      // 欠落フィールドの直接参照(.concat等)はTypeErrorになり、computedや初期化経路で
      // 投げられると検索条件まわりの機構が丸ごと無言で死ぬ
      const old_generation_json = {
        query_id: 'old-gen',
        use_words: true,
        keywords: '古いキーワード',
        tags: ['tagA'],
        reps: ['Kmemo'],
        // period_of_time_* / hide_tags / include_*_mi / devices_in_sidebar 等は存在しない
      }

      const restored = FindKyouQuery.parse_find_kyou_query(old_generation_json)

      // セット済みの値を優先。use_words=true は「キーワードグループ有効」として
      // words / not_words の非null化（[] 物質化）に変換される
      expect(restored.query_id).toBe('old-gen')
      expect(restored.keywords).toBe('古いキーワード')
      expect(restored.words).toEqual([])
      expect(restored.not_words).toEqual([])
      expect(restored.tags).toEqual(['tagA'])
      expect(restored.reps).toEqual(['Kmemo'])

      // 欠落フィールドはコンストラクタ既定へフォールバック
      const fresh = new FindKyouQuery()
      expect(restored.period_of_time_week_of_days).toEqual(fresh.period_of_time_week_of_days)
      expect(restored.hide_tags).toEqual(fresh.hide_tags)
      expect(restored.include_create_mi).toBe(fresh.include_create_mi)
      expect(restored.devices_in_sidebar).toEqual(fresh.devices_in_sidebar)

      // clone・再JSON化も安全で、欠落していたフィールドが復活する(固定化の自己治癒)。
      // 旧キーは再JSON化にも現れない
      const reserialized = JSON.parse(JSON.stringify(restored.clone()))
      expect(Array.isArray(reserialized.hide_tags)).toBe(true)
      expect(reserialized.period_of_time_week_of_days).toBeNull()
      expect('use_words' in reserialized).toBe(false)
    })
  })

  describe('generate_default_query_for_rykv の記録先詳細算出', () => {
    test('ツリーのis_checkedが古くても check_when_inited から reps を算出する', () => {
      // devices_in_sidebar / rep_types_in_sidebar は check_when_inited から作るのに、
      // repsだけツリーの永続化されたis_checkedから作ると、実環境(is_checkedが古い)で
      // 既定のrepsが空になり「列追加で条件が全部空」の一因になる
      const config = asConfig({
        device_struct: makeDeviceStructElement({
          name: 'root', key: '__root__',
          children: [makeDeviceStructElement({ name: 'なし', device_name: 'なし', key: 'なし', check_when_inited: true, is_checked: false })],
        }),
        rep_type_struct: makeRepTypeStructElement({
          name: 'root', key: '__root__',
          children: [
            makeRepTypeStructElement({ name: 'Kmemo', rep_type_name: 'Kmemo', key: 'Kmemo', check_when_inited: true, is_checked: false }),
            makeRepTypeStructElement({ name: 'URLog', rep_type_name: 'URLog', key: 'URLog', check_when_inited: false, is_checked: false }),
          ],
        }),
        rep_struct: makeRepStructElement({
          name: 'root', key: '__root__',
          children: [
            makeRepStructElement({ name: 'Kmemo', rep_name: 'Kmemo', key: 'Kmemo' }),
            makeRepStructElement({ name: 'URLog', rep_name: 'URLog', key: 'URLog' }),
          ],
        }),
      })

      const query = FindKyouQuery.generate_default_query_for_rykv(config)

      expect(query.devices_in_sidebar).toEqual(['なし'])
      expect(query.rep_types_in_sidebar).toEqual(['Kmemo'])
      expect(query.reps, 'ツリーのis_checked(全false)ではなくcheck_when_initedから算出すること').toEqual(['Kmemo'])
    })

    test('ignore_check_rep_rykv の rep は既定の reps から除外される', () => {
      const config = asConfig({
        device_struct: makeDeviceStructElement({
          name: 'root', key: '__root__',
          children: [makeDeviceStructElement({ name: 'なし', device_name: 'なし', key: 'なし', check_when_inited: true })],
        }),
        rep_type_struct: makeRepTypeStructElement({
          name: 'root', key: '__root__',
          children: [makeRepTypeStructElement({ name: 'Kmemo', rep_type_name: 'Kmemo', key: 'Kmemo', check_when_inited: true })],
        }),
        rep_struct: makeRepStructElement({
          name: 'root', key: '__root__',
          children: [
            makeRepStructElement({ name: 'Kmemo', rep_name: 'Kmemo', key: 'Kmemo', ignore_check_rep_rykv: true }),
          ],
        }),
      })

      const query = FindKyouQuery.generate_default_query_for_rykv(config)

      expect(query.rep_types_in_sidebar).toEqual(['Kmemo'])
      expect(query.reps).toEqual([])
    })

    test('rep_name を持たない混入ノードがあっても既定条件の生成は生き残る', () => {
      // 実環境の「列追加でデフォルト検索条件が入らない・行追加ができない」の正体。
      // REP_TYPE_STRUCT の内容が REP_STRUCT キーへ保存されていた実DBを模す
      const config = asConfig({
        device_struct: makeDeviceStructElement({
          name: 'root', key: '__root__',
          children: [makeDeviceStructElement({ name: 'なし', device_name: 'なし', key: 'なし', check_when_inited: true })],
        }),
        rep_type_struct: makeRepTypeStructElement({
          name: 'root', key: '__root__',
          children: [makeRepTypeStructElement({ name: 'Kmemo', rep_type_name: 'Kmemo', key: 'Kmemo', check_when_inited: true })],
        }),
        rep_struct: makeRepStructElement({
          name: 'root', key: '__root__',
          children: [
            makeRepStructElement({ rep_name: undefined, name: 'Box', rep_type_name: 'Box', key: 'Box', check_when_inited: true }),
            makeRepStructElement({ name: 'Kmemo', rep_name: 'Kmemo', key: 'Kmemo' }),
          ],
        }),
      })

      const query = FindKyouQuery.generate_default_query_for_rykv(config)

      expect(query.reps).toEqual(['Kmemo'])
    })
  })
})
