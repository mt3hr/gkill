'use strict';

import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query';
import type { Kyou } from '@/classes/datas/kyou';
import type { GkillPropsBase } from './gkill-props-base';

export interface KyouListViewProps extends GkillPropsBase {
    query: FindKyouQuery;
    matched_kyous: Array<Kyou>;
}
