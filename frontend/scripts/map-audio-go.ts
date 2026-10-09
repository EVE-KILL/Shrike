import { createAIIDAudio } from '../src/utils/map/aiidAudioCore'

export const { primeAIIDAudio, playAIIDAlarm } = createAIIDAudio(() => new AudioContext())
