document.querySelectorAll('[data-image]').forEach(b=>b.addEventListener('click',()=>select(b.dataset.image)));
function motionViewport(){
const rect=scene.getBoundingClientRect();let {left,top,width,height}=rect;
if(document.fullscreenElement===scene){
const aspect=textureSize[0]/textureSize[1];
if(width/height>aspect){const fitted=height*aspect;left+=(width-fitted)*.5;width=fitted;}
else{const fitted=width/aspect;top+=(height-fitted)*.5;height=fitted;}
}
return {left,top,width,height};
}
const touchPoints=new Map();let touchAnchor=null,touchPinch=null;
const clampView=value=>Math.max(-1,Math.min(1,value));
function disableGyro(){state.gyro=false;window.removeEventListener('deviceorientation',onOrientation);clearTimeout(gyroTimer);gyroOrigin=null;motionButton.textContent='Telefonu eğerek gez';motionButton.setAttribute('aria-pressed','false');}
function gestureCenter(){const points=[...touchPoints.values()];return {x:points.reduce((sum,p)=>sum+p.x,0)/points.length,y:points.reduce((sum,p)=>sum+p.y,0)/points.length};}
function anchorTouch(){
 if(!touchPoints.size){touchAnchor=null;touchPinch=null;return;}
 const center=gestureCenter();touchAnchor={...center,origin:[...state.target]};
 if(touchPoints.size>1){const [a,b]=[...touchPoints.values()];touchPinch={distance:Math.max(1,Math.hypot(a.x-b.x,a.y-b.y)),zoom:state.touchZoom};}else touchPinch=null;
}
function feedPointerField(point,r){
 if(fieldPointer&&!reduced.matches&&motionSettings.fluidity>0){const dx=point[0]-fieldPointer[0],dy=point[1]-fieldPointer[1];const steps=Math.min(8,Math.max(1,Math.ceil(Math.hypot(dx,dy)*40)));for(let i=1;i<=steps;i++)flowEngine.splat(fieldPointer[0]+dx*i/steps,fieldPointer[1]+dy*i/steps,dx/steps,dy/steps,r.width/r.height);}
 fieldPointer=point;
}
function resetView(){state.inputRevision++;setDemo(false);disableGyro();state.touchZoom=1;state.target=[0,0];fieldPointer=null;wake();}
scene.addEventListener('dragstart',e=>e.preventDefault());
scene.addEventListener('dblclick',e=>{if(!e.target.closest('button'))resetView();});
scene.addEventListener('pointerdown',e=>{
 if(e.target.closest('button')||e.pointerType!=='touch')return;
 state.inputRevision++;if(state.demo)setDemo(false);disableGyro();touchPoints.set(e.pointerId,{x:e.clientX,y:e.clientY});anchorTouch();scene.setPointerCapture(e.pointerId);fieldPointer=null;
});
scene.addEventListener('pointermove',e=>{
 if(e.target.closest('button'))return;const r=motionViewport();
 if(e.pointerType==='touch'){
  if(!touchPoints.has(e.pointerId))return;touchPoints.set(e.pointerId,{x:e.clientX,y:e.clientY});const center=gestureCenter();
  if(!touchAnchor)anchorTouch();state.target=[clampView(touchAnchor.origin[0]+(center.x-touchAnchor.x)/r.width*2.2),clampView(touchAnchor.origin[1]-(center.y-touchAnchor.y)/r.height*2.2)];
  if(touchPinch&&touchPoints.size>1){const [a,b]=[...touchPoints.values()];state.touchZoom=Math.max(1,Math.min(1.14,touchPinch.zoom*Math.hypot(a.x-b.x,a.y-b.y)/touchPinch.distance));}
  state.inputRevision++;feedPointerField([clampView((center.x-r.left)/r.width*2-1)*.5+.5,clampView(1-(center.y-r.top)/r.height*2)*.5+.5],r);wake();return;
 }
 if(state.gyro||e.isPrimary===false)return;
 if(e.clientX<r.left||e.clientX>r.left+r.width||e.clientY<r.top||e.clientY>r.top+r.height){fieldPointer=null;return;}
 state.inputRevision++;if(state.demo)setDemo(false);state.target=[clampView((e.clientX-r.left)/r.width*2-1),clampView(1-(e.clientY-r.top)/r.height*2)];
 feedPointerField([(state.target[0]+1)*.5,(state.target[1]+1)*.5],r);wake();
});
function endTouch(e){
 if(!touchPoints.has(e.pointerId))return;touchPoints.delete(e.pointerId);anchorTouch();fieldPointer=null;
 if(scene.hasPointerCapture(e.pointerId))scene.releasePointerCapture(e.pointerId);wake();
}
scene.addEventListener('pointerup',endTouch);
scene.addEventListener('lostpointercapture',endTouch);
scene.addEventListener('pointerleave',()=>{if(touchPoints.size)return;fieldPointer=null;if(!state.demo&&!state.gyro){state.target=[0,0];wake();}});
scene.addEventListener('pointercancel',e=>{endTouch(e);fieldPointer=null;});
scene.addEventListener('keydown',e=>{
 if(e.target.closest('button'))return;const dirs={ArrowLeft:[-.25,0],ArrowRight:[.25,0],ArrowUp:[0,.25],ArrowDown:[0,-.25]};
 if(dirs[e.key]){e.preventDefault();state.inputRevision++;setDemo(false);disableGyro();state.target=state.current.map((v,i)=>clampView(v+dirs[e.key][i]));wake();}
 else if(e.key==='Escape'||e.key==='Home'){e.preventDefault();resetView();}
});
function syncMotionControls(){
for(const key of Object.keys(motionDefaults)){
const input=document.getElementById(key);input.value=String(motionSettings[key]);
if(key!=='strength')document.getElementById(key+'Value').textContent=key==='speed'?(motionSettings[key]/100).toFixed(2)+'×':motionSettings[key]+'%';
}state.strength=motionSettings.strength/100;syncProfiles();
}
function saveMotionSettings(){try{localStorage.setItem('atlas-motion-v1',JSON.stringify(motionSettings));}catch(e){}}
for(const key of Object.keys(motionDefaults))document.getElementById(key).addEventListener('input',e=>{
const value=Number(e.target.value);if(!Number.isFinite(value))return;
motionSettings[key]=Math.max(key==='speed'?40:0,Math.min(key==='speed'?180:100,value));syncMotionControls();saveMotionSettings();wake();
});
document.getElementById('resetMotion').addEventListener('click',()=>{Object.assign(motionSettings,motionDefaults);syncMotionControls();saveMotionSettings();flowEngine.reset();wake();});
syncMotionControls();
document.getElementById('demo').addEventListener('click',()=>{state.inputRevision++;if(reduced.matches){notice.textContent='Cihazınızda hareketi azalt ayarı açık. Fare veya ok tuşlarıyla kontrollü olarak gezebilirsiniz.';return;}disableGyro();setDemo(!state.demo);});
const motionButton=document.getElementById('motion');let gyroOrigin=null,gyroTimer=null;
if('DeviceOrientationEvent' in window&&matchMedia('(pointer: coarse)').matches){motionButton.hidden=false;}
function onOrientation(e){if(!state.gyro||e.gamma===null||e.beta===null)return;if(!gyroOrigin){gyroOrigin=[e.gamma,e.beta];clearTimeout(gyroTimer);notice.textContent='Telefonunuzu hafifçe eğerek manzarayı keşfedin.';}state.target=[Math.max(-1,Math.min(1,(e.gamma-gyroOrigin[0])/18)),Math.max(-1,Math.min(1,(e.beta-gyroOrigin[1])/18))];wake();}
motionButton.addEventListener('click',async()=>{if(state.gyro){disableGyro();state.target=[0,0];wake();return;}try{state.inputRevision++;if(!window.isSecureContext)throw Error('secure');if(typeof DeviceOrientationEvent.requestPermission==='function'){const permission=await DeviceOrientationEvent.requestPermission();if(permission!=='granted')throw Error('denied');}setDemo(false);state.gyro=true;gyroOrigin=null;motionButton.textContent='Eğim kontrolünü kapat';motionButton.setAttribute('aria-pressed','true');window.addEventListener('deviceorientation',onOrientation);notice.textContent='Hareket algılayıcı bekleniyor…';gyroTimer=setTimeout(()=>{if(!gyroOrigin)notice.textContent='Eğim verisi alınamadı. Parmağınızla sürükleyebilir veya otomatik gezintiyi açabilirsiniz.';},2500);}catch(e){notice.textContent='Eğim kontrolü için desteklenen bir cihazda HTTPS ve hareket izni gerekir. Parmağınızla da sürükleyebilirsiniz.';}});
const expand=document.getElementById('expand');if(!scene.requestFullscreen)expand.hidden=true;expand.addEventListener('click',async()=>{try{if(document.fullscreenElement)await document.exitFullscreen();else await scene.requestFullscreen();}catch(e){notice.textContent='Tam ekran açılamadı; görseli burada keşfetmeye devam edebilirsiniz.';}});
new ResizeObserver(wake).observe(scene);document.addEventListener('fullscreenchange',()=>{expand.setAttribute('aria-label',document.fullscreenElement?'Tam ekrandan çık':'Tam ekran');wake();});
new IntersectionObserver(entries=>{state.visible=entries[0].isIntersecting;if(state.visible){state.last=0;wake();}}).observe(scene);
document.addEventListener('visibilitychange',()=>{if(document.hidden){videoRecorder.stop();touchPoints.clear();touchAnchor=null;touchPinch=null;fieldPointer=null;}state.last=0;wake();});
reduced.addEventListener('change',()=>{if(reduced.matches){setDemo(false);disableGyro();state.touchZoom=1;flowEngine.reset();}wake();});
canvas.addEventListener('webglcontextlost',e=>{e.preventDefault();fallback('3D görüntüleme durdu; sabit görsel gösteriliyor. Yenilemek için sayfayı yeniden açın.');});

// Capture panorama gestures before the planar pointer and touch handlers.
function advancePanorama(dt){
if(!panorama.enabled)return;
if(state.demo&&!reduced.matches&&!panorama.drag)panorama.targetYaw+=dt*.10*state.renderedSpeed;
const follow=reduced.matches?1:1-Math.exp(-7*dt);
panorama.yaw+=(panorama.targetYaw-panorama.yaw)*follow;
panorama.pitch+=(panorama.targetPitch-panorama.pitch)*follow;
// Rebase both angles together to retain precision after many revolutions.
if(Math.abs(panorama.yaw)>Math.PI*20){const turns=Math.trunc(panorama.yaw/(Math.PI*2))*Math.PI*2;panorama.yaw-=turns;panorama.targetYaw-=turns;}
}
const panoramaButton=document.getElementById('panoramaMode');
panoramaButton.addEventListener('click',()=>{
panorama.enabled=!panorama.enabled;panorama.drag=null;setDemo(false);
panoramaButton.setAttribute('aria-pressed',String(panorama.enabled));
scene.style.cursor=panorama.enabled?'grab':'';
notice.textContent=panorama.enabled?'Sürükleyerek veya ok tuşlarıyla 360° gezin. Tek görselden hazırlanmış, kenarları örtüştürülmüş panorama.':'Derinlikli fotoğraf görünümü.';
wake();
});
scene.addEventListener('pointerdown',e=>{
if(!panorama.enabled||e.target.closest('button')||panorama.drag)return;
e.stopImmediatePropagation();e.preventDefault();state.inputRevision++;if(state.demo)setDemo(false);
panorama.drag={id:e.pointerId,x:e.clientX,y:e.clientY,yaw:panorama.targetYaw,pitch:panorama.targetPitch};scene.setPointerCapture(e.pointerId);scene.style.cursor='grabbing';
},true);
scene.addEventListener('pointermove',e=>{
if(!panorama.enabled||e.target.closest('button'))return;
e.stopImmediatePropagation();
const drag=panorama.drag;if(!drag||drag.id!==e.pointerId)return;
const rect=scene.getBoundingClientRect();
panorama.targetYaw=drag.yaw-(e.clientX-drag.x)/Math.max(1,rect.width)*Math.PI;
panorama.targetPitch=Math.max(-.22,Math.min(.22,drag.pitch+(e.clientY-drag.y)/Math.max(1,rect.height)*.6));wake();
},true);
for(const type of ['pointerup','pointercancel','lostpointercapture'])scene.addEventListener(type,e=>{
if(!panorama.enabled||panorama.drag?.id!==e.pointerId)return;
e.stopImmediatePropagation();panorama.drag=null;scene.style.cursor='grab';if(scene.hasPointerCapture(e.pointerId))scene.releasePointerCapture(e.pointerId);
},true);
scene.addEventListener('keydown',e=>{
if(!panorama.enabled||e.target.closest('button'))return;
if(!['ArrowLeft','ArrowRight','ArrowUp','ArrowDown','Home','Escape'].includes(e.key))return;
e.preventDefault();e.stopImmediatePropagation();
if(e.key==='Home'||e.key==='Escape'){panorama.targetYaw=0;panorama.targetPitch=0;}
else if(e.key==='ArrowLeft')panorama.targetYaw-=.18;
else if(e.key==='ArrowRight')panorama.targetYaw+=.18;
else panorama.targetPitch=Math.max(-.22,Math.min(.22,panorama.targetPitch+(e.key==='ArrowUp'?.04:-.04)));
wake();
},true);
