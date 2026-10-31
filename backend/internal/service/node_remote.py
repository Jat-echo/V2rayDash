"""Run through authenticated SSH stdin. Never receives credentials in argv."""
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import socket
import stat
import subprocess
import sys
import tempfile
import time
import urllib.request
import uuid


LAYOUTS = {
    'relay': ('/usr/local/etc/xray-relay/config.json', '/usr/local/bin/xray', 'xray-relay'),
    'exit': ('/usr/local/etc/xray-exit/config.json', '/usr/local/bin/xray', 'xray-exit'),
    'legacy': ('/etc/v2ray-agent/xray/conf/07_VLESS_vision_reality_inbounds.json', '/etc/v2ray-agent/xray/xray', 'xray'),
}
BACKUP_ROOT = Path('/var/lib/v2raydash/backups')


def public_port(config, inbound):
    port = int(inbound['port'])
    loopback = ('127.0.0.1', '::1', 'localhost')
    if inbound.get('listen') not in loopback:
        return port
    # Older v2ray-agent layouts expose a dokodemo-door ahead of local Reality.
    ports = set()
    for item in config.get('inbounds', []):
        settings = item.get('settings', {})
        if (item.get('protocol') == 'dokodemo-door' and item.get('listen') not in loopback
                and settings.get('address') in loopback and settings.get('port') == port
                and 'tcp' in settings.get('network', 'tcp').split(',') and not settings.get('followRedirect')):
            ports.add(int(item['port']))
    if len(ports) != 1:
        raise ValueError('Cannot identify a unique public port for the loopback Reality inbound')
    return ports.pop()


def inspect(layout):
    candidates = [layout] if layout else list(LAYOUTS)
    for name in candidates:
        path, binary, service = LAYOUTS[name]
        if name == 'legacy':
            path = next((p for p in [path, '/etc/v2ray-agent/xray/conf/07_VLESS_reality_vision_inbounds.json'] if Path(p).is_file()), path)
        if not Path(path).is_file():
            continue
        raw = Path(path).read_bytes()
        config = json.loads(raw)
        inbounds = [i for i in config.get('inbounds', []) if i.get('protocol') == 'vless' and i.get('streamSettings', {}).get('security') == 'reality']
        if len(inbounds) != 1:
            raise ValueError('Expected one Reality inbound in the selected configuration')
        if not Path(binary).is_file():
            raise ValueError('Xray binary not found')
        return {'layout': name, 'path': path, 'binary': binary, 'service': service, 'hash': hashlib.sha256(raw).hexdigest(), 'config': config, 'inbound': inbounds[0], 'public_port': public_port(config, inbounds[0])}
    raise ValueError('No supported existing Xray installation found; adoption does not reinstall services')


def probe(info, probes):
    results = []
    for item in probes:
        with tempfile.TemporaryDirectory(prefix='v2raydash-probe-') as tmp:
            with socket.socket() as reservation:
                reservation.bind(('127.0.0.1', 0))
                port = reservation.getsockname()[1]
            config = {'log': {'loglevel': 'error'}, 'inbounds': [{'listen': '127.0.0.1', 'port': port, 'protocol': 'socks', 'settings': {'auth': 'noauth', 'udp': False}}], 'outbounds': [{'protocol': 'vless', 'settings': {'vnext': [{'address': '127.0.0.1', 'port': info['inbound']['port'], 'users': [{'id': item['uuid'], 'encryption': 'none', 'flow': 'xtls-rprx-vision'}]}]}, 'streamSettings': {'network': 'tcp', 'security': 'reality', 'realitySettings': {'serverName': item['sni'], 'publicKey': item['public_key'], 'shortId': item['short_id'], 'fingerprint': 'chrome'}}}]}
            path = Path(tmp) / 'client.json'
            path.write_text(json.dumps(config))
            path.chmod(0o600)
            proc = subprocess.Popen([info['binary'], 'run', '-config', str(path)], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            try:
                for attempt in range(50):
                    if proc.poll() is not None:
                        raise ValueError('Temporary verification client stopped')
                    try:
                        with socket.create_connection(('127.0.0.1', port), timeout=.2):
                            break
                    except OSError:
                        time.sleep(.1)
                def fetch(url):
                    return subprocess.check_output(['curl', '--fail', '--silent', '--show-error', '--socks5-hostname', '127.0.0.1:' + str(port), '--max-time', '12', url], stderr=subprocess.DEVNULL, timeout=15)
                ip = fetch('https://api.ipify.org').decode().strip()
                if item.get('expected_ip') and ip != item['expected_ip']:
                    raise ValueError('Observed exit IP differs from the selected exit')
                body = fetch('https://www.google.com/')
                if len(body) < 5000:
                    raise ValueError('HTTPS page verification received an incomplete body')
                results.append({'account_id': item['account_id'], 'exit_ip': ip, 'html_bytes': len(body), 'viewpoint': 'node'})
            finally:
                proc.terminate()
                try:
                    proc.wait(timeout=3)
                except subprocess.TimeoutExpired:
                    proc.kill()
                    proc.wait()
    return results


def publish(info, request):
    if info['hash'] != request['expected_hash']:
        raise ValueError('Remote configuration changed; inspect and adopt it again before publishing')
    path = Path(info['path'])
    original = path.read_bytes()
    attributes = path.stat()
    candidate = path.with_name('.v2raydash-' + uuid.uuid4().hex + '.candidate')
    fd = os.open(str(candidate), os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(fd, 'w') as file:
        file.write(json.dumps(request['config'], indent=2) + '\n')
    candidate.chmod(stat.S_IMODE(attributes.st_mode))
    os.chown(candidate, attributes.st_uid, attributes.st_gid)
    backup_dir = BACKUP_ROOT
    backup_dir.mkdir(parents=True, mode=0o700, exist_ok=True)
    backup_dir.chmod(0o700)
    backup = backup_dir / (uuid.uuid4().hex + '.json')
    applied = False
    try:
        with tempfile.TemporaryDirectory(prefix='v2raydash-validate-') as tmp:
            if info['layout'] == 'legacy':
                for source in path.parent.glob('*.json'):
                    shutil.copyfile(source, Path(tmp) / source.name)
                shutil.copyfile(candidate, Path(tmp) / path.name)
                args = ['-confdir', tmp]
            else:
                # Xray requires a supported extension even when JSON format is explicit.
                validation = Path(tmp) / path.name
                shutil.copyfile(candidate, validation)
                args = ['-config', str(validation)]
            checked = subprocess.run([info['binary'], 'run', '-test'] + args, capture_output=True, timeout=20)
            if checked.returncode:
                raise ValueError('Xray rejected the candidate configuration; running configuration was preserved')
        fd = os.open(str(backup), os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(fd, 'wb') as file:
            file.write(original)
        os.replace(candidate, path)
        applied = True
        subprocess.run(['systemctl', 'restart', info['service']], check=True, capture_output=True, timeout=20)
        subprocess.run(['systemctl', 'is-active', '--quiet', info['service']], check=True, timeout=10)
        updated = inspect(info['layout'])
        results = probe(updated, request.get('probes', []))
        return {'hash': updated['hash'], 'checks': results, 'backup': str(backup)}
    except BaseException:
        if applied:
            rollback = path.with_name('.v2raydash-rollback-' + uuid.uuid4().hex)
            rollback.write_bytes(original)
            rollback.chmod(stat.S_IMODE(attributes.st_mode))
            os.chown(rollback, attributes.st_uid, attributes.st_gid)
            os.replace(rollback, path)
            subprocess.run(['systemctl', 'restart', info['service']], check=True, capture_output=True, timeout=20)
        raise
    finally:
        candidate.unlink(missing_ok=True)


def rollback(info, request):
    if info['hash'] != request['expected_hash']:
        raise ValueError('Remote configuration changed; automatic rollback stopped')
    backup = Path(request['backup']).resolve()
    if backup.parent != BACKUP_ROOT.resolve() or not re.fullmatch(r'[a-f0-9]{32}\.json', backup.name):
        raise ValueError('Invalid backup path')
    path = Path(info['path'])
    attributes = path.stat()
    candidate = path.with_name('.v2raydash-rollback-' + uuid.uuid4().hex)
    try:
        fd = os.open(str(candidate), os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(fd, 'wb') as file:
            file.write(backup.read_bytes())
        candidate.chmod(stat.S_IMODE(attributes.st_mode))
        os.chown(candidate, attributes.st_uid, attributes.st_gid)
        os.replace(candidate, path)
        subprocess.run(['systemctl', 'restart', info['service']], check=True, capture_output=True, timeout=20)
        subprocess.run(['systemctl', 'is-active', '--quiet', info['service']], check=True, timeout=10)
        return {'hash': hashlib.sha256(path.read_bytes()).hexdigest()}
    finally:
        candidate.unlink(missing_ok=True)


if __name__ == '__main__':
    request = json.load(sys.stdin)
    try:
        info = inspect(request.get('layout', ''))
        operation = request['operation']
        if operation == 'inspect':
            if request.get('egress'):
                try:
                    with urllib.request.urlopen('https://api.ipify.org', timeout=8) as response:
                        info['egress_ip'] = response.read(128).decode().strip()
                except OSError:
                    info['egress_ip'] = ''
            result = info
        elif operation == 'publish':
            result = publish(info, request)
        elif operation == 'rollback':
            result = rollback(info, request)
        elif operation == 'probe':
            result = {'checks': probe(info, request['probes'])}
        elif operation == 'restart':
            subprocess.run(['systemctl','restart',info['service']],check=True,capture_output=True,timeout=20)
            subprocess.run(['systemctl','is-active','--quiet',info['service']],check=True,timeout=10)
            result = {'hash':info['hash']}
        else:
            raise ValueError('Unsupported operation')
        print(json.dumps(result))
    except Exception as error:
        # No subprocess output or input configuration is reflected in HTTP errors.
        print(json.dumps({'error': str(error) if isinstance(error, ValueError) else 'Remote operation failed; check service availability and connectivity'}))
        sys.exit(1)
