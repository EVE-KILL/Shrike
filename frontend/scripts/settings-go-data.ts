// Read the original settings definitions, retaining their order and labels.
// Ordinary Go builds consume the checked-in JSON and browser module.
export {DEFAULT_THEME, CSS_VAR_MAP, THEME_PRESETS} from '../src/utils/themeData'
export * from '../src/utils/adminFormat'
export {formatDateTime} from '../src/utils/formatters'
export {extractFetchError} from '../src/utils/fetchError'

export {DEFAULT_NAVBAR} from '../src/models/domainConfig'
export * from '../src/utils/domainEditor'
