'use strict';

import { GkillAPI } from '../api/gkill-api';
import type { GkillAPIResponse } from '../api/gkill-api-response';
import { KFTLRequestBase } from './kftl-request-base';
import type { KFTLStatementLineContext } from './kftl-statement-line-context';


export abstract class KFTLRequest extends KFTLRequestBase {


    private request_id: string;

    private tags: Array<string>;

    private current_text_id: string;

    private texts_map: Map<string, string>;

    private related_time: Date;

    private context: KFTLStatementLineContext;

    private api: GkillAPI;

    constructor(request_id: string, context: KFTLStatementLineContext) {
        super()
        this.api = new GkillAPI();

        this.request_id = ""
        this.tags = new Array<string>
        this.current_text_id = ""
        this.texts_map = new Map<string, string>()
        this.related_time = new Date(0)
        this.context = context
    }

    abstract do_request(): Promise<Array<GkillAPIResponse>>;

    async get_request_id(): Promise<string> {
        throw new Error('Not implemented');
    }

    async get_tags(): Promise<Array<string>> {
        throw new Error('Not implemented');
    }

    async set_tags(tags: Array<string>): Promise<void> {
        throw new Error('Not implemented');
    }

    async get_texts(): Promise<Array<string>> {
        throw new Error('Not implemented');
    }

    async set_texts(texts: Array<string>): Promise<void> {
        throw new Error('Not implemented');
    }

    async get_related_time(): Promise<Date> {
        throw new Error('Not implemented');
    }

    async set_related_time(time: Date): Promise<void> {
        throw new Error('Not implemented');
    }

    async add_tag(tag: string): Promise<void> {
        throw new Error('Not implemented');
    }

    async get_current_text_id(): Promise<string> {
        throw new Error('Not implemented');
    }

    async set_current_text_id(text_id: string): Promise<void> {
        throw new Error('Not implemented');
    }

    async add_text_line(text_id: string, text_line: string): Promise<void> {
        throw new Error('Not implemented');
    }

    protected async get_api(): Promise<GkillAPI> {
        throw new Error('Not implemented');
    }


}



