// Continuous position and velocity across retargeting: no abrupt easing restarts.
function transitionValue(key,target,frequency,dt){
if(reduced.matches){state[key]=target;state.transitionVelocity[key]=0;return;}
const value=state[key],velocity=state.transitionVelocity[key]||0;
const offset=value-target,impulse=velocity+frequency*offset,decay=Math.exp(-frequency*dt);
state[key]=target+(offset+impulse*dt)*decay;
state.transitionVelocity[key]=(velocity-frequency*impulse*dt)*decay;
}
// Smooth only the camera input; the original per-pixel depth shader stays intact.
function advanceParallax(dt){
transitionValue('renderedSpeed',motionSettings.speed/100,12,dt);
transitionValue('renderedElasticity',motionSettings.elasticity/100,12,dt);
const frequency=reduced.matches?30:5.8*state.renderedSpeed;
const damping=reduced.matches?1:1-.19*state.renderedElasticity;
for(let i=0;i<2;i++){
const offset=state.current[i]-state.target[i],velocity=state.velocity[i];
if(damping===1){
const impulse=velocity+frequency*offset,decay=Math.exp(-frequency*dt);
state.current[i]=state.target[i]+(offset+impulse*dt)*decay;
state.velocity[i]=(velocity-frequency*impulse*dt)*decay;
}else{
const attenuation=frequency*damping,oscillation=frequency*Math.sqrt(1-damping*damping);
const decay=Math.exp(-attenuation*dt),cos=Math.cos(oscillation*dt),sin=Math.sin(oscillation*dt);
state.current[i]=state.target[i]+decay*(offset*cos+(velocity+attenuation*offset)/oscillation*sin);
state.velocity[i]=decay*(velocity*cos-(attenuation*velocity+frequency*frequency*offset)/oscillation*sin);
}
}
}
// A separate wake decays gradually after the camera stops moving.
function advanceFlow(dt){
const follow=1-Math.exp(-4.5*dt);
for(let i=0;i<2;i++){
const goal=reduced.matches?0:Math.tanh(state.velocity[i]*.12);
state.flow[i]=reduced.matches?0:state.flow[i]+(goal-state.flow[i])*follow;
}
const response=state.renderedSpeed;
const nearFollow=1-Math.exp(-(4.8-1.4*state.renderedElasticity)*response*dt),pointerFollow=1-Math.exp(-8*response*dt);
for(let i=0;i<2;i++){
state.near[i]+=(state.current[i]-state.near[i])*nearFollow;
state.pointer[i]+=(state.target[i]-state.pointer[i])*pointerFollow;
}
// Gentle dolly follows the same motion, with slower easing than lateral movement.
const distance=Math.min(1,Math.hypot(...state.current)/Math.SQRT2);
const momentum=Math.tanh(Math.hypot(...state.velocity)*.16);
state.zoomTarget=reduced.matches?1:Math.min(1.18,(1+(distance*.06+momentum*.006)*(motionSettings.zoom/100)*state.strength)*state.touchZoom);
transitionValue('zoom',state.zoomTarget,5*response,dt);
transitionValue('renderedStrength',state.strength,12,dt);
const fluid=motionSettings.fluidity/100;
transitionValue('renderedFluidity',fluid*(1+.35*fluid),12,dt);
transitionValue('life',state.demo&&!reduced.matches?1:0,8,dt);
state.flowTime+=dt;
}
function advanceDemo(dt){
state.demoPhase+=dt*.18*state.renderedSpeed;state.demoElapsed+=dt;
const progress=Math.min(1,state.demoElapsed/2.8);
const blend=progress*progress*progress*(progress*(progress*6-15)+10),t=state.demoPhase;
const orbit=[Math.sin(t)*.72+Math.sin(t*.43)*.08,Math.cos(t*.68)*.32];
state.target=state.demoOrigin.map((v,i)=>v+(orbit[i]-v)*blend);
}
// One simulation clock drives camera, dolly, settings and surface deformation.
// Small shared substeps prevent a slow frame from changing their relative timing.
function advanceScene(elapsed){
const duration=Math.max(0,Math.min(elapsed,.1));
if(!duration)return;
const count=Math.max(1,Math.ceil(duration*120-1e-8)),dt=duration/count;
for(let step=0;step<count;step++){
if(state.demo&&!reduced.matches)advanceDemo(dt);
advanceParallax(dt);advanceFlow(dt);
if(reduced.matches||state.renderedFluidity<.0001)flowEngine.reset();
else{
if(state.demo)flowEngine.splat(state.pointer[0]*.5+.5,state.pointer[1]*.5+.5,state.velocity[0]*dt*.18,state.velocity[1]*dt*.18,textureSize[0]/textureSize[1]);
flowEngine.step(dt,state.renderedElasticity);
}
}
}
