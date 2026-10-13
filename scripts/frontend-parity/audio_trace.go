package main

import "net/http"

// Opt-in browser regression observer. The analyser passes the app's own audio
// to its real destination; it records neither microphone nor system audio.
func fixtureAudioTrace(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != "/__fixture/audio-trace.js" {
		return false
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store")
	_, _ = w.Write([]byte(fixtureAudioTraceScript))
	return true
}

const fixtureAudioTraceScript = `(() => {
 const Original=window.AudioContext;
 const contexts=[];let starts=0,peak=0;
 function report(){
  if(!document.body)return;
  let output=document.querySelector('[data-fixture-audio]');
  if(!output){output=document.createElement('output');output.dataset.fixtureAudio='';output.style.cssText='position:fixed;left:8px;bottom:48px;z-index:20000;padding:6px;background:#111;border:1px solid #555;color:#fff;font:11px monospace';document.body.append(output);}
  output.dataset.starts=String(starts);output.dataset.peak=String(peak);output.dataset.state=contexts.map(c=>c.state).join(',');
  output.textContent='Fixture audio: '+(output.dataset.state||'not primed')+' | '+starts+' pulses | peak '+peak.toFixed(5);
 }
 if(!Original){document.addEventListener('DOMContentLoaded',report);return;}
 window.AudioContext=class extends Original{
  constructor(...args){super(...args);this.trace=this.createAnalyser();this.trace.fftSize=512;this.samples=new Float32Array(512);this.trace.connect(super.destination);contexts.push(this);this.addEventListener('statechange',report);report();}
  get destination(){return this.trace||super.destination;}
  createOscillator(){const node=super.createOscillator(),start=node.start.bind(node);node.start=(...args)=>{starts++;report();return start(...args);};return node;}
 };
 document.addEventListener('DOMContentLoaded',report);
 setInterval(()=>{for(const c of contexts){c.trace.getFloatTimeDomainData(c.samples);for(const sample of c.samples)peak=Math.max(peak,Math.abs(sample));}report();},30);
})();`
