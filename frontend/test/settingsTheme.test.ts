import {readFile} from 'node:fs/promises'
import {expect, test} from 'vitest'
import {DEFAULT_THEME, CSS_VAR_MAP, THEME_PRESETS} from '../src/utils/themeData'
import {DEFAULT_THEME as bundledDefaults, THEME_PRESETS as bundledPresets} from '../src/browser/settings-data'
import {minimalTheme, normaliseTheme, detectThemePreset} from '../src/browser/settings-theme'

test('settings use every original preset, default and color mapping',async()=>{
  const data=JSON.parse(await readFile(new URL('../../internal/frontend/data/settings.json',import.meta.url),'utf8'))
  expect(data.defaults).toEqual(DEFAULT_THEME);expect(data.variables).toEqual(CSS_VAR_MAP);expect(data.presets).toEqual(THEME_PRESETS)
  expect(bundledDefaults).toEqual(DEFAULT_THEME);expect(bundledPresets).toEqual(THEME_PRESETS)
  for(const preset of THEME_PRESETS)expect(detectThemePreset(preset.overrides)).toBe(preset.id)
})
test('save snapshots retain newer local edits as dirty',()=>{
  const sent={brandPrimary:'#abcdef',colorError:DEFAULT_THEME.colorError}
  const snapshot=normaliseTheme(sent)
  expect(minimalTheme(sent)).toEqual({brandPrimary:'#abcdef'})
  expect(normaliseTheme({brandPrimary:'#fedcba'})).not.toBe(snapshot)
  expect(normaliseTheme({colorError:DEFAULT_THEME.colorError,brandPrimary:'#abcdef'})).toBe(snapshot)
  expect(minimalTheme({unexpectedColor:'#000000',brandPrimary:DEFAULT_THEME.brandPrimary})).toEqual({})
})
