// Test the actual panorama texture compositor with diagnostic image data.
import {readFileSync} from 'node:fs';
import vm from 'node:vm';
import assert from 'node:assert/strict';
const renderer=readFileSync(new URL('runtime/renderer.js',import.meta.url),'utf8');
const stitch=vm.runInNewContext(renderer.slice(renderer.indexOf('function stitchPanorama('),renderer.indexOf('function uploadPanorama('))+';stitchPanorama');
const w=160,h=64,overlap=Math.floor(w*.24),middle=w-2*overlap;
function input(pixel){const out=new Uint8ClampedArray(w*h*4);for(let y=0;y<h;y++)for(let x=0;x<w;x++){const p=pixel(x,y);for(let c=0;c<3;c++)out[(y*w+x)*4+c]=p[c];out[(y*w+x)*4+3]=255;}return out;}
const constant=input(()=>[77,128,203]),flat=stitch(constant,w,h);
assert.ok(flat.pixels.every((v,i)=>v===([77,128,203,255][i%4])),'constant colors survive all pyramid levels');
const source=input((x,y)=>[x,80+y,200-Math.floor(x/2)]),saved=source.slice(),result=stitch(source,w,h);
assert.deepEqual(source,saved,'original image stays untouched');
assert.equal(result.width,w-overlap);assert.equal(result.height,h);
for(let y=0;y<h;y++){
 for(let x=0;x<middle;x++)for(let c=0;c<3;c++)assert.equal(result.pixels[(y*result.width+x)*4+c],source[(y*w+overlap+x)*4+c],'nonoverlap artwork stays exact');
 for(const x of [0,middle])for(let c=0;c<3;c++){
 const previous=x===0?result.width-1:x-1;
 assert.ok(Math.abs(result.pixels[(y*result.width+x)*4+c]-result.pixels[(y*result.width+previous)*4+c])<=1,'no discontinuity at wrap or overlap entry');
 }
}
assert.ok(result.pixels.every((v,i)=>i%4!==3||v===255),'no transparent gaps');
assert.ok(!renderer.includes('panoramaCoordinate'),'no mirrored reversal');
assert.ok(!renderer.includes('float d=texture2D(uDepth,panoramaUv(uv))'),'unstitched depth cannot tear the join');
console.log('PASS: unchanged source, constant-color reconstruction, exact interior, opaque output and continuous panorama boundaries.');
