# Cloudflare image delivery

Status: deferred until `eve-kill.com` is served by the Go application in
production.

## Goal

Keep every public image on the canonical URL:

```text
https://eve-kill.com/images/...
```

Cloudflare should serve images already present in the public
`evekill-media` Backblaze B2 bucket. The Go image server should only receive a
request when the final image does not exist yet. On that cold request, Go
fetches or generates the image, stores the exact response in B2, and returns it
to the caller.

This avoids:

- routing ordinary B2 downloads through Kubernetes;
- paying for a Cloudflare Worker on every image request;
- adding RustFS and more persistent storage to the cluster;
- exposing a second image URL to the frontend.

## Why a Snippet

`eve-kill.com` is on Cloudflare Pro. Pro includes Snippets without additional
invocation charges and permits two subrequests per invocation. The image
fallback uses exactly two subrequests on a cold miss:

1. fetch the final object from B2;
2. continue the original request to the Go origin if B2 does not return it.

A B2 hit only uses the first subrequest. The B2 request must use its final
download URL without a redirect because every redirect counts as another
subrequest.

The Snippet performs routing and caching only. All image transformation,
validation, upstream fetching, and persistence remains in Go.

## Request flow

```text
Browser: GET https://eve-kill.com/images/...
                         |
                         v
                 Cloudflare Snippet
                         |
            deterministic delivery object key
                         |
                         v
       GET public B2 delivery/v1/<digest>
                |                    |
              2xx                  missing/error
                |                    |
                v                    v
       return B2 response       fetch(request)
                                      |
                                      v
                              Go image handler
                                      |
                       fetch/generate/process image
                                      |
                    PUT exact response into B2 delivery key
                                      |
                                      v
                              return response
```

The browser always sees `eve-kill.com/images/...`; the B2 download URL is an
internal Snippet subrequest.

## Storage layout

Existing source assets remain in their current prefixes:

```text
static/regions/
static/systems/
static/constellations/
static/ui/
types/dust514/
types/overlays/
types/{typeID}_64.png
types/{typeID}_bpc_64.png
types/{typeID}_512.jpg
types/manifest.json
oldcharacters/
```

Responses that are ready to send to a browser should use a separate namespace:

```text
delivery/v1/<sha256>
```

The digest must be calculated identically in Go and JavaScript from a
canonical representation containing:

- the image route and path parameters;
- only query parameters that affect the output, in a stable order;
- the negotiated output format, including WebP negotiation;
- a delivery schema version.

Do not use the raw query string without normalization. Irrelevant or reordered
query parameters must not create duplicate objects, and `Accept` negotiation
must not let one format poison another format's cache entry.

The stored object is the exact final response body, with its final
`Content-Type`. Go should finish the B2 upload before reporting a successful
cold response so another Cloudflare location cannot immediately observe a
false miss.

## Illustrative Snippet

This is deliberately not deployable until Go implements and tests the matching
delivery-key function.

```js
const B2_DELIVERY =
  "https://f003.backblazeb2.com/file/evekill-media/delivery/v1/";

export default {
  async fetch(request) {
    if (request.method !== "GET") {
      return fetch(request);
    }

    // This function must exactly match the canonicalization and SHA-256
    // implementation in Go.
    const objectDigest = await deliveryDigest(request);
    const b2Request = new Request(B2_DELIVERY + objectDigest, {
      method: "GET",
      headers: {
        "User-Agent": "EVE-KILL Cloudflare image cache",
      },
      redirect: "manual",
    });

    const stored = await fetch(b2Request, {
      cf: {
        cacheEverything: true,
        cacheTtlByStatus: {
          "200-299": 2592000,
          "300-399": 0,
          "400-499": 0,
          "500-599": 0,
        },
      },
    });

    if (stored.ok) {
      return stored;
    }

    if (stored.body) {
      stored.body.cancel();
    }

    // This must be the only continuation to the normal eve-kill.com origin.
    return fetch(request);
  },
};
```

Do not add an authentication request, pointer lookup, redirect, or another
fallback. Pro's two-subrequest limit has no room for it.

## Cloudflare rule

Start with a rule scoped to image GET requests only:

```text
http.host eq "eve-kill.com"
and starts_with(http.request.uri.path, "/images/")
and http.request.method eq "GET"
```

Deploy it as a canary against a narrower path first, such as one static image
family, before enabling every image route.

The Snippet contains no B2 key. It reads only from the public bucket; the Go
application retains the write-only operational credentials.

## Work required before deployment

1. Implement a versioned delivery-key function in Go.
2. Implement the identical canonicalization and digest function in the
   Snippet.
3. Add shared fixtures proving Go and JavaScript produce the same keys.
4. Store every successful generated/fetched final response under that key.
5. Verify public B2 GETs for representative PNG, JPEG, and WebP objects.
6. Confirm a B2 `404` is not cached.
7. Confirm a B2 success is cached for 30 days.
8. Confirm an unavailable B2 origin falls back to Go.
9. Confirm Go is not contacted for a warm B2 object.
10. Deploy narrowly, inspect Cloudflare and Go request metrics, then expand the
    rule.

## Purging and updates

The delivery namespace is versioned so a canonicalization or processing change
can move from `delivery/v1` to `delivery/v2` without mutating the old contract.

For ordinary portrait updates, Go may overwrite the current delivery object,
but Cloudflare can retain the prior response for up to 30 days. If an update
must be visible immediately, purge its B2 delivery URL from Cloudflare or bump
the delivery version/key input.

Never cache B2 `404`, authorization failures, or `5xx` responses. A cached miss
would hide a newly generated object until that negative entry expired.

## Rollback

Disable the Snippet rule. Requests then continue directly to the Go image
handlers on the same `/images/*` URLs; no frontend change or redirect rollback
is required.

## Seed status

As of 2026-07-27:

- `evekill-media` is public-readable;
- 22,155 static map, UI, Dust 514, and overlay files have been seeded;
- TurtleTools type assets use the directly hostable
  `Image.Export.Collection.zip`, not the deduplicated Service Bundle;
- 66,808 type images are seeded at direct `types/*` keys, including 4,798
  blueprint-copy icons;
- `types/manifest.json` records the SHA-256 of every mirrored type image so
  later syncs upload only new or changed files;
- stable type-ID objects use a one-day browser and 30-day edge cache policy;
  content-hashed generated variants remain immutable;
- the EVE Ref old-character portrait archive is seeded separately with the
  `images` CLI.

## References

- [Cloudflare Snippets](https://developers.cloudflare.com/rules/snippets/)
