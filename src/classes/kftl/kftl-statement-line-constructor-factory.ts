'use strict'

import type { KFTLStatementLine } from "./kftl-statement-line"
import type { KFTLStatementLineContext } from "./kftl-statement-line-context"
import { KFTLKmemoStatementLine } from "./kftl_kmemo/kftl-kmemo-statement-line"
import { KFTLStartLantanaStatementLine } from "./kftl_lantana/kftl-start-lantana-statement-line"
import { KFTLStartMiStatementLine } from "./kftl_mi/kftl-start-mi-statement-line"
import { KFTLStartNlogStatementLine } from "./kftl_nlog/kftl-start-nlog-statement-line"
import { KFTLNoneStatementLine } from "./kftl_none/kftl-none-statement-line"
import { KFTLRelatedTimeStatementLine } from "./kftl_related_time/kftl-related-time-statement-line"
import { KFTLSplitAndNextSecondStatementLine } from "./kftl_split/kftl-split-and-next-second-statement-line"
import { KFTLSplitStatementLine } from "./kftl_split/kftl-split-statement-line"
import { KFTLTagStatementLine } from "./kftl_tag/kftl-tag-statement-line"
import { KFTLStartTextStatementLine } from "./kftl_text/kftl-start-text-statement-line"
import { KFTLStartTimeIsStatementLine } from "./kftl_timeis/kftl-start-time-is-statement-line"
import { KFTLTimeIsEndTitleStatementLine } from "./kftl_timeis/kftl_timeis_end/kftl-time-is-end-title-statement-line"
import { KFTLStartTimeIsEndIfExistStatementLine } from "./kftl_timeis/kftl_timeis_end/kftl_timeis_end_exist/kftl-start-time-is-end-if-exist-statement-line"
import { KFTLStartTimeIsEndByTagStatementLine } from "./kftl_timeis/kftl_timeis_end/kftl_timeis_end_tag/kftl-start-time-is-end-by-tag-statement-line"
import { KFTLStartTimeIsEndByTagIfExistStatementLine } from "./kftl_timeis/kftl_timeis_end/kftl_timeis_end_tag_exist/kftl-start-time-is-end-by-tag-if-exist-statement-line"
import { KFTLStartTimeIsStartStatementLine } from "./kftl_timeis/kftl_timeis_start/kftl-start-time-is-start-statement-line"
import { KFTLStartURLogStatementLine } from "./kftl_urlog/kftl-start-ur-log-statement-line"

export class KFTLStatementLineConstructorFactory {

    private static instance: KFTLStatementLineConstructorFactory = new KFTLStatementLineConstructorFactory()

    private prev_line_is_meta_info: boolean

    private constructor() {
        this.prev_line_is_meta_info = false
    }

    static get_instance(): KFTLStatementLineConstructorFactory {
        return this.instance
    }

    reset(): void {
        this.prev_line_is_meta_info = true
    }

    generate_none_constructor(line_text: string): { (line_text: string, context: KFTLStatementLineContext): KFTLStatementLine } {
        return this.generate_default_constructor(line_text, (line_text: string, context: KFTLStatementLineContext) => {
            this.prev_line_is_meta_info = true
            return new KFTLNoneStatementLine(line_text, context)
        })
    }

    generate_kmemo_constructor(line_text: string): { (line_text: string, context: KFTLStatementLineContext): KFTLStatementLine } {
        return this.generate_default_constructor(line_text, (line_text: string, context: KFTLStatementLineContext) => {
            this.prev_line_is_meta_info = false
            return new KFTLKmemoStatementLine(line_text, context)
        })
    }

    generate_nlog_constructor(line_text: string): { (line_text: string, context: KFTLStatementLineContext): KFTLStatementLine } {
        return this.generate_default_constructor(line_text, (line_text: string, context: KFTLStatementLineContext) => {
            this.prev_line_is_meta_info = false
            return new KFTLKmemoStatementLine(line_text, context)
        })
    }

    private generate_default_constructor(line_text: string, last_func: { (line_text: string, context: KFTLStatementLineContext): KFTLStatementLine }): { (line_text: string, context: KFTLStatementLineContext): KFTLStatementLine } {
        if (KFTLTagStatementLine.is_this_type(line_text)) {
            return (line_text: string, context: KFTLStatementLineContext) => { return new KFTLTagStatementLine(line_text, context, this.prev_line_is_meta_info) }
        }
        if (KFTLStartTextStatementLine.is_this_type(line_text)) {
            return (line_text: string, context: KFTLStatementLineContext) => { return new KFTLStartTextStatementLine(line_text, context, this.prev_line_is_meta_info) }
        }
        if (KFTLRelatedTimeStatementLine.is_this_type(line_text)) {
            return (line_text: string, context: KFTLStatementLineContext) => { return new KFTLRelatedTimeStatementLine(line_text, context, this.prev_line_is_meta_info) }
        }
        if (KFTLSplitStatementLine.is_this_type(line_text)) {
            return (line_text: string, context: KFTLStatementLineContext) => {
                this.prev_line_is_meta_info = true
                return new KFTLSplitStatementLine(line_text, context)
            }
        }
        if (KFTLSplitAndNextSecondStatementLine.is_this_type(line_text)) {
            return (line_text: string, context: KFTLStatementLineContext) => {
                this.prev_line_is_meta_info = true
                return new KFTLSplitAndNextSecondStatementLine(line_text, context)
            }
        }
        if (KFTLStartMiStatementLine.is_this_type(line_text)) {
            return (line_text: string, context: KFTLStatementLineContext) => {
                this.prev_line_is_meta_info = false
                return new KFTLStartMiStatementLine(line_text, context)
            }
        }
        if (KFTLStartLantanaStatementLine.is_this_type(line_text)) {
            return (line_text: string, context: KFTLStatementLineContext) => {
                this.prev_line_is_meta_info = false
                return new KFTLStartLantanaStatementLine(line_text, context)
            }
        }
        if (KFTLStartNlogStatementLine.is_this_type(line_text)) {
            return (line_text: string, context: KFTLStatementLineContext) => {
                this.prev_line_is_meta_info = false
                return new KFTLStartNlogStatementLine(line_text, context)
            }
        }
        if (KFTLStartTimeIsStartStatementLine.is_this_type(line_text)) {
            return (line_text: string, context: KFTLStatementLineContext) => {
                this.prev_line_is_meta_info = false
                return new KFTLStartTimeIsStartStatementLine(line_text, context)
            }
        }
        if (KFTLTimeIsEndTitleStatementLine.is_this_type(line_text)) {
            return (line_text: string, context: KFTLStatementLineContext) => {
                this.prev_line_is_meta_info = false
                return new KFTLTimeIsEndTitleStatementLine(line_text, context)
            }
        }
        if (KFTLStartTimeIsStatementLine.is_this_type(line_text)) {
            return (line_text: string, context: KFTLStatementLineContext) => {
                this.prev_line_is_meta_info = false
                return new KFTLStartTimeIsStatementLine(line_text, context)
            }
        }
        if (KFTLStartTimeIsEndIfExistStatementLine.is_this_type(line_text)) {
            return (line_text: string, context: KFTLStatementLineContext) => {
                this.prev_line_is_meta_info = false
                return new KFTLStartTimeIsEndIfExistStatementLine(line_text, context)
            }
        }
        if (KFTLStartTimeIsEndByTagStatementLine.is_this_type(line_text)) {
            return (line_text: string, context: KFTLStatementLineContext) => {
                this.prev_line_is_meta_info = false
                return new KFTLStartTimeIsEndByTagStatementLine(line_text, context)
            }
        }
        if (KFTLStartTimeIsEndByTagIfExistStatementLine.is_this_type(line_text)) {
            return (line_text: string, context: KFTLStatementLineContext) => {
                this.prev_line_is_meta_info = false
                return new KFTLStartTimeIsEndByTagIfExistStatementLine(line_text, context)
            }
        }
        if (KFTLStartURLogStatementLine.is_this_type(line_text)) {
            return (line_text: string, context: KFTLStatementLineContext) => {
                this.prev_line_is_meta_info = false
                return new KFTLStartURLogStatementLine(line_text, context)
            }
        }
        return last_func
    }
}


