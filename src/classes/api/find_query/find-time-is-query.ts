// ˅
'use strict';

import { FindQueryBase } from './find-query-base';
import { FindKyouQuery } from './find-kyou-query';

// ˄

export class FindTimeIsQuery extends FindQueryBase {
    // ˅

    // ˄

    plaing_only: boolean;

    async clone(): Promise<FindTimeIsQuery> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async generate_find_kyou_query(): Promise<FindKyouQuery> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    constructor() {
        // ˅
        super()
        this.plaing_only = false
        // ˄
    }

    // ˅

    // ˄
}

// ˅

// ˄