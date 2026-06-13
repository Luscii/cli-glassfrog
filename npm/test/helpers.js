// Shared test helpers for the npm channel's node --test suites (spec 037).
//
// Hermetic by construction: every helper builds a throwaway package layout or a
// localhost fixture server, so the suites touch no network and no real registry
// (mirroring 027's hermetic Go exec-test).
'use strict';

const fs = require('fs');
const os = require('os');
const path = require('path');
const http = require('http');
const zlib = require('zlib');
const crypto = require('crypto');
const { spawnSync, execFileSync } = require('child_process');

const SRC = path.join(__dirname, '..');

// mkTmp(prefix) — a fresh temp directory; caller removes it.
function mkTmp(prefix) {
	return fs.mkdtempSync(path.join(os.tmpdir(), prefix || 'glassfrog-test-'));
}

// rmrf(dir) — best-effort recursive removal.
function rmrf(dir) {
	fs.rmSync(dir, { recursive: true, force: true });
}

// copyUmbrellaSources(dest) — copy the committed umbrella sources (launcher,
// postinstall, lib) into dest so dest is a self-contained umbrella package root.
function copyUmbrellaSources(dest) {
	fs.mkdirSync(path.join(dest, 'bin'), { recursive: true });
	fs.mkdirSync(path.join(dest, 'lib'), { recursive: true });
	fs.copyFileSync(path.join(SRC, 'bin', 'glassfrog'), path.join(dest, 'bin', 'glassfrog'));
	fs.chmodSync(path.join(dest, 'bin', 'glassfrog'), 0o755);
	fs.copyFileSync(path.join(SRC, 'lib', 'platform.js'), path.join(dest, 'lib', 'platform.js'));
	// postinstall.js is copied when present (it lands in T002); the launcher suite
	// stands alone without it.
	const postinstall = path.join(SRC, 'postinstall.js');
	if (fs.existsSync(postinstall)) {
		fs.copyFileSync(postinstall, path.join(dest, 'postinstall.js'));
	}
}

// writeStubBinary(file) — write an executable stub that stands in for the real
// glassfrog binary. It echoes its arguments (so passthrough is observable),
// honours `--exit N` (so the launcher's exit-code propagation is observable), and
// `--signal` (so signal re-raising is observable).
function writeStubBinary(file) {
	const body = [
		'#!/usr/bin/env node',
		"'use strict';",
		'const args = process.argv.slice(2);',
		"process.stdout.write('ARGS:' + JSON.stringify(args) + '\\n');",
		"if (args.includes('--signal')) { process.kill(process.pid, 'SIGTERM'); return; }",
		"const i = args.indexOf('--exit');",
		'const code = i >= 0 ? Number(args[i + 1]) : 0;',
		'process.exit(code);',
		'',
	].join('\n');
	fs.mkdirSync(path.dirname(file), { recursive: true });
	fs.writeFileSync(file, body);
	fs.chmodSync(file, 0o755);
}

// runLauncher(pkgRoot, args, env) — exec the launcher copy in pkgRoot as a child
// process (so real exit codes and signals are observable). Returns the spawn
// result ({ status, signal, stdout, stderr }).
function runLauncher(pkgRoot, args = [], env = {}) {
	return spawnSync(process.execPath, [path.join(pkgRoot, 'bin', 'glassfrog'), ...args], {
		encoding: 'utf8',
		env: { ...process.env, ...env },
	});
}

// makeArchive(dir, binaryBody) — write a stub glassfrog binary, tar.gz it the way
// 022 does (a `glassfrog` entry at the archive root), and return
// { archivePath, sha256, binaryBody }. Uses system tar so the fixture matches a
// real release archive (and so a missing tar surfaces in the helper, not the SUT).
function makeArchive(dir, binaryBody) {
	fs.mkdirSync(dir, { recursive: true });
	const body = binaryBody || '#!/bin/sh\necho stub-glassfrog\n';
	const binPath = path.join(dir, 'glassfrog');
	fs.writeFileSync(binPath, body);
	fs.chmodSync(binPath, 0o755);
	const archivePath = path.join(dir, 'archive.tar.gz');
	execFileSync('tar', ['-czf', archivePath, '-C', dir, 'glassfrog']);
	const sha256 = crypto.createHash('sha256').update(fs.readFileSync(archivePath)).digest('hex');
	return { archivePath, sha256, binaryBody: body };
}

// startReleaseServer({ tag, ver, assets }) — a localhost fixture server emulating
// the GitHub release-download surface. `assets` maps an asset filename to a Buffer
// (or string). Returns { baseUrl, hits, close }. `hits` counts served requests so
// a test can assert "no network" on the no-op path.
function startReleaseServer({ tag, assets }) {
	const hits = { count: 0, paths: [] };
	const server = http.createServer((req, res) => {
		hits.count += 1;
		hits.paths.push(req.url);
		// Match <base>/Luscii/cli-glassfrog/releases/download/<tag>/<name>.
		const prefix = `/Luscii/cli-glassfrog/releases/download/${tag}/`;
		if (req.url.startsWith(prefix)) {
			const name = decodeURIComponent(req.url.slice(prefix.length));
			if (Object.prototype.hasOwnProperty.call(assets, name)) {
				res.writeHead(200);
				res.end(assets[name]);
				return;
			}
		}
		res.writeHead(404);
		res.end('not found');
	});
	return new Promise((resolve) => {
		server.listen(0, '127.0.0.1', () => {
			const { port } = server.address();
			resolve({
				baseUrl: `http://127.0.0.1:${port}`,
				hits,
				close: () => new Promise((r) => server.close(r)),
			});
		});
	});
}

// hasTar() — whether system tar is on PATH (gates the archive-building helper).
function hasTar() {
	const r = spawnSync('tar', ['--version'], { stdio: 'ignore' });
	return !r.error;
}

module.exports = {
	SRC,
	mkTmp,
	rmrf,
	copyUmbrellaSources,
	writeStubBinary,
	runLauncher,
	makeArchive,
	startReleaseServer,
	hasTar,
	zlib,
	crypto,
};
