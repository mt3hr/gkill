'use strict'

export class KFTLTemplateElementData {

    id: string

    title: string

    template: string

    children: Array<KFTLTemplateElementData>

    constructor() {
        this.id = ""
        this.title = ""
        this.template = ""
        this.children = new Array<KFTLTemplateElementData>()
    }

}


