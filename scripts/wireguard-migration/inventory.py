#!/usr/bin/env python3
"""Restricted parsing; never print configurations, private keys, PSKs or Docker env."""
import base64,hashlib,ipaddress,json,os,pathlib,re,stat,sys

class SafeError(ValueError):
    pass

def parse(text):
    interface={};peers=[];section=None;seen=set();hooks={k:0 for k in ('PreUp','PostUp','PreDown','PostDown')}
    for line in text.splitlines():
        line=line.split('#',1)[0].strip()
        if not line:continue
        if line in ('[Interface]','[Peer]'):
            if line=='[Interface]':
                if section is not None:raise SafeError('duplicate/misplaced interface section')
                section=interface
            else:section={};peers.append(section)
            continue
        if section is None or '=' not in line or line.startswith('['):raise SafeError('invalid WireGuard configuration structure')
        name,value=(x.strip() for x in line.split('=',1))
        if name in hooks:hooks[name]+=1;continue
        if name=='AllowedIPs':section[name]=section.get(name,'')+(',' if name in section else '')+value
        elif name in section:raise SafeError('duplicate WireGuard field')
        else:section[name]=value
    def key(value):
        try:raw=base64.b64decode(value,validate=True)
        except Exception:raise SafeError('invalid WireGuard key') from None
        if len(raw)!=32:raise SafeError('invalid WireGuard key length')
        return value
    private=key(interface.get('PrivateKey',''));port=int(interface.get('ListenPort','0'))
    if not 1<=port<=65535:raise SafeError('explicit listen port required')
    out=[]
    for p in peers:
        public=key(p.get('PublicKey',''))
        if public in seen:raise SafeError('duplicate peer identity')
        seen.add(public)
        allowed=sorted(set(str(ipaddress.ip_network(x.strip(),strict=False)) for x in p.get('AllowedIPs','').split(',') if x.strip()))
        if not allowed:raise SafeError('peer without AllowedIPs')
        psk=p.get('PresharedKey','')
        if psk:key(psk)
        if psk==base64.b64encode(bytes(32)).decode():psk=''
        keep=p.get('PersistentKeepalive','0');keep=0 if keep=='off' else int(keep)
        out.append({'public':public,'psk':psk,'allowed':allowed,'keepalive':keep})
    fwmark=interface.get('FwMark','0');fwmark=0 if fwmark=='off' else int(fwmark,0)
    return {'private':private,'port':port,'fwmark':fwmark,'peers':sorted(out,key=lambda p:p['public']),'hooks':hooks}

def signature(parsed):return hashlib.sha256(json.dumps({k:v for k,v in parsed.items() if k!='hooks'},sort_keys=True,separators=(',',':')).encode()).hexdigest()
def summary(parsed):return {'configuration_signature':signature(parsed),'listen_port':parsed['port'],'peer_count':len(parsed['peers']),'psk_count':sum(bool(p['psk']) for p in parsed['peers']),'hook_counts':parsed['hooks'],'fwmark_nonzero':parsed['fwmark']!=0}
def secure(path,directory=False):
    p=pathlib.Path(path)
    if not p.is_absolute() or p.resolve()!=p:raise SafeError('path must be absolute without symlinks')
    for item in [p,*p.parents]:
        s=item.lstat()
        if s.st_uid!=0 or s.st_mode & 0o022:raise SafeError('path must be root-owned and not writable by group/others')
    if directory and not p.is_dir():raise SafeError('directory required')
    if not directory and not p.is_file():raise SafeError('regular file required')
    return p

def check_proof(baseline,proof,config_hash,model_file=None):
    data=json.loads(pathlib.Path(proof).read_text())
    if data.get('schema_version')!=1 or any(data.get(k) is not True for k in ('passed','legacy_peers_preserved','psks_preserved','reconcile_passed','cold_restore_passed')):raise SafeError('legacy import/reconcile/cold-restore evidence required')
    if data.get('configuration_signature')!=signature(parse(pathlib.Path(baseline).read_text())) or data.get('target_config_sha256')!=config_hash:raise SafeError('restore evidence does not match current peers or target configuration')
    private_test=False
    if model_file is not None:
        model=json.loads(pathlib.Path(model_file).read_text())
        private_test=model.get('services',{}).get('api',{}).get('environment',{}).get('VPN_MODE')=='container-private-test'
    if not (private_test and data.get('private_test_only') is True and data.get('terms_deferred') is True):
        if any(data.get(k) is not True for k in ('routing_firewall_reviewed','production_terms_reviewed')):raise SafeError('final production review required outside private test mode')
    return data

def validate_model(model):
    services=model.get('services',{})
    if set(services)!= {'api','wireguard'}:raise SafeError('target must contain exactly API and WireGuard services')
    for s in services.values():
        if s.get('build') or s.get('privileged') or s.get('network_mode')=='host':raise SafeError('no build/privileged/host networking during cutover')
        if not re.fullmatch(r'sha256:[a-f0-9]{64}',s.get('image','')):raise SafeError('target image IDs must be pinned and already built')
        caps={c.removeprefix('CAP_') for c in s.get('cap_add',[])}
        if caps!={'NET_ADMIN'}:raise SafeError('unsupported target capabilities')
        if 'ALL' not in s.get('cap_drop',[]):raise SafeError('target must drop all default capabilities')
        if s.get('env_file'):raise SafeError('additional service env files unsupported; use reviewed root plan env only')
        if not s.get('healthcheck') or s['healthcheck'].get('disable'):raise SafeError('target healthchecks required')
        for mount in s.get('volumes',[]):
            if 'docker.sock' in mount.get('source','') or mount.get('target') in ('/','/etc/wireguard'):raise SafeError('host Docker/WireGuard mounts forbidden')
            if mount.get('type')=='bind':secure(mount['source'],directory=pathlib.Path(mount['source']).is_dir())
    if services['api'].get('network_mode')!='service:wireguard' or services['api'].get('ports') or services['api'].get('networks'):raise SafeError('API must share WireGuard namespace')
    if set(services['api'].get('networks',{})):raise SafeError('API independent networking forbidden')
    ports=services['wireguard'].get('ports',[])
    if not any(str(p.get('published'))=='51442' and p.get('protocol')=='udp' for p in ports):raise SafeError('target must preserve UDP 51442')
    for p in ports:
        if p.get('protocol','tcp')=='tcp' and p.get('host_ip') not in ('127.0.0.1','::1'):raise SafeError('API TCP publish must remain host-loopback')
    for kind in ('secrets','configs'):
        for definition in model.get(kind,{}).values():
            if definition.get('file'):secure(definition['file'])
    env=services['api'].get('environment',{})
    if env.get('VPN_WIREGUARD_MODE')!='kernel':raise SafeError('target must use real kernel controller')
    if 'lab-fictional' in str(env.get('VPN_TERMS_VERSION','')) or env.get('VPN_ADMIN_USERNAME')=='lab-admin':raise SafeError('laboratory fixtures cannot become production')

def main():
    command=sys.argv[1]
    if command=='freeze':
        data=json.loads(pathlib.Path(sys.argv[2]).read_text())
        def escape(v):
            if isinstance(v,str):return v.replace('$','$$')
            if isinstance(v,list):return [escape(x) for x in v]
            if isinstance(v,dict):return {k:escape(x) for k,x in v.items()}
            return v
        print(json.dumps(escape(data),sort_keys=True,indent=2));return
    if command=='secure':secure(sys.argv[2],directory=len(sys.argv)>3);return
    if command=='get':
        value=json.loads(pathlib.Path(sys.argv[2]).read_text())[sys.argv[3]]
        if not isinstance(value,(str,int)) or '\n' in str(value):raise SafeError('invalid scalar plan value')
        print(value);return
    if command=='summary':
        report=summary(parse(pathlib.Path(sys.argv[2]).read_text()))
        if len(sys.argv)>3:report['hook_counts']=parse(pathlib.Path(sys.argv[3]).read_text())['hooks']
        if len(sys.argv)>4:
            report['interface_addresses']=[a['local']+'/'+str(a['prefixlen']) for item in json.loads(pathlib.Path(sys.argv[4]).read_text()) for a in item.get('addr_info',[])]
        print(json.dumps(report,sort_keys=True));return
    if command=='compare':
        if signature(parse(pathlib.Path(sys.argv[2]).read_text()))!=signature(parse(pathlib.Path(sys.argv[3]).read_text())):raise SafeError('identity/port/peers/PSKs/AllowedIPs drift')
        return
    if command=='plan':
        plan=json.loads(secure(sys.argv[2]).read_text())
        if plan.get('schema_version')!=1 or not re.fullmatch(r'darc-vpn-prod-[a-z0-9-]+',str(plan.get('project',''))):raise SafeError('invalid production plan/project')
        for field in ('compose_file','env_file','restore_proof'):secure(plan.get(field,''))
        if secure(plan['env_file']).stat().st_mode & 0o077:raise SafeError('environment file must be private to root')
        traffic=pathlib.Path(plan.get('traffic_proof',''))
        if not traffic.is_absolute():raise SafeError('traffic attestation path required')
        secure(str(traffic.parent),directory=True)
        if not isinstance(plan.get('rollback_delay_seconds'),int) or not 300<=plan['rollback_delay_seconds']<=1800:raise SafeError('rollback delay must be 300..1800 seconds')
        print('PLAN_PATHS_VALID');return
    if command=='model':validate_model(json.loads(pathlib.Path(sys.argv[2]).read_text()));return
    if command=='proof':check_proof(sys.argv[2],secure(sys.argv[3]),hashlib.sha256(pathlib.Path(sys.argv[4]).read_bytes()).hexdigest(),sys.argv[4]);return
    if command=='traffic':
        evidence=json.loads(secure(sys.argv[2]).read_text())
        if evidence.get('run_id')!=sys.argv[3] or any(evidence.get(k) is not True for k in ('existing_client_connected','ipv4_tunnel','ipv6_tunnel','https_ipv4','https_ipv6')):raise SafeError('root-owned external client test attestation missing/incomplete')
        return
    raise SafeError('unknown inventory command')
if __name__=='__main__':
    try:main()
    except Exception as exc:
        # Parser exceptions may contain a raw line/key/address. Never echo them.
        if isinstance(exc,SafeError):print(str(exc),file=sys.stderr)
        else:print('Validation rejected; incomplete/unsafe input. No configuration or secret printed.',file=sys.stderr)
        raise SystemExit(1)
