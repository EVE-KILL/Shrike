export type AIIDAlarmBand = 'near' | 'outer'

let audioContext: AudioContext | null = null
const lastPlayedAt: Record<AIIDAlarmBand, number> = { near: 0, outer: 0 }

function context(): AudioContext | null {
    if (!import.meta.client) return null
    audioContext ??= new AudioContext()
    return audioContext
}

export async function primeAIIDAudio() {
    const current = context()
    if (current?.state === 'suspended') await current.resume()
}

function pulse(current: AudioContext, start: number, frequency: number, duration: number, gain: number, type: OscillatorType = 'sine') {
    const oscillator = current.createOscillator()
    const envelope = current.createGain()
    oscillator.type = type
    oscillator.frequency.setValueAtTime(frequency, start)
    oscillator.frequency.exponentialRampToValueAtTime(frequency * 0.72, start + duration)
    envelope.gain.setValueAtTime(0.0001, start)
    envelope.gain.exponentialRampToValueAtTime(gain, start + 0.018)
    envelope.gain.exponentialRampToValueAtTime(0.0001, start + duration)
    oscillator.connect(envelope)
    envelope.connect(current.destination)
    oscillator.start(start)
    oscillator.stop(start + duration + 0.02)
}

export function playAIIDAlarm(band: AIIDAlarmBand) {
    const current = context()
    if (!current || current.state !== 'running') return
    const nowMs = performance.now()
    if (nowMs - lastPlayedAt[band] < 750) return
    lastPlayedAt[band] = nowMs
    const now = current.currentTime + 0.015

    if (band === 'near') {
        // Urgent three-beat proximity alarm.
        for (let beat = 0; beat < 3; beat++) {
            pulse(current, now + beat * 0.21, beat % 2 === 0 ? 920 : 690, 0.15, 0.055, 'sawtooth')
        }
        return
    }

    // Softer two-beat directional scanner cue for the outer watch ring.
    pulse(current, now, 560, 0.22, 0.035, 'sine')
    pulse(current, now + 0.27, 740, 0.22, 0.035, 'triangle')
}
