// Deterministic runtime tests. The browser suite separately exercises real WebGL.
import {readFileSync} from 'node:fs';
import vm from 'node:vm';
import assert from 'node:assert/strict';
const root=new URL('../',import.meta.url);
const files=['state.js','flow.js','renderer.js','motion.js','features.js','viewer.js','controls.js','boot.js'];
const source=`'use strict';const COUNTRIES=[{code:'tr',name:'Türkiye'},{code:'jp',name:'Japonya'}];const ASSETS={tr:{cover:'cover.avif',avatar:'avatar.avif'}};\n`+files.map(name=>readFileSync(new URL('scripts/runtime/'+name,root),'utf8')).join('\n');
const elements=new Map(),timers=new Map(),storage=new Map();let nextTimer=1,now=0;
const context2d={drawImage(){},fillRect(){},clearRect(){},globalAlpha:1};
function element(id){
 if(elements.has(id))return elements.get(id);
 const value={id,value:'',hidden:false,disabled:false,style:{},dataset:{},textContent:'',children:[],width:900,height:600,naturalWidth:900,naturalHeight:600,src:'',capture:null,attributes:{},listeners:{},classList:{add(){},remove(){},toggle(){}},
 setAttribute(k,v){this.attributes[k]=v;},addEventListener(k,v,capture){this.listeners[(capture?'capture:':'')+k]=v;},replaceChildren(){this.children=[];},append(v){this.children.push(v);},remove(){},click(){this.clicked=true;},
 getContext(type){return type==='2d'?context2d:null;},getBoundingClientRect(){return {left:0,top:0,width:900,height:600};},setPointerCapture(id){this.capture=id;},hasPointerCapture(id){return this.capture===id;},releasePointerCapture(){this.capture=null;}};
 elements.set(id,value);return value;
}
const tabs=['cover','avatar','hero'].map(kind=>{const x=element('tab-'+kind);x.dataset.image=kind;return x;});
const doc={hidden:false,fullscreenElement:null,body:element('body'),getElementById:element,querySelector:element,querySelectorAll:s=>s==='[data-image]'?tabs:[element('p1'),element('p2'),element('p3')],createElement:type=>element(type+'-'+elements.size),addEventListener(){}};
const matchMedia=()=>({matches:false,addEventListener(){}});
let stoppedTracks=0;
class FakeRecorder{
 static isTypeSupported(type){return type.startsWith('video/webm');}
 constructor(stream,options){this.stream=stream;this.options=options;this.state='inactive';}
 start(){this.state='recording';}
 stop(){this.state='inactive';this.ondataavailable?.({data:new Blob(['video'])});this.onstop?.();}
}
const globals={document:doc,window:{matchMedia,removeEventListener(){},addEventListener(){}},matchMedia,performance:{now:()=>now},Intl,console,Image:class{constructor(){this.naturalWidth=900;this.naturalHeight=600;}decode(){return Promise.resolve();}},ResizeObserver:class{observe(){}},IntersectionObserver:class{observe(){}},requestAnimationFrame:()=>1,setTimeout:(f,delay)=>{const id=nextTimer++;timers.set(id,{f,delay});return id;},clearTimeout:id=>timers.delete(id),localStorage:{getItem:k=>storage.get(k)||null,setItem:(k,v)=>storage.set(k,v)},Blob,MediaRecorder:FakeRecorder,URL:{createObjectURL:()=> 'blob:qa',revokeObjectURL(){}}};
const context=vm.createContext(globals);vm.runInContext(source,context);
await new Promise(resolve=>setImmediate(resolve));
const run=code=>vm.runInContext(code,context);
// Profiles apply all controls, preserve personal settings and reject invalid data.
element('profile-cinematic').listeners.click();assert.equal(run('motionSettings.speed'),65);assert.equal(element('profile-cinematic').attributes['aria-pressed'],'true');
element('saveCustomMotion').listeners.click();assert.equal(JSON.parse(storage.get('atlas-custom-motion-v1')).speed,65);
element('profile-lively').listeners.click();assert.equal(run('motionSettings.elasticity'),85);
element('loadCustomMotion').listeners.click();assert.equal(run('motionSettings.speed'),65);
assert.equal(run('sanitizeMotion({speed:200})'),null);
// Quality adapts after sustained pressure, not a single missed frame.
run('globalThis.qualityTest=new AdaptiveQuality();');
run('qualityTest.observe(.1)');assert.equal(run('qualityTest.scale'),1);
for(let i=0;i<150;i++)run('qualityTest.observe(.04)');assert.ok(run('qualityTest.scale')<1);
const reducedScale=run('qualityTest.scale');
for(let i=0;i<850;i++)run('qualityTest.observe(1/60)');assert.ok(run('qualityTest.scale')>reducedScale);
run('qualityTest.locked=.8');for(let i=0;i<200;i++)run('qualityTest.observe(.04)');assert.equal(run('qualityTest.scale'),.8);
run('qualityTest.locked=null;qualityTest.setMode("high")');assert.equal(run('qualityTest.scale'),1);
// Touch begins at the current camera angle, retains release momentum and pinches.
const scene=element('scene');const pointer=(id,x,y)=>({pointerId:id,pointerType:'touch',clientX:x,clientY:y,target:{closest:()=>null},preventDefault(){}});
run('state.current=[0,0];state.target=[0,0];state.touchZoom=1;');
scene.listeners.pointerdown(pointer(1,200,300));scene.listeners.pointermove(pointer(1,350,300));assert.ok(run('state.target[0]')>.3);
scene.listeners.pointerdown(pointer(2,450,300));scene.listeners.pointermove(pointer(2,550,300));assert.ok(run('state.touchZoom')>1);
scene.listeners.pointerup(pointer(2,550,300));const beforeRelease=run('state.target[0]');scene.listeners.pointerup(pointer(1,350,300));assert.equal(run('state.target[0]'),beforeRelease);assert.equal(run('touchPoints.size'),0);
scene.listeners.keydown({key:'Escape',target:{closest:()=>null},preventDefault(){}});assert.equal(run('state.touchZoom'),1);assert.equal(run('state.target[0]'),0);
// Recording uses a bounded output canvas, stops its stream and emits one file.
const nativeCreate=doc.createElement;doc.createElement=type=>{const x=nativeCreate(type);if(type==='canvas')x.captureStream=()=>({getTracks:()=>[{stop(){stoppedTracks++;}}]});return x;};
element('canvas').captureStream=()=>({});
context.savedVideos=[];context.recordStates=[];
run('globalThis.recordTest=new CanvasVideoRecorder(canvas,value=>recordStates.push(value),message=>{throw Error(message)},(blob,name)=>savedVideos.push({size:blob.size,name}));');
assert.equal(run('recordTest.start("tr_qa")'),true);assert.equal(run('recordTest.start("duplicate")'),false);
run('recordTest.frame();recordTest.stop();recordTest.stop()');assert.equal(context.savedVideos.length,1);assert.equal(context.savedVideos[0].name,'tr_qa.webm');assert.ok(stoppedTracks>0);assert.equal(run('recordTest.active'),false);
// Snapshot crossfade keeps the old scene until the new one is ready.
run('drawScene=()=>{};state.ready=true;gl={};beginVisualTransition()');assert.equal(element('transitionCanvas').hidden,false);
run('finishVisualTransition(state.token)');assert.equal(element('transitionCanvas').style.opacity,'0');
const transitionTimer=run('sceneTransition.timer');timers.get(transitionTimer).f();assert.equal(element('transitionCanvas').hidden,true);
run('gl=null;state.ready=false');
console.log('PASS: profiles/custom storage; adaptive quality; touch pan/pinch/inertia; bounded video export and stream cleanup; visual snapshot transition.');
// A failing new image keeps the existing artwork and country metadata.
run('globalThis.originalImage=Image;Image=class{constructor(){this.naturalWidth=900;this.naturalHeight=600;}decode(){return this.src==="broken.avif"?Promise.reject(Error("bad image")):Promise.resolve();}};imageCache.clear();state.displayCountry="tr";state.image="cover";country="tr";photo.src="cover.avif";ASSETS.jp={cover:"broken.avif"};');
assert.equal(await run('chooseCountry("jp")'),false);assert.equal(run('country'),'tr');assert.equal(element('photo').src,'cover.avif');assert.equal(run('state.image'),'cover');
run('Image=originalImage;delete ASSETS.jp');
// Validate ping-pong resource ownership and absence of read/write feedback.
function gpuContext({supported=true,complete=true}={}){
 let serial=0,frame=null,unit=0;const bound=new Map(),result={draws:0,deletedTextures:0,deletedFrames:0};
 const g={};for(const name of ['VERTEX_SHADER','FRAGMENT_SHADER','COMPILE_STATUS','LINK_STATUS','ARRAY_BUFFER','STATIC_DRAW','FLOAT','TEXTURE_2D','TEXTURE_MIN_FILTER','TEXTURE_MAG_FILTER','LINEAR','TEXTURE_WRAP_S','TEXTURE_WRAP_T','CLAMP_TO_EDGE','RGBA','FRAMEBUFFER','COLOR_ATTACHMENT0','FRAMEBUFFER_COMPLETE','DITHER','TRIANGLES','TEXTURE0'])g[name]=++serial;
 Object.assign(g,{getExtension:()=>supported?{}:null,createProgram:()=>({}),createShader:()=>({}),shaderSource(){},compileShader(){},getShaderParameter:()=>true,attachShader(){},linkProgram(){},getProgramParameter:()=>true,getAttribLocation:()=>0,getUniformLocation:(_,key)=>key,createBuffer:()=>({}),bindBuffer(){},bufferData(){},createTexture:()=>({id:++serial}),bindTexture:(_,texture)=>bound.set(unit,texture),texParameteri(){},texImage2D(){},createFramebuffer:()=>({}),bindFramebuffer:(_,value)=>{frame=value;},framebufferTexture2D:(_,__,___,texture)=>{frame.texture=texture;},checkFramebufferStatus:()=>complete?g.FRAMEBUFFER_COMPLETE:-1,deleteFramebuffer(){result.deletedFrames++;},deleteTexture(){result.deletedTextures++;},deleteBuffer(){},deleteProgram(){},deleteShader(){},useProgram(){},enableVertexAttribArray(){},vertexAttribPointer(){},disable(){},viewport(){},uniform1i(){},uniform2f(){},uniform1f(){},uniform4fv(){},activeTexture:value=>{unit=value-g.TEXTURE0;},texSubImage2D(){},drawArrays(){assert.ok(frame,'offscreen output framebuffer');assert.notEqual(frame.texture,bound.get(0),'input and output textures must differ');result.draws++;}});
 return {g,result};
}
const gpu=gpuContext();context.gpuContext=gpu.g;
run('globalThis.gpuTest=new GPUFlow(gpuContext,48,32);gpuTest.splat(.5,.5,.1,.04,1.5);gpuTest.step(1/60,.65)');
assert.equal(gpu.result.draws,2);assert.ok(run('gpuTest.energy')>0);
run('gpuTest.reset();gpuTest.step(1/60,.65)');assert.equal(gpu.result.draws,2,'idle field performs no GPU draws');
run('gpuTest.destroy()');assert.equal(gpu.result.deletedFrames,2);assert.equal(gpu.result.deletedTextures,2);
context.gpuContext=gpuContext({supported:false}).g;assert.throws(()=>run('new GPUFlow(gpuContext)'),/Float flow unavailable/);
const badGPU=gpuContext({complete:false});context.gpuContext=badGPU.g;assert.throws(()=>run('new GPUFlow(gpuContext)'),/Float framebuffer unavailable/);
assert.equal(badGPU.result.deletedFrames,1);assert.equal(badGPU.result.deletedTextures,1);
console.log('PASS: failed image rollback; GPU ping-pong ownership, idle suspension, unsupported-device fallback and resource cleanup.');

// Panorama gestures rotate freely, clamp vertical travel and retain angle precision.
element('panoramaMode').listeners.click();assert.equal(run('panorama.enabled'),true);
const panoEvent=(x,y)=>({pointerId:7,clientX:x,clientY:y,target:{closest:()=>null},preventDefault(){},stopImmediatePropagation(){}});
scene.listeners['capture:pointerdown'](panoEvent(800,300));
scene.listeners['capture:pointermove'](panoEvent(-1600,3000));
assert.ok(run('panorama.targetYaw')>Math.PI*2);assert.equal(run('panorama.targetPitch'),.22);
run('advancePanorama(1/60)');assert.ok(run('panorama.yaw>0&&panorama.yaw<panorama.targetYaw'));
scene.listeners['capture:pointerup'](panoEvent(-1600,3000));assert.equal(run('panorama.drag'),null);
run('panorama.yaw=1000;panorama.targetYaw=1001;advancePanorama(0)');assert.ok(Math.abs(run('panorama.yaw'))<Math.PI*2);assert.ok(Math.abs(run('panorama.targetYaw-panorama.yaw')-1)<1e-9);
element('panoramaMode').listeners.click();assert.equal(run('panorama.enabled'),false);
console.log('PASS: panorama full turns, vertical bounds, camera easing, precision and mode toggle.');
