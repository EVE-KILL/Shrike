import type {AdminEsiOverviewResponses} from '../../../shared/api/types.gen'

export type AdminSummary=AdminEsiOverviewResponses[200]
// The parent owns the request. Overview and ESI panels share its successful result.
