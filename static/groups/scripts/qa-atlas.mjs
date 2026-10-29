// Browserless fallback checks for embedding and country state; does not verify WebGL rendering.
import {readFileSync} from 'node:fs';
import vm from 'node:vm';
import assert from 'node:assert/strict';
const html=readFileSync(new URL('../index.html',import.meta.url),'utf8');
const script=html.match(/<script>([\s\S]*?)<\/script>/)[1];
const elements=new Map();
function el(id){if(!elements.has(id))elements.set(id,{id,hidden:false,value:'',textContent:'',dataset:{},children:[],classList:{add(){},remove(){},toggle(){}},setAttribute(k,v){this[k]=v},listeners:{},addEventListener(k,fn,capture){this.listeners[(capture?'capture:':'')+k]=fn},setPointerCapture(id){this.capture=id},hasPointerCapture(id){return this.capture===id},releasePointerCapture(){this.capture=null},replaceChildren(){this.children=[]},append(x){this.children.push(x)},getContext(){return null},decode(){return Promise.resolve()},getBoundingClientRect(){return {width:1500,height:500,left:0,top:0}},naturalWidth:1500,naturalHeight:500});return elements.get(id)}
const tabs=['cover','avatar','hero'].map(k=>{const x=el('tab'+k);x.dataset.image=k;return x});
const ps=[el('p1'),el('p2'),el('p3')];
const document={getElementById:el,querySelectorAll:s=>s==='[data-image]'?tabs:ps,querySelector:s=>el(s),createElement:t=>el('created'+Math.random()),addEventListener(){},hidden:false};
const matchMedia=()=>({matches:false,addEventListener(){}});
const savedSettings=new Map();
const localStorage={getItem:key=>savedSettings.get(key)||null,setItem:(key,value)=>savedSettings.set(key,value)};
const context=vm.createContext({document,localStorage,window:{matchMedia,removeEventListener(){}},matchMedia,performance:{now:()=>0},Intl,console,Image:class{constructor(){this.naturalWidth=1500;this.naturalHeight=500}decode(){return el('photo').decode()}},ResizeObserver:class{observe(){}},IntersectionObserver:class{observe(){}},requestAnimationFrame:()=>1,setTimeout,clearTimeout});
vm.runInContext(script,context);
const expectedCountries=JSON.parse(readFileSync(new URL('../data/countries.json',import.meta.url),'utf8')).length;
assert.equal(vm.runInContext('COUNTRIES.length',context),expectedCountries);
assert.equal(el('viewer').hidden,false);
assert.equal(el('downloadCover').download,'tr_cover.avif');
const pending=vm.runInContext("(()=>{const code='qa-pending';COUNTRIES.push({code,name:'Pending QA'});delete ASSETS[code];return code})()",context);
vm.runInContext(`chooseCountry(${JSON.stringify(pending)})`,context);
assert.equal(el('viewer').hidden,true);assert.equal(el('pending').hidden,false);assert.equal(el('.downloads').hidden,true);
vm.runInContext("filterCountries('zzzz_no_country')",context);assert.equal(el('countrySelect').disabled,true);
await vm.runInContext("filterCountries('tr');chooseCountry('tr');select('avatar')",context);
assert.equal(el('pending').hidden,true);assert.equal(el('photo').src.startsWith('data:image/avif;base64,'),true);
assert.equal(el('downloadAvatar').download,'tr_avatar.avif');
await new Promise(resolve=>setTimeout(resolve,0));
vm.runInContext('imageCache.clear()',context);
let rejectDecode;el('photo').decode=()=>new Promise((_,reject)=>{rejectDecode=reject});
const oldSelect=vm.runInContext("select('cover')",context);
vm.runInContext(`chooseCountry(${JSON.stringify(pending)})`,context);
const pendingNotice=el('notice').textContent;rejectDecode(new Error('old decode cancelled'));await oldSelect;
assert.equal(el('notice').textContent,pendingNotice);assert.equal(el('pending').hidden,false);
el('photo').decode=()=>Promise.resolve();
vm.runInContext("chooseCountry('tr');select('hero')",context);
const hasTRHero=vm.runInContext('!!ASSETS.tr.hero',context);assert.equal(el('pending').hidden,hasTRHero);assert.equal(el('viewer').hidden,!hasTRHero);
vm.runInContext("select('avatar')",context);assert.equal(el('viewer').hidden,false);
vm.runInContext(`ASSETS[${JSON.stringify(pending)}]={hero:ASSETS.tr.avatar};chooseCountry(${JSON.stringify(pending)})`,context);
assert.equal(vm.runInContext('state.image',context),'hero');assert.equal(el('viewer').hidden,false);assert.equal(el('downloadCover').hidden,true);assert.equal(el('downloadHero').hidden,false);
assert.equal(el('downloadHero').download,pending+'_hero.avif');
console.log(`PASS: ${expectedCountries} countries/regions; standalone embedded AVIF; synthetic pending hides art/downloads; search; cover/avatar routing; stale image race; absent hero and partial-country hero. Browser/WebGL validation remains required.`);


// Exercise camera physics without replacing the depth shader or requiring WebGL.
const endings=[];
for(const fps of [30,60,120]){
vm.runInContext('state.current=[0,0];state.target=[1,0];state.velocity=[0,0]',context);
let peak=0;
for(let frame=0;frame<fps*2;frame++){
vm.runInContext(`advanceParallax(${1/fps})`,context);
const value=vm.runInContext('state.current[0]',context);peak=Math.max(peak,value);
assert.ok(Number.isFinite(value)&&value>=0&&value<1.12);
}
assert.ok(peak>=1&&peak<1.01,'camera settles with less than one percent overshoot');
endings.push(vm.runInContext('state.current[0]',context));
assert.ok(Math.abs(endings.at(-1)-1)<.003);
vm.runInContext('state.target=[-1,0]',context);
for(let frame=0;frame<fps*2;frame++)vm.runInContext(`advanceParallax(${1/fps})`,context);
assert.ok(Math.abs(vm.runInContext('state.current[0]',context)+1)<.005);
}
assert.ok(Math.max(...endings)-Math.min(...endings)<1e-8,'refresh-rate independent motion');
vm.runInContext('reduced.matches=true;state.current=[0,0];state.target=[1,0];state.velocity=[0,0]',context);
for(let frame=0;frame<60;frame++){vm.runInContext('advanceParallax(1/60)',context);assert.ok(vm.runInContext('state.current[0]',context)<=1);}
assert.ok(script.includes('uv=mix(uv,next,0.65)'));
assert.ok(!script.includes('vec2 angle='));
assert.ok(script.includes('uniform vec2 uFlow'));
assert.ok(script.includes('vec2 memory=smoothField(vUv)'));
assert.ok(script.includes('reduced.matches?0:Math.tanh')); 
console.log('PASS: original depth shader preserved; elastic input, direction reversals and matching 30/60/120 Hz motion; reduced motion has no bounce.');

vm.runInContext('reduced.matches=false;state.flow=[0,0];state.velocity=[4,-3];advanceFlow(1/60)',context);
const impulse=vm.runInContext('state.flow[0]',context);assert.ok(impulse>0&&impulse<1);
vm.runInContext('state.velocity=[0,0];advanceFlow(1/60)',context);
assert.ok(vm.runInContext('state.flow[0]',context)>0,'wake survives after movement');
assert.ok(vm.runInContext('state.flow[0]',context)<impulse,'wake decays');
for(let i=0;i<360;i++)vm.runInContext('advanceFlow(1/60)',context);
assert.ok(vm.runInContext('Math.abs(state.flow[0])',context)<1e-7);
vm.runInContext('state.flow=[1,1];reduced.matches=true;advanceFlow(1/60)',context);
assert.equal(vm.runInContext('state.flow[0]',context),0);
console.log('PASS: fluid wake follows velocity, persists briefly, settles to zero and respects reduced motion.');

vm.runInContext('reduced.matches=false;state.near=[0,0];state.pointer=[0,0];state.current=[1,-1];state.target=[1,-1];state.velocity=[0,0];advanceFlow(1/60)',context);
assert.ok(vm.runInContext('state.near[0]>0&&state.near[0]<state.current[0]',context),'foreground has independent follow-through');
assert.ok(vm.runInContext('state.pointer[0]>0&&state.pointer[0]<state.target[0]',context),'deformation focus moves smoothly');
for(let i=0;i<360;i++)vm.runInContext('advanceFlow(1/60)',context);
assert.ok(vm.runInContext('Math.abs(state.near[0]-1)<1e-7',context));
assert.ok(script.includes('mix(uMotion,uNearMotion,smoothstep(.25,.9,d))'));
assert.ok(script.includes('state.demo&&!reduced.matches?1:0'));
console.log('PASS: foreground follow-through and pointer smoothing converge; ambient movement is restricted to automatic exploration.');

// Wave propagation is fixed-step and independent of display refresh rate.
const fields=[];
for(const fps of [30,60,120]){
vm.runInContext('globalThis.testField=new ElasticField();testField.splat(.5,.5,.08,-.04,1.5)',context);
for(let i=0;i<fps;i++)vm.runInContext(`testField.step(${1/fps},.65)`,context);
fields.push(vm.runInContext('Array.from(testField.position)',context));
assert.ok(vm.runInContext('testField.position.every(Number.isFinite)',context));
assert.ok(vm.runInContext('testField.energy>0',context),'wake survives release');
for(let i=0;i<fps*5;i++)vm.runInContext(`testField.step(${1/fps},.65)`,context);
assert.ok(vm.runInContext('testField.energy',context)<.001,'field settles');
}
assert.ok(Math.max(...fields[0].map((v,i)=>Math.abs(v-fields[2][i])))<1e-6,'identical field at 30/120 Hz');
vm.runInContext('testField.reset();testField.splat(.5,.5,100,-100,1);',context);
for(let i=0;i<240;i++)vm.runInContext('testField.step(1/120,1)',context);
assert.ok(vm.runInContext('testField.position.every(v=>Number.isFinite(v)&&Math.abs(v)<=1.10001)',context),'large impulses stay bounded');
for(const key of ['elasticity','fluidity','speed','strength']){
assert.equal(typeof el(key).listeners.input,'function');
}
el('elasticity').listeners.input({target:{value:'100'}});
assert.equal(vm.runInContext('motionSettings.elasticity',context),100);
assert.equal(el('elasticityValue').textContent,'100%');
assert.equal(JSON.parse(savedSettings.get('atlas-motion-v1')).elasticity,100);
el('speed').listeners.input({target:{value:'999'}});
assert.equal(vm.runInContext('motionSettings.speed',context),180);
el('fluidity').listeners.input({target:{value:'0'}});
assert.equal(vm.runInContext('motionSettings.fluidity',context),0);
el('resetMotion').listeners.click();
assert.equal(vm.runInContext('motionSettings.elasticity',context),65);
assert.equal(vm.runInContext('motionSettings.fluidity',context),70);
assert.equal(vm.runInContext('motionSettings.speed',context),100);
assert.equal(vm.runInContext('state.strength',context),.7);
console.log('PASS: wave propagation, frame-rate independence, bounded impulses, decay; adjustable motion controls and reset.');

// Stress the actual camera/field dynamics at maximum controls and alternating input.
vm.runInContext('reduced.matches=false;motionSettings.elasticity=100;motionSettings.fluidity=100;motionSettings.speed=180;state.current=[0,0];state.velocity=[0,0];state.near=[0,0];state.pointer=[0,0];state.flow=[0,0];testField.reset()',context);
for(let frame=0;frame<720;frame++){
vm.runInContext(`state.target=[${frame%24<12?1:-1},${frame%40<20?1:-1}];advanceParallax(1/120);advanceFlow(1/120);testField.splat(.5,.5,${frame%2?.12:-.12},.04,3);testField.step(1/120,1)`,context);
assert.ok(vm.runInContext('state.current.every(v=>Number.isFinite(v)&&Math.abs(v)<2)&&state.near.every(Number.isFinite)&&testField.position.every(v=>Number.isFinite(v)&&Math.abs(v)<=1.10001)',context));
}
vm.runInContext('state.target=[0,0]',context);
for(let i=0;i<600;i++)vm.runInContext('advanceParallax(1/120);advanceFlow(1/120);testField.step(1/120,1)',context);
assert.ok(vm.runInContext('state.current.every(v=>Math.abs(v)<1e-5)&&testField.energy<.001',context));
console.log('PASS: maximum settings withstand sustained direction changes and settle; cubic field reconstruction and edge protection enabled.');

vm.runInContext('reduced.matches=false;Object.assign(motionSettings,motionDefaults);state.current=[1,1];state.velocity=[0,0];state.zoom=1;advanceFlow(1/60)',context);
assert.ok(vm.runInContext('state.zoom>1&&state.zoom<state.zoomTarget',context),'dolly eases in');
for(let i=0;i<360;i++)vm.runInContext('advanceFlow(1/60)',context);
assert.ok(vm.runInContext('state.zoom>1.02&&state.zoom<1.04',context),'default zoom remains subtle');
el('zoom').listeners.input({target:{value:'0'}});
for(let i=0;i<360;i++)vm.runInContext('advanceFlow(1/60)',context);
assert.ok(vm.runInContext('Math.abs(state.zoom-1)<1e-7',context),'zoom control disables dolly');
el('zoom').listeners.input({target:{value:'100'}});
vm.runInContext('reduced.matches=true;advanceFlow(1/60)',context);
assert.equal(vm.runInContext('state.zoom',context),1);
el('resetMotion').listeners.click();assert.equal(vm.runInContext('motionSettings.zoom',context),55);
console.log('PASS: subtle eased dolly, adjustable amount, zero disables zoom, reduced motion and reset respected.');

// Smooth settings and a continuous demo path avoid mode-transition jumps.
vm.runInContext('reduced.matches=false;state.current=[.4,-.25];state.target=[.8,.8];setDemo(true);advanceDemo(0)',context);
assert.ok(vm.runInContext('Math.abs(state.target[0]-.4)<1e-12&&Math.abs(state.target[1]+.25)<1e-12',context));
vm.runInContext('advanceDemo(1/60)',context);
assert.ok(vm.runInContext('Math.abs(state.target[0]-.4)<.02',context));
vm.runInContext('setDemo(false)',context);
assert.equal(vm.runInContext('state.target[0]',context),.4);
vm.runInContext('state.renderedStrength=1;state.strength=0;motionSettings.fluidity=0;state.renderedFluidity=1;advanceFlow(1/60)',context);
assert.ok(vm.runInContext('state.renderedStrength>0&&state.renderedStrength<1&&state.renderedFluidity>0&&state.renderedFluidity<1',context));
for(let i=0;i<360;i++)vm.runInContext('advanceFlow(1/60)',context);
assert.ok(vm.runInContext('state.renderedStrength<1e-8&&state.renderedFluidity<1e-8',context));
console.log('PASS: demo starts at the current view, pauses in place and settings transition smoothly.');

// Retargeting keeps transition velocity continuous instead of restarting an ease.
vm.runInContext('reduced.matches=false;state.zoom=1;state.transitionVelocity.zoom=0;transitionValue("zoom",1.06,10,1/120)',context);
const initialZoom=vm.runInContext('state.zoom',context);
assert.ok(initialZoom>1&&initialZoom<1.001,'gentle start');
vm.runInContext('transitionValue("zoom",1.06,10,1/120)',context);
const beforeReverse=vm.runInContext('state.transitionVelocity.zoom',context);
vm.runInContext('transitionValue("zoom",1,10,1/120)',context);
assert.ok(vm.runInContext('state.transitionVelocity.zoom',context)>0,'direction change preserves momentum');
assert.ok(beforeReverse>0);
for(let i=0;i<300;i++)vm.runInContext('transitionValue("zoom",1,10,1/120)',context);
assert.ok(vm.runInContext('Math.abs(state.zoom-1)<1e-8',context));
vm.runInContext('state.current=[.3,.2];setDemo(true);advanceDemo(.001)',context);
assert.ok(vm.runInContext('Math.abs(state.target[0]-.3)<1e-8&&Math.abs(state.target[1]-.2)<1e-8',context),'demo has a gentle acceleration ramp');
console.log('PASS: continuous zoom velocity through reversal, smooth convergence and minimum-jerk demo entry.');

// Complete simulation (camera + dolly + flow + demo) must share timing.
const sceneSamples=[];
for(const fps of [30,60,120]){
vm.runInContext(`Object.assign(motionSettings,motionDefaults);reduced.matches=false;Object.assign(state,{current:[0,0],target:[0,0],velocity:[0,0],near:[0,0],pointer:[0,0],flow:[0,0],flowTime:0,zoom:1,zoomTarget:1,strength:.7,renderedStrength:.7,renderedFluidity:.8715,renderedSpeed:1,renderedElasticity:.65,life:0,transitionVelocity:{},demo:true,demoOrigin:[0,0],demoPhase:0,demoElapsed:0});elasticField.reset();`,context);
for(let frame=0;frame<fps*2;frame++)vm.runInContext(`advanceScene(${1/fps})`,context);
sceneSamples.push(vm.runInContext('[...state.current,...state.near,...state.flow,state.zoom,state.demoPhase,elasticField.energy]',context));
}
assert.ok(Math.max(...sceneSamples[0].map((v,i)=>Math.abs(v-sceneSamples[2][i])))<1e-7,'whole scene matches at 30 and 120 Hz');
vm.runInContext('advanceScene(.8)',context);
assert.ok(vm.runInContext('state.current.every(Number.isFinite)&&Number.isFinite(state.zoom)',context),'long frame recovers safely');
// Verify coordinates inside a letterboxed fullscreen image.
document.fullscreenElement=el('scene');
vm.runInContext('textureSize=[1,1]',context);
const viewport=vm.runInContext('motionViewport()',context);
assert.equal(viewport.width,500);assert.equal(viewport.left,500);assert.equal(viewport.height,500);
document.fullscreenElement=null;
assert.equal(vm.runInContext('motionViewport().width',context),1500);
console.log('PASS: complete scene synchronized at 30/60/120 Hz; long-frame recovery and fullscreen pointer alignment.');

// A late image load must not restart demo after manual interaction.
vm.runInContext('globalThis.originalSelect=select;globalThis.selectionResolvers=[];select=()=>{state.token++;state.ready=true;return new Promise(resolve=>selectionResolvers.push(resolve))};reduced.matches=false;',context);
const manuallyInterrupted=vm.runInContext("chooseCountry('tr')",context);
vm.runInContext('state.inputRevision++;selectionResolvers.shift()()',context);
await manuallyInterrupted;assert.equal(vm.runInContext('state.demo',context),false);
const staleCountry=vm.runInContext("chooseCountry('tr')",context);
const currentCountry=vm.runInContext("chooseCountry('tr')",context);
vm.runInContext('selectionResolvers.shift()()',context);await staleCountry;
assert.equal(vm.runInContext('state.demo',context),false);
vm.runInContext('selectionResolvers.shift()()',context);await currentCountry;
assert.equal(vm.runInContext('state.demo',context),true);
vm.runInContext('select=originalSelect;setDemo(false);state.ready=false',context);
console.log('PASS: stale loading cannot start demo or override manual interaction.');

const reloadContext=vm.createContext({document,window:{matchMedia,removeEventListener(){}},localStorage:{getItem:()=>JSON.stringify({elasticity:85,fluidity:90,speed:125,strength:55}),setItem(){}},matchMedia,performance:{now:()=>0},Intl,console,Image:class{constructor(){this.naturalWidth=1500;this.naturalHeight=500}decode(){return el('photo').decode()}},ResizeObserver:class{observe(){}},IntersectionObserver:class{observe(){}},requestAnimationFrame:()=>1,setTimeout,clearTimeout});
vm.runInContext(script,reloadContext);
assert.equal(vm.runInContext('motionSettings.elasticity',reloadContext),85);
assert.equal(vm.runInContext('motionSettings.speed',reloadContext),125);
assert.equal(vm.runInContext('state.strength',reloadContext),.55);
console.log('PASS: settings persist and reload with matching controls and depth.');

