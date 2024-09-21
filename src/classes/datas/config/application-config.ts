'use strict'

import type { GkillError } from '@/classes/api/gkill-error'
import { DeviceStruct } from './device-struct'
import { KFTLTemplateStruct } from './kftl-template-struct'
import { RepStruct } from './rep-struct'
import { RepTypeStruct } from './rep-type-struct'
import { TagStruct } from './tag-struct'
import { KFTLTemplateElement } from '../kftl-template-element'
import { TagStructElementData } from './tag-struct-element-data'
import { RepStructElementData } from './rep-struct-element-data'
import { DeviceStructElementData } from './device-struct-element-data'
import { RepTypeStructElementData } from './rep-type-struct-element-data'

export class ApplicationConfig {
    is_loaded: boolean
    user_id: string
    device: string
    enable_browser_cache: boolean
    google_map_api_key: string
    rykv_image_list_column_number: Number
    rykv_hot_reload: boolean
    mi_default_board: string
    parsed_kftl_template: KFTLTemplateElement
    parsed_tag_struct: TagStructElementData
    parsed_rep_struct: RepStructElementData
    parsed_device_struct: DeviceStructElementData
    parsed_rep_type_struct: RepTypeStructElementData
    tag_struct: Array<TagStruct>
    rep_struct: Array<RepStruct>
    device_struct: Array<DeviceStruct>
    rep_type_struct: Array<RepTypeStruct>
    kftl_template: Array<KFTLTemplateStruct>
    async parse_template_and_struct(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }
    async clone(): Promise<ApplicationConfig> {
        throw new Error('Not implemented')
    }
    async parse_kftl_template(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }
    async parse_tag_struct(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }
    async parse_rep_struct(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }
    async parse_device_struct(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }
    async parse_rep_type_struct(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }
    constructor() {
        this.is_loaded = false
        this.user_id = ""
        this.device = ""
        this.enable_browser_cache = false
        this.google_map_api_key = ""
        this.rykv_image_list_column_number = 3
        this.rykv_hot_reload = false
        this.mi_default_board = ""
        this.parsed_kftl_template = new KFTLTemplateElement()
        this.parsed_tag_struct = new TagStructElementData()
        this.parsed_rep_struct = new RepStructElementData()
        this.parsed_device_struct = new DeviceStructElementData()
        this.parsed_rep_type_struct = new RepTypeStructElementData()
        this.tag_struct = new Array<TagStruct>()
        this.rep_struct = new Array<RepStruct>()
        this.device_struct = new Array<DeviceStruct>()
        this.rep_type_struct = new Array<RepTypeStruct>()
        this.kftl_template = new Array<KFTLTemplateStruct>()
    }
}
