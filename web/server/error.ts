import {
    getRequestHost,
    getRequestURL,
    send,
    setResponseHeaders,
    setResponseStatus,
} from 'h3'
import { defineNitroErrorHandler } from 'nitropack/runtime'
import { logRequestError } from './utils/errorLogging'
import { isCanonicalSeoHost } from './utils/seoHost'

export default defineNitroErrorHandler(async (error, event, options) => {
    const url = getRequestURL(event, {
        xForwardedHost: true,
        xForwardedProto: true,
    })

    if (error.unhandled || error.fatal) {
        logRequestError(error, event.method, url.href)
    }

    const response = await options.defaultHandler(error, event, { silent: true })
    if (!event.node?.res.headersSent) {
        setResponseHeaders(event, response.headers)
        if (!isCanonicalSeoHost(getRequestHost(event))) {
            setResponseHeaders(event, { 'x-robots-tag': 'noindex, nofollow' })
        }
    }
    setResponseStatus(event, response.status, response.statusText)

    return send(
        event,
        typeof response.body === 'string'
            ? response.body
            : JSON.stringify(response.body, null, 2),
    )
})
