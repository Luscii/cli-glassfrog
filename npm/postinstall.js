// glassfrog postinstall fallback (spec 037, ADR-3 / ADR-5).
//
// Runs after npm installs the umbrella package. If the matching platform package
// is bundled (npm resolved the optional dependency for this os/cpu) it is a no-op
// success with NO network — the offline-capable primary path. Otherwise it falls
// back to 027's acquisition pattern: map the host to a supported target, construct
// the 022 release-asset URLs against GLASSFROG_DOWNLOAD_BASE_URL, download the
// archive + checksums, verify the archive's sha256 against its checksums entry,
// extract with system tar, and place the binary atomically inside the umbrella
// (temp → verify → move; nothing placed on any failure). An unsupported platform,
// a checksum mismatch, a download/extract failure, or a missing tar each refuses
// with a clear cause + next step (CONSTITUTION II) and a non-zero exit, leaving
// nothing runnable behind.
//
// Zero runtime npm dependencies — Node built-ins only.
'use strict';

const fs = require('fs');
const os = require('os');
const path = require('path');
const http = require('http');
const https = require('https');
const crypto = require('crypto');
const { spawnSync } = require('child_process');
const platform = require('./lib/platform.js');

// An install-time failure that fails `npm install` with a clear message. Carrying
// the message on the error keeps the throw/catch seam testable (tests assert the
// rejection) while the main wrapper renders it to stderr + a non-zero exit.
class InstallError extends Error {}

// downloadToFile(url, dest) — fetch url to dest, following redirects, rejecting on
// any non-2xx. http vs https is chosen by the URL scheme so the localhost fixture
// server (http) works through the same path as a real https release download.
function downloadToFile(url, dest, redirectsLeft = 5) {
	return new Promise((resolve, reject) => {
		const client = url.startsWith('https:') ? https : http;
		const req = client.get(url, (res) => {
			const { statusCode, headers } = res;
			if (statusCode >= 300 && statusCode < 400 && headers.location) {
				res.resume();
				if (redirectsLeft <= 0) {
					reject(new Error(`too many redirects for ${url}`));
					return;
				}
				const next = new URL(headers.location, url).toString();
				resolve(downloadToFile(next, dest, redirectsLeft - 1));
				return;
			}
			if (statusCode !== 200) {
				res.resume();
				reject(new Error(`HTTP ${statusCode} for ${url}`));
				return;
			}
			const out = fs.createWriteStream(dest);
			res.pipe(out);
			out.on('finish', () => out.close(resolve));
			out.on('error', reject);
		});
		req.on('error', reject);
	});
}

// hasTar() — whether a system tar extractor is on PATH.
function hasTar() {
	const r = spawnSync('tar', ['--version'], { stdio: 'ignore' });
	return !r.error;
}

// checksumFor(file, name) — the sha256 hex recorded for `name` in a GoReleaser
// checksums file ("<hash>  <filename>" lines), or null if absent.
function checksumFor(file, name) {
	const text = fs.readFileSync(file, 'utf8');
	for (const line of text.split('\n')) {
		const m = line.trim().match(/^([0-9a-f]+)\s+(.+)$/i);
		if (m && m[2] === name) return m[1].toLowerCase();
	}
	return null;
}

// sha256(file) — lowercase hex sha256 of a file.
function sha256(file) {
	return crypto.createHash('sha256').update(fs.readFileSync(file)).digest('hex');
}

// postinstall(opts) — the install-time logic. Resolves on no-op success or a
// completed fallback placement; throws InstallError on any refusal (nothing
// placed). Inputs are injectable so node --test drives it against a localhost
// fixture server with no network.
async function postinstall(opts = {}) {
	const plat = opts.platform || process.platform;
	const arch = opts.arch || process.arch;
	const pkgRoot = opts.pkgRoot || __dirname;
	const baseUrl =
		opts.baseUrl || process.env.GLASSFROG_DOWNLOAD_BASE_URL || 'https://github.com';
	const log = opts.log || ((m) => process.stdout.write(m + '\n'));

	const version = opts.version || require(path.join(pkgRoot, 'package.json')).version;

	// 1. Unsupported platform → refuse, place nothing (fork #1, ADR-5).
	const target = platform.detectTarget(plat, arch);
	if (!target) {
		throw new InstallError(platform.unsupportedMessage(plat, arch));
	}

	// 2. Bundled platform package present → no-op success, no network.
	if (platform.resolveBundledBinary(target, pkgRoot)) {
		log(`glassfrog: using the bundled ${plat}-${arch} binary (no download needed).`);
		return { placed: false, reason: 'bundled' };
	}

	// 3. Fallback download + verify + place (ADR-3, conforming to 027).
	const tag = `v${version}`;
	const { archive, checksums } = platform.assetNames(version, target.goos, target.goarch);
	const dlBase = platform.downloadBase(baseUrl, tag);

	// Probe the extractor before any network/placement (027's tooling-probe).
	if (!hasTar()) {
		throw new InstallError(
			'glassfrog: tar was not found, so the downloaded archive cannot be extracted.\n' +
				'Next step: install tar (preinstalled on macOS/Linux) and re-run the install, ' +
				'or use the install script (curl … | sh) channel.',
		);
	}

	// Everything is fetched/verified/extracted in a temp dir INSIDE the package, so
	// the final move is an atomic same-filesystem rename and nothing lands until the
	// checksum verifies.
	const tmp = fs.mkdtempSync(path.join(pkgRoot, '.glassfrog-dl-'));
	try {
		const archivePath = path.join(tmp, archive);
		const checksumsPath = path.join(tmp, checksums);

		try {
			await downloadToFile(`${dlBase}/${checksums}`, checksumsPath);
			await downloadToFile(`${dlBase}/${archive}`, archivePath);
		} catch (err) {
			throw new InstallError(
				`glassfrog: failed to download a release asset from ${dlBase} (${err.message}).\n` +
					'Next step: check network access to the release host and re-run the install; ' +
					`verify that version ${tag} exists.`,
			);
		}

		const expected = checksumFor(checksumsPath, archive);
		if (!expected) {
			throw new InstallError(
				`glassfrog: no checksum entry for ${archive} in ${checksums}.\n` +
					'Next step: re-run the install to retry the download; if it persists, the ' +
					'release asset may be corrupt — report it.',
			);
		}
		const actual = sha256(archivePath);
		if (actual !== expected) {
			throw new InstallError(
				`glassfrog: integrity check failed for ${archive} (expected ${expected}, got ${actual}).\n` +
					'Next step: re-run the install to retry the download; if it persists, the ' +
					'release asset may be corrupt — report it.',
			);
		}

		// Extract ONLY the expected member, the way the generator (build.mjs) does —
		// not the whole archive. The archive is already sha256-verified, but pulling
		// just `glassfrog` avoids writing any other entry (e.g. a path-traversal
		// member) to disk at all.
		const extract = spawnSync('tar', ['-xzf', archivePath, '-C', tmp, platform.BINARY], {
			encoding: 'utf8',
		});
		if (extract.status !== 0) {
			throw new InstallError(
				`glassfrog: failed to extract ${platform.BINARY} from ${archive} (${(extract.stderr || '').trim()}).\n` +
					'Next step: re-run the install; if it persists, the release asset may be corrupt — report it.',
			);
		}
		// Require a regular file (lstat does not follow symlinks) before placing — a
		// symlink or directory entry named `glassfrog` is never runnable and must not
		// be moved into place.
		const extracted = path.join(tmp, platform.BINARY);
		let extractedStat = null;
		try {
			extractedStat = fs.lstatSync(extracted);
		} catch {
			extractedStat = null;
		}
		if (!extractedStat || !extractedStat.isFile()) {
			throw new InstallError(
				`glassfrog: the archive ${archive} did not contain a regular ${platform.BINARY} binary.\n` +
					'Next step: re-run the install; if it persists, report it.',
			);
		}

		// Atomic placement: rename within the same filesystem (the binary becomes
		// runnable only here, after verification).
		const placed = platform.placedBinaryPath(pkgRoot);
		fs.mkdirSync(path.dirname(placed), { recursive: true });
		fs.renameSync(extracted, placed);
		fs.chmodSync(placed, 0o755);

		log(`glassfrog: downloaded and verified ${archive}; placed the ${plat}-${arch} binary.`);
		return { placed: true, reason: 'fallback', path: placed };
	} finally {
		fs.rmSync(tmp, { recursive: true, force: true });
	}
}

module.exports = { postinstall, InstallError };

// Run only when invoked as the npm postinstall script, not when a test requires
// this file. A refusal prints to stderr and fails the install (non-zero exit).
if (require.main === module) {
	postinstall().then(
		() => {},
		(err) => {
			process.stderr.write((err && err.message ? err.message : String(err)) + '\n');
			process.exit(1);
		},
	);
}
