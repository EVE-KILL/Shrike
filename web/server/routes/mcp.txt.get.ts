/**
 * /mcp.txt — machine-readable usage guide for the EVE-KILL MCP server.
 * Shares its body with /llms.txt via buildMcpGuide (Nitro auto-imports
 * from server/utils/).
 */
export default defineEventHandler(async (event) => {
    const config = useRuntimeConfig()
    const mcpBase = config.public.publicMcpUrl || 'https://eve-kill.com/api'
    const body = await buildMcpGuide(mcpBase)

    setResponseHeaders(event, {
        'content-type': 'text/plain; charset=utf-8',
        // Short cache — tool list can change after an mcp deploy.
        'cache-control': 'public, max-age=300, s-maxage=300',
    })
    return body
})
