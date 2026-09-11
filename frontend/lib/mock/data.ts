// data.ts — mock data for the /demo route (PRD 9.16, 38). Pure client-side
// fixtures: the demo never calls the API, never writes to any database.

export interface DemoLink {
  id: string;
  short_code: string;
  short_url: string;
  destination_url: string;
  title: string;
  status: "active" | "expired" | "disabled" | "max_clicks_reached";
  password_protected: boolean;
  click_count: number;
  tags: string[];
  utm_source: string;
  utm_medium: string;
  utm_campaign: string;
  created_at: string;
  expires_at: string | null;
}

export interface DemoClick {
  id: string;
  clicked_at: string;
  browser: string;
  os: string;
  device_type: string;
  referrer: string;
}

/** daysAgo returns an ISO timestamp N*24h before now (deterministic per call). */
function daysAgo(n: number, hour = 12): string {
  const d = new Date();
  d.setUTCDate(d.getUTCDate() - n);
  d.setUTCHours(hour, (n * 17) % 60, 0, 0);
  return d.toISOString();
}

/** dailySeries builds a pseudo-random but stable 30-day click series. */
function dailySeries(total: number): { date: string; clicks: number }[] {
  const weights: number[] = [];
  let sum = 0;
  for (let i = 0; i < 30; i++) {
    // Deterministic wave: weekly rhythm + gentle growth + fixed wobble.
    const w = 1 + 0.35 * Math.sin((i / 7) * Math.PI * 2) + (i / 30) * 0.4 + ((i * 13) % 7) / 22;
    weights.push(w);
    sum += w;
  }
  return weights.map((w, i) => {
    const d = new Date();
    d.setUTCDate(d.getUTCDate() - (29 - i));
    return {
      date: d.toISOString().slice(0, 10),
      clicks: Math.round((w / sum) * total),
    };
  });
}

export const demoUser = {
  id: "demo-user-id",
  name: "Demo User",
  email: "demo@linkpulse.local",
};

export const demoTenant = {
  id: "demo-tenant-id",
  name: "Demo Workspace",
  slug: "demo-workspace",
  role: "owner",
};

const SHORT_DOMAIN = process.env.NEXT_PUBLIC_DEFAULT_SHORT_DOMAIN ?? "https://linkpulse.example";

export const demoLinks: DemoLink[] = [
  {
    id: "demo-link-1",
    short_code: "demo2026",
    short_url: `${SHORT_DOMAIN}/demo2026`,
    destination_url: "https://example.com/products/september-launch?utm_source=instagram&utm_medium=social&utm_campaign=september-promo",
    title: "September Product Launch",
    status: "active",
    password_protected: false,
    click_count: 4820,
    tags: ["campaign", "social"],
    utm_source: "instagram",
    utm_medium: "social",
    utm_campaign: "september-promo",
    created_at: daysAgo(28),
    expires_at: null,
  },
  {
    id: "demo-link-2",
    short_code: "webinar-q4",
    short_url: `${SHORT_DOMAIN}/webinar-q4`,
    destination_url: "https://example.com/webinar/q4-roadmap?utm_source=x&utm_medium=social&utm_campaign=q4-webinar",
    title: "Q4 Roadmap Webinar",
    status: "active",
    password_protected: false,
    click_count: 2317,
    tags: ["campaign", "webinar"],
    utm_source: "x",
    utm_medium: "social",
    utm_campaign: "q4-webinar",
    created_at: daysAgo(21),
    expires_at: null,
  },
  {
    id: "demo-link-3",
    short_code: "press-kit",
    short_url: `${SHORT_DOMAIN}/press-kit`,
    destination_url: "https://example.com/press/kit.pdf",
    title: "Press Kit (password)",
    status: "active",
    password_protected: true,
    click_count: 640,
    tags: ["press"],
    utm_source: "",
    utm_medium: "",
    utm_campaign: "",
    created_at: daysAgo(19),
    expires_at: null,
  },
  {
    id: "demo-link-4",
    short_code: "bf-sale",
    short_url: `${SHORT_DOMAIN}/bf-sale`,
    destination_url: "https://example.com/sale/black-friday",
    title: "Black Friday Sale",
    status: "expired",
    password_protected: false,
    click_count: 9310,
    tags: ["campaign", "ecommerce"],
    utm_source: "newsletter",
    utm_medium: "email",
    utm_campaign: "bf-2025",
    created_at: daysAgo(60),
    expires_at: daysAgo(-2),
  },
  {
    id: "demo-link-5",
    short_code: "docs-guide",
    short_url: `${SHORT_DOMAIN}/docs-guide`,
    destination_url: "https://example.com/docs/getting-started",
    title: "Getting Started Guide",
    status: "active",
    password_protected: false,
    click_count: 1804,
    tags: ["docs"],
    utm_source: "",
    utm_medium: "",
    utm_campaign: "",
    created_at: daysAgo(14),
    expires_at: null,
  },
  {
    id: "demo-link-6",
    short_code: "flash-giveaway",
    short_url: `${SHORT_DOMAIN}/flash-giveaway`,
    destination_url: "https://example.com/giveaway?utm_source=tiktok&utm_medium=social&utm_campaign=flash",
    title: "Flash Giveaway (limit reached)",
    status: "max_clicks_reached",
    password_protected: false,
    click_count: 1000,
    tags: ["campaign", "social"],
    utm_source: "tiktok",
    utm_medium: "social",
    utm_campaign: "flash",
    created_at: daysAgo(9),
    expires_at: null,
  },
  {
    id: "demo-link-7",
    short_code: "old-landing",
    short_url: `${SHORT_DOMAIN}/old-landing`,
    destination_url: "https://example.com/legacy",
    title: "Legacy Landing (disabled)",
    status: "disabled",
    password_protected: false,
    click_count: 512,
    tags: [],
    utm_source: "",
    utm_medium: "",
    utm_campaign: "",
    created_at: daysAgo(40),
    expires_at: null,
  },
  {
    id: "demo-link-8",
    short_code: "careers",
    short_url: `${SHORT_DOMAIN}/careers`,
    destination_url: "https://example.com/careers",
    title: "Careers Page",
    status: "active",
    password_protected: false,
    click_count: 866,
    tags: ["evergreen"],
    utm_source: "linkedin",
    utm_medium: "social",
    utm_campaign: "",
    created_at: daysAgo(11),
    expires_at: null,
  },
];

export const demoOverview = {
  total_clicks: 21269,
  unique_clicks_estimate: 8102,
  active_links: 5,
  expired_links: 1,
  password_protected_links: 1,
  disabled_links: 1,
  clicks_over_time: dailySeries(21269),
  top_links: [
    { link_id: "demo-link-1", short_code: "demo2026", title: "September Product Launch", clicks: 4820 },
    { link_id: "demo-link-4", short_code: "bf-sale", title: "Black Friday Sale", clicks: 9310 },
    { link_id: "demo-link-2", short_code: "webinar-q4", title: "Q4 Roadmap Webinar", clicks: 2317 },
    { link_id: "demo-link-5", short_code: "docs-guide", title: "Getting Started Guide", clicks: 1804 },
    { link_id: "demo-link-8", short_code: "careers", title: "Careers Page", clicks: 866 },
  ],
  top_referrers: [
    { name: "instagram.com", clicks: 7420 },
    { name: "x.com", clicks: 4210 },
    { name: "linkedin.com", clicks: 3380 },
    { name: "google.com", clicks: 2860 },
    { name: "newsletter", clicks: 1910 },
    { name: "(direct)", clicks: 1489 },
  ],
  top_browsers: [
    { name: "chrome", clicks: 11260 },
    { name: "safari", clicks: 5940 },
    { name: "firefox", clicks: 2480 },
    { name: "edge", clicks: 1589 },
  ],
  top_devices: [
    { name: "mobile", clicks: 13450 },
    { name: "desktop", clicks: 6920 },
    { name: "tablet", clicks: 899 },
  ],
  top_os: [
    { name: "android", clicks: 8120 },
    { name: "ios", clicks: 6240 },
    { name: "windows", clicks: 4680 },
    { name: "macos", clicks: 2229 },
  ],
  top_campaigns: [
    { name: "september-promo", clicks: 4820 },
    { name: "bf-2025", clicks: 9310 },
    { name: "q4-webinar", clicks: 2317 },
    { name: "flash", clicks: 1000 },
  ],
};

/** demoRecentClicks synthesizes a newest-first click feed for a link. */
export function demoRecentClicks(link: DemoLink): DemoClick[] {
  const browsers = ["chrome", "safari", "chrome", "firefox", "edge", "chrome"];
  const oses = ["android", "ios", "windows", "macos", "android", "ios"];
  const devices = ["mobile", "mobile", "desktop", "desktop", "mobile", "tablet"];
  const referrers = ["instagram.com", "x.com", "google.com", "linkedin.com", "(direct)", "newsletter"];
  return Array.from({ length: 12 }, (_, i) => {
    const d = new Date();
    d.setUTCMinutes(d.getUTCMinutes() - i * 7 - 2);
    return {
      id: `${link.id}-click-${i}`,
      clicked_at: d.toISOString(),
      browser: browsers[(i + link.short_code.length) % browsers.length],
      os: oses[(i * 2 + link.short_code.length) % oses.length],
      device_type: devices[(i + 3) % devices.length],
      referrer: referrers[(i * 5 + link.click_count) % referrers.length],
    };
  });
}

/** demoLinkAnalytics builds the per-link analytics shape from fixtures. */
export function demoLinkAnalytics(link: DemoLink) {
  return {
    link_id: link.id,
    short_code: link.short_code,
    total_clicks: link.click_count,
    unique_clicks_estimate: Math.round(link.click_count * 0.62),
    clicks_over_time: dailySeries(link.click_count),
    top_referrers: demoOverview.top_referrers.slice(0, 4),
    top_browsers: demoOverview.top_browsers.slice(0, 4),
    top_devices: demoOverview.top_devices,
    top_os: demoOverview.top_os.slice(0, 4),
    top_sources: [
      { name: link.utm_source || "(direct)", clicks: Math.round(link.click_count * 0.55) },
      { name: "google.com", clicks: Math.round(link.click_count * 0.25) },
      { name: "newsletter", clicks: Math.round(link.click_count * 0.2) },
    ],
    top_campaigns: link.utm_campaign
      ? [{ name: link.utm_campaign, clicks: link.click_count }]
      : [],
  };
}
