'use strict';

import type { Kyou } from "@/classes/datas/kyou";
import type { GkillPropsBase } from "../views/gkill-props-base";

export interface DecideRelatedTimeUploadedFileDialogProps extends GkillPropsBase {
    app_content_height: Number;
    app_content_width: Number;
    uploaded_kyous: Array<Kyou>;
}
