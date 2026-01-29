#!/usr/bin/env node

/**
 * Generate favicon PNG files from SVG source
 *
 * This script uses one of the following (in order of preference):
 * 1. sharp (npm package) - fastest and best quality
 * 2. svg2png (npm package) - good alternative
 * 3. Falls back to manual instructions
 *
 * Install dependencies:
 *   npm install --save-dev sharp
 *   or
 *   npm install --save-dev svg2png
 */

const fs = require('fs');
const path = require('path');

// Add website node_modules to require path
const websiteNodeModules = path.join(__dirname, '../website/node_modules');
if (fs.existsSync(websiteNodeModules)) {
  module.paths.unshift(websiteNodeModules);
}

const SVG_SOURCE = path.join(__dirname, '../website/public/favicon.svg');
const OUTPUT_DIR = path.join(__dirname, '../website/public');

// Favicon sizes to generate
const SIZES = {
  'favicon-16x16.png': 16,
  'favicon-32x32.png': 32,
  'favicon-48x48.png': 48,
  'apple-touch-icon.png': 180,
  'apple-touch-icon-152x152.png': 152,
  'apple-touch-icon-167x167.png': 167,
  'apple-touch-icon-180x180.png': 180,
  'android-chrome-192x192.png': 192,
  'android-chrome-512x512.png': 512,
  'mstile-150x150.png': 150,
};

async function generateWithSharp() {
  try {
    const sharp = require('sharp');
    console.log('Using sharp for conversion (best quality)\n');

    const svgBuffer = fs.readFileSync(SVG_SOURCE);

    for (const [filename, size] of Object.entries(SIZES)) {
      const outputPath = path.join(OUTPUT_DIR, filename);

      await sharp(svgBuffer)
        .resize(size, size)
        .png()
        .toFile(outputPath);

      console.log(`  ✓ Generated ${filename} (${size}x${size})`);
    }

    return true;
  } catch (error) {
    return false;
  }
}

async function generateWithSvg2Png() {
  try {
    const svg2png = require('svg2png');
    console.log('Using svg2png for conversion\n');

    const svgBuffer = fs.readFileSync(SVG_SOURCE);

    for (const [filename, size] of Object.entries(SIZES)) {
      const outputPath = path.join(OUTPUT_DIR, filename);

      const pngBuffer = await svg2png(svgBuffer, { width: size, height: size });
      fs.writeFileSync(outputPath, pngBuffer);

      console.log(`  ✓ Generated ${filename} (${size}x${size})`);
    }

    return true;
  } catch (error) {
    return false;
  }
}

function showManualInstructions() {
  console.log('\n❌ No conversion libraries found!\n');
  console.log('Please install one of the following:\n');
  console.log('Option 1 (Recommended):');
  console.log('  cd website && npm install --save-dev sharp\n');
  console.log('Option 2:');
  console.log('  cd website && npm install --save-dev svg2png\n');
  console.log('Option 3 (System tools):');
  console.log('  brew install librsvg  # macOS');
  console.log('  apt install librsvg2-bin  # Ubuntu');
  console.log('  Then run: ./generate-favicons.sh\n');
  console.log('Option 4 (Online):');
  console.log('  Visit https://realfavicongenerator.net');
  console.log('  Upload: website/public/favicon.svg');
  console.log('  Download generated favicons to website/public/\n');
}

async function main() {
  console.log('Generating favicons from SVG source...\n');

  // Check if source exists
  if (!fs.existsSync(SVG_SOURCE)) {
    console.error(`Error: SVG source not found at ${SVG_SOURCE}`);
    process.exit(1);
  }

  // Try sharp first (best quality, fastest)
  let success = await generateWithSharp();

  // Try svg2png if sharp fails
  if (!success) {
    success = await generateWithSvg2Png();
  }

  // Show manual instructions if all automated methods fail
  if (!success) {
    showManualInstructions();
    process.exit(1);
  }

  console.log('\n✅ Favicon generation complete!\n');
  console.log('Generated files:');

  for (const filename of Object.keys(SIZES)) {
    const filePath = path.join(OUTPUT_DIR, filename);
    if (fs.existsSync(filePath)) {
      const stats = fs.statSync(filePath);
      const sizeKb = (stats.size / 1024).toFixed(1);
      console.log(`  ${filename} (${sizeKb} KB)`);
    }
  }

  console.log('\nNext steps:');
  console.log('1. Check that the generated images look good');
  console.log('2. The BaseLayout.astro has been updated with favicon meta tags');
  console.log('3. Commit the new favicon files');
  console.log('4. Test in browser: http://localhost:4321\n');
}

main().catch(error => {
  console.error('Error:', error.message);
  process.exit(1);
});
