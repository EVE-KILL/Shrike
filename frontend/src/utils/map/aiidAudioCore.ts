export type AIIDAlarmBand = 'near' | 'outer'

// The original oscillator/envelope recipe, shared by both browser frontends.
export function createAIIDAudio(createContext: () => AudioContext | null, now = () => performance.now()) {
    let audioContext: AudioContext | null = null
    const lastPlayedAt: Record<AIIDAlarmBand, number> = { near: -Infinity, outer: -Infinity }
    const context = () => audioContext ??= createContext()

    async function primeAIIDAudio() {
        const current = context()
        if (current?.state === 'suspended') await current.resume()
    }
    function pulse(current: AudioContext, start: number, frequency: number, duration: number, gain: number, type: OscillatorType = 'sine') {
        const oscillator = current.createOscillator(), envelope = current.createGain()
        oscillator.type = type
        oscillator.frequency.setValueAtTime(frequency, start)
        oscillator.frequency.exponentialRampToValueAtTime(frequency * .72, start + duration)
        envelope.gain.setValueAtTime(.0001, start)
        envelope.gain.exponentialRampToValueAtTime(gain, start + .018)
        envelope.gain.exponentialRampToValueAtTime(.0001, start + duration)
        oscillator.connect(envelope); envelope.connect(current.destination)
        oscillator.start(start); oscillator.stop(start + duration + .02)
    }
    function playAIIDAlarm(band: AIIDAlarmBand) {
        const current = context()
        if (!current || current.state !== 'running') return
        const stamp = now()
        if (stamp - lastPlayedAt[band] < 750) return
        lastPlayedAt[band] = stamp
        const start = current.currentTime + .015
        if (band === 'near') {
            for (let beat = 0; beat < 3; beat++) pulse(current, start + beat * .21, beat % 2 ? 690 : 920, .15, .055, 'sawtooth')
        } else {
            pulse(current, start, 560, .22, .035)
            pulse(current, start + .27, 740, .22, .035, 'triangle')
        }
    }
    return { primeAIIDAudio, playAIIDAlarm }
}
