function render(now){state.raf=0;if(!state.ready||!state.visible||document.hidden)return;
const dt=state.last?Math.max(0,(now-state.last)/1000):1/60;state.last=now;
advanceScene(dt);
advancePanorama(Math.min(dt,.1));
drawScene();
if(videoRecorder.active){videoRecorder.frame();const seconds=Math.max(0,10-Math.floor((now-videoRecorder.started)/1000));recordButton.textContent='Kaydı bitir · '+seconds+' sn';}
else qualityManager.observe(dt);
const settingsMoving=Math.abs(state.renderedStrength-state.strength)>.0001||Math.abs(state.renderedFluidity-(motionSettings.fluidity/100)*(1+.35*motionSettings.fluidity/100))>.0001;
const moving=state.current.some((v,i)=>Math.abs(v-state.target[i])>.0001||Math.abs(state.velocity[i])>.001||Math.abs(state.flow[i])>.0001||Math.abs(state.near[i]-v)>.0001||Math.abs(state.pointer[i]-state.target[i])>.0001);
if(panorama.enabled&&(Math.abs(panorama.yaw-panorama.targetYaw)>.00001||Math.abs(panorama.pitch-panorama.targetPitch)>.00001)||videoRecorder.active||state.demo&&!reduced.matches||moving||settingsMoving||Math.abs(state.life-(state.demo&&!reduced.matches?1:0))>.0001||Object.values(state.transitionVelocity).some(v=>Math.abs(v)>.0001)||Math.abs(state.zoom-state.zoomTarget)>.00001||flowEngine.energy>.0001)state.raf=requestAnimationFrame(render);
else{state.current=[...state.target];state.velocity=[0,0];state.flow=[0,0];state.last=0;}}
function drawScene(){
prepareMainDraw();uploadField();
resize();gl.uniform1f(locations.uPanorama,panorama.enabled?1:0);gl.uniform2f(locations.uOrbit,panorama.yaw,panorama.pitch);gl.uniform1f(locations.uViewportAspect,canvas.width/canvas.height);gl.uniform1f(locations.uZoom,state.zoom);gl.uniform2f(locations.uMotion,...state.current);
gl.uniform2f(locations.uFlow,...state.flow);
gl.uniform1f(locations.uFluidity,state.renderedFluidity);
gl.uniform2f(locations.uPhotoStep,2/textureSize[0],2/textureSize[1]);
gl.uniform2f(locations.uFieldSize,flowEngine.w,flowEngine.h);
gl.uniform2f(locations.uPointer,...state.pointer);
gl.uniform2f(locations.uNearMotion,...state.near);
gl.uniform1f(locations.uLife,state.life);
gl.uniform1f(locations.uTime,state.flowTime);
gl.uniform1f(locations.uAspect,textureSize[0]/textureSize[1]);gl.uniform1f(locations.uStrength,state.renderedStrength);gl.drawArrays(gl.TRIANGLES,0,6);
}
function wake(){if(!state.raf&&state.ready&&state.visible&&!document.hidden){state.last=0;state.raf=requestAnimationFrame(render);}}
function setDemo(value){state.demo=value&&!reduced.matches;document.getElementById('demo').setAttribute('aria-pressed',String(state.demo));document.getElementById('demoIcon').textContent=state.demo?'Ⅱ':'▷';document.getElementById('demoLabel').textContent=state.demo?'Gezintiyi durdur':'Otomatik gezinti';if(state.demo){state.demoOrigin=[...state.current];state.demoPhase=0;state.demoElapsed=0;}else state.target=[...state.current];wake();}
const imageCache=new Map();
function loadImage(source){
 if(imageCache.has(source))return imageCache.get(source);
 const image=new Image();image.src=source;
 const pending=image.decode().then(()=>image).catch(error=>{if(imageCache.get(source)===pending)imageCache.delete(source);throw error;});
 imageCache.set(source,pending);if(imageCache.size>4)imageCache.delete(imageCache.keys().next().value);return pending;
}
function setImageTabs(kind){document.querySelectorAll('[data-image]').forEach(button=>button.setAttribute('aria-pressed',String(button.dataset.image===kind)));}
function resetMotionForImage(){
 Object.assign(state,{target:[0,0],current:[0,0],velocity:[0,0],flow:[0,0],near:[0,0],pointer:[0,0],flowTime:0,transitionVelocity:{},life:0,renderedSpeed:motionSettings.speed/100,renderedElasticity:motionSettings.elasticity/100,renderedStrength:state.strength,renderedFluidity:(motionSettings.fluidity/100)*(1+.35*motionSettings.fluidity/100),zoom:1,zoomTarget:1,touchZoom:1,last:0});
 flowEngine.reset();fieldPointer=null;
}
async function select(kind){
 if(videoRecorder.active){notice.textContent='Görsel değiştirmek için önce video kaydını bitirin.';return false;}
 if(state.demo)setDemo(false);
 const entry=ASSETS[country],token=++state.token;
 const previous={country:state.displayCountry,image:state.image,ready:state.ready};
 beginVisualTransition();state.ready=false;state.image=kind;setImageTabs(kind);
 const available=!!entry?.[kind];document.getElementById('viewer').hidden=!available;document.getElementById('pending').hidden=available;document.getElementById('details').hidden=!available;
 if(!available){cancelVisualTransition();loading.hidden=true;document.querySelector('#pending h2').textContent=({cover:'Kapak',avatar:'Avatar',hero:'Hero'})[kind]+' görseli henüz hazırlanmadı.';notice.textContent='Bu görselin üretimi bekleniyor.';return false;}
 loading.hidden=sceneTransition.visible;
 notice.textContent=!gl?'Bu tarayıcıda 3D görüntüleme kullanılamıyor; sabit görsel gösteriliyor.':entry.depth?.[kind]&&entry.depthMode!=='approximate'?'Görsele özel derinlikle etkileşimli önizleme.':'Yaklaşık derinlik kullanılıyor; bazı detaylar birlikte hareket edebilir.';
 try{
  const artwork=await loadImage(entry[kind]);if(token!==state.token)return false;
  let depth=null;
  if(gl&&entry.depth?.[kind]){
   try{depth=await loadImage(entry.depth[kind]);}catch(error){if(token===state.token)notice.textContent='Derinlik verisi yüklenemedi; yaklaşık derinlikle devam ediliyor.';}
   if(token!==state.token)return false;
  }
  textureSize=[artwork.naturalWidth,artwork.naturalHeight];
  scene.classList.toggle('square',kind==='avatar');scene.classList.toggle('hero',kind==='hero');
  photo.src=entry[kind];photo.alt=(COUNTRIES.find(c=>c.code===country)?.name||country)+' kültürel miras illüstrasyonu';state.displayCountry=country;
  if(gl){uploadPanorama(artwork);texture(0,photoTexture,artwork);texture(1,depthTexture,depth||depthMap(kind));resetMotionForImage();state.ready=true;scene.classList.add('ready');drawScene();finishVisualTransition(token);wake();}
  else{scene.classList.remove('ready');cancelVisualTransition();}
  return true;
 }catch(error){
  if(token!==state.token)return false;
  if(previous.country){country=previous.country;updateCountryInfo(country);state.image=previous.image;setImageTabs(state.image);state.ready=previous.ready;finishVisualTransition(token);wake();notice.textContent='Yeni görsel yüklenemedi; önceki görünüm korunuyor.';}
  else{cancelVisualTransition();notice.textContent='Görsel yüklenemedi. Sayfayı yeniden açmayı deneyin.';}
  return false;
 }finally{if(token===state.token){loading.hidden=true;recordButton.disabled=!videoRecorder.active&&(!state.ready||sceneTransition.visible);}}
}
function updateCountryInfo(code){
 const item=COUNTRIES.find(c=>c.code===code),entry=ASSETS[code],ready=!!(entry?.cover||entry?.avatar||entry?.hero);
 document.getElementById('countryTitle').textContent=item?.name||code;
 document.getElementById('edition').textContent=code.toUpperCase()+' · '+(ready?(['cover','avatar','hero'].filter(k=>entry[k]).length+' / 3 GÖRSEL HAZIR'):'ÜRETİM BEKLİYOR');
 document.getElementById('viewer').hidden=!ready;document.getElementById('pending').hidden=ready;document.getElementById('details').hidden=!ready;document.querySelector('.downloads').hidden=!ready;document.querySelector('.tabs').hidden=!ready;document.getElementById('countrySelect').value=code;
 scene.setAttribute('aria-label',(item?.name||code)+' üç boyutlu fotoğraf. Fare, sürükleme veya ok tuşlarıyla keşfedin.');
 if(!ready){notice.textContent='Bu ülke için tamamlanmış görsel bulunmuyor.';document.querySelector('#pending h2').textContent='Bu ülkenin görselleri henüz hazırlanmadı.';return false;}
 for(const kind of ['cover','avatar','hero']){const link=document.getElementById('download'+kind[0].toUpperCase()+kind.slice(1));link.hidden=!entry[kind];if(entry[kind]){link.href=entry[kind];link.download=code+'_'+kind+'.avif';}}
 const paragraphs=document.querySelectorAll('.detail p');paragraphs[0].textContent=entry.description||'Ülkenin farklı bölgelerinden seçilmiş kültürel ve doğal miraslar.';paragraphs[2].textContent=entry.animals||'Ülkenin doğal yaşamı ve kültürel detaylar.';return true;
}
function chooseCountry(code){
 if(videoRecorder.active){document.getElementById('countrySelect').value=country;notice.textContent='Ülke değiştirmek için önce video kaydını bitirin.';return Promise.resolve(false);}
 country=code;state.token++;setDemo(false);
 if(!updateCountryInfo(code)){state.ready=false;scene.classList.remove('ready');cancelVisualTransition();loading.hidden=true;return Promise.resolve(false);}
 const entry=ASSETS[code],revision=state.inputRevision,selected=select(entry.hero?'hero':entry.cover?'cover':'avatar'),token=state.token;
 return selected.then(loaded=>{if(loaded!==false&&token===state.token&&revision===state.inputRevision&&state.ready&&!reduced.matches){state.started=performance.now();setDemo(true);}return loaded;});
}
const regionNames=new Intl.DisplayNames(['tr'],{type:'region'});for(const c of COUNTRIES){c.name_en=c.name_en||c.name;if(c.name_tr){c.name=c.name_tr;continue;}try{c.name=regionNames.of(c.code.toUpperCase())||c.name;}catch(e){}}
const countrySelect=document.getElementById('countrySelect');
function filterCountries(query=''){const previous=country;countrySelect.replaceChildren();const fold=s=>s.toLocaleLowerCase('tr').normalize('NFD').replace(/[\u0300-\u036f]/g,'');const found=COUNTRIES.filter(c=>fold(c.name+' '+(c.name_en||'')+' '+c.code).includes(fold(query)));for(const c of found){const option=document.createElement('option');option.value=c.code;option.textContent=c.name+({constituent_country:' · BK bileşeni',special_status:' · Özel statülü',self_governing_state:' · Özyönetimli devlet',country_or_territory:' · Ülke/bölge'}[c.type]||'')+' · '+(ASSETS[c.code]?(['cover','avatar','hero'].filter(k=>ASSETS[c.code][k]).length+'/3 hazır'):'Bekliyor');countrySelect.append(option);}countrySelect.value=found.some(c=>c.code===previous)?previous:(found[0]?.code||'');countrySelect.disabled=!found.length;}
filterCountries();document.getElementById('progress').textContent=Object.values(ASSETS).reduce((n,a)=>n+['cover','avatar','hero'].filter(k=>a[k]).length,0)+' / '+(COUNTRIES.length*3)+' görsel · '+Object.values(ASSETS).filter(a=>a.cover&&a.avatar&&a.hero).length+' ülke tamamlandı';document.getElementById('countrySearch').addEventListener('input',e=>filterCountries(e.target.value));countrySelect.addEventListener('change',e=>chooseCountry(e.target.value));
function fallback(message){videoRecorder.stop();cancelVisualTransition();state.ready=false;scene.classList.remove('ready');notice.textContent=message;document.getElementById('demo').disabled=true;document.getElementById('strength').disabled=true;loading.hidden=true;}
