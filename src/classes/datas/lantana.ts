// ˅
'use strict';

import type { GkillError } from '../api/gkill-error';
import { InfoBase } from './info-base';

// ˄

export class Lantana extends InfoBase {
    // ˅
    
    // ˄

    mood: Number;

    attached_histories: Array<Lantana>;

    async load_attached_histories(): Promise<Array<GkillError>> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async load_attached_datas(): Promise<Array<GkillError>> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async clear_attached_histories(): Promise<Array<GkillError>> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async clear_attached_datas(): Promise<Array<GkillError>> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async clone(): Promise<Lantana> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    constructor() {
        // ˅
        super()
        this.mood = 0
        this.attached_histories = new Array<Lantana>()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
