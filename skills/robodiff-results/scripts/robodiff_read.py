#!/usr/bin/env python3
"""Read local Robodiff diagnostic JSON using a narrowly approvable command."""
import argparse
import json
import re
import sys
import urllib.error
import urllib.parse
import urllib.request

MAX_BYTES = 2 * 1024 * 1024
ID = r"[A-Za-z0-9_-]+"
ROUTE = re.compile(
    rf"/api/agent/runs(?:/{ID}/(?:analysis-pack|triage|errors|tests|tests/{ID}|nodes/{ID}|groups/{ID}/tests))?"
)


class NoRedirects(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def request_url(base, endpoint):
    # Validate before urlsplit, which otherwise strips some control characters.
    if any(ord(c) < 33 or ord(c) == 127 for c in base + endpoint):
        raise ValueError("URLs must not contain whitespace or control characters")
    origin = urllib.parse.urlsplit(base)
    if (origin.scheme != "http" or origin.hostname not in ("127.0.0.1", "localhost", "::1")
            or origin.username is not None or origin.password is not None
            or origin.path not in ("", "/") or origin.query or origin.fragment):
        raise ValueError("--base must be an HTTP loopback origin, e.g. http://127.0.0.1:8080")
    port = 80 if origin.port is None else origin.port
    if not 1 <= port <= 65535:
        raise ValueError("invalid port")
    route = urllib.parse.urlsplit(endpoint)
    if route.scheme or route.netloc or route.fragment or not ROUTE.fullmatch(route.path):
        raise ValueError("only /api/agent/ diagnostic routes are supported")
    params = urllib.parse.parse_qsl(route.query, keep_blank_values=True, strict_parsing=True)
    seen = set()
    for key, value in params:
        if key in seen:
            raise ValueError("duplicate query parameter")
        seen.add(key)
        if key == "status":
            if value not in ("FAIL", "PASS", "SKIP"):
                raise ValueError("status must be FAIL, PASS or SKIP")
        elif key in ("offset", "textOffset", "limit", "textLimit"):
            if not re.fullmatch(r"[0-9]{1,10}", value):
                raise ValueError("pagination values must be nonnegative integers")
            maximum = 100 if key == "limit" else 2000 if key == "textLimit" else 1000000000
            minimum = 1 if key == "limit" else 0
            if not minimum <= int(value) <= maximum:
                raise ValueError(f"{key} must be {minimum}..{maximum}")
        else:
            raise ValueError("unsupported query parameter")
    # Avoid hostname resolution for localhost and ignore environment proxies.
    host = "[::1]" if origin.hostname == "::1" else "127.0.0.1"
    query = urllib.parse.urlencode(params)
    return f"http://{host}:{port}{route.path}" + ("?" + query if query else "")


def read(base, endpoint):
    url = request_url(base, endpoint)
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirects())
    request = urllib.request.Request(url, headers={"Accept": "application/json"}, method="GET")
    with opener.open(request, timeout=15) as response:
        body = response.read(MAX_BYTES + 1)
    if len(body) > MAX_BYTES:
        raise ValueError("response too large; request a smaller page")
    data = json.loads(body)
    if not isinstance(data, dict) or data.get("schemaVersion") != 1:
        raise ValueError("not a supported Robodiff diagnostic response; check the port/backend version")
    return data


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base", required=True, help="verified local Robodiff origin")
    parser.add_argument("endpoint", help="diagnostic path, optionally including pagination query")
    args = parser.parse_args()
    try:
        data = read(args.base, args.endpoint)
    except (ValueError, OSError, urllib.error.URLError) as error:
        print(f"Robodiff read failed: {error}", file=sys.stderr)
        return 1
    print(json.dumps(data, ensure_ascii=False, separators=(",", ":")))
    return 0


if __name__ == "__main__":
    sys.exit(main())
