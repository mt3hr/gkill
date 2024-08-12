// ˅
'use strict';

import type { KFTLStatementLine } from "./kftl-statement-line";

// ˄

export class KFTLStatementLineContext {
    // ˅
    
    // ˄

    private this_statement_line_text: string;

    private this_statement_line_target_id: string;

    private this_is_prototype: boolean;

    private next_statement_line_text: string;

    private next_statement_line_target_id: string;

    private next_is_prototype: boolean;

    private next_statement_line_constructor: { (line_text: string, context: KFTLStatementLineContext): KFTLStatementLine } | null;

    private kftl_statement_lines: Array<KFTLStatementLine>;

    private add_second: Number;

    constructor(statement_line_text: string, target_id: string, next_statement_line_text: string, kftl_statement_lines: Array<KFTLStatementLine>, is_prototype: boolean) {
        // ˅
        this.this_statement_line_text = ""
        this.this_statement_line_target_id = ""
        this.this_is_prototype = false;
        this.next_statement_line_text = ""
        this.next_statement_line_target_id = ""
        this.next_is_prototype = false;
        this.kftl_statement_lines = new Array<KFTLStatementLine>()
        this.add_second = 0;
        this.next_statement_line_constructor = null
        
        // ˄
    }

    async get_this_statement_line_text(): Promise<string> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async get_this_statement_line_target_id(): Promise<string> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async set_this_statement_line_target_id(this_statement_line_target_id: string): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async get_next_statement_line_text(): Promise<string> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async get_next_statement_line_target_id(): Promise<string> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async set_next_statement_line_target_id(target_id: string|null): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async get_next_statement_line_constructor(): Promise<{ (line_text: string,context: KFTLStatementLineContext): KFTLStatementLine } | null> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async set_next_statement_line_constructor(next_statement_line_constructor: { (line_text: string, context: KFTLStatementLineContext): KFTLStatementLine } | null): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async get_prev_statement_line(): Promise<KFTLStatementLine|null> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async get_kftl_statement_lines(): Promise<Array<KFTLStatementLine>> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async get_add_second(): Promise<Number> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async set_add_second(add_second: Number): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async is_this_prototype(): Promise<boolean> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async set_is_this_prototype(is_this_prototype: boolean): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async is_next_prototype(): Promise<boolean> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async set_is_next_prototype(is_next_prototype: boolean): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
