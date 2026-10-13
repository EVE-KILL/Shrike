#!/usr/bin/env python3
"""Read-only HTTP inventory audit. Browser parity is a separate required gate."""
import argparse
import concurrent.futures
import datetime
import html.parser
import json
import pathlib
import re
import time
import urllib.error
import urllib.parse
import urllib.request

ROOT = pathlib.Path(__file__).resolve().parents[1]


class Page(html.parser.HTMLParser):
    def __init__(self):
        super().__init__()
        self.title = ""
        self.in_head = False
        self.in_title = False
        self.links = []
        self.assets = set()
        self.canonical = ""
        self.robots = ""

    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if tag == "head":
            self.in_head = True
        if tag == "title" and self.in_head:
            self.in_title = True
        if tag == "a" and attrs.get("href", "").startswith("/"):
            self.links.append(attrs["href"])
        if tag == "link" and attrs.get("rel") == "canonical":
            self.canonical = attrs.get("href", "")
        if tag == "meta" and attrs.get("name") == "robots":
            self.robots = attrs.get("content", "")
        asset = attrs.get("src", "") if tag == "script" else attrs.get("href", "") if tag == "link" else ""
        if asset.startswith("/assets/"):
            self.assets.add(asset)

    def handle_endtag(self, tag):
        if tag == "title":
            self.in_title = False
        if tag == "head":
            self.in_head = False
            self.in_title = False

    def handle_data(self, data):
        if self.in_title:
            self.title += data


def request(origin, path, method="GET"):
    started = time.monotonic()
    try:
        response = urllib.request.urlopen(urllib.request.Request(origin + path, method=method), timeout=30)
    except urllib.error.HTTPError as error:
        response = error
    except (OSError, TimeoutError) as error:
        return {"path": path, "error": str(error), "seconds": round(time.monotonic() - started, 3)}, Page()
    body = response.read()
    page = Page()
    if "text/html" in response.headers.get("Content-Type", ""):
        page.feed(body.decode("utf-8", errors="replace"))
    return {"path": path, "status": response.code, "finalPath": urllib.parse.urlsplit(response.url).path,
            "seconds": round(time.monotonic() - started, 3), "bytes": len(body), "title": page.title,
            "canonical": page.canonical, "robots": page.robots,
            "cache": response.headers.get("Cache-Control", ""), "serverTiming": response.headers.get("Server-Timing", ""),
            "contentType": response.headers.get("Content-Type", "")}, page


def inventory(origin):
    directories = {}
    for path in ["/", "/wars", "/battles", "/campaigns", "/fits", "/coalitions", "/blog", "/legacy", "/market", "/item/11186"]:
        result, page = request(origin, path)
        directories[path] = page.links if result.get("status") == 200 else []

    def link(directory, prefix):
        return next((href.split("?", 1)[0] for href in directories[directory] if href.startswith(prefix)), "")

    examples = {"/kill/{id}": link("/", "/kill/"), "/battle/{id}": link("/battles", "/battle/"),
                "/war/{id}": link("/wars", "/war/"), "/campaign/{id}": link("/campaigns", "/campaign/"),
                "/fit/{fit_id}": link("/fits", "/fit/"), "/coalitions/{slug}": link("/coalitions", "/coalitions/"),
                "/blog/{slug}": link("/blog", "/blog/"), "/legacy/kill/{id}": link("/legacy", "/legacy/kill/"),
                "/group/{id}": link("/item/11186", "/group/"), "/map/region/{id}": "/map/region/10000042",
                "/market/item/{id}": "/market/item/11186", "/item/{item_id}": "/item/11186",
                "/ship/{id}": "/ship/11186", "/wars/{id}": link("/wars", "/war/").replace("/war/", "/wars/"),
                "/killmail/{id}": link("/", "/kill/").replace("/kill/", "/killmail/")}
    group = examples["/group/{id}"]
    examples["/shipgroup/{id}"] = group.replace("/group/", "/shipgroup/")
    # Area campaigns intentionally have no intel tab. Use a real participant
    # campaign that exposes that route instead of inventing an invalid case.
    for candidate in dict.fromkeys(directories["/campaigns"]):
        if candidate.startswith("/campaign/") and candidate.count("/") == 2:
            result, page = request(origin, candidate)
            if result.get("status") == 200 and candidate + "/intel" in page.links:
                examples["/campaign/{id}/intel"] = candidate + "/intel"
                break
    kinds_source = (ROOT / "frontend/shared/utils/killListTypes.ts").read_text()
    kinds = re.findall(r"'([^']+)'", kinds_source.split("export const KILL_LIST_TYPES = [", 1)[1].split("] as const", 1)[0])
    routes = set()
    unresolved = set()
    for line in (ROOT / "TODO.md").read_text().splitlines():
        match = re.match(r"- \[[ x]\] `([^`]+)`", line)
        if not match:
            continue
        pattern = match[1]
        # Account/private routes use the disposable authenticated API audit.
        if pattern.startswith(("/admin", "/settings", "/tracker", "/blog/preview")):
            continue
        if pattern == "/kills/{type}":
            routes.update("/kills/" + kind for kind in kinds)
            continue
        if pattern == "/market/{category/...}":
            entries = [href for href in directories["/market"] if href.startswith("/market/") and not href.startswith("/market/item/")]
            routes.update(entries)
            if not entries:
                unresolved.add(pattern)
            continue
        if pattern == "/faction-war/{matchup}":
            entries = [href for href in directories["/wars"] if href.startswith("/faction-war/")]
            routes.update(entries)
            if not entries:
                unresolved.add(pattern)
            continue
        if pattern.startswith("/_sitemap-data/"):
            # Indexing routes have a separate production-host test gate.
            continue
        if "[/{tab}]" in pattern:
            base = pattern.replace("[/{tab}]", "")
            patterns = [base, base + "/kills"]
        else:
            patterns = [pattern]
        for pattern in patterns:
            for prefix, replacement in sorted(examples.items(), key=lambda entry: -len(entry[0])):
                if pattern.startswith(prefix) and replacement:
                    pattern = replacement + pattern[len(prefix):]
                    break
            choice = re.search(r"\{([^{}]+,[^{}]+)\}", pattern)
            expanded = [pattern[:choice.start()] + value + pattern[choice.end():] for value in choice[1].split(",")] if choice else [pattern]
            for path in expanded:
                if examples["/campaign/{id}"] and path == examples["/campaign/{id}"] + "/intel":
                    path = examples.get("/campaign/{id}/intel", path)
                if "{" in path or "}" in path or "..." in path:
                    unresolved.add(path)
                else:
                    routes.add(path)
    # Exercise a populated search in addition to the empty entry form.
    routes.add("/search?q=Rifter")
    routes.update(f"/map?activity={layer}&hours={hours}" for layer in ["activity", "danger", "ship-kills", "pod-kills", "npc-kills", "jumps"] for hours in [1, 6, 24, 168])
    return sorted(routes), sorted(unresolved)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--origin", default="http://127.0.0.1:4201")
    parser.add_argument("--output", default="/tmp/shrike-frontend-http-review.json")
    args = parser.parse_args()
    parsed = urllib.parse.urlsplit(args.origin)
    if parsed.scheme != "http" or parsed.hostname not in {"127.0.0.1", "localhost"} or parsed.path or parsed.query or parsed.username:
        parser.error("origin must be a plain loopback HTTP origin")
    routes, unresolved = inventory(args.origin)
    assets = set()
    results = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=2) as workers:
        futures = {workers.submit(request, args.origin, route): route for route in routes}
        for future in concurrent.futures.as_completed(futures):
            result, page = future.result()
            results.append(result)
            assets.update(page.assets)
            if result.get("status") != 200:
                print(json.dumps(result), flush=True)
    asset_results = [request(args.origin, asset, "HEAD")[0] for asset in sorted(assets)]
    report = {"at": datetime.datetime.now(datetime.timezone.utc).isoformat(), "origin": args.origin,
              "scope": "HTTP transport only; visual, mobile, theme and behavior checks remain separate",
              "routes": sorted(results, key=lambda result: result["path"]), "assets": asset_results, "unresolved": unresolved}
    pathlib.Path(args.output).write_text(json.dumps(report, indent=2) + "\n")
    print(json.dumps({"routes": len(results), "assets": len(asset_results), "httpFailures": sum(result.get("status") != 200 for result in results),
                      "assetFailures": sum(result.get("status") != 200 for result in asset_results), "unresolved": unresolved, "report": args.output}), flush=True)


if __name__ == "__main__":
    main()
