'use strict'

import { GkillError } from '@/classes/api/gkill-error'
import { DeviceStruct } from './device-struct'
import { RepStruct } from './rep-struct'
import { RepTypeStruct } from './rep-type-struct'
import { TagStruct } from './tag-struct'
import { TagStructElementData } from './tag-struct-element-data'
import { RepStructElementData } from './rep-struct-element-data'
import { DeviceStructElementData } from './device-struct-element-data'
import { RepTypeStructElementData } from './rep-type-struct-element-data'
import { KFTLTemplateElementData } from '../kftl-template-element-data'
import type { KFTLTemplateStruct } from './kftl-template-struct'

export class ApplicationConfig {
    is_loaded: boolean
    user_id: string
    device: string
    enable_browser_cache: boolean
    google_map_api_key: string
    rykv_image_list_column_number: Number
    rykv_hot_reload: boolean
    mi_default_board: string
    parsed_kftl_template: KFTLTemplateElementData
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
        const errors = Array<GkillError>()
        errors.concat(await this.parse_tag_struct())
        errors.concat(await this.parse_rep_struct())
        errors.concat(await this.parse_device_struct())
        errors.concat(await this.parse_rep_type_struct())
        errors.concat(await this.parse_kftl_template())
        return errors
    }
    async clone(): Promise<ApplicationConfig> {
        const application_config = new ApplicationConfig()
        application_config.is_loaded = this.is_loaded
        application_config.user_id = this.user_id
        application_config.device = this.device
        application_config.enable_browser_cache = this.enable_browser_cache
        application_config.google_map_api_key = this.google_map_api_key
        application_config.rykv_image_list_column_number = this.rykv_image_list_column_number
        application_config.rykv_hot_reload = this.rykv_hot_reload
        application_config.mi_default_board = this.mi_default_board
        application_config.tag_struct = this.tag_struct
        application_config.rep_struct = this.rep_struct
        application_config.device_struct = this.device_struct
        application_config.rep_type_struct = this.rep_type_struct
        application_config.kftl_template = this.kftl_template
        return application_config
    }
    async parse_kftl_template(): Promise<Array<GkillError>> {
        const added_list = new Array<KFTLTemplateStruct>()
        const not_added_list: Array<KFTLTemplateStruct> = this.kftl_template.concat()
        const struct = new KFTLTemplateElementData()
        struct.children = new Array<KFTLTemplateElementData>()

        // 親を持たないものをルートに追加する
        for (let i = 0; i < not_added_list.length; i++) {
            let item = not_added_list[i]
            if (!item.parent_folder_id || item.parent_folder_id === "") {
                const struct_element = new KFTLTemplateElementData()
                struct_element.id = item.id
                struct_element.title = item.title
                if (item.template) {
                    struct_element.template = item.template
                }

                struct.children.push(struct_element)
                added_list.push(item)
                not_added_list.splice(i, 1)
                i--;
            }
        }
        // 再帰呼び出し用
        let walk = (struct: KFTLTemplateElementData, target_id: string): KFTLTemplateElementData | null => {
            throw new Error('Not implemented')
        }
        // structを潜ってIDが一致するものを取得する
        walk = (struct: KFTLTemplateElementData, target_id: string): KFTLTemplateElementData | null => {
            if (struct.id === target_id) {
                return struct
            }
            if (!struct.children) {
                return null
            }
            let target: KFTLTemplateElementData | null = null
            for (let i = 0; i < struct.children.length; i++) {
                const child = struct.children[i]
                target = walk(child, target_id)
                if (target) {
                    break
                }
            }
            return target
        }
        while (not_added_list.length !== 0) {
            for (let i = 0; i < not_added_list.length; i++) {
                let item = not_added_list[i]
                const parent_struct = walk(struct, item.parent_folder_id)
                if (parent_struct) {
                    const struct_element = new KFTLTemplateElementData()
                    struct_element.id = item.id
                    struct_element.title = item.title
                    if (item.template) {
                        struct_element.template = item.template
                    }

                    if (!parent_struct.children) {
                        parent_struct.children = new Array<KFTLTemplateElementData>()
                    }
                    parent_struct.children.push(struct_element)
                    added_list.push(item)
                    not_added_list.splice(i, 1)
                    i--;
                }
            }
        }
        this.parsed_kftl_template = struct
        return new Array<GkillError>()
    }
    async parse_tag_struct(): Promise<Array<GkillError>> {
        const added_list = new Array<TagStruct>()
        const not_added_list: Array<TagStruct> = this.tag_struct.concat()
        const struct = new TagStructElementData()
        struct.children = new Array<TagStructElementData>()

        // 親を持たないものをルートに追加する
        for (let i = 0; i < not_added_list.length; i++) {
            let item = not_added_list[i]
            if (!item.parent_folder_id || item.parent_folder_id === "") {
                const struct_element = new TagStructElementData()
                struct_element.check_when_inited = item.check_when_inited
                struct_element.id = item.id
                struct_element.indeterminate = false
                struct_element.is_checked = item.check_when_inited
                struct_element.is_force_hide = item.is_force_hide
                struct_element.key = item.tag_name
                struct_element.tag_name = item.tag_name

                struct.children.push(struct_element)
                added_list.push(item)
                not_added_list.splice(i, 1)
                i--;
            }
        }
        // 再帰呼び出し用
        let walk = (struct: TagStructElementData, target_id: string): TagStructElementData | null => {
            throw new Error('Not implemented')
        }
        // structを潜ってIDが一致するものを取得する
        walk = (struct: TagStructElementData, target_id: string): TagStructElementData | null => {
            if (struct.id === target_id) {
                return struct
            }
            if (!struct.children) {
                return null
            }
            let target: TagStructElementData | null = null
            for (let i = 0; i < struct.children.length; i++) {
                const child = struct.children[i]
                target = walk(child, target_id)
                if (target) {
                    break
                }
            }
            return target
        }
        while (not_added_list.length !== 0) {
            for (let i = 0; i < not_added_list.length; i++) {
                let item = not_added_list[i]
                const parent_struct = walk(struct, item.parent_folder_id)
                if (parent_struct) {
                    const struct_element = new TagStructElementData()
                    struct_element.check_when_inited = item.check_when_inited
                    struct_element.id = item.id
                    struct_element.indeterminate = false
                    struct_element.is_checked = item.check_when_inited
                    struct_element.is_force_hide = item.is_force_hide
                    struct_element.key = item.tag_name
                    struct_element.tag_name = item.tag_name

                    if (!parent_struct.children) {
                        parent_struct.children = new Array<TagStructElementData>()
                    }
                    parent_struct.children.push(struct_element)
                    added_list.push(item)
                    not_added_list.splice(i, 1)
                    i--;
                }
            }
        }
        this.parsed_tag_struct = struct
        return new Array<GkillError>()
    }
    async parse_rep_struct(): Promise<Array<GkillError>> {
        const added_list = new Array<RepStruct>()
        const not_added_list: Array<RepStruct> = this.rep_struct.concat()
        const struct = new RepStructElementData()
        struct.children = new Array<RepStructElementData>()

        // 親を持たないものをルートに追加する
        for (let i = 0; i < not_added_list.length; i++) {
            let item = not_added_list[i]
            if (!item.parent_folder_id || item.parent_folder_id === "") {
                const struct_element = new RepStructElementData()
                struct_element.check_when_inited = item.check_when_inited
                struct_element.id = item.id
                struct_element.indeterminate = false
                struct_element.is_checked = item.check_when_inited
                struct_element.ignore_check_rep_rykv = struct_element.ignore_check_rep_rykv
                struct_element.key = item.rep_name
                struct_element.rep_name = item.rep_name

                struct.children.push(struct_element)
                added_list.push(item)
                not_added_list.splice(i, 1)
                i--;
            }
        }
        // 再帰呼び出し用
        let walk = (struct: RepStructElementData, target_id: string): RepStructElementData | null => {
            throw new Error('Not implemented')
        }
        // structを潜ってIDが一致するものを取得する
        walk = (struct: RepStructElementData, target_id: string): RepStructElementData | null => {
            if (struct.id === target_id) {
                return struct
            }
            if (!struct.children) {
                return null
            }
            let target: RepStructElementData | null = null
            for (let i = 0; i < struct.children.length; i++) {
                const child = struct.children[i]
                target = walk(child, target_id)
                if (target) {
                    break
                }
            }
            return target
        }
        while (not_added_list.length !== 0) {
            for (let i = 0; i < not_added_list.length; i++) {
                let item = not_added_list[i]
                const parent_struct = walk(struct, item.parent_folder_id)
                if (parent_struct) {
                    const struct_element = new RepStructElementData()
                    struct_element.check_when_inited = item.check_when_inited
                    struct_element.id = item.id
                    struct_element.indeterminate = false
                    struct_element.is_checked = item.check_when_inited
                    struct_element.ignore_check_rep_rykv = struct_element.ignore_check_rep_rykv
                    struct_element.key = item.rep_name
                    struct_element.rep_name = item.rep_name

                    if (!parent_struct.children) {
                        parent_struct.children = new Array<RepStructElementData>()
                    }
                    parent_struct.children.push(struct_element)
                    added_list.push(item)
                    not_added_list.splice(i, 1)
                    i--;
                }
            }
        }
        this.parsed_rep_struct = struct
        return new Array<GkillError>()
    }
    async parse_device_struct(): Promise<Array<GkillError>> {
        const added_list = new Array<DeviceStruct>()
        const not_added_list: Array<DeviceStruct> = this.device_struct.concat()
        const struct = new DeviceStructElementData()
        struct.children = new Array<DeviceStructElementData>()

        // 親を持たないものをルートに追加する
        for (let i = 0; i < not_added_list.length; i++) {
            let item = not_added_list[i]
            if (!item.parent_folder_id || item.parent_folder_id === "") {
                const struct_element = new DeviceStructElementData()
                struct_element.check_when_inited = item.check_when_inited
                struct_element.id = item.id
                struct_element.indeterminate = false
                struct_element.is_checked = item.check_when_inited
                struct_element.key = item.device_name
                struct_element.device_name = item.device_name

                struct.children.push(struct_element)
                added_list.push(item)
                not_added_list.splice(i, 1)
                i--;
            }
        }
        // 再帰呼び出し用
        let walk = (struct: DeviceStructElementData, target_id: string): DeviceStructElementData | null => {
            throw new Error('Not implemented')
        }
        // structを潜ってIDが一致するものを取得する
        walk = (struct: DeviceStructElementData, target_id: string): DeviceStructElementData | null => {
            if (struct.id === target_id) {
                return struct
            }
            if (!struct.children) {
                return null
            }
            let target: DeviceStructElementData | null = null
            for (let i = 0; i < struct.children.length; i++) {
                const child = struct.children[i]
                target = walk(child, target_id)
                if (target) {
                    break
                }
            }
            return target
        }
        while (not_added_list.length !== 0) {
            for (let i = 0; i < not_added_list.length; i++) {
                let item = not_added_list[i]
                const parent_struct = walk(struct, item.parent_folder_id)
                if (parent_struct) {
                    const struct_element = new DeviceStructElementData()
                    struct_element.check_when_inited = item.check_when_inited
                    struct_element.id = item.id
                    struct_element.indeterminate = false
                    struct_element.is_checked = item.check_when_inited
                    struct_element.key = item.device_name
                    struct_element.device_name = item.device_name

                    if (!parent_struct.children) {
                        parent_struct.children = new Array<DeviceStructElementData>()
                    }
                    parent_struct.children.push(struct_element)
                    added_list.push(item)
                    not_added_list.splice(i, 1)
                    i--;
                }
            }
        }
        this.parsed_device_struct = struct
        return new Array<GkillError>()
    }
    async parse_rep_type_struct(): Promise<Array<GkillError>> {
        const added_list = new Array<RepTypeStruct>()
        const not_added_list: Array<RepTypeStruct> = this.rep_type_struct.concat()
        const struct = new RepTypeStructElementData()
        struct.children = new Array<RepTypeStructElementData>()

        // 親を持たないものをルートに追加する
        for (let i = 0; i < not_added_list.length; i++) {
            let item = not_added_list[i]
            if (!item.parent_folder_id || item.parent_folder_id === "") {
                const struct_element = new RepTypeStructElementData()
                struct_element.check_when_inited = item.check_when_inited
                struct_element.id = item.id
                struct_element.indeterminate = false
                struct_element.is_checked = item.check_when_inited
                struct_element.key = item.rep_type_name
                struct_element.rep_type_name = item.rep_type_name

                struct.children.push(struct_element)
                added_list.push(item)
                not_added_list.splice(i, 1)
                i--;
            }
        }
        // 再帰呼び出し用
        let walk = (struct: RepTypeStructElementData, target_id: string): RepTypeStructElementData | null => {
            throw new Error('Not implemented')
        }
        // structを潜ってIDが一致するものを取得する
        walk = (struct: RepTypeStructElementData, target_id: string): RepTypeStructElementData | null => {
            if (struct.id === target_id) {
                return struct
            }
            if (!struct.children) {
                return null
            }
            let target: RepTypeStructElementData | null = null
            for (let i = 0; i < struct.children.length; i++) {
                const child = struct.children[i]
                target = walk(child, target_id)
                if (target) {
                    break
                }
            }
            return target
        }
        while (not_added_list.length !== 0) {
            for (let i = 0; i < not_added_list.length; i++) {
                let item = not_added_list[i]
                const parent_struct = walk(struct, item.parent_folder_id)
                if (parent_struct) {
                    const struct_element = new RepTypeStructElementData()
                    struct_element.check_when_inited = item.check_when_inited
                    struct_element.id = item.id
                    struct_element.indeterminate = false
                    struct_element.is_checked = item.check_when_inited
                    struct_element.key = item.rep_type_name
                    struct_element.rep_type_name = item.rep_type_name

                    if (!parent_struct.children) {
                        parent_struct.children = new Array<RepTypeStructElementData>()
                    }
                    parent_struct.children.push(struct_element)
                    added_list.push(item)
                    not_added_list.splice(i, 1)
                    i--;
                }
            }
        }
        this.parsed_rep_type_struct = struct
        return new Array<GkillError>()

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
        this.parsed_kftl_template = new KFTLTemplateElementData()
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
