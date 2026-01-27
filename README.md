# MovieVault

A multimedia file scanner that discovers movie files, scrapes metadata from TMDB API, generates MDX files, and uses Astro to build a beautiful static website for your personal movie collection.

## Features

- 🎬 Automatic movie file discovery and metadata fetching from TMDB
- 📝 MDX file generation (one per movie) with rich frontmatter
- 🌐 Beautiful static website built with Astro
- 🔍 Search and filter by genre
- 📊 Collection statistics (total movies, size, runtime)
- 🐳 Docker support for easy deployment
- ⚡ Smart scanning - only processes new files (existing movies are skipped)
- 🖼️ Automatic cover and backdrop image downloads
- 📱 Fully responsive design with dark theme

## Tech Stack

- **Scanner**: Go 1.21+
- **Metadata**: TMDB API
- **Static Site**: Astro with MDX
- **Styling**: Modern CSS with dark theme
- **Deployment**: Docker or native (nginx/Apache/Caddy)

## Quick Start (Docker)

### Prerequisites
- Docker and Docker Compose
- TMDB API key ([get one here](https://www.themoviedb.org/settings/api))

### Setup

1. Clone or setup the project:
```bash
cd movieVault
```

2. Create environment file:
```bash
cp .env.example .env
```

3. Edit `.env` and add your TMDB API key:
```env
TMDB_API_KEY=your_actual_api_key_here
```

4. Update `docker-compose.yml` to mount your movie directories:
```yaml
volumes:
  - /path/to/your/movies:/movies:ro
  - /path/to/external/drive:/external-movies:ro
```

5. Create configuration file:
```bash
cp config/config.example.yaml config/config.yaml
```

6. Edit `config/config.yaml` with your settings:
```yaml
tmdb:
  api_key: "${TMDB_API_KEY}"

scanner:
  directories:
    - "/movies"              # Docker mount paths
    - "/external-movies"
```

7. Start the services:
```bash
docker-compose up -d
```

8. Access your movie collection:
```
http://localhost:8080
```

### Docker Commands

```bash
# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Rebuild after code changes
docker-compose build && docker-compose up -d

# Manual scan trigger
docker exec movievault scanner

# Force refresh all metadata
docker exec movievault scanner --force-refresh
```

## Quick Start (Native)

### Prerequisites
- Go 1.21+
- Node.js 18+
- TMDB API key

### Setup

1. Install dependencies:
```bash
# Go dependencies (automatic with go build)
go mod download

# Node.js dependencies
cd website
npm install
cd ..
```

2. Create configuration:
```bash
cp config/config.example.yaml config/config.yaml
```

3. Edit `config/config.yaml` with your settings:
```yaml
tmdb:
  api_key: "your_api_key_here"

scanner:
  directories:
    - "/home/marco/Movies"
    - "/media/external/Films"

output:
  mdx_dir: "./website/src/content/movies"
  covers_dir: "./website/public/covers"
  auto_build: true
```

4. Build the scanner:
```bash
go build -o scanner cmd/scanner/main.go
```

5. Run the scanner:
```bash
./scanner
```

6. Start the development server:
```bash
cd website
npm run dev
```

7. Access your collection:
```
http://localhost:4321
```

## Usage

### Scanner Command-Line Flags

```bash
# Default behavior (only scans new files)
./scanner

# Force refresh all metadata
./scanner --force-refresh

# Skip Astro build
./scanner --no-build

# Dry run (show what would be done)
./scanner --dry-run

# Verbose output
./scanner --verbose

# Custom config file
./scanner --config /path/to/config.yaml
```

### Update Script

For convenience, use the update script:

```bash
./scripts/update-site.sh
```

This will:
1. Run the scanner
2. Install npm dependencies (if needed)
3. Build the Astro website

## Project Structure

```
movieVault/
├── cmd/scanner/          # CLI scanner application
├── internal/             # Go packages
│   ├── config/          # Configuration loading
│   ├── metadata/        # TMDB API client
│   ├── scanner/         # File system scanner
│   └── writer/          # MDX file generator
├── website/             # Astro static site
│   ├── src/
│   │   ├── pages/      # Routes (index, movie detail, search)
│   │   ├── components/ # Astro components
│   │   ├── layouts/    # Base layout
│   │   ├── styles/     # Global CSS
│   │   └── content/    # Content collections
│   │       └── movies/ # Generated MDX files
│   └── public/
│       └── covers/     # Downloaded cover images
├── config/             # Configuration files
├── docker/             # Docker files
├── scripts/            # Utility scripts
└── data/               # Generated data (Docker)
```

## Configuration Options

### Scanner Settings

- `directories`: Array of paths to scan for movie files
- `extensions`: Supported video file extensions

### Output Settings

- `mdx_dir`: Where to write MDX files
- `covers_dir`: Where to save cover images
- `auto_build`: Automatically build Astro after scanning
- `cleanup_missing`: Remove MDX for deleted movie files

### Options

- `rate_limit_delay`: Milliseconds between TMDB API requests (250 recommended)
- `download_covers`: Download cover images locally
- `download_backdrops`: Download backdrop images

## How the Scanner Works

### First Run
- Scans all video files in configured directories
- Fetches metadata from TMDB for each movie
- Downloads cover and backdrop images
- Creates MDX files
- **Time**: ~1 second per movie

### Subsequent Runs (Default)
- Scans all video files
- **Skips movies that already have MDX files**
- Only fetches TMDB data for new movies
- **Time**: ~0.01 second per existing movie, ~1 second per new movie

### Force Refresh
- Re-fetches ALL metadata from TMDB
- Updates all MDX files
- Useful when TMDB data has been updated
- **Time**: Same as first run

## Filename Parsing

The scanner extracts movie titles from filenames by removing:

- Year markers: `(2023)`, `[2023]`
- Quality: `1080p`, `720p`, `4K`, `BluRay`, `WEB-DL`
- Codecs: `x264`, `x265`, `HEVC`
- Audio: `AAC`, `DTS`, `DD5.1`
- Release groups: `-GROUP`

**Examples:**
- `The.Matrix.1999.1080p.BluRay.x264-GROUP.mkv` → "The Matrix" (1999)
- `Inception (2010) [1080p].mp4` → "Inception" (2010)

## Production Deployment

### Build Static Site

```bash
cd website
npm run build
```

Static files will be in `website/dist/`

### Serve with Nginx

```nginx
server {
    listen 80;
    server_name movies.yourdomain.com;
    root /path/to/movieVault/website/dist;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /covers/ {
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
}
```

### Automated Scanning (Cron)

Add to crontab:

```cron
# Run scanner daily at 3 AM
0 3 * * * cd /path/to/movieVault && ./scripts/update-site.sh >> /var/log/movievault.log 2>&1
```

## Troubleshooting

### "No results found" from TMDB

- Check your API key is valid
- Ensure the movie title is correctly extracted
- Try using `--verbose` to see what title is being searched
- Manually rename the file to have a clearer title

### Images Not Loading

- Check that `download_covers` and `download_backdrops` are enabled
- Ensure the `covers_dir` is accessible to the web server
- Verify images exist in `website/public/covers/`

### Astro Build Fails

- Run `cd website && npm install` to ensure dependencies are installed
- Check for syntax errors in generated MDX files
- Try building with `npm run build` to see detailed errors

### Scanner Not Finding Files

- Verify directory paths are correct and accessible
- Check file extensions match the configured extensions
- Ensure you have read permissions for the directories

## API Rate Limits

TMDB has generous rate limits, but to be safe:
- Default delay between requests: 250ms
- Adjust with `rate_limit_delay` in config
- Use default scanning (skip existing) to minimize API calls

## Future Enhancements

- TV show support with episode tracking
- Watched status tracking
- User ratings and notes
- Duplicate detection
- Video thumbnail generation
- Collections/playlists
- Statistics dashboard
- PWA support for offline access

## Credits

- Movie data from [The Movie Database (TMDB)](https://www.themoviedb.org)
- Built with [Astro](https://astro.build)
- Powered by Go

## License

This project is for personal use. TMDB API usage subject to [TMDB Terms of Use](https://www.themoviedb.org/terms-of-use).

## Support

For issues or questions, please open an issue on GitHub.
