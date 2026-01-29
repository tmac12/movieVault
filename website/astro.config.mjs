import { defineConfig } from 'astro/config';
import mdx from '@astrojs/mdx';

export default defineConfig({
  integrations: [mdx()],
  // Use relative URLs for deployment flexibility
  // For proper social sharing and SEO, set site to your actual domain
  // In production: site: 'https://yourdomain.com'
  // In development: site: 'http://localhost:4321'
  site: import.meta.env.SITE || 'http://localhost:4321',
  output: 'static',
  trailingSlash: 'never', // Prevent trailing slash redirects
});
