// ˅
'use strict';

import type { GkillError } from '../api/gkill-error';
import { IDFKyou } from './idf-kyou';
import { InfoBase } from './info-base';
import { Kmemo } from './kmemo';
import { Lantana } from './lantana';
import { Mi } from './mi';
import { Nlog } from './nlog';
import { TimeIs } from './time-is';
import { URLog } from './ur-log';

// ˄

export class Kyou extends InfoBase {
    // ˅

    // ˄

    image_source: string;

    attached_histories: Array<Kyou>;

    typed_kmemo: Kmemo;

    typed_urlog: URLog;

    typed_nlog: Nlog;

    typed_timeis: TimeIs;

    typed_mi: Mi;

    typed_lantana: Lantana;

    typed_idf_kyou: IDFKyou;

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

    async load_typed_kmemo(): Promise<Array<GkillError>> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async load_typed_urlog(): Promise<Array<GkillError>> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async load_typed_nlog(): Promise<Array<GkillError>> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async load_typed_timeis(): Promise<Array<GkillError>> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async load_typed_mi(): Promise<Array<GkillError>> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async load_typed_lantana(): Promise<Array<GkillError>> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async load_typed_idf_kyou(): Promise<Array<GkillError>> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async clear_typed_datas(): Promise<Array<GkillError>> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    clone(): Promise<Kyou> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    constructor() {
        // ˅
        super()
        this.image_source = ""

        this.attached_histories = new Array<Kyou>()

        this.typed_kmemo = new Kmemo()

        this.typed_urlog = new URLog()

        this.typed_nlog = new Nlog()

        this.typed_timeis = new TimeIs()

        this.typed_mi = new Mi()

        this.typed_lantana = new Lantana()

        this.typed_idf_kyou = new IDFKyou()
        // ˄
    }

    // ˅

    // ˄
}

// ˅

// ˄
