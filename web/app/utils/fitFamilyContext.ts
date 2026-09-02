export interface FitFamilySecurityContext {
    name: string;
    count: number;
    pct: number;
}

export interface FitFamilyRegionContext {
    region_id: number;
    name: string | null;
    count: number;
    pct: number;
}

export interface FitFamilyContext {
    security_distribution: FitFamilySecurityContext[];
    top_region: FitFamilyRegionContext | null;
    median_attackers: number;
    median_loss_value: number;
}

export function fitFamilyAdvancedSearchQuery(windowDays: number): string {
    return JSON.stringify({ timeRange: { preset: `${windowDays}d` } });
}

export function fitFamilyContextParts(context?: FitFamilyContext | null): string[] {
    if (!context) return [];
    const distribution = [...context.security_distribution].sort((a, b) => b.pct - a.pct);
    const primary = distribution[0];
    const secondary = distribution[1];
    const parts: string[] = [];

    if (primary) {
        if (primary.pct >= 60 || !secondary) {
            parts.push(`${Math.round(primary.pct)}% ${primary.name.toLowerCase()}`);
        } else {
            parts.push(`${Math.round(primary.pct)}% ${primary.name.toLowerCase()} / ${Math.round(secondary.pct)}% ${secondary.name.toLowerCase()}`);
        }
    }

    if (context.top_region?.name
        && context.top_region.pct >= 35
        && context.top_region.name.toLowerCase() !== primary?.name.toLowerCase()) {
        parts.push(`${context.top_region.name} ${Math.round(context.top_region.pct)}%`);
    }

    if (context.median_attackers > 0) {
        const attackers = Math.round(context.median_attackers);
        parts.push(attackers === 1 ? "median solo loss" : `median ${attackers} attackers`);
    }

    if (context.median_loss_value > 0) {
        const value = context.median_loss_value;
        const compact = value >= 1_000_000_000
            ? `${(value / 1_000_000_000).toFixed(value >= 10_000_000_000 ? 0 : 1)}b`
            : `${(value / 1_000_000).toFixed(value >= 100_000_000 ? 0 : 1)}m`;
        parts.push(`median loss ${compact}`);
    }

    return parts;
}
