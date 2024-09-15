'use strict';

import type { GkillError } from "../api/gkill-error";
import { InfoBase } from "./info-base";


export class GitCommitLog extends InfoBase {


    commit_message: string;

    async load_attached_histories(): Promise<Array<GkillError>> {
        throw new Error('Not implemented');
    }

    async load_attached_datas(): Promise<Array<GkillError>> {
        throw new Error('Not implemented');
    }

    async clear_attached_histories(): Promise<Array<GkillError>> {
        throw new Error('Not implemented');
    }

    async clear_attached_datas(): Promise<Array<GkillError>> {
        throw new Error('Not implemented');
    }

    async clone(): Promise<GitCommitLog> {
        throw new Error('Not implemented');
    }

    constructor() {
        super()
        this.commit_message = ""
    }


}



