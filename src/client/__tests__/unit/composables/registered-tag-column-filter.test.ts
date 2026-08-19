/**
 * 「利用者がその場で作った新しいタグを、開いている列の検索条件へ足す」経路のテスト。
 *
 * 直している不具合は「タグを付けて追加した記録が、追加した直後に一覧から消える」。
 * 既定クエリが「絞らない」を tags の列挙として物質化するため、
 * タグが1つも無い時期に作られた列は tags = ["no tags"] で凍り、
 * タグの付いた記録が1件も通らなくなる（エラーも警告も出ない）。
 *
 * 使ってよい情報は「そのタグがタグツリーに無かった」という決定可能な事実だけ。
 * 既知のタグは「利用者が意図的に外した」可能性があるので触らない。
 */
import { describe, test, expect, vi, beforeEach } from 'vitest'
import { nextTick, ref, type Ref } from 'vue'

import {
    decide_new_tag_column_patch,
    useRegisteredTagColumnFilter,
} from '@/classes/use-registered-tag-column-filter'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { Tag } from '@/classes/datas/tag'

function make_query(options: {
    query_id: string
    tags: Array<string> | null
    tags_and?: boolean
}): FindKyouQuery {
    const query = new FindKyouQuery()
    query.query_id = options.query_id
    query.tags = options.tags
    query.tags_and = options.tags_and ?? false
    return query
}

function make_tag(tag_name: string): Tag {
    const tag = new Tag()
    tag.tag = tag_name
    return tag
}

describe('decide_new_tag_column_patch', () => {
    test('tags が null の列は触らない（タグで絞っていないので記録は元から通る）', () => {
        const decision = decide_new_tag_column_patch(make_query({ query_id: 'a', tags: null }), ['新タグ'])
        expect(decision).toEqual({ kind: 'skip', reason: 'tags_unused' })
    })

    test('tags_and の列は触らない（"no tags" との積が空になり列が丸ごと消える）', () => {
        // AND は query.tags.every(...)。["no tags","新タグ"] は
        //   新タグ付きの記録 → has_no_tags === false で落ちる
        //   タグ無しの記録   → has_tag_name(新タグ) で落ちる
        // となり必ず空になる
        const decision = decide_new_tag_column_patch(
            make_query({ query_id: 'a', tags: ['no tags'], tags_and: true }), ['新タグ'])
        expect(decision).toEqual({ kind: 'skip', reason: 'tags_and' })
    })

    test('"no tags" だけの OR 列には末尾へ足す', () => {
        const decision = decide_new_tag_column_patch(
            make_query({ query_id: 'a', tags: ['no tags'] }), ['新タグ'])
        expect(decision).toEqual({ kind: 'patch', tags: ['no tags', '新タグ'] })
    })

    test('既に含まれていれば触らない（差分ゼロで空検索を投げない）', () => {
        const decision = decide_new_tag_column_patch(
            make_query({ query_id: 'a', tags: ['no tags', '新タグ'] }), ['新タグ'])
        expect(decision).toEqual({ kind: 'skip', reason: 'already_included' })
    })

    test('tags が空配列（有効・0件指定）でも足す', () => {
        const decision = decide_new_tag_column_patch(make_query({ query_id: 'a', tags: [] }), ['新タグ'])
        expect(decision).toEqual({ kind: 'patch', tags: ['新タグ'] })
    })

    test('複数の新タグは渡された順で末尾へ足す', () => {
        const decision = decide_new_tag_column_patch(
            make_query({ query_id: 'a', tags: ['既存'] }), ['A', 'B'])
        expect(decision).toEqual({ kind: 'patch', tags: ['既存', 'A', 'B'] })
    })

    test('空文字のタグ名は足さない', () => {
        const decision = decide_new_tag_column_patch(make_query({ query_id: 'a', tags: ['既存'] }), [''])
        expect(decision).toEqual({ kind: 'skip', reason: 'already_included' })
    })

    test('渡されたクエリ自身は書き換えない（呼び出し側が clone して差し替える）', () => {
        const query = make_query({ query_id: 'a', tags: ['no tags'] })
        decide_new_tag_column_patch(query, ['新タグ'])
        expect(query.tags).toEqual(['no tags'])
    })
})

describe('useRegisteredTagColumnFilter', () => {
    let querys: Ref<Array<FindKyouQuery>>
    let querys_backup: Ref<Array<FindKyouQuery>>
    let known_tag_names: Set<string>
    let reload_list_by_query_id: ReturnType<typeof vi.fn>
    let suppressed_depth: number
    let wrote_inside_suppression: boolean

    function build() {
        return useRegisteredTagColumnFilter({
            querys,
            querys_backup,
            is_known_tag_name: (tag_name: string) => known_tag_names.has(tag_name),
            reload_list_by_query_id: reload_list_by_query_id as unknown as (query_id: string) => Promise<void>,
            run_with_sidebar_search_suppressed: (fn: () => void) => {
                suppressed_depth++
                const before = querys.value.map(q => q.tags?.join(',') ?? 'null').join('|')
                fn()
                const after = querys.value.map(q => q.tags?.join(',') ?? 'null').join('|')
                if (before !== after) {
                    wrote_inside_suppression = true
                }
                suppressed_depth--
            },
        })
    }

    beforeEach(() => {
        querys = ref([make_query({ query_id: 'col1', tags: ['no tags'] })])
        querys_backup = ref([make_query({ query_id: 'col1', tags: ['no tags'] })])
        known_tag_names = new Set<string>()
        reload_list_by_query_id = vi.fn().mockResolvedValue(undefined)
        suppressed_depth = 0
        wrote_inside_suppression = false
    })

    test('既知のタグでは列も backup も動かず、引き直しも走らない', async () => {
        known_tag_names.add('既知タグ')
        const { onRegisteredTag } = build()

        expect(onRegisteredTag(make_tag('既知タグ'))).toBe(false)
        await nextTick()

        expect(querys.value[0].tags).toEqual(['no tags'])
        expect(querys_backup.value[0].tags).toEqual(['no tags'])
        expect(reload_list_by_query_id).not.toHaveBeenCalled()
    })

    test('未知のタグでは querys と querys_backup の両方が書き換わり、その列を引き直す', async () => {
        const { onRegisteredTag } = build()

        expect(onRegisteredTag(make_tag('新タグ'))).toBe(true)
        await nextTick()

        expect(querys.value[0].tags).toEqual(['no tags', '新タグ'])
        expect(querys_backup.value[0].tags).toEqual(['no tags', '新タグ'])
        expect(reload_list_by_query_id).toHaveBeenCalledTimes(1)
        expect(reload_list_by_query_id).toHaveBeenCalledWith('col1')
    })

    test('列の同一性(query_id)は保たれる', async () => {
        const { onRegisteredTag } = build()
        onRegisteredTag(make_tag('新タグ'))
        await nextTick()
        expect(querys.value[0].query_id).toBe('col1')
        expect(querys_backup.value[0].query_id).toBe('col1')
    })

    test('1tickに3件来ても引き直しは列あたり1回で、3タグとも入る', async () => {
        const { onRegisteredTag } = build()

        onRegisteredTag(make_tag('A'))
        onRegisteredTag(make_tag('B'))
        onRegisteredTag(make_tag('C'))
        await nextTick()

        expect(querys.value[0].tags).toEqual(['no tags', 'A', 'B', 'C'])
        expect(reload_list_by_query_id).toHaveBeenCalledTimes(1)
    })

    test('既知判定は onRegisteredTag を呼んだその場で行う（フラッシュ前に既知へ転じても足される）', async () => {
        const { onRegisteredTag } = build()

        onRegisteredTag(make_tag('新タグ'))
        // check_tag_update が先に着地してツリーへ足した状況を模す
        known_tag_names.add('新タグ')
        await nextTick()

        expect(querys.value[0].tags).toEqual(['no tags', '新タグ'])
    })

    test('列の書き換えはサイドバー検索の抑止の中で起きる', async () => {
        const { onRegisteredTag } = build()
        onRegisteredTag(make_tag('新タグ'))
        await nextTick()
        expect(wrote_inside_suppression).toBe(true)
        expect(suppressed_depth).toBe(0)
    })

    test('tags が null の列と tags_and の列は引き直さない', async () => {
        querys.value = [
            make_query({ query_id: 'null列', tags: null }),
            make_query({ query_id: 'and列', tags: ['no tags'], tags_and: true }),
            make_query({ query_id: 'or列', tags: ['no tags'] }),
        ]
        querys_backup.value = querys.value.map(q => q.clone())
        const { onRegisteredTag } = build()

        onRegisteredTag(make_tag('新タグ'))
        await nextTick()

        expect(querys.value[0].tags).toBeNull()
        expect(querys.value[1].tags).toEqual(['no tags'])
        expect(querys.value[2].tags).toEqual(['no tags', '新タグ'])
        expect(reload_list_by_query_id).toHaveBeenCalledTimes(1)
        expect(reload_list_by_query_id).toHaveBeenCalledWith('or列')
    })

    test('apply_new_tag_names は既知判定をやり直さない（他画面からの通知用）', async () => {
        known_tag_names.add('新タグ')
        const { apply_new_tag_names } = build()

        apply_new_tag_names(['新タグ'])
        await nextTick()

        expect(querys.value[0].tags).toEqual(['no tags', '新タグ'])
        expect(reload_list_by_query_id).toHaveBeenCalledTimes(1)
    })
})
