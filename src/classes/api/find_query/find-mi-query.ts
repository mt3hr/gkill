'use strict';

import { FindQueryBase } from './find-query-base';
import { MiCheckState } from './mi-check-state';
import { MiSortType } from './mi-sort-type';
import { FindKyouQuery } from './find-kyou-query';

export class FindMiQuery extends FindQueryBase {
    board_name: string
    include_check_mi: boolean;
    include_limit_mi: boolean;
    include_start_mi: boolean;
    include_end_mi: boolean;
    sort_type: MiSortType;
    check_state: MiCheckState;

    async clone(): Promise<FindMiQuery> {
        throw new Error('Not implemented');
    }

    generate_find_kyou_query(): FindKyouQuery {
        throw new Error('Not implemented');
    }
    constructor() {
        super()
        this.board_name = ""
        this.sort_type = MiSortType.estimate_start_time
        this.check_state = MiCheckState.uncheck
        this.include_check_mi = false
        this.include_limit_mi = false
        this.include_start_mi = false
        this.include_end_mi = false
    }
}
