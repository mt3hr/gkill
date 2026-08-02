import { expect } from 'vitest'

/**
 * clone() のフィールド網羅チェック。
 *
 * データモデルの clone() は「全フィールドを1行ずつ手で写す」実装になっている。
 * フィールドを増やしたときに clone() への追記を忘れるのが典型的な壊れ方で、
 * その場合コピー元では値が入っているのにコピー先だけ既定値になる。
 * 画面上はエラーにならず、編集ダイアログで値が消える等の形で表面化する。
 *
 * 各エンティティの clone テストが「代表的な4フィールドだけ確認する」形だと
 * この壊れ方を検出できないので、全フィールドに既定値と異なる値を入れてから
 * clone し、値が変わっていないことを機械的に確認する。
 */

/**
 * clone() が意図的にコピーしないフィールド。
 *
 * いずれも「必要になった時点で load_attached_* / load_typed_* で読み直す」
 * 遅延ロード用の入れ物と、その読み込み済みフラグ。clone した側は空の状態から
 * 読み直すのが正しいので、コピーされていなくても不具合ではない。
 * is_checked_kyou は一覧の選択状態を持つ画面側の値で、これも引き継がない。
 *
 * ただしこれは「多くのクラスでコピーされない」既定であって、
 * クラスによってはコピーする（例: Kyou.clone は attached_* を slice() で写す）。
 * そのため既定リストは「コピーされていなくても許す」だけで、
 * 「コピーされていたら怒る」ことはしない。
 */
const DEFAULT_EXCLUDED_FIELDS = [
  'attached_histories',
  'attached_kyou',
  'attached_tags',
  'attached_texts',
  'attached_notifications',
  'attached_timeis_kyou',
  'is_attached_tags_loaded',
  'is_attached_texts_loaded',
  'is_attached_notifications_loaded',
  'is_attached_timeis_loaded',
  'is_checked_kyou',
]

/** JSON化して値を比較する（Date や配列もこれで比較できる）。 */
function serialize(value: unknown): string {
  return JSON.stringify(value) ?? String(value)
}

/**
 * インスタンスの各フィールドに、既定値と異なる値を型に応じて詰める。
 * 型を推測できないフィールド（null / オブジェクト）は触らず、名前を返す。
 */
function populateAllFields(target: Record<string, unknown>): { populated: string[], skipped: string[] } {
  const populated: string[] = []
  const skipped: string[] = []

  for (const key of Object.keys(target)) {
    const current = target[key]

    if (typeof current === 'string') {
      target[key] = `clone-probe-${key}`
    } else if (typeof current === 'number') {
      target[key] = current === 4649 ? 5150 : 4649
    } else if (typeof current === 'boolean') {
      target[key] = !current
    } else if (current instanceof Date) {
      target[key] = new Date('2026-03-04T05:06:07.000Z')
    } else if (Array.isArray(current)) {
      // 中身の型までは合わせない。clone() が slice()/concat() で写しているか、
      // そもそも写していないかだけを見たいので番兵で足りる。
      target[key] = [`clone-probe-${key}`]
    } else {
      // null や入れ子オブジェクトは既定値から動かせないので対象外。
      skipped.push(key)
      continue
    }
    populated.push(key)
  }

  return { populated, skipped }
}

/**
 * clone() が（意図的な除外を除く）全フィールドをコピーすることを確認する。
 *
 * @param fresh 対象クラスの新規インスタンス（この関数が値を書き換える）
 * @param options.exclude 追加で除外するフィールド名
 */
export function expectCloneCopiesAllFields<T extends object>(
  fresh: T & { clone(): T },
  options: { exclude?: string[] } = {},
): void {
  const callerExcluded = options.exclude ?? []
  const excluded = new Set([...DEFAULT_EXCLUDED_FIELDS, ...callerExcluded])

  const source = fresh as unknown as Record<string, unknown>
  const { populated } = populateAllFields(source)

  // 何も詰められていないと、以降の比較が素通りしてしまう
  expect(populated.length, 'clone を検証できるフィールドが1つも無い').toBeGreaterThan(0)

  const cloned = fresh.clone() as unknown as Record<string, unknown>

  expect(cloned, 'clone が同じインスタンスを返している').not.toBe(fresh)

  const dropped = populated.filter(
    (key) => !excluded.has(key) && serialize(cloned[key]) !== serialize(source[key]),
  )
  expect(dropped, 'clone() でコピーされていないフィールドがある').toEqual([])

  // 呼び出し側が明示した除外指定の腐り防止。
  // 「このクラスではコピーされない」と書いたフィールドが実はコピーされていたら、
  // 除外指定のほうを消す必要がある（既定リストはクラスによって挙動が違うので対象外）。
  const unnecessarilyExcluded = callerExcluded.filter(
    (key) => populated.includes(key) && serialize(cloned[key]) === serialize(source[key]),
  )
  expect(unnecessarilyExcluded, 'コピーされているのに除外指定が残っている').toEqual([])
}
