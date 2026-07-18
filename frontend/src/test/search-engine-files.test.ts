import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const publicDir = resolve(process.cwd(), 'public')

describe('search engine crawl files', () => {
  it('publishes a robots.txt that points to the canonical sitemap and blocks private surfaces', () => {
    const robots = readFileSync(resolve(publicDir, 'robots.txt'), 'utf8')

    expect(robots).toContain('User-agent: *')
    expect(robots).toContain('Allow: /$')
    expect(robots).toContain('Disallow: /api/')
    expect(robots).toContain('Disallow: /admin')
    expect(robots).toContain('Sitemap: https://earnlearning.com/sitemap.xml')
  })

  it('publishes a canonical XML sitemap with only the public landing page', () => {
    const sitemap = readFileSync(resolve(publicDir, 'sitemap.xml'), 'utf8')

    expect(sitemap).toContain('<?xml version="1.0" encoding="UTF-8"?>')
    expect(sitemap).toContain('<loc>https://earnlearning.com/</loc>')
    expect(sitemap).not.toContain('/admin')
    expect(sitemap).not.toContain('/feed')
  })
})
