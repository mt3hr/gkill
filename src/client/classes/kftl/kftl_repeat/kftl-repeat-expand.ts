'use strict'

import { i18n } from '@/i18n'
import { GkillAPI } from '../../api/gkill-api'
import type { ApplicationConfig } from '../../datas/config/application-config'
import type { KFTLRequest } from '../kftl-request'
import { days_between, occurrences_of, REPEAT_MAX_RECORDS, validate_repeat_spec, type RepeatSpec } from './kftl-repeat-spec'

/**
 * 繰り返しブロック「？？」の展開。
 *
 * **展開は送信時にだけ行う。** `use-kftl-view.ts` は本文が変わるたびに
 * `get_invalid_line_indexs()`（= 全行の apply）を回すので、そこで複製すると
 * 打鍵1回あたり最大 REPEAT_MAX_RECORDS 件を作ることになる。
 * 打鍵のたびに走る経路は validate_repeats（複製しない検査だけ）を使う。
 *
 * グループは**同じ spec の参照を共有しているか**で決まる。支出ブロックは
 * 全支払いが同じ参照を見るので、ブロックまるごとが1グループになる。
 *
 * Mirrors: expandRepeats (src/server/gkill/api/kftl/kftl_repeat_lines.go)
 */

interface RepeatGroup {
    spec: RepeatSpec
    members: Array<KFTLRequest>
}

interface RepeatSlot {
    request: KFTLRequest | null
    group: RepeatGroup | null
}

/** 挿入順を保ったままグループにまとめる。グループは最初の1件が居た位置に置く。 */
function build_slots(requests: Array<KFTLRequest>): Array<RepeatSlot> {
    const slots: Array<RepeatSlot> = []
    const groups = new Map<RepeatSpec, RepeatGroup>()
    for (const request of requests) {
        const spec = request.get_repeat_spec()
        if (spec === null) {
            slots.push({ request: request, group: null })
            continue
        }
        let group = groups.get(spec)
        if (group === undefined) {
            group = { spec: spec, members: [] }
            groups.set(spec, group)
            slots.push({ request: null, group: group })
        }
        group.members.push(request)
    }
    return slots
}

/**
 * 展開せずに検査だけする。**打鍵のたびに走る経路から呼ぶ。**
 * 不正だったブロックの開始行の位置を返す（複製はしないので候補の計算だけで済む）。
 */
export function validate_repeats(requests: Array<KFTLRequest>, base: Date): Array<number> {
    const invalid_line_indexs: Array<number> = []
    for (const slot of build_slots(requests)) {
        if (slot.group === null) {
            continue
        }
        try {
            resolve_occurrences(slot.group, base)
        } catch (_e: unknown) {
            invalid_line_indexs.push(slot.group.spec.line_index)
        }
    }
    return invalid_line_indexs
}

/** 必須2行・アンカー・候補の検査をまとめて行い、候補日時を返す。 */
function resolve_occurrences(group: RepeatGroup, base: Date): { anchor: Date; occurrences: Array<Date> } {
    validate_repeat_spec(group.spec)
    // アンカーはグループの先頭から取る。支出ブロックは関連時刻をブロックで共有するので、
    // どの支払いから取っても同じ
    const anchor = group.members[0].anchor_time_for_repeat()
    if (anchor === null) {
        throw new Error(i18n.global.t("KFTL_REPEAT_NO_DATE_FIELD_MESSAGE_TITLE"))
    }
    const occurrences = occurrences_of(group.spec, anchor, base)
    if (occurrences.length === 0) {
        throw new Error(i18n.global.t("KFTL_REPEAT_NO_OCCURRENCE_MESSAGE_TITLE"))
    }
    return { anchor: anchor, occurrences: occurrences }
}

/**
 * 繰り返し指定を持つリクエストを候補日時のぶんだけ複製する。
 *
 * gkill_api が null のときは既存判定を飛ばす（単体テストのようにAPIが無い場合）。
 * Go 側で repositories が未設定なら「既存なし」として扱うのと対。
 */
export async function expand_repeats(
    requests: Array<KFTLRequest>,
    base: Date,
    gkill_api: GkillAPI | null,
    application_config: ApplicationConfig | null,
): Promise<Array<KFTLRequest>> {
    const slots = build_slots(requests)
    if (!slots.some((slot) => slot.group !== null)) {
        return requests
    }

    const expanded: Array<KFTLRequest> = []
    for (const slot of slots) {
        if (slot.group === null) {
            expanded.push(slot.request!)
            continue
        }
        expanded.push(...await expand_repeat_group(slot.group, base, gkill_api, application_config))
    }
    if (expanded.length > REPEAT_MAX_RECORDS) {
        throw new Error(i18n.global.t("KFTL_REPEAT_LIMIT_EXCEEDED_MESSAGE_TITLE"))
    }
    return expanded
}

async function expand_repeat_group(
    group: RepeatGroup,
    base: Date,
    gkill_api: GkillAPI | null,
    application_config: ApplicationConfig | null,
): Promise<Array<KFTLRequest>> {
    const resolved = resolve_occurrences(group, base)
    const occurrences = resolved.occurrences

    // 既存があればその回を飛ばす（3行目の既定が no なので、既定でこちらを通る）。
    //
    // **メンバーごとに引く。** 支出ブロックは支払いごとに品名が違うので、
    // グループでまとめて判定すると「片方だけ既にある」ときに残りも作られなくなる。
    // 引くのはメンバー数ぶんだけで、候補の回数ぶんは引かない。
    const existing_by_member: Array<Set<number>> = group.members.map(() => new Set<number>())
    if (!group.spec.add_if_exists && gkill_api !== null) {
        const from = new Date(occurrences[0].getTime())
        from.setDate(from.getDate() - 1)
        const to = new Date(occurrences[occurrences.length - 1].getTime())
        to.setDate(to.getDate() + 1)
        for (let i = 0; i < group.members.length; i++) {
            existing_by_member[i] = await group.members[i].find_existing_for_repeat(gkill_api, application_config, from, to)
        }
    }

    const out: Array<KFTLRequest> = []
    // 元のIDを引き継ぐのは「実際に作る最初の1件」。
    // 既存で先頭の回が飛ばされても、作られた1件目が引き継ぐ
    const id_taken = group.members.map(() => false)
    for (const occurrence of occurrences) {
        // 全日時欄を同じ日数だけずらす。欄どうしの相対差は保たれる
        const day_shift = days_between(resolved.anchor, occurrence)
        for (let i = 0; i < group.members.length; i++) {
            if (existing_by_member[i].has(occurrence.getTime())) {
                continue
            }
            let id = GkillAPI.get_gkill_api().generate_uuid()
            if (!id_taken[i]) {
                id = group.members[i].get_request_id()
                id_taken[i] = true
            }
            out.push(group.members[i].clone_for_repeat(id, day_shift))
        }
    }
    return out
}
