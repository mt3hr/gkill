import moment from 'moment'
import type { FindKyouQuery } from "../api/find_query/find-kyou-query";
import type { Kyou } from "../datas/kyou";
import type DnoteAggregateTarget from "./dnote-aggregate-target";
import type DnotePredicate from "./dnote-predicate";
import load_kyous from "./kyou-loader";
import type DnoteTrendPoint from "./dnote-trend/dnote-trend-point";
import type { DnoteTrendGranularity } from "./dnote-trend/dnote-trend-types";
import aggregated_value_to_number from "./dnote-trend/aggregated-value-to-number";

// バケット数の上限（広すぎる検索範囲による暴走防止。超過時は新しい側を優先）
const max_bucket_count = 400

function start_of_unit(granularity: DnoteTrendGranularity): moment.unitOfTime.StartOf {
    switch (granularity) {
        case 'week': return 'isoWeek'
        case 'month': return 'month'
        default: return 'day'
    }
}

function add_unit(granularity: DnoteTrendGranularity): moment.unitOfTime.DurationConstructor {
    switch (granularity) {
        case 'week': return 'week'
        case 'month': return 'month'
        default: return 'day'
    }
}

function bucket_label(bucket_start: moment.Moment, granularity: DnoteTrendGranularity): string {
    switch (granularity) {
        case 'month': return bucket_start.format('YYYY/M')
        default: return bucket_start.format('M/D')
    }
}

export class DnoteTrendAggregator {
    private dnote_predicate: DnotePredicate
    private dnote_aggregate_target: DnoteAggregateTarget
    private granularity: DnoteTrendGranularity

    constructor(dnote_predicate: DnotePredicate, dnote_aggregate_target: DnoteAggregateTarget, granularity: DnoteTrendGranularity) {
        this.dnote_predicate = dnote_predicate
        this.dnote_aggregate_target = dnote_aggregate_target
        this.granularity = granularity
    }

    // 渡されたkyousを粒度単位のバケットに区切って集計する。
    // バケット期間はfind_kyou_queryのcalendar範囲、なければkyousのrelated_timeのmin/maxから導出する。
    public async aggregate_trend(abort_controller: AbortController, kyous: Array<Kyou>, find_kyou_query: FindKyouQuery, kyou_is_loaded: boolean): Promise<Array<DnoteTrendPoint>> {
        const unit = start_of_unit(this.granularity)
        const step = add_unit(this.granularity)

        // 渡されたデータの全項目を取得
        const get_latest_data = false
        const cloned_kyous = await load_kyous(abort_controller, kyous, get_latest_data, !kyou_is_loaded)

        // バケット期間の決定
        let window_start: Date
        let window_end: Date
        if (find_kyou_query.calendar_start_date || find_kyou_query.calendar_end_date) {
            window_start = find_kyou_query.calendar_start_date ?? find_kyou_query.calendar_end_date!
            window_end = find_kyou_query.calendar_end_date ?? new Date(Date.now())
        } else if (cloned_kyous.length > 0) {
            // 全期間検索などcalendar範囲がない場合はkyousのrelated_timeから導出
            let min_time = cloned_kyous[0].related_time.getTime()
            let max_time = min_time
            for (let i = 1; i < cloned_kyous.length; i++) {
                const time = cloned_kyous[i].related_time.getTime()
                if (time < min_time) min_time = time
                if (time > max_time) max_time = time
            }
            window_start = new Date(min_time)
            window_end = new Date(max_time)
        } else {
            window_start = new Date(Date.now())
            window_end = new Date(Date.now())
        }

        // バケット列を先に生成する（ゼロ埋め・昇順を保証）
        // 上限超過時は新しい側を残すため、開始位置を末尾から遡って決める
        const end = moment(window_end)
        let cursor = moment(window_start).startOf(unit)
        const total_bucket_count = end.diff(cursor, step) + 1
        if (total_bucket_count > max_bucket_count) {
            cursor = end.clone().startOf(unit).subtract(max_bucket_count - 1, step)
        }

        const points = new Array<DnoteTrendPoint>()
        const buckets = new Map<string, { point: DnoteTrendPoint, aggregated_value: unknown, bucket_query: FindKyouQuery, bucket_start_ms: number, bucket_end_exclusive_ms: number }>()
        const first_bucket_start_ms = cursor.valueOf()
        let last_bucket_start_ms = first_bucket_start_ms
        while (cursor.isSameOrBefore(end) && points.length < max_bucket_count) {
            const bucket_start = cursor.clone()
            // 終端は排他的（翌単位の0:00）。TimeIsのTrimを0:00ちょうどで区切るため
            const bucket_end_exclusive = cursor.clone().add(1, step)
            const point: DnoteTrendPoint = {
                bucket_key: bucket_start.format('YYYY-MM-DD'),
                label: bucket_label(bucket_start, this.granularity),
                value: 0,
                value_string: "",
                match_kyous: new Array<Kyou>(),
            }
            const bucket_query = typeof find_kyou_query.clone === 'function' ? find_kyou_query.clone() : find_kyou_query
            if (bucket_query !== find_kyou_query) {
                bucket_query.calendar_start_date = bucket_start.toDate()
                bucket_query.calendar_end_date = bucket_end_exclusive.toDate()
            }
            points.push(point)
            buckets.set(point.bucket_key, { point, aggregated_value: null, bucket_query, bucket_start_ms: bucket_start.valueOf(), bucket_end_exclusive_ms: bucket_end_exclusive.valueOf() })
            last_bucket_start_ms = bucket_start.valueOf()
            cursor = cursor.clone().add(1, step)
        }

        // predicateにマッチしたKyouをバケットへ振り分けて集計
        for (let i = 0; i < cloned_kyous.length; i++) {
            const kyou = cloned_kyous[i]
            if (!(await this.dnote_predicate.is_match(kyou, null))) {
                continue
            }
            if (kyou.typed_timeis) {
                // TimeIsは日付をまたぐことがあるため、期間が重なる全バケットへ振り分ける
                // （バケット内への切り詰めは各AggregateTargetがbucket_queryのcalendar範囲で行う）
                const span_start = kyou.typed_timeis.start_time.getTime()
                const raw_end = kyou.typed_timeis.end_time ? kyou.typed_timeis.end_time.getTime() : Date.now()
                const span_end = Math.max(raw_end, span_start + 1)
                let span_cursor = moment(Math.max(span_start, first_bucket_start_ms)).startOf(unit)
                while (span_cursor.valueOf() < span_end && span_cursor.valueOf() <= last_bucket_start_ms) {
                    const bucket = buckets.get(span_cursor.format('YYYY-MM-DD'))
                    if (bucket && span_start < bucket.bucket_end_exclusive_ms && span_end > bucket.bucket_start_ms) {
                        bucket.aggregated_value = await this.dnote_aggregate_target.append_aggregate_element_value(bucket.aggregated_value, kyou, bucket.bucket_query)
                        bucket.point.match_kyous.push(kyou.clone())
                    }
                    span_cursor = span_cursor.clone().add(1, step)
                }
                continue
            }
            const bucket_key = moment(kyou.related_time).startOf(unit).format('YYYY-MM-DD')
            const bucket = buckets.get(bucket_key)
            if (!bucket) {
                continue
            }
            bucket.aggregated_value = await this.dnote_aggregate_target.append_aggregate_element_value(bucket.aggregated_value, kyou, bucket.bucket_query)
            bucket.point.match_kyous.push(kyou.clone())
        }

        for (const bucket of buckets.values()) {
            if (bucket.aggregated_value === null) {
                continue
            }
            bucket.point.value = aggregated_value_to_number(bucket.aggregated_value)
            bucket.point.value_string = await this.dnote_aggregate_target.result_to_string(bucket.aggregated_value)
        }
        return points
    }
}
