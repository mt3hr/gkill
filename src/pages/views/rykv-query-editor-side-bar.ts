// ˅
'use strict';

import { CalendarQuery } from './calendar-query';
import { KeywordQuery } from './keyword-query';
import { MapQuery } from './map-query';
import { RepQuery } from './rep-query';
import { SidebarHeader } from './sidebar-header';
import { TagQuery } from './tag-query';
import { TimeIsQuery } from './time-is-query';
import { rykvQueryEditorSidebarEmits } from './rykv-query-editor-sidebar-emits';
import { rykvQueryEditorSidebarProps } from './rykv-query-editor-sidebar-props';

// ˄

export class rykvQueryEditorSideBar {
    // ˅
    
    // ˄

    private props: rykvQueryEditorSidebarProps;

    private emits: rykvQueryEditorSidebarEmits;

    private header: SidebarHeader;

    private keyword_query: KeywordQuery;

    private timeis_query: TimeIsQuery;

    private rep_query: RepQuery;

    private tag_query: TagQuery;

    private calendar_query: CalendarQuery;

    private map_query: MapQuery;

    async generate_query(): Promise<FindKyouQuery> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
