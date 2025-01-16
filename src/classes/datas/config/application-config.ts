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
import { GkillAPI } from '@/classes/api/gkill-api'
import { GetAllRepNamesRequest } from '@/classes/api/req_res/get-all-rep-names-request'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import { GetAllTagNamesRequest } from '@/classes/api/req_res/get-all-tag-names-request'
import { MiBoardStructElementData } from './mi-board-struct-element-data'
import { MiBoardStruct } from './mi-board-struct'
import { GetMiBoardRequest } from '@/classes/api/req_res/get-mi-board-request'

export class ApplicationConfig {
    is_loaded: boolean
    user_id: string
    device: string
    enable_browser_cache: boolean
    google_map_api_key: string
    rykv_image_list_column_number: number
    rykv_hot_reload: boolean
    mi_default_board: string
    account_is_admin: boolean
    parsed_kftl_template: KFTLTemplateElementData
    parsed_tag_struct: TagStructElementData
    parsed_rep_struct: RepStructElementData
    parsed_device_struct: DeviceStructElementData
    parsed_rep_type_struct: RepTypeStructElementData
    parsed_mi_boad_struct: MiBoardStructElementData
    tag_struct: Array<TagStruct>
    rep_struct: Array<RepStruct>
    device_struct: Array<DeviceStruct>
    rep_type_struct: Array<RepTypeStruct>
    kftl_template_struct: Array<KFTLTemplateStruct>
    mi_board_struct: Array<MiBoardStruct>
    async parse_template_and_struct(): Promise<Array<GkillError>> {
        const awaitPromises = new Array<Promise<any>>()
        awaitPromises.push(this.parse_tag_struct())
        awaitPromises.push(this.parse_rep_struct())
        awaitPromises.push(this.parse_device_struct())
        awaitPromises.push(this.parse_rep_type_struct())
        awaitPromises.push(this.parse_kftl_template_struct())
        awaitPromises.push(this.parse_mi_board_struct())
        return Promise.all(awaitPromises).then((errors_list) => {
            const errors = new Array<GkillError>()
            errors_list.forEach(e => {
                errors.push(...e)
            })
            return errors
        })
    }
    clone(): ApplicationConfig {
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
        application_config.kftl_template_struct = this.kftl_template_struct
        application_config.account_is_admin = this.account_is_admin
        application_config.mi_board_struct = this.mi_board_struct
        return application_config
    }
    async load_all(): Promise<Array<GkillError>> {
        const errors = Array<GkillError>()
        errors.concat(await this.append_not_found_reps())
        errors.concat(await this.append_not_found_devices())
        errors.concat(await this.append_not_found_rep_types())
        errors.concat(await this.append_not_found_tags())
        errors.concat(await this.append_not_found_mi_boards())
        errors.concat(await this.append_no_devices())
        errors.concat(await this.append_no_tags())
        errors.concat(await this.append_all_mi_board())
        errors.concat(await this.parse_template_and_struct())
        return errors
    }
    async append_not_found_reps(): Promise<Array<GkillError>> {
        const req = new GetAllRepNamesRequest()
        
        const res = await GkillAPI.get_gkill_api().get_all_rep_names(req)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }

        const gkill_info_req = new GetGkillInfoRequest()
        const gkill_info_res = await GkillAPI.get_gkill_api().get_gkill_info(gkill_info_req)
        if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
            return gkill_info_res.errors
        }

        const not_found = new Set<string>()
        res.rep_names.forEach(rep_name => {
            let exist = false
            this.rep_struct.forEach(rep => {
                if (rep.rep_name === rep_name) {
                    exist = true
                }
            })
            if (!exist) {
                not_found.add(rep_name)
            }
        })

        let i = 0
        not_found.forEach(rep_name => {
            const rep_struct = new RepStruct()
            rep_struct.check_when_inited = true
            rep_struct.device = gkill_info_res.device
            rep_struct.id = GkillAPI.get_gkill_api().generate_uuid()
            rep_struct.ignore_check_rep_rykv = false
            rep_struct.parent_folder_id = null
            rep_struct.rep_name = rep_name
            rep_struct.seq = 1000 + i++
            rep_struct.user_id = gkill_info_res.user_id
            this.rep_struct.push(rep_struct)
        })
        return new Array<GkillError>()
    }
    async append_not_found_tags(): Promise<Array<GkillError>> {
        const req = new GetAllTagNamesRequest()
        
        const res = await GkillAPI.get_gkill_api().get_all_tag_names(req)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }

        const gkill_info_req = new GetGkillInfoRequest()
        const gkill_info_res = await GkillAPI.get_gkill_api().get_gkill_info(gkill_info_req)
        if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
            return gkill_info_res.errors
        }

        const not_found = new Set<string>()
        res.tag_names.forEach(tag_name => {
            let exist = false
            this.tag_struct.forEach(tag => {
                if (tag.tag_name === tag_name) {
                    exist = true
                }
            })
            if (!exist) {
                not_found.add(tag_name)
            }
        })

        let i = 0
        not_found.forEach(tag_name => {
            const tag_struct = new TagStruct()
            tag_struct.check_when_inited = true
            tag_struct.device = gkill_info_res.device
            tag_struct.id = GkillAPI.get_gkill_api().generate_uuid()
            tag_struct.is_force_hide = false
            tag_struct.parent_folder_id = null
            tag_struct.tag_name = tag_name
            tag_struct.seq = 1000 + i++
            tag_struct.user_id = gkill_info_res.user_id
            this.tag_struct.push(tag_struct)
        })
        return new Array<GkillError>()
    }

    async append_not_found_mi_boards(): Promise<Array<GkillError>> {
        const req = new GetMiBoardRequest()
        
        const res = await GkillAPI.get_gkill_api().get_mi_board_list(req)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }

        const gkill_info_req = new GetGkillInfoRequest()
        const gkill_info_res = await GkillAPI.get_gkill_api().get_gkill_info(gkill_info_req)
        if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
            return gkill_info_res.errors
        }

        const not_found = new Set<string>()
        res.boards.forEach(board => {
            let exist = false
            this.mi_board_struct.forEach(mi_board => {
                if (mi_board.board_name === board) {
                    exist = true
                }
            })
            if (!exist) {
                not_found.add(board)
            }
        })

        let i = 0
        not_found.forEach(board_name => {
            const board_struct = new MiBoardStruct()
            board_struct.check_when_inited = true
            board_struct.device = gkill_info_res.device
            board_struct.id = GkillAPI.get_gkill_api().generate_uuid()
            board_struct.parent_folder_id = null
            board_struct.board_name = board_name
            board_struct.seq = 1000 + i++
            board_struct.user_id = gkill_info_res.user_id
            this.mi_board_struct.push(board_struct)
        })
        return new Array<GkillError>()
    }

    async append_no_devices(): Promise<Array<GkillError>> {
        const gkill_info_req = new GetGkillInfoRequest()
        const gkill_info_res = await GkillAPI.get_gkill_api().get_gkill_info(gkill_info_req)
        if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
            return gkill_info_res.errors
        }

        let exist = false
        this.device_struct.forEach(device => {
            if (device.device_name === "なし") {
                exist = true
            }
        })

        if (!exist) {
            const device_struct = new DeviceStruct()
            device_struct.check_when_inited = true
            device_struct.device = gkill_info_res.device
            device_struct.id = GkillAPI.get_gkill_api().generate_uuid()
            device_struct.device_name = "なし"
            device_struct.parent_folder_id = null
            device_struct.seq = -1000
            device_struct.user_id = gkill_info_res.user_id
            this.device_struct.push(device_struct)
        }
        return new Array<GkillError>()
    }

    async append_not_found_devices(): Promise<Array<GkillError>> {
        const req = new GetAllRepNamesRequest()
        
        const res = await GkillAPI.get_gkill_api().get_all_rep_names(req)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }

        const gkill_info_req = new GetGkillInfoRequest()
        const gkill_info_res = await GkillAPI.get_gkill_api().get_gkill_info(gkill_info_req)
        if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
            return gkill_info_res.errors
        }

        const not_found = new Set<string>()
        res.rep_names.forEach(rep_name => {
            let exist = false
            const device_name = this.get_device_from_rep_name(rep_name)
            this.device_struct.forEach(device => {
                if (device.device_name === device_name) {
                    exist = true
                }
            })
            if (!exist && device_name) {
                not_found.add(device_name)
            }
        })

        let i = 0
        not_found.forEach(device_name => {
            const device_struct = new DeviceStruct()
            device_struct.check_when_inited = true
            device_struct.device = gkill_info_res.device
            device_struct.id = GkillAPI.get_gkill_api().generate_uuid()
            device_struct.device_name = device_name
            device_struct.parent_folder_id = null
            device_struct.seq = 1000 + i++
            device_struct.user_id = gkill_info_res.user_id
            this.device_struct.push(device_struct)
        })
        return new Array<GkillError>()
    }
    async append_not_found_rep_types(): Promise<Array<GkillError>> {
        const req = new GetAllRepNamesRequest()
        
        const res = await GkillAPI.get_gkill_api().get_all_rep_names(req)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }

        const gkill_info_req = new GetGkillInfoRequest()
        const gkill_info_res = await GkillAPI.get_gkill_api().get_gkill_info(gkill_info_req)
        if (res.errors && res.errors.length !== 0) {
            return gkill_info_res.errors
        }

        const not_found = new Set<string>()
        res.rep_names.forEach(rep_name => {
            let exist = false
            const rep_type_name = this.get_rep_type_from_rep_name(rep_name)
            this.rep_type_struct.forEach(rep_type => {
                if (rep_type.rep_type_name === rep_type_name) {
                    exist = true
                }
            })
            if (!exist && rep_type_name) {
                not_found.add(rep_type_name)
            }
        })

        let i = 0
        not_found.forEach(rep_type => {
            const rep_type_struct = new RepTypeStruct()
            rep_type_struct.check_when_inited = true
            rep_type_struct.device = gkill_info_res.device
            rep_type_struct.id = GkillAPI.get_gkill_api().generate_uuid()
            rep_type_struct.rep_type_name = rep_type
            rep_type_struct.parent_folder_id = null
            rep_type_struct.seq = 1000 + i++
            rep_type_struct.user_id = gkill_info_res.user_id
            this.rep_type_struct.push(rep_type_struct)
        })
        return new Array<GkillError>()
    }
    async parse_kftl_template_struct(): Promise<Array<GkillError>> {
        const added_list = new Array<KFTLTemplateStruct>()
        const not_added_list: Array<KFTLTemplateStruct> = this.kftl_template_struct.concat()
        const struct = new KFTLTemplateElementData()
        struct.children = new Array<KFTLTemplateElementData>()

        // 親を持たないものをルートに追加する
        for (let i = 0; i < not_added_list.length; i++) {
            const item = not_added_list[i]
            if (!item.parent_folder_id || item.parent_folder_id === "") {
                const struct_element = new KFTLTemplateElementData()
                struct_element.id = item.id
                struct_element.title = item.title
                struct_element.key = item.title
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
        let walk = (_struct: KFTLTemplateElementData, _target_id: string | null): KFTLTemplateElementData | null => {
            throw new Error('Not implemented')
        }
        // structを潜ってIDが一致するものを取得する
        walk = (struct: KFTLTemplateElementData, target_id: string | null): KFTLTemplateElementData | null => {
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
                const item = not_added_list[i]
                const parent_struct = walk(struct, item.parent_folder_id)
                if (parent_struct) {
                    const struct_element = new KFTLTemplateElementData()
                    struct_element.id = item.id
                    struct_element.title = item.title
                    struct_element.key = item.title
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
            const item = not_added_list[i]
            if (!item.parent_folder_id || item.parent_folder_id === "") {
                const struct_element = new TagStructElementData()
                struct_element.seq_in_parent = item.seq.valueOf()
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
        let walk = (_struct: TagStructElementData, _target_id: string | null): TagStructElementData | null => {
            throw new Error('Not implemented')
        }
        // structを潜ってIDが一致するものを取得する
        walk = (struct: TagStructElementData, target_id: string | null): TagStructElementData | null => {
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
                const item = not_added_list[i]
                const parent_struct = walk(struct, item.parent_folder_id)
                if (parent_struct) {
                    const struct_element = new TagStructElementData()
                    struct_element.seq_in_parent = item.seq.valueOf()
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
        let sort = (_struct: TagStructElementData): void => { }
        sort = (struct: TagStructElementData): void => {
            if (struct.children) {
                struct.children.sort((a, b): number => a.seq_in_parent - b.seq_in_parent)
                for (let i = 0; i < struct.children.length; i++) {
                    sort(struct.children[i])
                }
            }
        }
        sort(struct)
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
            const item = not_added_list[i]
            if (!item.parent_folder_id || item.parent_folder_id === "") {
                const struct_element = new RepStructElementData()
                struct_element.seq_in_parent = item.seq.valueOf()
                struct_element.check_when_inited = item.check_when_inited
                struct_element.id = item.id
                struct_element.indeterminate = false
                struct_element.is_checked = item.check_when_inited
                struct_element.ignore_check_rep_rykv = item.ignore_check_rep_rykv
                struct_element.key = item.rep_name
                struct_element.rep_name = item.rep_name

                struct.children.push(struct_element)
                added_list.push(item)
                not_added_list.splice(i, 1)
                i--;
            }
        }
        // 再帰呼び出し用
        let walk = (_struct: RepStructElementData, _target_id: string | null): RepStructElementData | null => {
            throw new Error('Not implemented')
        }
        // structを潜ってIDが一致するものを取得する
        walk = (struct: RepStructElementData, target_id: string | null): RepStructElementData | null => {
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
                const item = not_added_list[i]
                const parent_struct = walk(struct, item.parent_folder_id)
                if (parent_struct) {
                    const struct_element = new RepStructElementData()
                    struct_element.seq_in_parent = item.seq.valueOf()
                    struct_element.check_when_inited = item.check_when_inited
                    struct_element.id = item.id
                    struct_element.indeterminate = false
                    struct_element.is_checked = item.check_when_inited
                    struct_element.ignore_check_rep_rykv = item.ignore_check_rep_rykv
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
        let sort = (_struct: RepStructElementData): void => { }
        sort = (struct: RepStructElementData): void => {
            if (struct.children) {
                struct.children.sort((a, b): number => a.seq_in_parent - b.seq_in_parent)
                for (let i = 0; i < struct.children.length; i++) {
                    sort(struct.children[i])
                }
            }
        }
        sort(struct)
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
            const item = not_added_list[i]
            if (!item.parent_folder_id || item.parent_folder_id === "") {
                const struct_element = new DeviceStructElementData()
                struct_element.seq_in_parent = item.seq.valueOf()
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
        let walk = (_struct: DeviceStructElementData, _target_id: string | null): DeviceStructElementData | null => {
            throw new Error('Not implemented')
        }
        // structを潜ってIDが一致するものを取得する
        walk = (struct: DeviceStructElementData, target_id: string | null): DeviceStructElementData | null => {
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
                const item = not_added_list[i]
                const parent_struct = walk(struct, item.parent_folder_id)
                if (parent_struct) {
                    const struct_element = new DeviceStructElementData()
                    struct_element.seq_in_parent = item.seq.valueOf()
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
        let sort = (_struct: DeviceStructElementData): void => { }
        sort = (struct: DeviceStructElementData): void => {
            if (struct.children) {
                struct.children.sort((a, b): number => a.seq_in_parent - b.seq_in_parent)
                for (let i = 0; i < struct.children.length; i++) {
                    sort(struct.children[i])
                }
            }
        }
        sort(struct)
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
            const item = not_added_list[i]
            if (!item.parent_folder_id || item.parent_folder_id === "") {
                const struct_element = new RepTypeStructElementData()
                struct_element.seq_in_parent = item.seq.valueOf()
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
        let walk = (_struct: RepTypeStructElementData, _target_id: string | null): RepTypeStructElementData | null => {
            throw new Error('Not implemented')
        }
        // structを潜ってIDが一致するものを取得する
        walk = (struct: RepTypeStructElementData, target_id: string | null): RepTypeStructElementData | null => {
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
                const item = not_added_list[i]
                const parent_struct = walk(struct, item.parent_folder_id)
                if (parent_struct) {
                    const struct_element = new RepTypeStructElementData()
                    struct_element.seq_in_parent = item.seq.valueOf()
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
        let sort = (_struct: RepTypeStructElementData): void => { }
        sort = (struct: RepTypeStructElementData): void => {
            if (struct.children) {
                struct.children.sort((a, b): number => a.seq_in_parent - b.seq_in_parent)
                for (let i = 0; i < struct.children.length; i++) {
                    sort(struct.children[i])
                }
            }
        }
        sort(struct)
        this.parsed_rep_type_struct = struct
        return new Array<GkillError>()

    }

    async parse_mi_board_struct(): Promise<Array<GkillError>> {
        const added_list = new Array<MiBoardStruct>()
        const not_added_list: Array<MiBoardStruct> = this.mi_board_struct.concat()
        const struct = new MiBoardStructElementData()
        struct.children = new Array<MiBoardStructElementData>()

        // 親を持たないものをルートに追加する
        for (let i = 0; i < not_added_list.length; i++) {
            const item = not_added_list[i]
            if (!item.parent_folder_id || item.parent_folder_id === "") {
                const struct_element = new MiBoardStructElementData()
                struct_element.seq_in_parent = item.seq.valueOf()
                struct_element.check_when_inited = item.check_when_inited
                struct_element.id = item.id
                struct_element.indeterminate = false
                struct_element.is_checked = item.check_when_inited
                struct_element.key = item.board_name

                struct.children.push(struct_element)
                added_list.push(item)
                not_added_list.splice(i, 1)
                i--;
            }
        }
        // 再帰呼び出し用
        let walk = (_struct: MiBoardStructElementData, _target_id: string | null): MiBoardStructElementData | null => {
            throw new Error('Not implemented')
        }
        // structを潜ってIDが一致するものを取得する
        walk = (struct: MiBoardStructElementData, target_id: string | null): MiBoardStructElementData | null => {
            if (struct.id === target_id) {
                return struct
            }
            if (!struct.children) {
                return null
            }
            let target: MiBoardStructElementData | null = null
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
                const item = not_added_list[i]
                const parent_struct = walk(struct, item.parent_folder_id)
                if (parent_struct) {
                    const struct_element = new MiBoardStructElementData()
                    struct_element.seq_in_parent = item.seq.valueOf()
                    struct_element.check_when_inited = item.check_when_inited
                    struct_element.id = item.id
                    struct_element.indeterminate = false
                    struct_element.is_checked = item.check_when_inited
                    struct_element.key = item.board_name
                    struct_element.board_name = item.board_name

                    if (!parent_struct.children) {
                        parent_struct.children = new Array<MiBoardStructElementData>()
                    }
                    parent_struct.children.push(struct_element)
                    added_list.push(item)
                    not_added_list.splice(i, 1)
                    i--;
                }
            }
        }
        let sort = (_struct: MiBoardStructElementData): void => { }
        sort = (struct: MiBoardStructElementData): void => {
            if (struct.children) {
                struct.children.sort((a, b): number => a.seq_in_parent - b.seq_in_parent)
                for (let i = 0; i < struct.children.length; i++) {
                    sort(struct.children[i])
                }
            }
        }
        sort(struct)
        this.parsed_mi_boad_struct = struct
        return new Array<GkillError>()
    }

    async append_no_tags(): Promise<Array<GkillError>> {
        const gkill_info_req = new GetGkillInfoRequest()
        const gkill_info_res = await GkillAPI.get_gkill_api().get_gkill_info(gkill_info_req)
        if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
            return gkill_info_res.errors
        }

        let exist = false
        this.tag_struct.forEach(tag => {
            if (tag.tag_name === "no tags") {
                exist = true
            }
        })

        if (!exist) {
            const tag_struct = new TagStruct()
            tag_struct.check_when_inited = true
            tag_struct.device = gkill_info_res.device
            tag_struct.id = GkillAPI.get_gkill_api().generate_uuid()
            tag_struct.is_force_hide = false
            tag_struct.parent_folder_id = null
            tag_struct.tag_name = "no tags"
            tag_struct.seq = -1000
            tag_struct.user_id = gkill_info_res.user_id
            this.tag_struct.unshift(tag_struct)
        }
        return new Array<GkillError>()
    }

    async append_all_mi_board(): Promise<Array<GkillError>> {
        const gkill_info_req = new GetGkillInfoRequest()
        const gkill_info_res = await GkillAPI.get_gkill_api().get_gkill_info(gkill_info_req)
        if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
            return gkill_info_res.errors
        }

        let exist = false
        this.mi_board_struct.forEach(board => {
            if (board.board_name === "すべて") {
                exist = true
            }
        })

        if (!exist) {
            const board_struct = new MiBoardStruct()
            board_struct.check_when_inited = true
            board_struct.device = gkill_info_res.device
            board_struct.id = GkillAPI.get_gkill_api().generate_uuid()
            board_struct.parent_folder_id = null
            board_struct.board_name = "すべて"
            board_struct.seq = -1000
            board_struct.user_id = gkill_info_res.user_id
            this.mi_board_struct.unshift(board_struct)
        }
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
        this.account_is_admin = false
        this.parsed_kftl_template = new KFTLTemplateElementData()
        this.parsed_tag_struct = new TagStructElementData()
        this.parsed_rep_struct = new RepStructElementData()
        this.parsed_device_struct = new DeviceStructElementData()
        this.parsed_rep_type_struct = new RepTypeStructElementData()
        this.tag_struct = new Array<TagStruct>()
        this.rep_struct = new Array<RepStruct>()
        this.device_struct = new Array<DeviceStruct>()
        this.rep_type_struct = new Array<RepTypeStruct>()
        this.kftl_template_struct = new Array<KFTLTemplateStruct>()
        this.parsed_mi_boad_struct = new MiBoardStructElementData()
        this.mi_board_struct = new Array<MiBoardStruct>()
    }


    get_device_from_rep_name(rep_name: string): string | null {
        const splited = rep_name.split("_")
        if (splited.length !== 3) {
            return null
        }
        if (splited.length < 2) {
            return null
        }
        return splited[1]
    }

    get_rep_type_from_rep_name(rep_name: string): string | null {
        const splited = rep_name.split("_")
        if (splited.length !== 3) {
            if (!rep_name || rep_name === "") {
                return null
            }
            return rep_name
        }
        return splited[0]
    }
}
