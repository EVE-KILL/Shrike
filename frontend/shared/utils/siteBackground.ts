/** Select lighter built-in artwork; custom URLs and the full-size viewer keep their originals. */
export function siteBackgroundCSS(source: string, original = false): string {
    const match = /^\/backgrounds\/(bg[1-8])\.webp$/.exec(source)
    if (!match || original) return `url(${JSON.stringify(source)})`
    const base = `/backgrounds/optimized-v1/${match[1]}`
    return `image-set(url("${base}.avif") type("image/avif"), url("${base}.webp") type("image/webp"))`
}
