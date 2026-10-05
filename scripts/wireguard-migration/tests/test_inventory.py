import base64,importlib.util,json,pathlib,tempfile,unittest
PATH=pathlib.Path(__file__).resolve().parents[1]/'inventory.py'
spec=importlib.util.spec_from_file_location('inventory',PATH);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
def key(n):return base64.b64encode(bytes([n])*32).decode()
def config():return '[Interface]\nPrivateKey = '+key(1)+'\nListenPort = 51442\n[Peer]\nPublicKey = '+key(2)+'\nPresharedKey = '+key(3)+'\nAllowedIPs = 10.66.66.2/32, fd42:42:42::2/128\n'
class InventoryTests(unittest.TestCase):
 def test_summary_never_exposes_keys_or_hooks(self):
  p=m.parse(config());report=m.summary(p);text=json.dumps(report)
  for n in (1,2,3):self.assertNotIn(key(n),text)
  self.assertEqual(report['peer_count'],1);self.assertEqual(report['psk_count'],1)
 def test_peer_key_psk_address_and_port_changes_are_detected(self):
  original=m.signature(m.parse(config()))
  for old,new in ((key(1),key(4)),(key(2),key(4)),(key(3),key(4)),('51442','51443'),('10.66.66.2','10.66.66.3')):
   self.assertNotEqual(original,m.signature(m.parse(config().replace(old,new))))
 def test_runtime_endpoint_change_does_not_change_identity(self):
  self.assertEqual(m.signature(m.parse(config())),m.signature(m.parse(config()+'Endpoint = 192.0.2.1:12345\n')))
 def test_duplicate_peer_and_bad_key_fail(self):
  with self.assertRaises(ValueError):m.parse(config()+'[Peer]\nPublicKey = '+key(2)+'\nAllowedIPs = 10.66.66.3/32\n')
  with self.assertRaises(ValueError):m.parse(config().replace(key(1),'bad-key'))
 def test_unknown_section_and_missing_address_fail(self):
  with self.assertRaises(ValueError):m.parse(config()+'[Other]\nthing=yes\n')
  with self.assertRaises(ValueError):m.parse(config().replace('AllowedIPs = 10.66.66.2/32, fd42:42:42::2/128\n',''))
 def test_restore_proof_not_accepted_without_matching_fingerprint(self):
  with tempfile.TemporaryDirectory() as d:
   p=pathlib.Path(d);(p/'baseline.conf').write_text(config());(p/'proof.json').write_text(json.dumps({'schema_version':1,'passed':True,'configuration_signature':'wrong','legacy_peers_preserved':True,'psks_preserved':True,'reconcile_passed':True,'cold_restore_passed':True,'target_config_sha256':'wrong'}))
   with self.assertRaises(ValueError):m.check_proof(p/'baseline.conf',p/'proof.json','expected')

class ModelTests(unittest.TestCase):
 def model(self):
  common={'image':'sha256:'+'a'*64,'cap_add':['NET_ADMIN'],'cap_drop':['ALL'],'healthcheck':{'test':['CMD','true']}}
  return {'services':{'wireguard':{**common,'ports':[{'published':'51442','protocol':'udp'}]},'api':{**common,'network_mode':'service:wireguard','environment':{'VPN_WIREGUARD_MODE':'kernel','VPN_TERMS_VERSION':'production-v1','VPN_ADMIN_USERNAME':'owner'}}}}
 def test_valid_static_production_model(self):m.validate_model(self.model())
 def test_lab_privileged_socket_build_and_public_api_rejected(self):
  for alteration in ('lab','privileged','socket','build','public_api','host'):
   model=self.model()
   if alteration=='lab':model['services']['api']['environment']['VPN_ADMIN_USERNAME']='lab-admin'
   if alteration=='privileged':model['services']['api']['privileged']=True
   if alteration=='socket':model['services']['api']['volumes']=[{'source':'/var/run/docker.sock','target':'/var/run/docker.sock'}]
   if alteration=='build':model['services']['api']['build']={'context':'.'}
   if alteration=='public_api':model['services']['wireguard']['ports'].append({'published':'8080','protocol':'tcp','host_ip':'0.0.0.0'})
   if alteration=='host':model['services']['api']['network_mode']='host'
   with self.subTest(alteration=alteration),self.assertRaises(ValueError):m.validate_model(model)
 def test_extra_env_file_and_missing_capability_rejected(self):
  for field,value in (('env_file',['/home/codex/env']),('cap_add',[])):
   model=self.model();model['services']['api'][field]=value
   with self.subTest(field=field),self.assertRaises(ValueError):m.validate_model(model)

class ProofTests(unittest.TestCase):
 def proof(self,cfg_hash):
  return {'schema_version':1,'configuration_signature':m.signature(m.parse(config())),'target_config_sha256':cfg_hash,**{k:True for k in ('passed','legacy_peers_preserved','psks_preserved','reconcile_passed','cold_restore_passed','routing_firewall_reviewed','production_terms_reviewed')}}
 def test_matching_proof_and_missing_psk_restore_claim(self):
  with tempfile.TemporaryDirectory() as d:
   root=pathlib.Path(d);(root/'baseline.conf').write_text(config());data=self.proof('expected');(root/'proof.json').write_text(json.dumps(data));m.check_proof(root/'baseline.conf',root/'proof.json','expected')
   data['psks_preserved']=False;(root/'proof.json').write_text(json.dumps(data))
   with self.assertRaises(ValueError):m.check_proof(root/'baseline.conf',root/'proof.json','expected')

class WrapperTests(unittest.TestCase):
 def test_named_wrappers_route_to_subcommand_and_help_without_root(self):
  import os,subprocess
  root=PATH.parent
  for name in ('01-inspect','02-prepare','03-cutover','04-verify','05-rollback'):
   p=root/(name+'.sh');self.assertTrue(p.is_file())
   help_result=subprocess.run(['/bin/bash',str(p),'--help'],capture_output=True,text=True)
   self.assertEqual(help_result.returncode,0);self.assertIn('Darc VPN migration bundle',help_result.stdout)
   if os.geteuid()!=0:
    result=subprocess.run(['/bin/bash',str(p)],capture_output=True,text=True)
    self.assertEqual(result.returncode,1);self.assertIn('Root required',result.stderr)

if __name__=='__main__':unittest.main()

class PrivateTestProofTests(unittest.TestCase):
 def test_technical_proof_allows_terms_deferred_only_for_private_test(self):
  import hashlib
  with tempfile.TemporaryDirectory() as d:
   p=pathlib.Path(d);(p/'baseline.conf').write_text(config());model={'services':{'api':{'environment':{'VPN_MODE':'container-private-test'}}}}
   raw=json.dumps(model);(p/'model.json').write_text(raw);sha=hashlib.sha256(raw.encode()).hexdigest()
   proof={'schema_version':1,'configuration_signature':m.signature(m.parse(config())),'target_config_sha256':sha,'private_test_only':True,'terms_deferred':True,**{k:True for k in ('passed','legacy_peers_preserved','psks_preserved','reconcile_passed','cold_restore_passed')}}
   (p/'proof.json').write_text(json.dumps(proof));m.check_proof(p/'baseline.conf',p/'proof.json',sha,p/'model.json')
   model['services']['api']['environment']['VPN_MODE']='container-private';raw=json.dumps(model);(p/'model.json').write_text(raw);sha=hashlib.sha256(raw.encode()).hexdigest();proof['target_config_sha256']=sha;(p/'proof.json').write_text(json.dumps(proof))
   with self.assertRaises(ValueError):m.check_proof(p/'baseline.conf',p/'proof.json',sha,p/'model.json')
