// These global controllers compile into site.js, rather than separate entries.
// Everything else retains its current public module name during migration.
export const shellControllers=new Set(['shell','navigation-progress','account','announcements','kill-submit','home'])
