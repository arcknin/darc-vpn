#!/usr/bin/env python3
"""Real disposable acceptance; never prints API responses, tokens or configurations."""
import contextlib,http.client,ipaddress,json,os,pathlib,re,subprocess,tempfile,time,urllib.error,urllib.request
ROOT=pathlib.Path(__file__).resolve().parents[2]
ENV=dict(line.split('=',1) for line in (ROOT/'.lab.env').read_text().splitlines() if '=' in line)
PROJECT=ENV['LAB_ID'];BASE='http://127.0.0.1:'+ENV['LAB_API_PORT'];CLIENT=PROJECT+'-client'
COMPOSE=['docker','compose','--env-file',str(ROOT/'.lab.env'),'-p',PROJECT,'-f',str(ROOT/'infra/wireguard-stack/compose.yaml')]
RESULTS=[]
def run(args,check=True):
    p=subprocess.run(args,text=True,capture_output=True,cwd=ROOT)
    if check and p.returncode:
        detail=p.stderr.strip()[:600]
        for value in ENV.values():
            if len(value)>20:detail=detail.replace(value,'[redacted]')
        detail=re.sub(r'[A-Za-z0-9+/]{43}=','[key redacted]',detail)
        operation=' '.join(args[3:5]) if args[:2]==['docker','exec'] else ' '.join(args[:2])
        raise RuntimeError('command '+operation+' exit '+str(p.returncode)+': '+detail)
    return p

def api(method,path,body=None,token=None,expected=200):
    data=json.dumps(body).encode() if body is not None else None
    req=urllib.request.Request(BASE+path,data=data,method=method,headers={'Content-Type':'application/json',**({'Authorization':'Bearer '+token} if token else {})})
    try:
        with urllib.request.urlopen(req,timeout=15) as r:status=r.status;payload=r.read()
    except urllib.error.HTTPError as r:status=r.code;payload=r.read()
    if status!=expected:raise RuntimeError('HTTP assertion failed: '+method+' '+path+' status '+str(status))
    return json.loads(payload) if payload and payload[:1] in (b'{',b'[') else None

def login(kind,user,password):return api('POST','/api/v1/auth/'+kind+'/login',{'username':user,'password':password})['accessToken']
def ready(timeout=120):
    deadline=time.monotonic()+timeout
    while time.monotonic()<deadline:
        try:api('GET','/readyz');return
        except (RuntimeError,urllib.error.URLError,OSError,http.client.HTTPException):time.sleep(2)
    raise RuntimeError('readiness deadline exceeded')
def mark(name):RESULTS.append(name);print('PASS '+name,flush=True)
def service(name):return run(COMPOSE+['ps','-q',name]).stdout.strip()
def inspect(container):return json.loads(run(['docker','inspect',container]).stdout)[0]
def client_exec(args,check=True):return run(['docker','exec',CLIENT]+args,check)
def private(account,user,password):
    token=login('friend',user,password);doc=api('GET','/api/v1/terms/current',token=token)
    api('GET','/api/v1/profiles/'+account['primaryProfileId']+'/private',token=token,expected=428)
    api('POST','/api/v1/terms/acceptance',{'accepted':True,'version':doc['version'],'sha256':doc['sha256']},token,201)
    return token,api('GET','/api/v1/profiles/'+account['primaryProfileId']+'/private',token=token)
def configure(profile):
    transport=ENV.get('LAB_ENDPOINT','187.124.240.109');ipaddress.ip_address(transport)
    content='[Interface]\nPrivateKey = '+profile['privateKey']+'\n[Peer]\nPublicKey = '+profile['serverPublicKey']+'\nEndpoint = '+transport+':'+str(profile['endpointPort'])+'\nAllowedIPs = 0.0.0.0/0, ::/0\nPersistentKeepalive = 5\n'
    written=subprocess.run(['docker','exec','-i',CLIENT,'sh','-c','umask 077; cat > /tmp/client.conf'],input=content,text=True,capture_output=True)
    if written.returncode:raise RuntimeError('client configuration write failed')
    # Namespace belongs exclusively to disposable client. Route transport outside tunnel.
    info=inspect(CLIENT);net=next(iter(info['NetworkSettings']['Networks'].values()));gateway=net['Gateway']
    client_exec(['sh','-c','set -eu; ip link del wgtest 2>/dev/null || true; ip link add wgtest type wireguard; wg setconf wgtest /tmp/client.conf; rm /tmp/client.conf'])
    for value in profile['address'].split(','):
        prefix=ipaddress.ip_interface(value.strip());client_exec(['ip']+(['-6'] if prefix.version==6 else [])+['address','add',str(prefix),'dev','wgtest'])
    client_exec(['ip','link','set','wgtest','mtu','1420','up'])
    client_exec(['ip','route','replace',transport+'/32','via',gateway,'dev','eth0'])
    for family,prefixes in (([],('0.0.0.0/1','128.0.0.0/1')),(['-6'],('::/1','8000::/1'))):
        for prefix in prefixes:client_exec(['ip']+family+['route','replace',prefix,'dev','wgtest'])
def traffic():
    for addr in ('10.77.77.1','fd42:77:77::1'):
        deadline=time.monotonic()+75
        while True:
            result=client_exec(['ping','-c','2','-W','3',addr],False)
            if result.returncode==0:break
            if time.monotonic()>=deadline:
                raw=client_exec(['wg','show','wgtest','allowed-ips'],False).stdout
                allowed=[line.split('\t',1)[-1] for line in raw.splitlines()]
                routes=client_exec(['ip']+(['-6'] if ':' in addr else [])+['route','get',addr],False).stdout.strip()
                print('CLIENT_DIAGNOSTIC '+json.dumps({'target':addr,'allowedIPs':allowed,'route':routes}),flush=True)
                raise RuntimeError('tunnel ping deadline exceeded '+addr+': '+result.stderr.strip())
            time.sleep(2)
    for family in ('-4','-6'):
        p=client_exec(['curl',family,'--fail','--silent','--show-error','--max-time','30','--output','/dev/null','--write-out','%{http_code}','https://www.cloudflare.com'])
        if p.stdout!='200':raise RuntimeError('HTTPS tunnel test failed '+family)
    # Confirm packet paths; counters before/after must show actual forwarding.
    for family,target in (([],'1.1.1.1'),(['-6'],'2606:4700:4700::1111')):
        if 'wgtest' not in client_exec(['ip']+family+['route','get',target]).stdout:raise RuntimeError('egress did not route through tunnel')
    stamps=client_exec(['wg','show','wgtest','latest-handshakes']).stdout.split()
    if len(stamps)!=2 or int(stamps[1])==0:raise RuntimeError('handshake absent')
def main():
    os.umask(0o077)
    if (ROOT/'.lab-owner').read_text().strip()!=PROJECT:raise RuntimeError('ownership mismatch')
    ready();admin=login('admin','lab-admin',ENV['LAB_ADMIN_PASSWORD'])
    def create(user):return api('POST','/api/v1/admin/accounts',{'username':user,'password':'fixture-password-123','primaryDisplayName':'Primary'},admin,201)
    a=create('lab-friend-a');extra=api('POST','/api/v1/admin/accounts/'+a['id']+'/profiles',{'displayName':'Extra'},admin,201);b=create('lab-friend-b')
    if a['profiles'][0].get('provisioningState')!='ready' or extra.get('provisioningState')!='ready':raise RuntimeError('profiles not ready')
    token,pa=private(a,'lab-friend-a','fixture-password-123');_,pb=private(b,'lab-friend-b','fixture-password-123')
    if pa['serverPublicKey']==pa['peerPublicKey'] or pa['endpointPort']!=int(ENV['LAB_WG_PORT']) or pa['address']==pb['address']:raise RuntimeError('invalid identity/port/allocation')
    api('POST','/api/v1/admin/accounts',{'username':'forbidden','password':'fixture','primaryDisplayName':'No'},token,403)
    api('GET','/api/v1/profiles/'+b['primaryProfileId']+'/private',token=token,expected=404)
    mark('API authorization, terms, distinct identities and dual-stack reservations')
    wg=inspect(service('wireguard'));api_info=inspect(service('api'))
    if api_info['HostConfig']['NetworkMode']!='container:'+wg['Id']:raise RuntimeError('namespace not shared')
    for info in (wg,api_info):
        if info['HostConfig']['Privileged'] or 'NET_ADMIN' not in [c.removeprefix('CAP_') for c in info['HostConfig']['CapAdd']] or any(m['Destination']=='/var/run/docker.sock' for m in info['Mounts']):raise RuntimeError('unsafe runtime configuration')
    original_network=next(iter(wg['NetworkSettings']['Networks']))
    run(['docker','run','-d','--name',CLIENT,'--label','darc.vpn.lab='+PROJECT,'--cap-drop','ALL','--cap-add','NET_ADMIN','--cap-add','NET_RAW','--network',original_network,'--entrypoint','/bin/sh',PROJECT+'-wireguard:1','-c','while :; do sleep 3600; done'])
    try:
        configure(pa);traffic();mark('real kernel handshake and IPv4/IPv6 tunnel plus verified HTTPS egress')
        op=api('POST','/api/v1/admin/accounts/'+a['id']+'/block',token=admin,expected=202)
        if op['status']!='completed' or len(op['peers'])!=2:raise RuntimeError('revocation incomplete')
        api('GET','/api/v1/profiles/'+a['primaryProfileId']+'/private',token=token,expected=401)
        if client_exec(['ping','-c','1','-W','2','10.77.77.1'],False).returncode==0:raise RuntimeError('revoked traffic still passes')
        mark('all account peers removed; session and live traffic rejected')
        configure(pb);traffic()
        run(COMPOSE+['restart','api']);ready();traffic();mark('API restart recovered active peers')
        run(COMPOSE+['restart','wireguard']);run(COMPOSE+['up','-d','--force-recreate','api','--wait','--wait-timeout','150']);ready();traffic();mark('WireGuard restart and API relink recovered active peers')
        run(COMPOSE+['up','-d','--force-recreate','--wait','--wait-timeout','150']);ready();traffic();mark('joint container recreation preserved server identity and active peers')
        admin=login('admin','lab-admin',ENV['LAB_ADMIN_PASSWORD']);op=api('GET','/api/v1/admin/accounts/'+a['id']+'/revocation',token=admin)
        if op['status']!='completed':raise RuntimeError('revocation state lost')
        configure(pa)
        if client_exec(['ping','-c','1','-W','2','10.77.77.1'],False).returncode==0:raise RuntimeError('revoked peer resurrected')
        mark('revoked peer stays rejected after recreation')
        # NET_ADMIN denied: do not let startup fake its way out.
        denied=run(['docker','run','--rm','--network','container:'+service('wireguard'),'--cap-drop','ALL','--entrypoint','wg',PROJECT+'-wireguard:1','show','wg0','public-key'],check=False)
        if denied.returncode==0:raise RuntimeError('kernel query succeeded without NET_ADMIN')
        mark('kernel access denied without NET_ADMIN')
        logs=run(COMPOSE+['logs','--no-color']).stdout
        secrets=[ENV['LAB_ADMIN_PASSWORD'],ENV['LAB_AUDIT_TOKEN'],admin,token,pa['privateKey'],pb['privateKey']]
        if any(value and value in logs for value in secrets):raise RuntimeError('secret found in runtime logs')
        mark('runtime logs contain no fixture tokens, passwords or private keys')
    finally:
        run(['docker','rm','-f',CLIENT],check=False)
    (ROOT/'acceptance-results.json').write_text(json.dumps({'results':RESULTS,'production_cutover':False,'host_reboot_tested':False},indent=2)+'\n')
if __name__=='__main__':
    try:main()
    except Exception as exc:print('FAIL '+str(exc),flush=True);raise SystemExit(1)
