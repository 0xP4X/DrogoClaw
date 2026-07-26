import * as esbuild from 'esbuild';
import { spawnSync } from 'child_process';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import JavaScriptObfuscator from 'javascript-obfuscator';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, '..');
const DIST_DIR = path.join(ROOT, 'dist');
const BUILD_TMP_DIR = path.join(DIST_DIR, '.binary-build');
const BUNDLE_PATH = path.join(BUILD_TMP_DIR, 'bundle.cjs');
const ENTRY_PATH = path.join(BUILD_TMP_DIR, 'pkg-entry.cjs');
const BIN_DIR = path.join(ROOT, 'bin');
const OUTPUT_BINARY = path.join(BIN_DIR, 'drogonclaw-linux-x64');
const PKG_TARGET = process.env.DROGON_PKG_TARGET || 'node22-linux-x64';

function assertFile(relativePath, label = relativePath) {
  const absolutePath = path.join(ROOT, relativePath);
  if (!fs.existsSync(absolutePath)) {
    throw new Error(`Missing ${label}: ${relativePath}. Run npm install in WSL before building the binary.`);
  }
  return absolutePath;
}

function run(command, args, options = {}) {
  const result = spawnSync(command, args, {
    cwd: ROOT,
    stdio: 'inherit',
    shell: false,
    ...options,
  });

  if (result.error) throw result.error;
  if (result.status !== 0) {
    throw new Error(`${command} ${args.join(' ')} failed with exit code ${result.status}`);
  }
}

async function build() {
  // Step 1: Clean output directories
  console.log('🧹 Cleaning output directories...');
  if (fs.existsSync(BIN_DIR)) fs.rmSync(BIN_DIR, { recursive: true, force: true });
  fs.mkdirSync(DIST_DIR, { recursive: true });
  fs.rmSync(BUILD_TMP_DIR, { recursive: true, force: true });
  fs.mkdirSync(BUILD_TMP_DIR, { recursive: true });

  // Step 2: Bundle with esbuild (TypeScript -> single CJS bundle)
  console.log('📦 Bundling DrogonClaw with esbuild...');
  await esbuild.build({
    entryPoints: [path.join(ROOT, 'src', 'cli', 'index.ts')],
    bundle: true,
    platform: 'node',
    target: 'node22',
    outfile: BUNDLE_PATH,
    format: 'cjs',
    // Keep native modules external so pkg can snapshot them as assets
    external: [
      'better-sqlite3',
      'playwright',
      'playwright-core',
      'puppeteer',
      'fsevents',
      'neo4j-driver',
      'ssh2',
      'cpu-features',
    ],
    minify: true,
  });

  if (!fs.existsSync(BUNDLE_PATH)) {
    throw new Error(`esbuild did not produce ${BUNDLE_PATH}`);
  }
  console.log(`✅ Bundle written to ${BUNDLE_PATH} (${(fs.statSync(BUNDLE_PATH).size / 1024 / 1024).toFixed(2)} MB)`);

  // Step 3: Obfuscate the bundle (safe settings for Node.js bundles)
  console.log('🔒 Obfuscating bundle...');
  const bundleCode = fs.readFileSync(BUNDLE_PATH, 'utf8');
  const obfuscated = JavaScriptObfuscator.obfuscate(bundleCode, {
    compact: true,
    controlFlowFlattening: false,      // keep off — avoids performance hit on large files
    deadCodeInjection: false,          // keep off — avoids size explosion
    identifierNamesGenerator: 'hexadecimal',
    renameGlobals: false,              // CRITICAL: keep false to avoid breaking require/module
    stringArray: true,
    stringArrayEncoding: [],           // no base64/rc4 encoding — safer for Node bundles
    stringArrayThreshold: 0.75,
    target: 'node',
    ignoreRequireImports: true,        // CRITICAL: don't touch require() calls (native modules)
    sourceMap: false,
  });
  fs.writeFileSync(BUNDLE_PATH, obfuscated.getObfuscatedCode(), 'utf8');
  console.log(`✅ Obfuscated bundle written (${(fs.statSync(BUNDLE_PATH).size / 1024 / 1024).toFixed(2)} MB)`);

  // pkg maps asset snapshot paths relative to the project root (ROOT).
  // The config file MUST be written at ROOT so that relative paths
  // resolve correctly (e.g. node_modules/playwright-core/browsers.json →
  // /snapshot/drogon/node_modules/playwright-core/browsers.json).
  //
  // PLAYWRIGHT NOTE: playwright-core dynamically loads browsers.json using __dirname
  // inside pkg's snapshot — which pkg cannot resolve correctly regardless of asset config.
  // Solution: we patch the entry to intercept playwright's require and redirect browsers.json
  // to a copy placed alongside the binary on the real filesystem.
  console.log('🛠️  Compiling standalone binaries with pkg...');

  const pkgAssets = [
    'node_modules/better-sqlite3/build/Release/better_sqlite3.node',
  ];

  // Verify required assets exist
  for (const a of pkgAssets) {
    if (!fs.existsSync(path.join(ROOT, a))) {
      throw new Error(`Missing required asset: ${a}. Run npm install in WSL first.`);
    }
  }

  const optionalAssets = [
    'node_modules/cpu-features/build/Release/cpufeatures.node',
  ];
  for (const asset of optionalAssets) {
    if (fs.existsSync(path.join(ROOT, asset))) {
      pkgAssets.push(asset);
    } else {
      console.log(`   Optional asset not present, skipping: ${asset}`);
    }
  }

  // Copy browsers.json alongside the binary so playwright can find it at runtime.
  // The entry point will set PLAYWRIGHT_BROWSERS_PATH to point pkg's extraction dir.
  const browsersJsonSrc = path.join(ROOT, 'node_modules/playwright-core/browsers.json');
  if (!fs.existsSync(browsersJsonSrc)) {
    throw new Error('Missing node_modules/playwright-core/browsers.json. Run npm install in WSL first.');
  }
  fs.mkdirSync(BIN_DIR, { recursive: true });
  fs.copyFileSync(browsersJsonSrc, path.join(BIN_DIR, 'browsers.json'));
  console.log('📋 Copied browsers.json alongside binary.');

  // Since the bundle is obfuscated, pkg cannot statically analyze require() calls for external modules.
  // We create a temporary plain-text entry point that:
  //   1. Patches Module._resolveFilename to intercept the playwright browsers.json require
  //      and redirect it to a copy on the real filesystem next to the binary.
  //   2. Pre-requires all external native modules so pkg can trace and snapshot them.
  //   3. Loads the obfuscated bundle.
  const entryCode = `
    'use strict';
    const path = require('path');
    const Module = require('module');
    const origResolve = Module._resolveFilename.bind(Module);

    // Redirect playwright-core's internal browsers.json require to the real filesystem.
    // When running as a pkg binary, process.execPath is the binary itself.
    // We place browsers.json next to the binary during build.
    const binDir = path.dirname(process.execPath);
    Module._resolveFilename = function(request, parent, isMain, options) {
      if (request && request.endsWith('browsers.json') && parent && parent.filename && parent.filename.includes('playwright-core')) {
        const real = path.join(binDir, 'browsers.json');
        return real;
      }
      return origResolve(request, parent, isMain, options);
    };

    try { require('better-sqlite3'); } catch(e) {}
    try { require('playwright'); } catch(e) {}
    try { require('playwright-core'); } catch(e) {}
    try { require('puppeteer'); } catch(e) {}
    try { require('neo4j-driver'); } catch(e) {}
    try { require('ssh2'); } catch(e) {}
    try { require('cpu-features'); } catch(e) {}
    require('./bundle.cjs');
  `;
  fs.writeFileSync(ENTRY_PATH, entryCode, 'utf8');

  // Write the pkg config at the project ROOT so that relative asset paths
  // are resolved from ROOT — this ensures correct snapshot path mapping.
  //
  // CRITICAL: bundle.cjs is listed under `scripts` (not `assets`) so pkg
  // embeds it verbatim without static analysis. This avoids the massive flood
  // of "Cannot resolve '_0x...'" warnings from obfuscated code that cause
  // pkg to hang for 10+ minutes on large bundles.
  const rootConfigPath = path.join(ROOT, 'pkg.config.json');
  const tmpConfig = {
    pkg: {
      assets: pkgAssets,
      scripts: [
        // Relative to ROOT — bundle is snapshotted as-is, no analysis
        path.relative(ROOT, BUNDLE_PATH).replace(/\\/g, '/'),
      ],
      targets: [PKG_TARGET],
      outputPath: BIN_DIR,
      compress: 'GZip',   // Brotli is extremely slow on large bundles; GZip is sufficient
    }
  };
  fs.writeFileSync(rootConfigPath, JSON.stringify(tmpConfig, null, 2));

  try {
    run('npx', [
      '--no-install',
      'pkg',
      ENTRY_PATH,
      '--config', rootConfigPath,
      '--target', PKG_TARGET,
      '--output', OUTPUT_BINARY,
      '--no-bytecode',   // skip V8 snapshot compilation — avoids additional slowdown
    ]);
  } finally {
    // Clean up temp files
    if (fs.existsSync(rootConfigPath)) fs.rmSync(rootConfigPath);
    fs.rmSync(BUILD_TMP_DIR, { recursive: true, force: true });
  }

  fs.chmodSync(OUTPUT_BINARY, 0o755);
  console.log('🔎 Smoke-testing standalone binary...');
  run(OUTPUT_BINARY, ['--version']);

  console.log('✅ Build complete! Binaries are in the bin/ directory.');
  for (const f of fs.readdirSync(BIN_DIR)) {
    const size = (fs.statSync(path.join(BIN_DIR, f)).size / 1024 / 1024).toFixed(1);
    console.log(`   📁 bin/${f} (${size} MB)`);
  }
}

build().catch(err => {
  console.error('❌ Build failed:', err.message || err);
  process.exit(1);
});
