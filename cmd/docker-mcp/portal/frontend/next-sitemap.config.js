/**
 * Sitemap configuration for MCP Portal
 * Uses NEXT_PUBLIC_SITE_URL environment variable for deployment flexibility
 */

/** @type {import('next-sitemap').IConfig} */
module.exports = {
  siteUrl: process.env.NEXT_PUBLIC_SITE_URL || 'http://localhost:3000',
  generateRobotsTxt: true,
  changefreq: 'weekly',
  priority: 0.7,
  sitemapSize: 5000,
  generateIndexSitemap: true,

  // Exclude paths
  exclude: [
    '/api/*',
    '/api/**',
    '/server-sitemap.xml',
    '/404',
    '/500',
    '/_error',
    '/admin/*',
    '/admin/**',
    '/settings/*',
    '/profile/*',
  ],

  // Robots.txt configuration
  robotsTxtOptions: {
    additionalSitemaps: [
      `${process.env.NEXT_PUBLIC_SITE_URL || 'http://localhost:3000'}/server-sitemap.xml`,
    ],
    policies: [
      {
        userAgent: '*',
        allow: '/',
        disallow: [
          '/api/',
          '/admin/',
          '/settings/',
          '/profile/',
          '/*.json$',
          '/_next/',
        ],
      },
      {
        userAgent: 'Googlebot',
        allow: '/',
        disallow: ['/api/', '/admin/'],
        crawlDelay: 0,
      },
      {
        userAgent: 'bingbot',
        allow: '/',
        disallow: ['/api/', '/admin/'],
        crawlDelay: 1,
      },
    ],
  },

  // Transform function for customizing entries
  transform: async (config, path) => {
    // Set higher priority for important pages
    const importantPages = ['/', '/dashboard', '/servers', '/catalog'];
    const isImportant = importantPages.includes(path);

    // Set different changefreq based on page type
    let changefreq = 'weekly';
    if (path === '/') {
      changefreq = 'daily';
    } else if (path.startsWith('/docs')) {
      changefreq = 'monthly';
    } else if (path.startsWith('/blog')) {
      changefreq = 'weekly';
    }

    return {
      loc: path,
      changefreq,
      priority: isImportant ? 1.0 : 0.7,
      lastmod: config.autoLastmod ? new Date().toISOString() : undefined,
      alternateRefs: config.alternateRefs ?? [],
    };
  },

  // Additional paths to include
  additionalPaths: async () => {
    const result = [];

    // Add dynamic server pages (example)
    // In production, fetch actual server IDs from database
    const serverIds = ['example-server-1', 'example-server-2'];
    for (const id of serverIds) {
      result.push({
        loc: `/servers/${id}`,
        priority: 0.6,
        changefreq: 'weekly',
        lastmod: new Date().toISOString(),
      });
    }

    return result;
  },
};
