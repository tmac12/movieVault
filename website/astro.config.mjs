import { defineConfig } from 'astro/config';
import mdx from '@astrojs/mdx';

export default defineConfig({
  integrations: [mdx()],
  // Use relative URLs for deployment flexibility
  // Set SITE env var if you need absolute URLs: SITE=http://your-domain.com npm run build
  output: 'static',
  trailingSlash: 'never', // Prevent trailing slash redirects
});
