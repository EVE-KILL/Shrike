// @vitest-environment happy-dom
import {afterEach,expect,test,vi} from 'vitest'
async function apply(){vi.resetModules();await import('../src/browser/preferences')}
afterEach(()=>{document.cookie='ek_theme=; Max-Age=0';document.cookie='siteBackground=; Max-Age=0';delete document.documentElement.dataset.domainTheme;delete document.documentElement.dataset.domainBackground;document.documentElement.removeAttribute('style');vi.unstubAllGlobals()})
test('domain defaults apply before paint while explicit personal preferences win',async()=>{
 vi.stubGlobal('CSS',{supports:(_kind:string,value:string)=>/^#[0-9a-f]{6}$/.test(value)})
 const root=document.documentElement;root.dataset.domainTheme=JSON.stringify({defaultThemePreset:'triglavian',defaultThemeOverrides:{brandPrimary:'#b91c1c',textPrimary:'url(https://unsafe.example)'}});root.dataset.domainBackground='/images/domains/background/27'
 await apply();expect(root.style.getPropertyValue('--color-brand-primary')).toBe('#b91c1c');expect(root.style.getPropertyValue('--site-bg')).toContain('/images/domains/background/27');expect(root.style.getPropertyValue('--color-text-primary')).toBe('')
 document.cookie='ek_theme='+encodeURIComponent(JSON.stringify({brandPrimary:'#123456'}));document.cookie='siteBackground='+encodeURIComponent('/backgrounds/bg4.webp');await apply()
 expect(root.style.getPropertyValue('--color-brand-primary')).toBe('#123456');expect(root.style.getPropertyValue('--site-bg')).toContain('/assets/backgrounds/bg4.avif')
})
test('unapproved background URLs cannot become CSS resources',async()=>{
 document.documentElement.dataset.domainBackground='https://unsafe.example/background';await apply();expect(document.documentElement.style.getPropertyValue('--site-bg')).toBe('')
})
