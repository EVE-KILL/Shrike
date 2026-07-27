/**
 * /llms.txt — the answer.ai-style AI-readable site guide.
 *
 * Serves the same body as /mcp.txt so crawlers that respect the /llms.txt
 * convention (Perplexity, some web-browsing LLMs) get the full usage
 * reference for the MCP server without having to know the /mcp.txt name.
 * `buildMcpGuide` comes in via Nitro auto-import from server/utils/.
 */
export default defineEventHandler(async (event) => {
    const config = useRuntimeConfig()
    const mcpBase = config.public.publicMcpUrl || 'https://mcp.eve-kill.com'
    const body = await buildMcpGuide(mcpBase)

    setResponseHeaders(event, {
        'content-type': 'text/plain; charset=utf-8',
        'cache-control': 'public, max-age=300, s-maxage=300',
    })
    return body
})
