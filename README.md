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
- 📄 Jellyfin NFO file support with TMDB fallback and NFO image downloads
- 👀 Watch mode - automatically scan when new files are added
- ⏰ Scheduled scanning - periodic scans at configurable intervals
- 🔁 Duplicate detection with quality comparison and recommendations
- 💾 SQLite cache for TMDB API responses with hit/miss statistics
- 🖼️ Automatic cover and backdrop image downloads
- 📱 Fully responsive design with dark theme
- 🚀 Concurrent file processing with configurable worker pool (5x faster)

## Tech Stack

- **Scanner**: Go 1.21+
- **Metadata**: TMDB API
- **Static Site**: Astro with MDX
- **Styling**: Modern CSS with dark theme
- **Deployment**: Docker or native (nginx/Apache/Caddy)

## Using Pre-Built Images

Pre-built Docker images are available on GitHub Container Registry for faster deployment:

```bash
# Pull latest version
docker pull ghcr.io/tmac12/movievault:latest

# Pull specific version
docker pull ghcr.io/tmac12/movievault:v1.2.0
```

To use pre-built images, update your `docker-compose.yml`:

```yaml
services:
  movievault:
    # Comment out the build line:
    # build: .

    # Use published image instead:
    image: ghcr.io/tmac12/movievault:latest
```

See [DOCKER_REGISTRY.md](DOCKER_REGISTRY.md) for complete publishing and usage instructions.

## Quick Start Without Cloning (Docker Image Only)

You can use MovieVault with just the Docker image, without cloning the entire repository.

### Option 1: Using `docker run` (Simplest)

**Minimal setup with just environment variables:**

```bash
# Create directories for data
mkdir -p ~/movievault-data/{movies,covers}

# Run the container
docker run -d \
  --name movievault \
  -p 8080:80 \
  -v /path/to/your/movies:/movies:ro \
  -v ~/movievault-data/movies:/data/movies \
  -v ~/movievault-data/covers:/data/covers \
  -e TMDB_API_KEY=your_tmdb_api_key_here \
  -e SCHEDULE_ENABLED=true \
  -e SCHEDULE_INTERVAL=60 \
  ghcr.io/tmac12/movievault:latest
```

**Then access your collection:**
```
http://localhost:8080
```

**Configuration notes:**
- `-v /path/to/your/movies:/movies:ro` - Mount your movie directory (read-only)
- `-v ~/movievault-data/movies:/data/movies` - Stores generated MDX files
- `-v ~/movievault-data/covers:/data/covers` - Stores downloaded images
- `-e TMDB_API_KEY=...` - Your TMDB API key (required)
- `-e SCHEDULE_ENABLED=true` - Enable scheduled periodic scans (new in v1.4.0)
- `-e SCHEDULE_INTERVAL=60` - Scan every 60 minutes (default: 60)

### Option 2: Minimal docker-compose.yml

**If you prefer docker-compose**, create just these two files:

**1. Create `.env` file:**
```env
TMDB_API_KEY=your_actual_api_key_here
SCHEDULE_ENABLED=true      # Enable scheduled periodic scans (new in v1.4.0)
SCHEDULE_INTERVAL=60       # Scan every 60 minutes (default: 60)
WEB_PORT=8080
```

**2. Create `docker-compose.yml` file:**
```yaml
services:
  movievault:
    image: ghcr.io/tmac12/movievault:latest
    container_name: movievault
    ports:
      - "${WEB_PORT:-8080}:80"
    volumes:
      # Your movie directories (read-only)
      - /path/to/your/movies:/movies:ro
      - /path/to/external/drive:/movies2:ro

      # Data persistence
      - ./data/movies:/data/movies
      - ./data/covers:/data/covers
    environment:
      - TMDB_API_KEY=${TMDB_API_KEY}
      - SCHEDULE_ENABLED=${SCHEDULE_ENABLED:-false}
      - SCHEDULE_INTERVAL=${SCHEDULE_INTERVAL:-60}
    restart: unless-stopped
```

**3. Start the container:**
```bash
docker-compose up -d
```

### Option 3: With Custom Configuration

**For advanced configuration**, create a config file:

**1. Download the config template:**
```bash
# Create config directory
mkdir -p config

# Create config file
cat > config/config.docker.yaml << 'EOF'
tmdb:
  api_key: "${TMDB_API_KEY}"
  language: "en-US"

scanner:
  directories:
    - "/movies"
    - "/movies2"
  extensions:
    - ".mkv"
    - ".mp4"
    - ".avi"
    - ".mov"
    - ".m4v"

output:
  mdx_dir: "/data/movies"
  covers_dir: "/data/covers"
  auto_build: true

options:
  rate_limit_delay: 250
  download_covers: true
  download_backdrops: true
  use_nfo: true
  nfo_fallback_tmdb: true
EOF
```

**2. Update docker-compose.yml to use it:**
```yaml
services:
  movievault:
    image: ghcr.io/tmac12/movievault:latest
    volumes:
      - /path/to/movies:/movies:ro
      - ./data:/data
      - ./config/config.docker.yaml:/config/config.yaml:ro  # Add this
    environment:
      - TMDB_API_KEY=${TMDB_API_KEY}
```

### Managing the Container

```bash
# View logs
docker logs -f movievault

# Trigger manual scan
docker exec movievault scanner

# Force refresh all metadata
docker exec movievault scanner --force-refresh

# Stop the container
docker stop movievault

# Update to latest version
docker pull ghcr.io/tmac12/movievault:latest
docker stop movievault
docker rm movievault
# Then run the docker run command again
```

**Scheduled Scanning in Docker:**

When `SCHEDULE_ENABLED=true`, the scanner runs continuously in the background, performing periodic scans every `SCHEDULE_INTERVAL` minutes. This is ideal for Docker deployments as it eliminates the need for external cron jobs or manual scans. The scanner will:
- Run an initial scan on startup (configurable via `schedule_on_startup`)
- Continue scanning at the configured interval
- Process only new files (incremental scans)
- Automatically rebuild the Astro site after each scan (if `auto_build` is enabled)
- Run alongside the nginx web server in the same container

### Get Your TMDB API Key

1. Create a free account at [themoviedb.org](https://www.themoviedb.org/signup)
2. Go to [API Settings](https://www.themoviedb.org/settings/api)
3. Request an API key (choose "Developer" option)
4. Use the "API Key (v3 auth)" value

---

## Quick Start (Docker)

### Prerequisites
- Docker and Docker Compose
- TMDB API key ([get one here](https://www.themoviedb.org/settings/api))

### Setup

**Choose one:** Build locally OR use pre-built images (see above)

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

# Watch mode - continuously monitor directories for new files
./scanner --watch

# Scheduled scanning - periodic scans every N minutes
./scanner --schedule --schedule-interval 30  # Scan every 30 minutes
./scanner --schedule                         # Use config defaults (60 min)

# Combined modes - watch + schedule together
./scanner --watch --schedule

# Concurrent processing - override number of workers
./scanner --workers 10  # Use 10 concurrent workers (default: 5)

# Test title extraction without running a full scan
./scanner --test-parser "Movie.Name.2020.1080p.BluRay.mkv"

# Find duplicate movies in your library
./scanner --find-duplicates
./scanner --find-duplicates --detailed

# Show cache hit/miss statistics
./scanner --cache-stats
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
- `concurrent_workers`: Number of concurrent workers for parallel scanning (default: `5`, range: 1-20)

### Output Settings

- `mdx_dir`: Where to write MDX files
- `covers_dir`: Where to save cover images
- `auto_build`: Automatically build Astro after scanning
- `cleanup_missing`: Remove MDX for deleted movie files

### Watch Mode Settings

- `watch_mode`: Enable continuous directory monitoring (`false` by default)
- `watch_debounce`: Seconds to wait after a file change before processing (default: `30`)
- `watch_recursive`: Watch subdirectories recursively (default: `true`)

### Scheduled Scanning Settings

- `schedule_enabled`: Enable scheduled periodic scans (`false` by default)
- `schedule_interval`: Minutes between scans (default: `60`)
- `schedule_on_startup`: Run immediately on startup (default: `true`)

**Note:** Watch mode and scheduled scanning can run simultaneously (watch = immediate, schedule = periodic validation)

### Options

- `rate_limit_delay`: Milliseconds between TMDB API requests (250 recommended)
- `download_covers`: Download cover images locally
- `download_backdrops`: Download backdrop images
- `use_nfo`: Enable Jellyfin `.nfo` file parsing (default: `true`)
- `nfo_fallback_tmdb`: Fall back to TMDB if NFO is missing or incomplete (default: `true`)
- `nfo_download_images`: Download images from NFO URLs before trying TMDB (default: `false`)

### Retry Settings

- `max_attempts`: Number of retries for transient API errors (default: `3`)
- `initial_backoff_ms`: Starting backoff delay in ms, doubles each retry (default: `1000`)

### Cache Settings

- `enabled`: Enable local SQLite cache for TMDB responses (default: `true`)
- `path`: Path to the SQLite cache database file (default: `./data/cache.db`)
- `ttl_days`: Days before a cache entry expires (default: `30`)

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
- Audio: `AAC`, `DTS`, `DD5.1`, `DTS-HD`, `TrueHD`, `Atmos`, `FLAC`
- Release groups: `-SPARKS`, `-YIFY`, `[YTS]`, `[RARBG]`
- Edition markers: `Extended`, `Director's Cut`, `IMAX`, `Remastered`, `Theatrical`
- Year-starting titles: `2001.A.Space.Odyssey.1968.mkv` → "2001 A Space Odyssey" (1968)

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

### Automated Scanning

**Option 1: Built-in Scheduled Scanning (Recommended)**

Use the built-in scheduler for continuous operation:

```bash
# Run scanner with scheduled scans every hour
./scanner --schedule --schedule-interval 60
```

Or configure in `config.yaml`:
```yaml
scanner:
  schedule_enabled: true
  schedule_interval: 60    # Minutes
  schedule_on_startup: true
```

**Option 2: Cron (Traditional)**

Add to crontab for periodic one-shot scans:

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
- Video thumbnail generation
- Collections/playlists
- Statistics dashboard
- PWA support for offline access
- IMDb integration (OMDb API)
- Multi-format NFO support (Kodi, Plex, Emby)
- NFO validation and repair

## Credits

- Movie data from [The Movie Database (TMDB)](https://www.themoviedb.org)
- Built with [Astro](https://astro.build)
- Powered by Go

## License

This project is for personal use. TMDB API usage subject to [TMDB Terms of Use](https://www.themoviedb.org/terms-of-use).

## Support

For issues or questions, please open an issue on GitHub.
