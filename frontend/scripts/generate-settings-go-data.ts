import {MY_COMMENT_TARGETS} from '../src/utils/myCommentTarget'
import {DEFAULT_THEME,CSS_VAR_MAP,THEME_PRESETS} from '../src/utils/themeData'
import {SETTINGS_METADATA} from '../src/models/settings'
import {writeChanged} from './io'
await writeChanged(new URL('../../internal/frontend/data/settings.json',import.meta.url),JSON.stringify({defaults:DEFAULT_THEME,variables:CSS_VAR_MAP,presets:THEME_PRESETS,colors:SETTINGS_METADATA.colors,commentTargets:MY_COMMENT_TARGETS,tabs:SETTINGS_METADATA.tabs},null,2)+'\n')
