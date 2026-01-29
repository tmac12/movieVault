# Favicon Files

This directory contains comprehensive favicon files for all platforms and devices.

## Generated Files

### Standard Favicons
- `favicon.ico` - Classic ICO format (16x16, 32x32, 48x48 combined)
- `favicon.svg` - Modern scalable vector format (source file)
- `favicon-16x16.png` - Small favicon
- `favicon-32x32.png` - Standard favicon
- `favicon-48x48.png` - Large favicon

### Apple Touch Icons (iOS/macOS)
- `apple-touch-icon.png` (180x180) - Default iOS home screen
- `apple-touch-icon-152x152.png` - iPad non-Retina
- `apple-touch-icon-167x167.png` - iPad Retina
- `apple-touch-icon-180x180.png` - iPhone Retina

### Android/Chrome
- `android-chrome-192x192.png` - Standard Android
- `android-chrome-512x512.png` - High-res Android

### Microsoft
- `mstile-150x150.png` - Windows tiles
- `browserconfig.xml` - Windows tile configuration

### Web Manifest
- `site.webmanifest` - PWA manifest with app metadata

## Design

The favicon features a **film reel design** with:
- Red/crimson gradient outer ring (cinema theme)
- Dark center hub
- Film sprocket holes at cardinal points
- Central play button icon
- Clean, recognizable at small sizes

**Color scheme:**
- Primary: #ff6b6b (coral red)
- Secondary: #c0392b (crimson)
- Accent: #2c3e50 (dark blue-grey)

## Regenerating Favicons

If you modify `favicon.svg`, regenerate all PNG formats:

### Using Node.js (Recommended)
```bash
cd scripts
node generate-favicons.js
```

**Requirements:**
- Node.js installed
- sharp package: `cd website && npm install --save-dev sharp`

### Using Shell Script
```bash
cd scripts
./generate-favicons.sh
```

**Requirements:**
- librsvg: `brew install librsvg` (macOS)
- Or ImageMagick: `brew install imagemagick`

### Using Online Tool
1. Visit https://realfavicongenerator.net
2. Upload `favicon.svg`
3. Download and extract to `website/public/`

## Browser Support

All major browsers and platforms are supported:

- ✅ Chrome/Edge (Windows, macOS, Linux, Android)
- ✅ Firefox (all platforms)
- ✅ Safari (macOS, iOS, iPadOS)
- ✅ Opera
- ✅ Samsung Internet
- ✅ iOS home screen icons
- ✅ Android home screen icons
- ✅ Windows tiles
- ✅ PWA manifest

## Meta Tags

The favicon meta tags are configured in `website/src/layouts/BaseLayout.astro`:

```html
<!-- Standard -->
<link rel="icon" type="image/svg+xml" href="/favicon.svg">
<link rel="icon" type="image/png" sizes="32x32" href="/favicon-32x32.png">
<link rel="shortcut icon" href="/favicon.ico">

<!-- Apple -->
<link rel="apple-touch-icon" sizes="180x180" href="/apple-touch-icon.png">

<!-- Android/Chrome -->
<link rel="manifest" href="/site.webmanifest">

<!-- Microsoft -->
<meta name="msapplication-TileColor" content="#ff6b6b">
<meta name="msapplication-config" content="/browserconfig.xml">

<!-- Theme -->
<meta name="theme-color" content="#ff6b6b">
```

## Testing

### Local Testing
1. Start dev server: `cd website && npm run dev`
2. Visit http://localhost:4321
3. Check browser tab for favicon
4. Check browser dev tools → Application → Icons

### Mobile Testing
1. Build and deploy: `npm run build`
2. Add to home screen on iOS/Android
3. Verify icon appears correctly

### Online Testing
- Favicon checker: https://realfavicongenerator.net/favicon_checker
- Lighthouse audit (PWA icons): Chrome DevTools → Lighthouse

## File Sizes

Total favicon size: ~92 KB

| File | Size | Purpose |
|------|------|---------|
| favicon.svg | 1.2 KB | Modern browsers (scalable) |
| favicon.ico | 1.3 KB | Legacy browsers |
| favicon-16x16.png | 0.6 KB | Browser tab (small) |
| favicon-32x32.png | 1.3 KB | Browser tab (standard) |
| favicon-48x48.png | 2.0 KB | Browser tab (large) |
| apple-touch-icon.png | 8.6 KB | iOS home screen |
| android-chrome-192x192.png | 9.1 KB | Android |
| android-chrome-512x512.png | 30.5 KB | High-res Android |
| mstile-150x150.png | 7.0 KB | Windows tiles |

## License

Part of the MovieVault project. Same license as parent project.
