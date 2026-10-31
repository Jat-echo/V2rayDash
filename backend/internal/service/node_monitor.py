import hashlib
import json
import os
from pathlib import Path
import pwd
import re
import subprocess
import sys
import uuid
from contextlib import contextmanager


@contextmanager
def update_monitor(paths):
    saved = {path: (path.read_bytes(), path.stat()) if path.exists() else None for path in paths}
    was_active = subprocess.run(['systemctl', 'is-active', '--quiet', 'v2raydash-monitor'], capture_output=True).returncode == 0
    was_enabled = subprocess.run(['systemctl', 'is-enabled', '--quiet', 'v2raydash-monitor'], capture_output=True).returncode == 0
    try:
        yield
    except BaseException:
        subprocess.run(['systemctl', 'stop', 'v2raydash-monitor'], capture_output=True)
        for path, previous in saved.items():
            if previous is None:
                path.unlink(missing_ok=True)
            else:
                content, attributes = previous
                temporary = path.with_name('.v2raydash-restore-' + uuid.uuid4().hex)
                fd = os.open(str(temporary), os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
                with os.fdopen(fd, 'wb') as file:
                    file.write(content)
                temporary.chmod(attributes.st_mode & 0o777)
                os.chown(temporary, attributes.st_uid, attributes.st_gid)
                os.replace(temporary, path)
        subprocess.run(['systemctl', 'daemon-reload'], check=True, capture_output=True)
        if not was_enabled:
            subprocess.run(['systemctl', 'disable', 'v2raydash-monitor'], capture_output=True)
        if was_active:
            subprocess.run(['systemctl', 'restart', 'v2raydash-monitor'], check=True, capture_output=True)
        raise


request = json.load(sys.stdin)
source = Path(request['source'])
assert re.fullmatch(r'/tmp/v2raydash-monitor-[a-f0-9]{32}', str(source)), 'Invalid upload path'
assert hashlib.sha256(source.read_bytes()).hexdigest() == request['sha256'], 'Agent checksum mismatch'
assert request['core_service'] in ('xray', 'xray-relay', 'xray-exit')
assert os.uname().machine == 'x86_64', 'This agent artifact requires an amd64 node'
try:
    user = pwd.getpwnam('v2raydash-monitor')
except KeyError:
    subprocess.run(['useradd', '--system', '--user-group', '--no-create-home', '--home-dir', '/nonexistent', '--shell', '/usr/sbin/nologin', 'v2raydash-monitor'], check=True, capture_output=True)
    user = pwd.getpwnam('v2raydash-monitor')

directory = Path('/etc/v2raydash-monitor')
directory.mkdir(mode=0o750, exist_ok=True)
os.chown(directory, 0, user.pw_gid)
config = directory / 'config.json'
target = Path('/usr/local/bin/v2raydash-monitor')
unit_path = Path('/etc/systemd/system/v2raydash-monitor.service')
unit = '''[Unit]
Description=V2rayDash node monitor
After=network-online.target
Wants=network-online.target

[Service]
User=v2raydash-monitor
Group=v2raydash-monitor
ExecStart=/usr/local/bin/v2raydash-monitor -config /etc/v2raydash-monitor/config.json
StateDirectory=v2raydash-monitor
StateDirectoryMode=0700
Restart=on-failure
RestartSec=5
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
UMask=0077

[Install]
WantedBy=multi-user.target
'''
try:
    with update_monitor([config, target, unit_path]):
        temporary = config.with_name('.config-' + uuid.uuid4().hex)
        fd = os.open(str(temporary), os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(fd, 'w') as file:
            json.dump({'server_id': request['server_id'], 'control_center_url': request['url'], 'psk': request['psk'], 'core_service': request['core_service'], 'state_path': '/var/lib/v2raydash-monitor/pending-heartbeats.json'}, file)
        temporary.chmod(0o640)
        os.chown(temporary, 0, user.pw_gid)
        os.replace(temporary, config)
        source.chmod(0o755)
        os.replace(source, target)
        unit_path.write_text(unit)
        subprocess.run(['systemctl', 'daemon-reload'], check=True, capture_output=True)
        subprocess.run(['systemctl', 'enable', 'v2raydash-monitor'], check=True, capture_output=True)
        subprocess.run(['systemctl', 'restart', 'v2raydash-monitor'], check=True, capture_output=True)
        subprocess.run(['systemctl', 'is-active', '--quiet', 'v2raydash-monitor'], check=True)
    # Retire only the known old Go monitor after the new one is running.
    # A proxy core or any other service using this name must remain untouched.
    old = subprocess.run(['systemctl', 'show', '-p', 'MainPID', '--value', 'v2ray-agent'], capture_output=True, text=True).stdout.strip()
    if old.isdigit() and int(old) > 0:
        try:
            args = Path('/proc/' + old + '/cmdline').read_bytes().split(bytes([0]))
            if Path(args[0].decode()).name == 'v2ray-agent' and any(b'agent.json' in arg for arg in args):
                subprocess.run(['systemctl', 'disable', '--now', 'v2ray-agent'], capture_output=True)
        except OSError:
            pass
finally:
    source.unlink(missing_ok=True)
print(json.dumps({'installed': True, 'proxy_restarted': False}))
