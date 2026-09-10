export interface TimePoint {
  date: string;
  clicks: number;
}

export interface TopLink {
  link_id: string;
  short_code: string;
  title: string;
  clicks: number;
}

export interface NameCount {
  name: string;
  clicks: number;
}

export interface OverviewData {
  total_clicks: number;
  unique_clicks_estimate: number;
  active_links: number;
  expired_links: number;
  password_protected_links: number;
  disabled_links: number;
  clicks_over_time: TimePoint[];
  top_links: TopLink[];
  top_referrers: NameCount[];
  top_browsers: NameCount[];
  top_devices: NameCount[];
  top_os: NameCount[];
  top_campaigns: NameCount[];
}
