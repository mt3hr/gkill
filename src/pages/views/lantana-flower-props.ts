'use strict';

import type { LantanaFlowerState } from "@/classes/lantana/lantana-flower-state";
import type { GkillPropsBase } from "./gkill-props-base";

export interface LantanaFlowerProps extends GkillPropsBase {
    state: LantanaFlowerState;
    editable: boolean;
}