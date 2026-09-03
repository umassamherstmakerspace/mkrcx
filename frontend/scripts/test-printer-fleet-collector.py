import importlib.util
import os
import unittest
from pathlib import Path
from unittest.mock import patch


MODULE_PATH = Path(__file__).with_name("printer-fleet-collector.py")
SPEC = importlib.util.spec_from_file_location("printer_fleet_collector", MODULE_PATH)
COLLECTOR = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(COLLECTOR)


class Response:
    status = 204

    def __enter__(self):
        return self

    def __exit__(self, *_args):
        return False


class PublishTests(unittest.TestCase):
    def setUp(self):
        self.snapshot = {"fetchedAt": "2026-09-03T12:00:00Z", "printers": []}

    def test_primary_target_remains_supported(self):
        requests = []

        def open_request(request, timeout):
            requests.append((request, timeout))
            return Response()

        environment = {
            "PRINTER_FLEET_ENDPOINT": "https://staging.example/ingest",
            "PRINTER_FLEET_INGEST_SECRET": "staging-secret",
        }
        with patch.dict(os.environ, environment, clear=True), patch.object(
            COLLECTOR.urllib.request, "urlopen", side_effect=open_request
        ):
            self.assertEqual(COLLECTOR.publish(self.snapshot), 1)

        self.assertEqual(len(requests), 1)
        self.assertEqual(requests[0][0].full_url, environment["PRINTER_FLEET_ENDPOINT"])
        self.assertEqual(requests[0][0].get_header("Authorization"), "Bearer staging-secret")

    def test_secondary_target_receives_the_same_snapshot(self):
        requests = []

        def open_request(request, timeout):
            requests.append((request, timeout))
            return Response()

        environment = {
            "PRINTER_FLEET_ENDPOINT": "https://staging.example/ingest",
            "PRINTER_FLEET_INGEST_SECRET": "staging-secret",
            "PRINTER_FLEET_SECONDARY_ENDPOINT": "https://production.example/ingest",
            "PRINTER_FLEET_SECONDARY_INGEST_SECRET": "production-secret",
        }
        with patch.dict(os.environ, environment, clear=True), patch.object(
            COLLECTOR.urllib.request, "urlopen", side_effect=open_request
        ):
            self.assertEqual(COLLECTOR.publish(self.snapshot), 2)

        self.assertEqual([request.full_url for request, _timeout in requests], [
            environment["PRINTER_FLEET_ENDPOINT"],
            environment["PRINTER_FLEET_SECONDARY_ENDPOINT"],
        ])
        self.assertEqual(requests[1][0].get_header("Authorization"), "Bearer production-secret")
        self.assertEqual(requests[0][0].data, requests[1][0].data)

    def test_one_failed_target_does_not_skip_the_other(self):
        requests = []

        def open_request(request, timeout):
            requests.append((request, timeout))
            if "staging" in request.full_url:
                raise OSError("staging unavailable")
            return Response()

        environment = {
            "PRINTER_FLEET_ENDPOINT": "https://staging.example/ingest",
            "PRINTER_FLEET_INGEST_SECRET": "staging-secret",
            "PRINTER_FLEET_SECONDARY_ENDPOINT": "https://production.example/ingest",
            "PRINTER_FLEET_SECONDARY_INGEST_SECRET": "production-secret",
        }
        with patch.dict(os.environ, environment, clear=True), patch.object(
            COLLECTOR.urllib.request, "urlopen", side_effect=open_request
        ):
            with self.assertRaisesRegex(RuntimeError, "primary:OSError"):
                COLLECTOR.publish(self.snapshot)

        self.assertEqual(len(requests), 2)

    def test_secondary_target_requires_both_values(self):
        environment = {
            "PRINTER_FLEET_ENDPOINT": "https://staging.example/ingest",
            "PRINTER_FLEET_INGEST_SECRET": "staging-secret",
            "PRINTER_FLEET_SECONDARY_ENDPOINT": "https://production.example/ingest",
        }
        with patch.dict(os.environ, environment, clear=True):
            with self.assertRaisesRegex(RuntimeError, "secondary collector target is incomplete"):
                COLLECTOR.publish(self.snapshot)


if __name__ == "__main__":
    unittest.main()
