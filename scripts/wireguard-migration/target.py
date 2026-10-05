#!/usr/bin/env python3
"""Root-only target materialization. No raw key/env output."""
import hashlib,importlib.util,ipaddress,json,os,pathlib,re,secrets,shutil,subprocess,sys
spec=importlib.util.spec_from_file_location('inventory',pathlib.Path(__file__).with_name('inventory.py'));inv=importlib.util.module_from_spec(spec);spec.loader.exec_module(inv)
def write_private(path,text):
    p=pathlib.Path(path)
    if p.exists() and p.is_symlink():raise ValueError('symlink rejected')
    p.write_text(text);p.chmod(0o600)
def run(args):
    p=subprocess.run(args,text=True,capture_output=True)
    if p.returncode:raise ValueError('target validation command failed')
    return p.stdout

def export(config,interface,root):
    root=inv.secure(root,True);parsed=inv.parse(inv.secure(config).read_text())
    addresses={a['local']+'/'+str(a['prefixlen']) for item in json.loads(inv.secure(interface).read_text()) for a in item.get('addr_info',[]) if a.get('scope')!='link'}
    if addresses!={'10.66.66.1/24','fd42:42:42::1/64'} or parsed['port']!=51442 or parsed['fwmark']!=0:raise ValueError('source differs from supported production target')
    key=root/'identity/server.key'
    if key.exists():
        if inv.secure(str(key)).read_text().strip()!=parsed['private']:raise ValueError('existing target identity conflicts')
    else:write_private(key,parsed['private']+'\n')
    peers=[{'publicKey':p['public'],'allowedIPs':p['allowed'],'presharedKey':p['psk'],'keepalive':p['keepalive']} for p in parsed['peers']]
    write_private(root/'legacy-import.json',json.dumps({'schemaVersion':1,'peers':peers}))

def check_mounts(path):
    ids=run(['docker','ps','-q']).split()
    if not ids:return
    for container in json.loads(run(['docker','inspect',*ids])):
        for mount in container.get('Mounts',[]):
            src=mount.get('Source','')
            if src and (pathlib.Path(path)==pathlib.Path(src) or pathlib.Path(src) in pathlib.Path(path).parents or pathlib.Path(path) in pathlib.Path(src).parents):raise ValueError('running container can access target data; offline import refused')

def materialize(bundle,root,state,run_id,api,wg,settings,private_test="0"):
    root=inv.secure(root,True);state=inv.secure(state,True);bundle=inv.secure(bundle,True)
    if (root/'compose.yaml').is_symlink():raise ValueError('symlink target rejected')
    shutil.copyfile(bundle/'payload/infra/wireguard-production/compose.yaml',root/'compose.yaml')
    write_private(root/'image-ids.json',json.dumps({'api':api,'wireguard':wg}))
    baseline=state/'backups'/run_id/'live.conf'
    # Actual comparison was performed by Bash; this record does not claim final terms/network reviewed.
    proof={'schema_version':1,'configuration_signature':inv.signature(inv.parse(baseline.read_text())),'passed':True,'legacy_peers_preserved':True,'psks_preserved':True,'reconcile_passed':True,'cold_restore_passed':True,'routing_firewall_reviewed':False,'production_terms_reviewed':False,'target_config_sha256':None}
    write_private(root/'cold-restore-proof.json',json.dumps(proof,indent=2)+'\n')
    example={'terms_file':'/root/terms-final.txt','terms_version':'production-v1','admin_username':'vpn-admin','routing_firewall_reviewed':False,'production_terms_reviewed':False}
    write_private(root/'production-settings.example.json',json.dumps(example,indent=2)+'\n')
    private_test=private_test=="1"
    if not settings and not private_test:
        print('LEGACY_READY; production Compose/image IDs saved. Final plan pending root-reviewed settings/terms. Example: '+str(root/'production-settings.example.json'))
        return
    cfg=json.loads(inv.secure(settings).read_text()) if settings else {}
    version="private-technical-test" if private_test else cfg.get('terms_version','')
    username=cfg.get('admin_username','vpn-admin')
    if not re.fullmatch(r'[a-zA-Z0-9_.-]{1,80}',version) or not re.fullmatch(r'[a-zA-Z0-9_.-]{1,80}',username):raise ValueError('invalid terms version/admin name')
    if not private_test:
        if any(cfg.get(k) is not True for k in ('routing_firewall_reviewed','production_terms_reviewed')):raise ValueError('root review flags missing; cold-restore proof retained, no cutover plan')
        terms=inv.secure(cfg.get('terms_file',''))
        if username=='lab-admin' or 'lab-fictional' in version:raise ValueError('lab fixtures rejected')
        if (root/'terms/terms.txt').is_symlink():raise ValueError('symlink terms rejected')
        shutil.copyfile(terms,root/'terms/terms.txt');(root/'terms/terms.txt').chmod(0o600)
    creds=root/'admin-credentials.json'
    if creds.exists():credentials=json.loads(inv.secure(str(creds)).read_text())
    else:
        credentials={'password':secrets.token_hex(24),'audit_token':secrets.token_hex(24)};write_private(creds,json.dumps(credentials))
    values={'PROD_ROOT':str(root),'PROD_API_IMAGE':api,'PROD_WG_IMAGE':wg,'PROD_TERMS_VERSION':version,'PROD_ADMIN_USERNAME':username,'PROD_ADMIN_PASSWORD':credentials['password'],'PROD_AUDIT_TOKEN':credentials['audit_token']}
    env=''.join(k+'='+v+'\n' for k,v in values.items());write_private(root/'production.env',env)
    # Validate real terms with the same Go terms loader by booting only API in an isolated cold WG namespace.
    # Bash performs this runtime step separately before accepting this proof.
    model=run(['docker','compose','--project-directory',str(root),'--env-file',str(root/'production.env'),'-p','darc-vpn-prod-main','-f',str(root/'compose.yaml'),'config','--format','json'])
    parsed=json.loads(model)
    if private_test:
        api_service=parsed['services']['api'];api_service['environment']['VPN_MODE']='container-private-test'
        for k in ('VPN_TERMS_VERSION','VPN_TERMS_PATH'):api_service['environment'].pop(k,None)
        api_service['volumes']=[v for v in api_service.get('volumes',[]) if v.get('target')!='/terms']
        # Freeze a non-interpolated, fully resolved test model; it retains loopback publishing.
        def escape(v):
            if isinstance(v,str):return v.replace('$','$$')
            if isinstance(v,list):return [escape(x) for x in v]
            if isinstance(v,dict):return {k:escape(x) for k,x in v.items()}
            return v
        write_private(root/'compose-private-test.json',json.dumps(escape(parsed),sort_keys=True,indent=2))
        model=run(['docker','compose','--project-directory',str(root),'-p','darc-vpn-prod-main','-f',str(root/'compose-private-test.json'),'config','--format','json']);parsed=json.loads(model)
    inv.validate_model(parsed)
    occupied=run(['ss','-H','-ltn'])
    if re.search(r':18082\b',occupied):raise ValueError('target API loopback port busy')
    existing=[]
    ids=run(['docker','network','ls','-q']).split()
    if ids:
        for network in json.loads(run(['docker','network','inspect',*ids])):
            existing.extend(v['Subnet'] for v in (network.get('IPAM',{}).get('Config') or []) if v.get('Subnet'))
    for family in ('-4','-6'):
        existing.extend(r['dst'] for r in json.loads(run(['ip',family,'-j','route','show','table','all'])) if r.get('dst') and r['dst']!='default')
    for value in existing:
        try:p=ipaddress.ip_network(value,strict=False)
        except ValueError:continue
        for wanted in ('172.30.88.0/24','fd42:30:88::/64'):
            w=ipaddress.ip_network(wanted)
            if p.version==w.version and p.overlaps(w):raise ValueError('target bridge network overlaps existing routes')
    write_private(root/'runtime-api.env',''.join(k+'='+str(v)+'\n' for k,v in parsed['services']['api']['environment'].items()))
    # Produce a candidate only. Actual API startup/readiness must pass before prepare can consume proof.
    proof.update(routing_firewall_reviewed=not private_test,production_terms_reviewed=not private_test,private_test_only=private_test,terms_deferred=private_test,target_config_sha256=hashlib.sha256(model.encode()).hexdigest(),passed=False)
    write_private(root/'restore-proof.pending.json',json.dumps(proof,indent=2)+'\n')
    write_private(root/'production-plan.json',json.dumps({'schema_version':1,'project':'darc-vpn-prod-main','compose_file':str(root/('compose-private-test.json' if private_test else 'compose.yaml')),'env_file':str(root/'production.env'),'restore_proof':str(root/'restore-proof.json'),'traffic_proof':str(root/'client-verification.json'),'rollback_delay_seconds':600},indent=2)+'\n')
    print('PRIVATE_TEST_RENDERED; no terms required; API runtime test still required before technical proof.' if private_test else 'SETTINGS_RENDERED; API runtime test still required before final proof.')

def main():
    command=sys.argv[1]
    if command=='export':export(*sys.argv[2:])
    elif command=='check-mounts':check_mounts(sys.argv[2])
    elif command=='materialize':materialize(*sys.argv[2:])
    elif command=='finalize-proof':
        root=inv.secure(sys.argv[2],True);data=json.loads(inv.secure(str(root/'restore-proof.pending.json')).read_text());data['passed']=True;write_private(root/'restore-proof.json',json.dumps(data,indent=2)+'\n');print('TARGET_PREPARATION_OK; run 02-prepare with production-plan.json next.')
    else:raise ValueError('unsupported operation')
if __name__=='__main__':
    try:main()
    except Exception:
        print('Target preparation rejected: input/identity/settings/permissions/network conflict. Private state retained; no secret printed.',file=sys.stderr);sys.exit(1)
