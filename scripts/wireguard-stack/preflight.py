#!/usr/bin/env python3
"""Read-only inventory. Run on the intended Linux VPS before creating resources."""
import ipaddress,json,os,re,subprocess,sys

def run(*args):
    p=subprocess.run(args,text=True,capture_output=True)
    if p.returncode: raise RuntimeError('preflight command failed: '+args[0])
    return p.stdout.strip()

def main():
    if run('id','-un')!='codex' or run('hostname')!='srv1519215': raise RuntimeError('unexpected target identity')
    if not os.path.isdir('/sys/module/wireguard'): raise RuntimeError('WireGuard kernel unavailable; host module changes not authorized')
    run('docker','compose','version')
    if run('systemctl','is-active','wg-quick@wg0')!='active': raise RuntimeError('production wg0 is not active')
    sockets=run('ss','-H','-lun')+'\n'+run('ss','-H','-ltn')
    for port in (51443,18081):
        if re.search(r':'+str(port)+r'\b',sockets):raise RuntimeError('lab port unavailable: '+str(port))
    wanted=[ipaddress.ip_network(p) for p in ('172.30.77.0/24','fd42:30:77::/64','10.77.77.0/24','fd42:77:77::/64')]
    ids=run('docker','network','ls','-q').split(); existing=[]
    if ids:
        for network in json.loads(run('docker','network','inspect',*ids)):
            existing.extend(c['Subnet'] for c in (network.get('IPAM',{}).get('Config') or []) if c.get('Subnet'))
    for family in ('-4','-6'):
        for r in json.loads(run('ip',family,'-j','route','show','table','all')):
            if r.get('dst') and r['dst']!='default':existing.append(r['dst'])
    for addr in json.loads(run('ip','-j','address','show')):
        for a in addr.get('addr_info',[]):existing.append(a['local']+'/'+str(a['prefixlen']))
    for prefix in existing:
        try:p=ipaddress.ip_network(prefix,strict=False)
        except ValueError:continue
        if any(p.version==w.version and p.overlaps(w) for w in wanted):raise RuntimeError('lab subnet overlaps existing network')
    print(json.dumps({'target':'srv1519215','user':'codex','kernel_wireguard':True,'ports_free':[51443,18081],'pools_nonoverlapping':True,'production_wg0':'active','production_addresses':run('ip','-brief','address','show','wg0')},sort_keys=True))
if __name__=='__main__':
    try:main()
    except Exception as exc:print(str(exc),file=sys.stderr);sys.exit(1)
