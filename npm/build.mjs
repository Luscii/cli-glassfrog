// Package generator for the npm channel (spec 037, ADR-2).
//
// Given a release version and the verified GoReleaser dist/ directory, this emits
// the umbrella package (@luscii-healthtech/glassfrog) and the four platform
// packages (@luscii-healthtech/glassfrog-<os>-<cpu>) into a gitignored output dir,
// each ready to `npm publish`. The per-platform package directories are NOT
// committed — they exist only as this generator's output. The generator owns the
// GoReleaser→npm arch map and the =<version> optional-dependency pinning (ADR-6),
// reusing lib/platform.js so those rules have one source of truth.
//
//   node npm/build.mjs --version v1.4.0 --dist dist --out dist/npm
//
// Zero npm dependencies — Node built-ins only.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';
import { execFileSync } from 'node:child_process';

const require = createRequire(import.meta.url);
const platform = require('./lib/platform.js');

const HERE = path.dirname(fileURLToPath(import.meta.url));
const TEMPLATE = path.join(HERE, 'package.json');

// npmVersion(tag) — the npm version for a git tag: the tag without a leading `v`
// (v1.4.0 → 1.4.0, v1.4.0-rc.1 → 1.4.0-rc.1). The same transform 022/027 use.
export function npmVersion(tag) {
	return String(tag).replace(/^v/, '');
}

// platformPackageJson(target, version) — the package.json for one platform
// package: scoped <os>-<cpu> name, the matching os/cpu arrays, the bundled binary
// in files, and NO bin (only the umbrella links the command).
export function platformPackageJson(target, version) {
	return {
		name: platform.platformPackageName(target.os, target.cpu),
		version,
		description: `The glassfrog binary for ${target.os}-${target.cpu}.`,
		homepage: 'https://github.com/Luscii/cli-glassfrog#readme',
		repository: { type: 'git', url: 'git+https://github.com/Luscii/cli-glassfrog.git' },
		license: 'UNLICENSED',
		os: [target.os],
		cpu: [target.cpu],
		files: [`bin/${platform.BINARY}`],
		publishConfig: { access: 'public' },
	};
}

// umbrellaPackageJson(template, version) — the umbrella package.json: the template
// stamped at `version`, with optionalDependencies listing all four platform
// packages pinned to the EXACT version (=X.Y.Z), and scripts reduced to the
// postinstall (the dev-only test script is dropped from the published package).
export function umbrellaPackageJson(template, version) {
	const optionalDependencies = {};
	for (const t of platform.SUPPORTED) {
		optionalDependencies[platform.platformPackageName(t.os, t.cpu)] = `=${version}`;
	}
	return {
		...template,
		version,
		scripts: { postinstall: 'node postinstall.js' },
		optionalDependencies,
	};
}

// extractBinary(archivePath, destBinPath) — extract the `glassfrog` member of a
// 022 tar.gz into destBinPath, executable. Uses system tar (present in CI).
function extractBinary(archivePath, destDir) {
	fs.mkdirSync(destDir, { recursive: true });
	execFileSync('tar', ['-xzf', archivePath, '-C', destDir, platform.BINARY]);
	fs.chmodSync(path.join(destDir, platform.BINARY), 0o755);
}

// generate({ version, distDir, outDir }) — emit the umbrella + four platform
// packages. Returns a manifest of what was written. Throws if a target's verified
// archive is missing from distDir (the hard coupling to 022's asset names).
export function generate({ version, distDir, outDir }) {
	const ver = npmVersion(version);
	const template = JSON.parse(fs.readFileSync(TEMPLATE, 'utf8'));

	fs.mkdirSync(outDir, { recursive: true });
	const platforms = [];

	for (const target of platform.SUPPORTED) {
		const { archive } = platform.assetNames(ver, target.goos, target.goarch);
		const archivePath = path.join(distDir, archive);
		if (!fs.existsSync(archivePath)) {
			throw new Error(
				`missing release archive ${archivePath} for ${target.os}-${target.cpu} ` +
					`(expected 022 asset ${archive})`,
			);
		}
		const pkgDir = path.join(outDir, `${platform.BINARY}-${target.os}-${target.cpu}`);
		extractBinary(archivePath, path.join(pkgDir, 'bin'));
		fs.writeFileSync(
			path.join(pkgDir, 'package.json'),
			JSON.stringify(platformPackageJson(target, ver), null, 2) + '\n',
		);
		platforms.push(pkgDir);
	}

	// Umbrella: copy the real sources (launcher, postinstall, shared lib) and stamp
	// the package.json with the version + pinned optional dependencies.
	const umbDir = path.join(outDir, platform.BINARY);
	fs.mkdirSync(path.join(umbDir, 'bin'), { recursive: true });
	fs.mkdirSync(path.join(umbDir, 'lib'), { recursive: true });
	fs.copyFileSync(path.join(HERE, 'bin', platform.BINARY), path.join(umbDir, 'bin', platform.BINARY));
	fs.chmodSync(path.join(umbDir, 'bin', platform.BINARY), 0o755);
	fs.copyFileSync(path.join(HERE, 'postinstall.js'), path.join(umbDir, 'postinstall.js'));
	fs.copyFileSync(path.join(HERE, 'lib', 'platform.js'), path.join(umbDir, 'lib', 'platform.js'));
	fs.writeFileSync(
		path.join(umbDir, 'package.json'),
		JSON.stringify(umbrellaPackageJson(template, ver), null, 2) + '\n',
	);

	return { version: ver, umbrella: umbDir, platforms };
}

// parseArgs(argv) — minimal --flag value parser.
function parseArgs(argv) {
	const out = {};
	for (let i = 0; i < argv.length; i += 1) {
		const a = argv[i];
		if (a.startsWith('--')) {
			out[a.slice(2)] = argv[i + 1];
			i += 1;
		}
	}
	return out;
}

// CLI entry — run only when invoked directly, not when a test imports this module.
if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
	const args = parseArgs(process.argv.slice(2));
	const version = args.version || process.env.GLASSFROG_VERSION;
	if (!version) {
		process.stderr.write('build.mjs: --version <tag> is required\n');
		process.exit(2);
	}
	const result = generate({
		version,
		distDir: args.dist || 'dist',
		outDir: args.out || 'dist/npm',
	});
	process.stdout.write(
		`Generated ${platform.UMBRELLA}@${result.version} + ${result.platforms.length} platform packages into ${path.dirname(result.umbrella)}\n`,
	);
}
