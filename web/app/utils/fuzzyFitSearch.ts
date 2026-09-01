const romanLevels: Record<string, string> = {
    i: "1",
    ii: "2",
    iii: "3",
    iv: "4",
    v: "5",
};

function tokens(value: string): string[] {
    return value
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, " ")
        .trim()
        .split(/\s+/)
        .filter(Boolean)
        .map(token => romanLevels[token] ?? token);
}

/**
 * Word-aware fitting search. Every query token must match, but abbreviated
 * word prefixes and Roman/Arabic tech levels are accepted. This deliberately
 * avoids loose character soup: `mega p 2` finds Mega Pulse Laser II without
 * making a short query match unrelated modules unpredictably.
 */
export function fuzzyFitMatch(query: string, candidate: string): boolean {
    const queryTokens = tokens(query);
    if (queryTokens.length === 0) return true;
    const candidateTokens = tokens(candidate);
    return queryTokens.every(queryToken =>
        candidateTokens.some(candidateToken =>
            candidateToken === queryToken || candidateToken.startsWith(queryToken),
        ),
    );
}

export function fuzzyFitScore(query: string, candidate: string): number {
    const queryTokens = tokens(query);
    const candidateTokens = tokens(candidate);
    let score = 0;
    for (const queryToken of queryTokens) {
        const exactIndex = candidateTokens.findIndex(candidateToken => candidateToken === queryToken);
        if (exactIndex >= 0) {
            score += exactIndex * 2;
            continue;
        }
        const prefixIndex = candidateTokens.findIndex(candidateToken => candidateToken.startsWith(queryToken));
        score += prefixIndex >= 0 ? 10 + prefixIndex * 2 : 1000;
    }
    return score + candidateTokens.length;
}
