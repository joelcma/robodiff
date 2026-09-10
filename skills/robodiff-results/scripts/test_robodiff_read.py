import io
import unittest
from unittest.mock import patch
import urllib.request

import robodiff_read as reader


class ReaderTests(unittest.TestCase):
    def test_local_routes_and_paging(self):
        for endpoint in ("/api/agent/runs", "/api/agent/runs/abc/triage",
                         "/api/agent/runs/abc/errors", "/api/agent/runs/abc/tests",
                         "/api/agent/runs/abc/tests/s0-t1", "/api/agent/runs/abc/nodes/s0-t1-k2",
                         "/api/agent/runs/abc/groups/g123/tests"):
            with self.subTest(endpoint=endpoint):
                self.assertEqual(reader.request_url("http://localhost:55827", endpoint),
                                 "http://127.0.0.1:55827" + endpoint)
        self.assertEqual(reader.request_url("http://[::1]:8080", "/api/agent/runs?limit=1&offset=2&textOffset=3&status=FAIL"),
                         "http://[::1]:8080/api/agent/runs?limit=1&offset=2&textOffset=3&status=FAIL")

    def test_rejects_other_targets_and_operations(self):
        for base in ("https://example.com", "http://example.com", "file:///tmp/data", "http://127.0.0.1@evil.test",
                     "http://user:pass@localhost", "http://127.0.0.1/path", "http://localhost?x=1",
                     "http://localhost:99999", "http://local\nhost", "http://127.0.0.2"):
            with self.subTest(base=base), self.assertRaises(ValueError):
                reader.request_url(base, "/api/agent/runs")
        for path in ("/api/delete-runs", "/api/http-try", "/api/run-file?path=secret", "//evil.test/api/agent/runs",
                     "/api/agent/runs/../delete-runs", "/api/agent/runs/%2e%2e/triage", "/api/agent/runs#x",
                     "/api/agent/runs?limit=101", "/api/agent/runs?limit=0", "/api/agent/runs?offset=-1",
                     "/api/agent/runs?command=delete", "/api/agent/runs?limit=1&limit=2"):
            with self.subTest(path=path), self.assertRaises(ValueError):
                reader.request_url("http://127.0.0.1:8080", path)

    def test_get_only_no_proxy_and_no_redirects(self):
        with patch.object(urllib.request, "build_opener") as build:
            build.return_value.open.return_value = io.BytesIO(b'{"schemaVersion":1,"runs":{}}')
            self.assertEqual(reader.read("http://127.0.0.1:8080", "/api/agent/runs")["schemaVersion"], 1)
            self.assertEqual(build.call_args.args[0].proxies, {})
            self.assertIsInstance(build.call_args.args[1], reader.NoRedirects)
            request = build.return_value.open.call_args.args[0]
            self.assertEqual(request.get_method(), "GET")
            self.assertIsNone(request.data)
        self.assertIsNone(reader.NoRedirects().redirect_request(None, None, 302, "", {}, "http://evil.test"))

    def test_invalid_and_oversized_responses(self):
        for body in (b"{}", b"[]", b"<html>old server</html>", b"x" * (reader.MAX_BYTES + 1)):
            with self.subTest(size=len(body)), patch.object(urllib.request, "build_opener") as build:
                build.return_value.open.return_value = io.BytesIO(body)
                with self.assertRaises(ValueError):
                    reader.read("http://127.0.0.1:8080", "/api/agent/runs")


if __name__ == "__main__":
    unittest.main()
