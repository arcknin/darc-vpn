import base64,importlib.util,json,pathlib,tempfile,unittest,unittest.mock
ROOT=pathlib.Path(__file__).resolve().parents[1]
spec=importlib.util.spec_from_file_location('target',ROOT/'target.py');t=importlib.util.module_from_spec(spec);spec.loader.exec_module(t)
def key(n):return base64.b64encode(bytes([n])*32).decode()
class ExportTests(unittest.TestCase):
 def test_export_keeps_server_identity_separate_and_detects_conflict(self):
  with tempfile.TemporaryDirectory() as tmp:
   p=pathlib.Path(tmp);(p/'identity').mkdir();cfg=p/'live.conf';cfg.write_text('[Interface]\nPrivateKey = '+key(1)+'\nListenPort = 51442\n[Peer]\nPublicKey = '+key(2)+'\nPresharedKey = '+key(3)+'\nAllowedIPs = 10.66.66.2/32\n');interface=p/'interface.json';interface.write_text(json.dumps([{'addr_info':[{'local':'10.66.66.1','prefixlen':24},{'local':'fd42:42:42::1','prefixlen':64}]}]))
   with unittest.mock.patch.object(t.inv,'secure',side_effect=lambda path,*args:pathlib.Path(path)):
    t.export(str(cfg),str(interface),str(p));manifest=(p/'legacy-import.json').read_text();self.assertNotIn(key(1),manifest);self.assertIn(key(3),manifest);self.assertEqual((p/'identity/server.key').read_text().strip(),key(1));t.export(str(cfg),str(interface),str(p));(p/'identity/server.key').write_text(key(4));
    with self.assertRaises(ValueError):t.export(str(cfg),str(interface),str(p))
 def test_02b_wrapper_help_and_nonroot_no_operation(self):
  import os,subprocess
  result=subprocess.run(['/bin/bash',str(ROOT/'02b-build-target.sh'),'--help'],capture_output=True,text=True);self.assertEqual(result.returncode,0)
  if os.geteuid()!=0:
   result=subprocess.run(['/bin/bash',str(ROOT/'02b-build-target.sh')],capture_output=True,text=True);self.assertEqual(result.returncode,1);self.assertIn('Root required',result.stderr)
if __name__=='__main__':unittest.main()
