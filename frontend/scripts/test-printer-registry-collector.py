import importlib.util
import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import Mock, patch

SPEC = importlib.util.spec_from_file_location("registry_collector", Path(__file__).with_name("printer-registry-collector.py"))
COLLECTOR = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(COLLECTOR)


class RegistryCollectorTests(unittest.TestCase):
    def setUp(self):
        self.record = {"id": "replacement", "host": "192.168.1.160", "mac": "fc:ee:28:00:30:aa"}

    def test_accepts_new_identity_without_source_edit(self):
        self.assertEqual(COLLECTOR.validate_roster({"printers": [self.record]}), [("replacement", "192.168.1.160", "fc:ee:28:00:30:aa")])

    def test_rejects_unsafe_and_duplicate_targets(self):
        for host in ("127.0.0.1", "169.254.169.254", "192.168.2.10", "192.168.1.255"):
            with self.assertRaises(ValueError):
                COLLECTOR.validate_roster({"printers": [{**self.record, "host": host}]})
        with self.assertRaises(ValueError):
            COLLECTOR.validate_roster({"printers": [self.record, self.record]})

    def test_outage_uses_previously_validated_cache(self):
        with tempfile.TemporaryDirectory() as directory:
            cache = Path(directory) / "roster.json"
            cache.write_text(json.dumps({"printers": [self.record]}))
            opener = Mock(); opener.open.side_effect = OSError("offline")
            self.assertEqual(COLLECTOR.load_roster(opener, "fixture", cache)[0][0], "replacement")

    def test_collects_activity_when_station_db_is_down(self):
        with patch.object(COLLECTOR.OBSERVER, "read_runtime", side_effect=OSError()), patch.object(COLLECTOR.OBSERVER, "read_printer", return_value={"id": "replacement", "condition": "unknown", "activity": "idle"}) as read:
            snapshot = COLLECTOR.collect([("replacement", "192.168.1.160", "fc:ee:28:00:30:aa")])
            self.assertEqual(snapshot["printers"][0]["activity"], "idle")
            self.assertEqual(read.call_args.args[1:], ({}, {}))

    def test_observation_requires_matching_mac_and_never_sends_commands(self):
        with patch.object(COLLECTOR.OBSERVER, "get_json", return_value={"result": {"system_info": {"network": {"eth0": {"mac_address": "fc:ee:28:00:30:bb"}}}}}) as get:
            result = COLLECTOR.OBSERVER.read_printer(("replacement", "192.168.1.160", "fc:ee:28:00:30:aa"), {}, {})
            self.assertEqual(result["activity"], "unknown")
            self.assertEqual(get.call_count, 1)


if __name__ == "__main__":
    unittest.main()
