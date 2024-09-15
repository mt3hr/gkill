'use strict';

import { FindQueryBase } from './find-query-base';
import { MoodOperator } from './mood-operator';
import { FindKyouQuery } from './find-kyou-query';


export class FindLantanaQuery extends FindQueryBase {


    use_mood_find: boolean;

    mood_value: Number;

    mood_operator: MoodOperator;

    async clone(): Promise<FindLantanaQuery> {
        throw new Error('Not implemented');
    }

    async generate_find_kyou_query(): Promise<FindKyouQuery> {
        throw new Error('Not implemented');
    }

    constructor() {
        super()
        this.use_mood_find = false
        this.mood_value = 0
        this.mood_operator = MoodOperator.all
    }


}



