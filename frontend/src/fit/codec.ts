// The existing v3 fit-link contract: gzip UTF-8 document, then base64.
export async function encodeFitText(text:string):Promise<string> {
  const compressed = await new Response(new Blob([text]).stream().pipeThrough(new CompressionStream('gzip'))).arrayBuffer()
  const bytes = new Uint8Array(compressed);let binary = ''
  for(let offset=0;offset<bytes.length;offset+=0x8000) binary += String.fromCharCode(...bytes.subarray(offset,offset+0x8000))
  return 'v3:'+btoa(binary)
}
