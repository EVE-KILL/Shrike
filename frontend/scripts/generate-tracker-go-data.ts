import {writeChanged} from './io'
import { trackerDashboardWidgets } from '../src/utils/trackerDashboard'
const output = new URL('../../internal/frontend/static/tracker-widgets.json', import.meta.url)
const content = JSON.stringify(trackerDashboardWidgets) + '\n'
await writeChanged(output,content)
