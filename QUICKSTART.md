# Quick Start Guide

## Option 1: Docker (Recommended)

### Step 1: Get TMDB API Key
1. Visit https://www.themoviedb.org/settings/api
2. Sign up/login and request an API key
3. Copy your API key

### Step 2: Configure
```bash
# Create environment file
cp .env.example .env

# Edit .env and add your API key
nano .env
# Set: TMDB_API_KEY=your_actual_key_here

# Create config file
cp config/config.example.yaml config/config.yaml

# Edit docker-compose.yml to mount your movie directories
nano docker-compose.yml
# Change: - /path/to/your/movies:/movies:ro
# To your actual movie directory path
```

### Step 3: Edit Config
```bash
nano config/config.yaml
```

Update the directories to match Docker paths:
```yaml
scanner:
  directories:
    - "/movies"  # This maps to your host directory in docker-compose.yml
```

### Step 4: Start
```bash
docker-compose up -d
```

### Step 5: Access
Open http://localhost:8080 in your browser

### View Logs
```bash
docker-compose logs -f
```

## Option 2: Native (macOS/Linux)

### Step 1: Prerequisites
```bash
# Install Go (if not installed)
brew install go  # macOS
# or download from https://golang.org/dl/

# Install Node.js (if not installed)
brew install node  # macOS
# or download from https://nodejs.org/
```

### Step 2: Configure
```bash
# Create config file
cp config/config.example.yaml config/config.yaml

# Edit config with your settings
nano config/config.yaml
```

Set your TMDB API key and movie directories:
```yaml
tmdb:
  api_key: "your_actual_key_here"

scanner:
  directories:
    - "/Users/marco/Movies"  # Change to your actual path
    - "/Volumes/External/Films"  # Add more as needed
```

### Step 3: Install Dependencies
```bash
# Go dependencies (automatic)
go mod download

# Node.js dependencies
cd website
npm install
cd ..
```

### Step 4: Build Scanner
```bash
go build -o scanner cmd/scanner/main.go
```

### Step 5: Run Scanner
```bash
./scanner
```

This will:
- Scan your movie directories
- Fetch metadata from TMDB
- Download cover images
- Generate MDX files
- Build the Astro website

### Step 6: View Your Collection

**Development Mode (auto-reload):**
```bash
cd website
npm run dev
```
Access: http://localhost:4321

**Production Mode:**
```bash
cd website
npm run build
npm run preview
```
Access: http://localhost:4321

## Testing with Sample Movie

If you want to test without a full movie library:

1. Create a test directory:
```bash
mkdir -p ~/test-movies
```

2. Create a dummy movie file:
```bash
touch ~/test-movies/The.Matrix.1999.1080p.BluRay.mkv
```

3. Update config to point to test directory:
```yaml
scanner:
  directories:
    - "/Users/marco/test-movies"  # Adjust path
```

4. Run scanner:
```bash
./scanner
```

The scanner will fetch real metadata from TMDB for "The Matrix" even though it's just an empty test file.

## Quick Update Workflow

After adding new movies:

**Docker:**
```bash
docker exec filmscanner scanner
```

**Native:**
```bash
./scanner
# or use the convenience script:
./scripts/update-site.sh
```

## Troubleshooting

### "TMDB API key is required"
- Check your API key in `config/config.yaml`
- Ensure it's not `your_api_key_here` (replace with actual key)

### "No results found for 'Movie Title'"
- Check the filename - scanner extracts title from filename
- Try renaming file to simpler format: `Movie.Title.2023.mkv`
- Use `--verbose` flag to see what title is being searched

### Scanner binary not found (native)
```bash
go build -o scanner cmd/scanner/main.go
```

### npm dependencies not installed (native)
```bash
cd website
npm install
cd ..
```

### Permission denied (Docker)
```bash
# Ensure your movie directories are readable
ls -la /path/to/your/movies

# Check Docker has access to the directory
```

## Next Steps

- Browse your collection at http://localhost:8080 (or :4321 for native)
- Use the search feature to find movies
- Filter by genre
- Click on movies to see detailed information
- Set up automated scanning with cron (native) or SCAN_INTERVAL (Docker)

## Need Help?

See the full README.md for:
- Detailed configuration options
- Production deployment guides
- Advanced features
- Troubleshooting tips
