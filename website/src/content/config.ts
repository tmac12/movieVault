import { defineCollection, z } from 'astro:content';

const moviesCollection = defineCollection({
  type: 'content',
  schema: z.object({
    title: z.string(),
    description: z.string(),
    coverImage: z.string(),
    backdropImage: z.string().optional(),
    filePath: z.preprocess((v) => (typeof v === 'string' ? v : ''), z.string()),
    fileName: z.preprocess((v) => (typeof v === 'string' ? v : ''), z.string()),
    rating: z.number(),
    releaseYear: z.number(),
    releaseDate: z.string(),
    runtime: z.number(),
    genres: z.array(z.string()),
    director: z.string(),
    cast: z.array(z.string()),
    tmdbId: z.number(),
    imdbId: z.string().optional(),
    scannedAt: z.coerce.date(),
    fileSize: z.number(),
    sourceDir: z.string().optional(),
  }),
});

export const collections = {
  movies: moviesCollection,
};
