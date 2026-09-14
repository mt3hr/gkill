'use strict'

import type { KFTLStatementLine } from "./kftl-statement-line"

export class KFTLStatementLineContext {
    private this_statement_line_text: string

    private this_statement_line_target_id: string

    private this_is_prototype: boolean

    private next_statement_line_text: string

    private next_statement_line_target_id: string | null

    private next_is_prototype: boolean

    private next_statement_line_constructor: { (line_text: string, context: KFTLStatementLineContext): KFTLStatementLine } | null

    private kftl_statement_lines: Array<KFTLStatementLine>

    constructor(statement_line_text: string, target_id: string, next_statement_line_text: string, kftl_statement_lines: Array<KFTLStatementLine>, is_prototype: boolean) {
        this.this_statement_line_text = statement_line_text
        this.this_statement_line_target_id = target_id
        this.this_is_prototype = is_prototype
        this.next_statement_line_text = next_statement_line_text
        this.next_statement_line_target_id = ""
        this.next_is_prototype = false
        this.kftl_statement_lines = kftl_statement_lines
        this.next_statement_line_constructor = null
    }

    get_this_statement_line_text(): string {
        return this.this_statement_line_text
    }

    get_this_statement_line_target_id(): string {
        return this.this_statement_line_target_id
    }

    set_this_statement_line_target_id(this_statement_line_target_id: string): void {
        this.this_statement_line_target_id = this_statement_line_target_id
    }

    get_next_statement_line_text(): string {
        return this.next_statement_line_text
    }

    get_next_statement_line_target_id(): string | null {
        return this.next_statement_line_target_id
    }

    set_next_statement_line_target_id(target_id: string | null): void {
        this.next_statement_line_target_id = target_id
    }

    get_next_statement_line_constructor(): { (line_text: string, context: KFTLStatementLineContext): KFTLStatementLine } | null {
        return this.next_statement_line_constructor
    }

    set_next_statement_line_constructor(next_statement_line_constructor: { (line_text: string, context: KFTLStatementLineContext): KFTLStatementLine } | null): void {
        this.next_statement_line_constructor = next_statement_line_constructor
    }

    get_kftl_statement_lines(): Array<KFTLStatementLine> {
        return this.kftl_statement_lines
    }

    is_this_prototype(): boolean {
        return this.this_is_prototype
    }

    set_is_this_prototype(is_this_prototype: boolean): void {
        this.this_is_prototype = is_this_prototype
    }

    is_next_prototype(): boolean {
        return this.next_is_prototype
    }

    set_is_next_prototype(is_next_prototype: boolean): void {
        this.next_is_prototype = is_next_prototype
    }
}


