'use strict'

import { GkillAPI } from '../api/gkill-api'
import type { GkillError } from '../api/gkill-error'
import { AddTagRequest } from '../api/req_res/add-tag-request'
import { AddTextRequest } from '../api/req_res/add-text-request'
import { GetGkillInfoRequest } from '../api/req_res/get-gkill-info-request'
import { KFTLRequestBase } from './kftl-request-base'
import type { KFTLStatementLineContext } from './kftl-statement-line-context'

export abstract class KFTLRequest extends KFTLRequestBase {

    private request_id: string

    private tags: Array<string>

    private current_text_id: string | null

    private texts_map: Map<string, string>

    private related_time: Date | null

    private context: KFTLStatementLineContext

    private api: GkillAPI

    constructor(request_id: string, context: KFTLStatementLineContext) {
        super()
        this.api = GkillAPI.get_instance()

        this.request_id = request_id
        this.tags = new Array<string>
        this.current_text_id = ""
        this.texts_map = new Map<string, string>()
        this.related_time = null
        this.context = context
    }

    async do_request(): Promise<Array<GkillError>> {
        let errors = Array<GkillError>()
        const time = this.get_related_time() != null ? this.get_related_time()!! : new Date(Date.now())
        const req = new GetGkillInfoRequest()
        req.session_id = GkillAPI.get_instance().get_session_id()
        const res = await GkillAPI.get_instance().get_gkill_info(req)

        for (let i = 0; i < this.tags.length; i++) {
            const tag = this.tags[i]
            const req = new AddTagRequest()
            req.session_id = GkillAPI.get_instance().get_session_id()
            req.tag.id = GkillAPI.get_instance().generate_uuid()
            req.tag.tag = tag
            req.tag.target_id = this.get_request_id()
            req.tag.related_time = time
            req.tag.create_app = "gkill_kftl"
            req.tag.create_device = res.device
            req.tag.create_time = time
            req.tag.create_user = res.user_id
            req.tag.update_app = "gkill_kftl"
            req.tag.update_device = res.device
            req.tag.update_time = time
            req.tag.update_user = res.user_id
            await GkillAPI.get_instance().add_tag(req).then((res) => {
                if (res.errors && res.errors.length !== 0) {
                    errors = errors.concat(res.errors)
                }
            })
        }
        for (const text_entry of this.texts_map) {
            const id = text_entry[0]
            const text = text_entry[1]

            const req = new AddTextRequest()
            req.session_id = GkillAPI.get_instance().get_session_id()
            req.text.id = id
            req.text.target_id = this.get_request_id()
            req.text.text = text
            req.text.related_time = time
            req.text.create_app = "gkill_kftl"
            req.text.create_device = res.device
            req.text.create_time = time
            req.text.create_user = res.user_id
            req.text.update_app = "gkill_kftl"
            req.text.update_device = res.device
            req.text.update_time = time
            req.text.update_user = res.user_id
            await GkillAPI.get_instance().add_text(req).then((res) => {
                if (res.errors && res.errors.length !== 0) {
                    errors = errors.concat(res.errors)
                }
            })
        }
        return errors
    }

    get_request_id(): string {
        return this.request_id
    }

    get_tags(): Array<string> {
        return this.tags
    }

    set_tags(tags: Array<string>): void {
        this.tags = tags
    }

    get_texts(): Array<string> {
        const texts = Array<string>()
        this.texts_map.forEach(text => {
            texts.push(text)
        });
        return texts
    }

    set_texts(texts: Array<string>): void {
        this.texts_map.clear()
        texts.forEach(text => {
            this.texts_map.set(GkillAPI.get_instance().generate_uuid(), text)
        });
    }

    get_related_time(): Date | null {
        return this.related_time
    }

    set_related_time(time: Date | null): void {
        this.related_time = time
    }

    add_tag(tag: string): void {
        this.tags.push(tag)
    }

    get_current_text_id(): string | null {
        return this.current_text_id
    }

    set_current_text_id(text_id: string | null): void {
        this.current_text_id = text_id
    }

    add_text_line(text_id: string, text_line: string): void {
        let text = this.texts_map.get(text_id)
        if (!text) {
            text = `${text_line}`
        } else {
            text += `\n${text_line}`
        }
        this.texts_map.set(text_id, text)
    }
}


