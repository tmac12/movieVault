#!/bin/bash
# Script to generate favicon PNG files from SVG source
# Requires: librsvg (rsvg-convert) or ImageMagick (convert)

set -e

SVG_SOURCE="../website/public/favicon.svg"
OUTPUT_DIR="../website/public"

echo "Generating favicons from $SVG_SOURCE..."

# Check for rsvg-convert (preferred) or ImageMagick
if command -v rsvg-convert &> /dev/null; then
    CONVERTER="rsvg"
    echo "Using rsvg-convert"
elif command -v convert &> /dev/null; then
    CONVERTER="imagemagick"
    echo "Using ImageMagick"
else
    echo "Error: Neither rsvg-convert nor ImageMagick found"
    echo "Install one of:"
    echo "  - macOS: brew install librsvg"
    echo "  - Ubuntu: apt install librsvg2-bin"
    echo "  - ImageMagick: brew install imagemagick"
    exit 1
fi

# Function to convert SVG to PNG
convert_svg() {
    local size=$1
    local output=$2

    if [ "$CONVERTER" = "rsvg" ]; then
        rsvg-convert -w $size -h $size "$SVG_SOURCE" -o "$output"
    else
        convert -background none -resize ${size}x${size} "$SVG_SOURCE" "$output"
    fi

    echo "  ✓ Generated $output (${size}x${size})"
}

# Generate all required sizes
cd "$(dirname "$0")"

echo ""
echo "Generating PNG favicons..."

# Standard favicon sizes
convert_svg 16 "$OUTPUT_DIR/favicon-16x16.png"
convert_svg 32 "$OUTPUT_DIR/favicon-32x32.png"
convert_svg 48 "$OUTPUT_DIR/favicon-48x48.png"

# Apple touch icons
convert_svg 180 "$OUTPUT_DIR/apple-touch-icon.png"
convert_svg 152 "$OUTPUT_DIR/apple-touch-icon-152x152.png"
convert_svg 167 "$OUTPUT_DIR/apple-touch-icon-167x167.png"
convert_svg 180 "$OUTPUT_DIR/apple-touch-icon-180x180.png"

# Android/Chrome
convert_svg 192 "$OUTPUT_DIR/android-chrome-192x192.png"
convert_svg 512 "$OUTPUT_DIR/android-chrome-512x512.png"

# Microsoft
convert_svg 150 "$OUTPUT_DIR/mstile-150x150.png"

# Generate multi-size ICO (requires ImageMagick)
if command -v convert &> /dev/null; then
    echo ""
    echo "Generating favicon.ico..."
    convert "$OUTPUT_DIR/favicon-16x16.png" \
            "$OUTPUT_DIR/favicon-32x32.png" \
            "$OUTPUT_DIR/favicon-48x48.png" \
            "$OUTPUT_DIR/favicon.ico"
    echo "  ✓ Generated favicon.ico"
fi

echo ""
echo "✅ Favicon generation complete!"
echo ""
echo "Generated files:"
ls -lh "$OUTPUT_DIR"/*.png "$OUTPUT_DIR"/*.ico 2>/dev/null | awk '{print "  " $9 " (" $5 ")"}'
echo ""
echo "Next steps:"
echo "1. Check the generated images look good"
echo "2. The BaseLayout.astro has been updated with favicon meta tags"
echo "3. Commit the new favicon files"
