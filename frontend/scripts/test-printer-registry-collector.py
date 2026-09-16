import importlib.util
import json
import tempfile
import sqlite3
import unittest
from pathlib import Path
from unittest.mock import Mock, patch

SPEC = importlib.util.spec_from_file_location("registry_collector", Path(__file__).with_name("printer-registry-collector.py"))
COLLECTOR = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(COLLECTOR)


class RegistryCollectorTests(unittest.TestCase):
    def test_saved_station_note_is_collected_when_printer_is_offline(self):
        with patch.object(COLLECTOR.OBSERVER,'read_runtime',return_value=({'replacement':{'condition':'out_of_service','problem_note':'Waiting for fan','system_status':'unavailable'}},{})), patch.object(COLLECTOR.OBSERVER,'read_printer',return_value={'id':'replacement','condition':'unknown','activity':'unknown'}), patch.object(COLLECTOR,'read_history',return_value=[]):
            snapshot=COLLECTOR.collect([('replacement','192.168.1.160','fc:ee:28:00:30:aa')])
            self.assertEqual(snapshot['printers'][0]['condition'],'out')
            self.assertEqual(snapshot['printers'][0]['note'],'Waiting for fan')
            self.assertEqual(snapshot['printers'][0]['activity'],'unknown')

    def test_history_projection_excludes_credentials_and_preserves_failure_details(self):
        with tempfile.TemporaryDirectory() as directory:
            path=Path(directory)/'station.sqlite'
            db=sqlite3.connect(path)
            db.executescript('''CREATE TABLE requests(id TEXT,printer_id TEXT,file_name TEXT,filament_type TEXT,user_display TEXT);
                CREATE TABLE events(id INTEGER,request_id TEXT,event_type TEXT,created_at TEXT,payload_json TEXT);
                CREATE TABLE printer_runtime_events(id INTEGER,printer_id TEXT,event_type TEXT,created_at TEXT,payload_json TEXT);''')
            db.execute('INSERT INTO requests VALUES (?,?,?,?,?)',('job','replacement','part.gcode','PLA','Fixture user'))
            db.execute('INSERT INTO events VALUES (?,?,?,?,?)',(10,'job','printer_failed','2026-09-15T12:00:00Z',json.dumps({'message':'Heater fault','printDurationSeconds':7510,'accessRef':'secret-access','token':'secret-token'})))
            db.execute('INSERT INTO printer_runtime_events VALUES (?,?,?,?,?)',(20,'replacement','staff_runtime_changed','2026-09-15T12:01:00Z',json.dumps({'newCondition':'out_of_service','newNote':'Fan failed','accessRef':'secret-access'})))
            db.commit(); db.close()
            history=COLLECTOR.read_history(['replacement'],path,include_identity=True)
            self.assertEqual(len(history),2)
            self.assertIn('Heater fault',json.dumps(history))
            self.assertIn('Fan failed',json.dumps(history))
            self.assertNotIn('secret-',json.dumps(history))
            job=next(event for event in history if event['eventType']=='printer_failed')
            self.assertEqual(job['person'],'Fixture user')
            self.assertEqual(job['durationSeconds'],7510)
            self.assertEqual(job['detail'],'Heater fault')
            self.assertEqual(history,COLLECTOR.read_history(['replacement'],path,include_identity=True))
            with patch.dict(COLLECTOR.os.environ, {}, clear=True):
                private_off=COLLECTOR.read_history(['replacement'],path)
            self.assertTrue(all('person' not in event and 'actorName' not in event and 'actorMethod' not in event for event in private_off))

            with patch.dict(COLLECTOR.os.environ, {'PRINTER_HISTORY_PRINT_USERS_ENABLED':'true'}, clear=True):
                users_only=COLLECTOR.read_history(['replacement'],path)
            self.assertEqual(next(e for e in users_only if e['eventType']=='printer_failed')['person'],'Fixture user')
            self.assertTrue(all('actorName' not in e and 'actorMethod' not in e for e in users_only))

    def test_change_projection_handles_independent_edits_clear_and_unknown(self):
        with tempfile.TemporaryDirectory() as directory:
            path=Path(directory)/'station.sqlite'; db=sqlite3.connect(path)
            db.executescript("CREATE TABLE requests(id TEXT,printer_id TEXT,file_name TEXT,filament_type TEXT,user_display TEXT); CREATE TABLE events(id INTEGER,request_id TEXT,event_type TEXT,created_at TEXT,payload_json TEXT); CREATE TABLE printer_runtime_events(id INTEGER,printer_id TEXT,event_type TEXT,created_at TEXT,payload_json TEXT);")
            changes=[('working','working','old','new'),('working','out_of_service','same','same'),('working','out_of_service','old','new'),('working','working','old',''),('working','working','','')]
            for i,(before,after,old_note,new_note) in enumerate(changes):
                payload={'oldCondition':before,'newCondition':after,'oldNote':old_note,'newNote':new_note}
                db.execute('INSERT INTO printer_runtime_events VALUES (?,?,?,?,?)',(i,'replacement','staff_runtime_changed','2026-09-16T12:00:00Z',json.dumps(payload)))
            db.execute('INSERT INTO printer_runtime_events VALUES (?,?,?,?,?)',(5,'replacement','staff_runtime_changed','2026-09-16T12:00:00Z','{"newCondition":"working","newNote":"legacy"}'))
            db.commit();db.close()
            events={e['sourceId']:e for e in COLLECTOR.read_history(['replacement'],path,include_identity=False)}
            self.assertEqual([(events[f'station:condition:{i}']['noteChanged'],events[f'station:condition:{i}']['conditionChanged']) for i in range(5)],[(True,False),(False,True),(True,True),(True,False),(False,False)])
            self.assertNotIn('noteChanged',events['station:condition:5'])
            self.assertNotIn('conditionChanged',events['station:condition:5'])

    def test_shutdown_message_survives_unavailable_print_stats(self):
        def get(host, route):
            if route == '/machine/system_info':
                return {'result':{'system_info':{'network':{'eth0':{'mac_address':self.record['mac']}}}}}
            if route == '/printer/info':
                return {'result':{'state':'shutdown','state_message':'MCU shutdown: Heater not heating'}}
            raise AssertionError('Do not query unavailable print stats')
        with patch.object(COLLECTOR.OBSERVER,'get_json',side_effect=get):
            reading=COLLECTOR.OBSERVER.read_printer(('replacement',self.record['host'],self.record['mac']),{}, {})
        self.assertEqual(reading['fault'],'MCU shutdown: Heater not heating')
        self.assertEqual(reading['activity'],'unknown')

    def test_station_attribution_exports_only_display_name_and_known_method(self):
        with tempfile.TemporaryDirectory() as directory:
            path=Path(directory)/'station.sqlite'
            db=sqlite3.connect(path)
            db.executescript('''CREATE TABLE requests(id TEXT,printer_id TEXT,file_name TEXT,filament_type TEXT,user_display TEXT);
                CREATE TABLE events(id INTEGER,request_id TEXT,event_type TEXT,created_at TEXT,payload_json TEXT);
                CREATE TABLE printer_runtime_events(id INTEGER,printer_id TEXT,event_type TEXT,created_at TEXT,payload_json TEXT);''')
            for i,method in enumerate(('ucard','local_pin',None,'unknown','printer_api')):
                payload={'oldCondition':'working','newCondition':'working','oldNote':'Fan broken','newNote':'Fan replaced','displayIdentity':'Fixture staff','method':method,
                    'actorId':'private-user-id','accessRef':'secret-access','cardCsn':'secret-card','pin':'secret-pin'}
                kind='system_runtime_changed' if method=='printer_api' else 'staff_runtime_changed'
                db.execute('INSERT INTO printer_runtime_events VALUES (?,?,?,?,?)',(i,'replacement',kind,'2026-09-15T12:00:00Z',json.dumps(payload)))
            db.commit(); db.close()
            events={event['sourceId']:event for event in COLLECTOR.read_history(['replacement'],path,include_identity=True)}
            self.assertEqual(len(events),5)
            self.assertTrue(all(event['noteChanged'] is True and event['conditionChanged'] is False for event in events.values()))
            self.assertEqual(events['station:condition:0']['actorName'],'Fixture staff')
            self.assertEqual(events['station:condition:0']['actorMethod'],'ucard')
            self.assertEqual(events['station:condition:1']['actorMethod'],'local_pin')
            for i in (1,2,3,4): self.assertNotIn('actorName',events[f'station:condition:{i}'])
            for i in (2,3): self.assertNotIn('actorMethod',events[f'station:condition:{i}'])
            self.assertEqual(events['station:condition:4']['actorMethod'],'printer_api')
            with patch.dict(COLLECTOR.os.environ, {}, clear=True):
                redacted=COLLECTOR.read_history(['replacement'],path,include_print_users=True)
            self.assertTrue(all('actorName' not in event and 'actorMethod' not in event for event in redacted))
            serialized=json.dumps(events)
            for private in ('secret-','private-user-id','accessRef','cardCsn','pin"'):
                self.assertNotIn(private,serialized.replace('local_pin"',''))

    def setUp(self):
        self.record = {"id": "replacement", "host": "192.168.1.160", "mac": "fc:ee:28:00:30:aa"}

    def test_backfill_expands_retained_window_without_changing_normal_polling(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'station.sqlite'
            db = sqlite3.connect(path)
            db.executescript('''CREATE TABLE requests(id TEXT,printer_id TEXT,file_name TEXT,filament_type TEXT,user_display TEXT);
                CREATE TABLE events(id INTEGER,request_id TEXT,event_type TEXT,created_at TEXT,payload_json TEXT);
                CREATE TABLE printer_runtime_events(id INTEGER,printer_id TEXT,event_type TEXT,created_at TEXT,payload_json TEXT);''')
            db.execute('INSERT INTO requests VALUES (?,?,?,?,?)', ('job','replacement','part.gcode','PLA','Fixture user'))
            for i in range(35):
                db.execute('INSERT INTO events VALUES (?,?,?,?,?)', (i,'job','printer_completed',f'2026-09-15T12:00:{i:02d}Z','{}'))
            for i in range(25):
                db.execute('INSERT INTO printer_runtime_events VALUES (?,?,?,?,?)', (i,'replacement','staff_runtime_changed',f'2026-09-15T12:01:{i:02d}Z','{"method":"local_pin"}'))
            db.commit(); db.close()
            normal = COLLECTOR.read_history(['replacement'], path)
            expanded = COLLECTOR.read_history(['replacement'], path, job_limit=1000, condition_limit=1000)
            self.assertEqual(len(normal), 50)
            self.assertEqual(len(expanded), 60)
            self.assertNotIn('station:job:0', [event['sourceId'] for event in normal])
            self.assertIn('station:job:0', [event['sourceId'] for event in expanded])
            self.assertEqual(COLLECTOR.read_history(['unmapped'], path, job_limit=1000, condition_limit=1000), [])
            self.assertEqual(normal, COLLECTOR.read_history(['replacement'], path))

    def test_history_limits_are_bounded(self):
        for invalid in (0, 1001, True, '100', 1.5):
            with self.assertRaises(ValueError):
                COLLECTOR.read_history([], job_limit=invalid)
            with self.assertRaises(ValueError):
                COLLECTOR.read_history([], condition_limit=invalid)

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
