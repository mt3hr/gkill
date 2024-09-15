'use strict';

import type { KFTLStatementLine } from "./kftl-statement-line";
import type { KFTLStatementLineContext } from "./kftl-statement-line-context";


export class KFTLStatementLineConstructorFactory {


    private instance: KFTLStatementLineConstructorFactory;

    private prev_line_is_meta_info: boolean;

    private constructor() {
        this.instance = new KFTLStatementLineConstructorFactory();
        this.prev_line_is_meta_info = false
    }

    async get_instance(): Promise<KFTLStatementLineConstructorFactory> {
        throw new Error('Not implemented');
    }

    async reset(): Promise<void> {
        throw new Error('Not implemented');
    }

    async generate_none_constructor(line_text: string): Promise<{ (line_text: string, context: KFTLStatementLineContext): KFTLStatementLine }> {
        throw new Error('Not implemented');
    }

    async generate_kmemo_constructor(line_text: string): Promise<{ (line_text: string, context: KFTLStatementLineContext): KFTLStatementLine }> {
        throw new Error('Not implemented');
    }

    async generate_nlog_constructor(line_text: string): Promise<{ (line_text: string, context: KFTLStatementLineContext): KFTLStatementLine }> {
        throw new Error('Not implemented');
    }

    async generate_default_constructor(line_text: string): Promise<{ (line_text: string, context: KFTLStatementLineContext): KFTLStatementLine }> {
        throw new Error('Not implemented');
    }


}



