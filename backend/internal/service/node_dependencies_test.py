"""Run the real shell bootstrap against isolated commands; never invoke a package manager."""
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


@unittest.skipUnless(os.name == 'posix', 'node bootstrap requires a POSIX shell')
class DependencyTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='v2raydash-dependencies-test-')
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.bin = self.root / 'bin'
        self.bin.mkdir()
        self.env = dict(os.environ, PATH=str(self.bin), FIXTURE=str(self.root), FAKE_UID='0', FAKE_ERROR='', FAKE_SUDO='yes')
        # Whitelist only shell utilities. Real package managers are outside PATH.
        for name in ['sh', 'env', 'mktemp', 'rm', 'grep', 'touch', 'true']:
            shutil.copy2(shutil.which(name), self.bin / name)
        self.stub('timeout', '''
            [ "$1" = -k ] && [ "$2" = 5 ] && [ "$3" = 90 ] || exit 1
            shift 3
            exec "$@"
        ''')
        self.stub('id', 'printf "%s\\n" "$FAKE_UID"')
        self.stub('python3', '''
            [ -f "$FIXTURE/python" ] || exit 1
            case "$*" in *create_default_context*) [ -f "$FIXTURE/certificates" ] || exit 1 ;; esac
        ''')
        self.script = Path(__file__).with_name('node_dependencies.sh').read_text()

    def stub(self, name, body):
        path = self.bin / name
        if path.is_symlink():
            path.unlink()
        path.write_text('#!/bin/sh\nset -eu\n' + body + '\n')
        path.chmod(0o700)

    def ready(self):
        for name in ['python', 'certificates']:
            (self.root / name).touch()

    def manager(self, name='apt-get'):
        self.stub(name, '''
            printf '%s\\n' "$*" >> "$FIXTURE/packages.log"
            if [ -n "$FAKE_ERROR" ]; then printf '%s\\n' "$FAKE_ERROR" >&2; exit 1; fi
            case " $* " in
              *' install '*)
                for package in "$@"; do
                  case "$package" in
                    python3) touch "$FIXTURE/python" ;;
                    ca-certificates) touch "$FIXTURE/certificates" ;;
                    curl) printf '#!/bin/sh\\nexit 0\\n' > "$FIXTURE/bin/curl"; chmod 700 "$FIXTURE/bin/curl" ;;
                  esac
                done
                ;;
            esac
        ''')
        shutil.copy2(shutil.which('chmod'), self.bin / 'chmod')

    def run_bootstrap(self, curl=False):
        return subprocess.run([shutil.which('sh'), '-s', '--'] + (['curl'] if curl else []), input=self.script, text=True, env=self.env, capture_output=True, timeout=10)

    def test_ready_root_requires_no_sudo_or_package_manager(self):
        self.ready()
        result = self.run_bootstrap()
        self.assertEqual((result.returncode, result.stdout), (0, 'root\n'))
        self.assertFalse((self.root / 'packages.log').exists())

    def test_missing_dependencies_install_once_without_leaking_logs(self):
        for manager in ['apt-get', 'dnf', 'yum']:
            with self.subTest(manager=manager):
                self.manager(manager)
                result = self.run_bootstrap(curl=True)
                self.assertEqual((result.returncode, result.stdout), (0, 'root\n'), result.stderr)
                calls = (self.root / 'packages.log').read_text()
                self.assertIn('python3', calls)
                self.assertIn('ca-certificates', calls)
                self.assertIn('curl', calls)
                self.assertNotIn('upgrade', calls.replace('--no-upgrade', ''))
                result = self.run_bootstrap(curl=True)
                self.assertEqual(result.returncode, 0, result.stdout)
                self.assertEqual((self.root / 'packages.log').read_text(), calls)
                (self.bin / manager).unlink()
                (self.bin / 'chmod').unlink()
                (self.bin / 'curl').unlink()
                for name in ['python', 'certificates', 'packages.log']:
                    (self.root / name).unlink()

    def test_read_does_not_install_curl(self):
        self.ready()
        self.manager()
        result = self.run_bootstrap()
        self.assertEqual(result.returncode, 0)
        self.assertFalse((self.root / 'packages.log').exists())

    def test_nonroot_requires_noninteractive_sudo(self):
        self.ready()
        self.env['FAKE_UID'] = '1000'
        result = self.run_bootstrap()
        self.assertEqual((result.returncode, result.stdout), (1, 'error:privilege\n'))
        self.stub('sudo', '''
            [ "$1" = -n ] || exit 1
            shift
            [ "$FAKE_SUDO" = yes ] || exit 1
            exec "$@"
        ''')
        result = self.run_bootstrap()
        self.assertEqual((result.returncode, result.stdout), (0, 'sudo\n'))
        (self.root / 'python').unlink()
        self.manager()
        result = self.run_bootstrap()
        self.assertEqual((result.returncode, result.stdout), (0, 'sudo\n'))
        self.env['FAKE_SUDO'] = 'no'
        result = self.run_bootstrap()
        self.assertEqual((result.returncode, result.stdout), (1, 'error:privilege\n'))

    def test_package_errors_are_classified_and_install_does_not_continue(self):
        result = self.run_bootstrap()
        self.assertEqual((result.returncode, result.stdout), (1, 'error:package_manager\n'))
        self.manager()
        errors = {
            'Could not get lock /var/lib/dpkg/lock-frontend': 'package_lock',
            'No space left on device': 'disk_space',
            'Failed to fetch https://user:secret@repo.invalid/?token=secret': 'package_network',
            'Package unavailable': 'install_failed',
        }
        for message, code in errors.items():
            with self.subTest(code=code):
                self.env['FAKE_ERROR'] = message
                result = self.run_bootstrap()
                self.assertEqual((result.returncode, result.stdout), (1, 'error:'+code+'\n'))
                self.assertEqual(result.stderr, '')
                self.assertFalse((self.root / 'python').exists())
        self.stub('timeout', 'exit 124')
        result = self.run_bootstrap()
        self.assertEqual((result.returncode, result.stdout), (1, 'error:install_timeout\n'))


if __name__ == '__main__':
    unittest.main()
