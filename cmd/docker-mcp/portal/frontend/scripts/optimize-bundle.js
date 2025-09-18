#!/usr/bin/env node
/* eslint-disable @typescript-eslint/no-require-imports */
/* eslint-disable no-console */

/**
 * Bundle Optimization Script
 *
 * This script analyzes and optimizes the Next.js bundle for production.
 * Run with: npm run analyze
 */

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const ANALYZE_ENV = 'ANALYZE=true';
const BUILD_CMD = 'next build';

console.log('🔍 Starting bundle analysis...\n');

// Check if bundle analyzer is installed
try {
  require.resolve('@next/bundle-analyzer');
} catch {
  console.error('❌ @next/bundle-analyzer not found. Installing...');
  execSync('npm install --save-dev @next/bundle-analyzer', {
    stdio: 'inherit',
  });
}

// Run build with analyzer
try {
  console.log('📦 Building with bundle analyzer...');
  execSync(`${ANALYZE_ENV} ${BUILD_CMD}`, {
    stdio: 'inherit',
    env: { ...process.env, ANALYZE: 'true' },
  });
} catch (error) {
  console.error('❌ Build failed:', error.message);
  process.exit(1);
}

// Read and analyze bundle stats
const statsPath = path.join(
  process.cwd(),
  '.next',
  'analyze',
  'bundle-stats.json'
);

if (fs.existsSync(statsPath)) {
  console.log('\n📊 Bundle Analysis Results:');
  console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n');

  try {
    const stats = JSON.parse(fs.readFileSync(statsPath, 'utf8'));

    // Analyze chunks
    const chunks = stats.chunks || [];
    const assets = stats.assets || [];

    console.log('📦 Chunk Analysis:');
    chunks.forEach(chunk => {
      const size = chunk.size || 0;
      const sizeKB = (size / 1024).toFixed(2);
      const files = chunk.files || [];

      console.log(`  ${chunk.name || 'unnamed'}: ${sizeKB}KB`);
      if (files.length > 0) {
        console.log(`    Files: ${files.join(', ')}`);
      }
    });

    console.log('\n📁 Large Assets (>100KB):');
    const largeAssets = assets
      .filter(asset => asset.size > 102400) // 100KB
      .sort((a, b) => b.size - a.size);

    largeAssets.forEach(asset => {
      const sizeKB = (asset.size / 1024).toFixed(2);
      console.log(`  ${asset.name}: ${sizeKB}KB`);
    });

    // Performance recommendations
    console.log('\n💡 Optimization Recommendations:');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n');

    // Check for heavy packages
    const heavyPackages = [
      { name: 'recharts', threshold: 180 },
      { name: 'react-grid-layout', threshold: 95 },
      { name: 'canvas-confetti', threshold: 15 },
      { name: 'motion', threshold: 85 },
      { name: '@azure/msal', threshold: 65 },
    ];

    heavyPackages.forEach(pkg => {
      const relevantAssets = assets.filter(
        asset =>
          asset.name.includes(pkg.name) ||
          asset.name.includes(pkg.name.replace('-', ''))
      );

      if (relevantAssets.length > 0) {
        const totalSize = relevantAssets.reduce(
          (sum, asset) => sum + asset.size,
          0
        );
        const totalKB = (totalSize / 1024).toFixed(2);

        if (totalSize > pkg.threshold * 1024) {
          console.log(
            `⚠️  ${pkg.name}: ${totalKB}KB (consider code splitting)`
          );
        } else {
          console.log(`✅ ${pkg.name}: ${totalKB}KB (optimized)`);
        }
      }
    });

    // General recommendations
    console.log('\n📋 General Recommendations:');
    console.log('  • Use dynamic imports for components > 50KB');
    console.log('  • Implement proper loading states for async components');
    console.log('  • Consider lazy loading for below-the-fold content');
    console.log('  • Use React.memo() for expensive components');
    console.log('  • Optimize images with Next.js Image component');
    console.log('  • Enable Turbopack for faster development builds');
  } catch (error) {
    console.error('❌ Failed to parse bundle stats:', error.message);
  }
} else {
  console.log('⚠️  Bundle stats file not found. Check build output.');
}

// Check for performance budget violations
console.log('\n🎯 Performance Budget Check:');
console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n');

const budgets = {
  'First Load JS': 250, // KB
  'Individual chunks': 100, // KB
  'Total bundle': 1000, // KB
};

// Read Next.js build output for First Load JS
const buildOutputPath = path.join(
  process.cwd(),
  '.next',
  'build-manifest.json'
);
if (fs.existsSync(buildOutputPath)) {
  try {
    JSON.parse(fs.readFileSync(buildOutputPath, 'utf8'));
    console.log(
      '✅ Build manifest found - check console output for First Load JS sizes'
    );
  } catch {
    console.log('⚠️  Could not parse build manifest');
  }
}

Object.entries(budgets).forEach(([metric, limit]) => {
  console.log(`  ${metric}: Target < ${limit}KB`);
});

console.log('\n🚀 Bundle analysis complete!');
console.log('📊 View detailed analysis: .next/analyze/client.html');
console.log('📈 Server analysis: .next/analyze/server.html\n');

// Performance tips
console.log('💡 Quick Performance Tips:');
console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n');
console.log('1. Move heavy components to separate route segments');
console.log('2. Use Suspense boundaries around dynamic imports');
console.log('3. Implement proper loading skeletons');
console.log('4. Consider server components for static content');
console.log('5. Use React.memo() for expensive re-renders');
console.log('6. Optimize third-party scripts with next/script');
console.log('7. Enable experimental.optimizeCss in next.config.js');
console.log('8. Use dynamic imports with { loading: Component }');
