package frontend

func accountMenuRow(mobile, danger bool) string {
	if mobile {
		return "flex items-center gap-3 px-4 py-3 rounded-lg " + killBoolClass(danger, "text-red-400 hover:bg-red-500/10", "text-white hover:bg-blue-500/[0.06]")
	}
	return "group/dropdown flex items-center gap-2.5 rounded-lg px-3 py-2.5 text-sm transition-colors " + killBoolClass(danger, "text-red-400 hover:bg-red-500/10 hover:text-red-300", "text-gray-400 hover:bg-blue-500/[0.08] hover:text-blue-300")
}
func accountMenuIcon(mobile, danger bool) string {
	if mobile {
		return "text-lg opacity-60"
	}
	return "flex h-7 w-7 shrink-0 items-center justify-center rounded-md border transition-colors " + killBoolClass(danger, "border-red-400/10 bg-red-500/[0.04] text-red-400/70 group-hover/dropdown:border-red-400/20 group-hover/dropdown:bg-red-500/[0.08] group-hover/dropdown:text-red-300", "border-white/[0.06] bg-white/[0.035] text-blue-400/70 group-hover/dropdown:border-blue-400/20 group-hover/dropdown:bg-blue-500/[0.08] group-hover/dropdown:text-blue-300")
}
