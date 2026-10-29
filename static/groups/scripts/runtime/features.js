// Product features share the motion settings without changing source artwork.
const motionProfiles={
 cinematic:{label:'Sinematik',elasticity:48,fluidity:48,speed:65,strength:58,zoom:72},
 lively:{label:'Canlı',elasticity:85,fluidity:90,speed:110,strength:78,zoom:65},
 calm:{label:'Sakin',elasticity:25,fluidity:25,speed:55,strength:42,zoom:30}
};
function sanitizeMotion(value){
 if(!value||typeof value!=='object')return null;
 const result={};for(const key of Object.keys(motionDefaults)){
  if(typeof value[key]!=='number'||!Number.isFinite(value[key]))return null;
  result[key]=Math.max(key==='speed'?40:0,Math.min(key==='speed'?180:100,value[key]));
 }return result;
}
let customMotion=null;
try{customMotion=sanitizeMotion(JSON.parse(localStorage.getItem('atlas-custom-motion-v1')||'null'));}catch(e){}
function syncProfiles(){
 let selected='Özel';
 for(const [key,profile] of Object.entries(motionProfiles)){
  const matches=Object.keys(motionDefaults).every(k=>motionSettings[k]===profile[k]);
  document.getElementById('profile-'+key).setAttribute('aria-pressed',String(matches));if(matches)selected=profile.label;
 }
 document.getElementById('profileState').textContent=selected;
 document.getElementById('loadCustomMotion').disabled=!customMotion;
}
function applyMotionProfile(profile){
 const values=sanitizeMotion(profile);if(!values)return;
 state.inputRevision++;Object.assign(motionSettings,values);syncMotionControls();saveMotionSettings();wake();
}
class AdaptiveQuality {
 constructor(mode='auto'){this.mode=['auto','high','eco'].includes(mode)?mode:'auto';this.automaticScale=1;this.mean=16.67;this.slow=0;this.fast=0;this.samples=0;this.locked=null;}
 get scale(){return this.locked??(this.mode==='high'?1:this.mode==='eco'?.65:this.automaticScale);}
 observe(interval){
  if(this.mode!=='auto'||this.locked!==null||interval<=0||interval>.12)return;
  this.mean+=(interval*1000-this.mean)*.08;this.samples++;
  if(this.samples<45)return;
  if(this.mean>25){this.slow++;this.fast=0;}else if(this.mean<19){this.fast++;this.slow=0;}else{this.fast=0;this.slow=0;}
  if(this.slow>=75){this.automaticScale=Math.max(.6,this.automaticScale*.86);this.slow=0;}
  if(this.fast>=360){this.automaticScale=Math.min(1,this.automaticScale+.06);this.fast=0;}
 }
 setMode(mode){if(!['auto','high','eco'].includes(mode))return;this.mode=mode;this.samples=0;this.slow=0;this.fast=0;}
}
let storedQuality='auto';try{storedQuality=localStorage.getItem('atlas-quality-v1')||'auto';}catch(e){}
const qualityManager=new AdaptiveQuality(storedQuality);
const sceneTransition={visible:false,timer:0};
const transitionCanvas=document.getElementById('transitionCanvas');
function cancelVisualTransition(){
 clearTimeout(sceneTransition.timer);sceneTransition.visible=false;transitionCanvas.hidden=true;if(typeof recordButton!=='undefined')recordButton.disabled=!state.ready;
}
function beginVisualTransition(){
 if(!state.ready||!gl)return;
 const context=transitionCanvas.getContext('2d');if(!context)return;
 drawScene();
 const copy=document.createElement('canvas');copy.width=canvas.width;copy.height=canvas.height;
 const c=copy.getContext('2d');if(!c)return;c.drawImage(canvas,0,0);
 if(sceneTransition.visible&&typeof getComputedStyle==='function'){
  c.globalAlpha=Number(getComputedStyle(transitionCanvas).opacity)||0;c.drawImage(transitionCanvas,0,0,copy.width,copy.height);
 }
 transitionCanvas.width=copy.width;transitionCanvas.height=copy.height;context.drawImage(copy,0,0);
 clearTimeout(sceneTransition.timer);transitionCanvas.style.transition='none';transitionCanvas.style.opacity='1';transitionCanvas.hidden=false;sceneTransition.visible=true;recordButton.disabled=true;
}
function finishVisualTransition(token){
 if(!sceneTransition.visible)return;
 if(reduced.matches){cancelVisualTransition();return;}
 // Force the visible snapshot to paint before beginning its opacity transition.
 transitionCanvas.getBoundingClientRect();transitionCanvas.style.transition='opacity 600ms cubic-bezier(.22,.61,.36,1)';transitionCanvas.style.opacity='0';
 sceneTransition.timer=setTimeout(()=>{if(token===state.token)cancelVisualTransition();},650);
}
class CanvasVideoRecorder {
 constructor(source,onChange,onError,onSave){this.source=source;this.onChange=onChange;this.onError=onError;this.onSave=onSave;this.active=false;this.recorder=null;this.stream=null;this.chunks=[];this.timer=0;this.stopping=false;}
 get available(){return typeof MediaRecorder!=='undefined'&&typeof this.source.captureStream==='function';}
 start(name){
  if(!this.available||this.active||this.stopping)return false;
  try{
   const mime=['video/webm;codecs=vp9','video/webm;codecs=vp8','video/webm','video/mp4'].find(type=>MediaRecorder.isTypeSupported(type));
   if(!mime)throw new Error('Bu tarayıcıda desteklenen video biçimi bulunamadı.');
   this.output=document.createElement('canvas');const scale=Math.min(1,1280/Math.max(this.source.width,this.source.height));
   this.output.width=Math.max(2,Math.round(this.source.width*scale/2)*2);this.output.height=Math.max(2,Math.round(this.source.height*scale/2)*2);
   this.context=this.output.getContext('2d');if(!this.context)throw new Error('Video yüzeyi oluşturulamadı.');
   this.stream=this.output.captureStream(30);this.recorder=new MediaRecorder(this.stream,{mimeType:mime,videoBitsPerSecond:6000000});
   this.chunks=[];this.stopping=false;this.name=name;this.mime=mime;this.failed=false;
   this.recorder.ondataavailable=e=>{if(e.data?.size)this.chunks.push(e.data);};
   this.recorder.onerror=()=>{this.failed=true;this.onError('Video kaydı tamamlanamadı.');this.stop();};
   this.recorder.onstop=()=>{
    const chunks=this.chunks;this.cleanup();
    if(!this.failed&&chunks.length)this.onSave(new Blob(chunks,{type:mime}),name+(mime.includes('mp4')?'.mp4':'.webm'));else if(!this.failed)this.onError('Kayıt çok kısa olduğu için video oluşturulamadı.');
   };
   this.started=performance.now();this.active=true;this.recorder.start(250);this.onChange(true);
   this.timer=setTimeout(()=>this.stop(),10000);return true;
  }catch(error){this.cleanup();this.onError(error.message||'Video kaydı başlatılamadı.');return false;}
 }
 frame(){
  if(!this.active||!this.context)return;
  const {width:w,height:h}=this.output;const scale=Math.min(w/this.source.width,h/this.source.height),dw=this.source.width*scale,dh=this.source.height*scale;
  this.context.fillStyle='#231e2b';this.context.fillRect(0,0,w,h);this.context.drawImage(this.source,(w-dw)/2,(h-dh)/2,dw,dh);
 }
 stop(){if(!this.active||this.stopping)return;this.stopping=true;clearTimeout(this.timer);if(this.recorder&&this.recorder.state!=='inactive')this.recorder.stop();else this.cleanup();}
 cleanup(){clearTimeout(this.timer);this.active=false;this.stopping=false;for(const track of this.stream?.getTracks()||[])track.stop();this.stream=null;this.onChange(false);}
}
const recordButton=document.getElementById('recordVideo');
const videoRecorder=new CanvasVideoRecorder(canvas,active=>{
 state.recording=active;
 if(active){recordButton.setAttribute('aria-pressed','true');qualityManager.locked=qualityManager.scale;notice.textContent='10 saniyelik kayıt başladı. Görsel üzerinde gezinerek hareketi kaydedebilirsiniz.';}
 else{qualityManager.locked=null;recordButton.textContent='Video kaydet';recordButton.setAttribute('aria-pressed','false');}
 document.getElementById('qualityMode').disabled=active;
 for(const button of document.querySelectorAll('[data-image]'))button.disabled=active;
 document.getElementById('countrySelect').disabled=active||!document.getElementById('countrySelect').value;
 document.getElementById('countrySearch').disabled=active;
 wake();
},message=>{notice.textContent=message;},(blob,name)=>{
 const url=URL.createObjectURL(blob),link=document.createElement('a');link.href=url;link.download=name;document.body.append(link);link.click();link.remove();
 setTimeout(()=>URL.revokeObjectURL(url),1000);notice.textContent='Video kaydı indirildi. Orijinal görseller korunuyor.';
});
function setupFeatures(){
 for(const [key,profile] of Object.entries(motionProfiles))document.getElementById('profile-'+key).addEventListener('click',()=>applyMotionProfile(profile));
 document.getElementById('saveCustomMotion').addEventListener('click',()=>{
  customMotion={...motionSettings};try{localStorage.setItem('atlas-custom-motion-v1',JSON.stringify(customMotion));notice.textContent='Kişisel hareket ayarınız kaydedildi.';}catch(e){notice.textContent='Ayarınız bu oturum için kaydedildi; tarayıcı kalıcı kayda izin vermiyor.';}syncProfiles();
 });
 document.getElementById('loadCustomMotion').addEventListener('click',()=>{if(customMotion)applyMotionProfile(customMotion);});
 const quality=document.getElementById('qualityMode');quality.value=qualityManager.mode;
 quality.addEventListener('change',e=>{qualityManager.setMode(e.target.value);try{localStorage.setItem('atlas-quality-v1',qualityManager.mode);}catch(error){}wake();});
 recordButton.hidden=!videoRecorder.available||!gl;recordButton.disabled=!state.ready;
 recordButton.addEventListener('click',()=>{
  if(videoRecorder.active){videoRecorder.stop();return;}
  if(!state.ready){notice.textContent='Video kaydı için görselin yüklenmesini bekleyin.';return;}
  if(sceneTransition.visible){notice.textContent='Görsel geçişi tamamlandıktan sonra kaydı başlatabilirsiniz.';return;}
  if(videoRecorder.start(country+'_'+state.image+'_hareket')){drawScene();videoRecorder.frame();}
 });
 syncProfiles();
}
