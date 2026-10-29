const scene=document.getElementById('scene'),canvas=document.getElementById('canvas'),photo=document.getElementById('photo');
const loading=document.getElementById('loading'),notice=document.getElementById('notice');
const reduced=window.matchMedia('(prefers-reduced-motion: reduce)');
let country='tr';
const motionDefaults={elasticity:65,fluidity:70,speed:100,strength:70,zoom:55};
const motionSettings={...motionDefaults};
try{const saved=JSON.parse(localStorage.getItem('atlas-motion-v1')||'{}');for(const key of Object.keys(motionDefaults)){const v=saved[key];if(typeof v==='number'&&Number.isFinite(v))motionSettings[key]=Math.max(key==='speed'?40:0,Math.min(key==='speed'?180:100,v));}}catch(e){}

const state={image:'cover',target:[0,0],current:[0,0],strength:.7,demo:false,gyro:false,ready:false,visible:true,token:0,raf:0,last:0,started:performance.now(),velocity:[0,0],flow:[0,0],flowTime:0,near:[0,0],pointer:[0,0],zoom:1,zoomTarget:1,renderedStrength:.7,renderedFluidity:1,demoPhase:0,demoElapsed:0,demoOrigin:[0,0],renderedSpeed:1,renderedElasticity:.65,life:0,transitionVelocity:{},inputRevision:0,recording:false,touchZoom:1,displayCountry:null};
let gl,program,locations,panoramaTexture,photoTexture,depthTexture,fieldTexture,mainBuffer,mainPosition,textureSize=[3,1];

const panorama={enabled:false,yaw:0,targetYaw:0,pitch:0,targetPitch:0,drag:null};
