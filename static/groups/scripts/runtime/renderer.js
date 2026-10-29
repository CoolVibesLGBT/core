function uploadField(){if(flowEngine.gpu)return;if(!elasticField.dirty)return;gl.activeTexture(gl.TEXTURE2);gl.bindTexture(gl.TEXTURE_2D,fieldTexture);gl.pixelStorei(gl.UNPACK_FLIP_Y_WEBGL,false);gl.texSubImage2D(gl.TEXTURE_2D,0,0,0,elasticField.w,elasticField.h,gl.RGBA,gl.UNSIGNED_BYTE,elasticField.encode());elasticField.dirty=false;}
const vertex='attribute vec2 aPosition; varying vec2 vUv; void main(){vUv=aPosition*0.5+0.5;gl_Position=vec4(aPosition,0.,1.);}';
const fragment=`precision highp float;
uniform sampler2D uPanoramaPhoto;uniform float uPanorama;uniform vec2 uOrbit;uniform float uViewportAspect;
varying vec2 vUv;uniform sampler2D uPhoto;uniform sampler2D uDepth;uniform sampler2D uField;uniform float uFluidity;uniform vec2 uMotion;uniform vec2 uNearMotion;uniform float uLife;uniform vec2 uFlow;uniform vec2 uPointer;uniform float uTime;uniform float uAspect;uniform float uStrength;uniform vec2 uScale;uniform float uZoom;uniform vec2 uPhotoStep;uniform vec2 uFieldSize;
// Cubic reconstruction avoids visible grid transitions at strong deformation.
vec4 cubicWeights(float v){
vec4 n=vec4(1.0,2.0,3.0,4.0)-v;
vec4 q=n*n*n;
float x=q.x,y=q.y-4.0*q.x,z=q.z-4.0*q.y+6.0*q.x;
return vec4(x,y,z,6.0-x-y-z)/6.0;
}
vec2 smoothField(vec2 uv){
vec2 p=uv*uFieldSize-.5,cell=floor(p),f=fract(p);
vec4 x=cubicWeights(f.x),y=cubicWeights(f.y);
vec2 gx=vec2(x.x+x.y,x.z+x.w),gy=vec2(y.x+y.y,y.z+y.w);
vec2 px=cell.x+vec2(-1.0+x.y/gx.x,1.0+x.w/gx.y)+.5;
vec2 py=cell.y+vec2(-1.0+y.y/gy.x,1.0+y.w/gy.y)+.5;
vec2 a=texture2D(uField,vec2(px.x,py.x)/uFieldSize).rg;
vec2 b=texture2D(uField,vec2(px.y,py.x)/uFieldSize).rg;
vec2 c=texture2D(uField,vec2(px.x,py.y)/uFieldSize).rg;
vec2 d=texture2D(uField,vec2(px.y,py.y)/uFieldSize).rg;
return (mix(mix(a,b,gx.y),mix(c,d,gx.y),gy.y)*255.0-128.0)/100.0;
}
vec2 panoramaUv(vec2 uv){return vec2(fract(uv.x),clamp(uv.y,.001,.999));}
vec3 panoramaColor(vec2 uv){return texture2D(uPanoramaPhoto,panoramaUv(uv)).rgb;}
void main(){
if(uPanorama>.5){
// A precomputed, same-direction panorama wraps without mirrored turnarounds.
float lens=.72;
vec2 ray=(vUv-.5)*2.0*lens;
ray.x*=uViewportAspect;
float angle=atan(ray.x,1.0);
float horizontal=length(vec2(ray.x,1.0));
vec2 uv=vec2(.5+(uOrbit.x+angle)/6.28318530718,.5+ray.y/horizontal*.46+uOrbit.y);
// Keep the stitched surface rigid: unstitched depth would reintroduce a tear.
gl_FragColor=vec4(panoramaColor(uv),1.0);return;
}
// A small off-axis perspective tilt gives the entire scene camera volume.
vec2 screen=(vUv-vec2(0.5))*uScale;
float perspective=1.0+dot(screen,uMotion*vec2(.024,.018))*uStrength;
vec2 base=vec2(0.5)+screen*(0.92/uZoom)/perspective;
// Broad, continuous deformation follows motion; it vanishes as the image settles.
// The fixed border and low amplitude keep the image inside its overscan margin.
float depth=texture2D(uDepth,base).r;
vec2 p=base-vec2(0.5);
float envelope=(1.0-smoothstep(0.22,0.51,abs(p.x)))*(1.0-smoothstep(0.22,0.51,abs(p.y)));
// A broad wake follows the pointer in image space, with aspect-correct distances.
vec2 focus=vec2(0.5)+uPointer*uScale*(0.46/uZoom);
vec2 delta=(base-focus)*vec2(uAspect,1.0);
float radius=length(delta);
float influence=exp(-dot(delta,delta)*3.4);
float wave=sin(radius*6.0-uTime*2.0)*exp(-radius*2.8);
vec2 bend=vec2(sin(base.y*5.0+uTime*.65),sin(base.x*5.0-uTime*.55));
vec2 wake=-uFlow*influence*(.7+.3*wave);
vec2 curl=vec2(-delta.y,delta.x)/vec2(uAspect,1.0);
float rotation=(uFlow.x-uFlow.y)*influence;
vec2 flow=(wake+curl*rotation*.3+bend*uFlow*.25)*(.35+depth*.65)*envelope*.006*uStrength;
vec2 memory=smoothField(vUv);
vec2 deformation=(flow+memory*envelope*(.4+.6*depth)*.006)*uFluidity;
// Preserve subject boundaries: deform along the edge rather than stretching across it.
vec2 depthGradient=vec2(texture2D(uDepth,base+vec2(.004,0.0)).r-texture2D(uDepth,base-vec2(.004,0.0)).r,texture2D(uDepth,base+vec2(0.0,.004)).r-texture2D(uDepth,base-vec2(0.0,.004)).r);
float edge=length(depthGradient);
vec2 normal=depthGradient/max(edge,.0001);
deformation-=normal*dot(deformation,normal)*smoothstep(.03,.22,edge)*.7;
vec3 color=texture2D(uPhoto,base).rgb;
float detail=length(texture2D(uPhoto,base+uPhotoStep).rgb-color)+length(texture2D(uPhoto,base-uPhotoStep).rgb-color);
deformation*=1.0-.35*smoothstep(.2,.8,detail);
// Smoothly bound high settings; no hard clipping or exposed image borders.
deformation/=sqrt(1.0+dot(deformation,deformation)/(.028*.028));
base+=deformation;
vec2 uv=base;
for(int i=0;i<16;i++){float d=texture2D(uDepth,clamp(uv,0.001,0.999)).r;vec2 layerMotion=mix(uMotion,uNearMotion,smoothstep(.25,.9,d));
// Foreground travels farther than the horizon, in image-aspect-correct units.
vec2 layerTravel=layerMotion*vec2(0.062/uAspect,0.062)*uStrength;
vec2 next=base-layerTravel*(mix(d,d*d,.35)-0.28);uv=mix(uv,next,0.65);}
gl_FragColor=vec4(texture2D(uPhoto,clamp(uv,0.001,0.999)).rgb,1.0);
}`;
function shader(type,source){if(type===gl.FRAGMENT_SHADER&&gl.getShaderPrecisionFormat&&!gl.getShaderPrecisionFormat(gl.FRAGMENT_SHADER,gl.HIGH_FLOAT).precision)source=source.replace('precision highp float','precision mediump float');const s=gl.createShader(type);gl.shaderSource(s,source);gl.compileShader(s);if(!gl.getShaderParameter(s,gl.COMPILE_STATUS))throw new Error(gl.getShaderInfoLog(s));return s;}
function init(){gl=canvas.getContext('webgl',{alpha:false,antialias:false,preserveDrawingBuffer:false,powerPreference:'low-power'});if(!gl)throw new Error('WebGL unavailable');program=gl.createProgram();gl.attachShader(program,shader(gl.VERTEX_SHADER,vertex));gl.attachShader(program,shader(gl.FRAGMENT_SHADER,fragment));gl.linkProgram(program);if(!gl.getProgramParameter(program,gl.LINK_STATUS))throw new Error(gl.getProgramInfoLog(program));gl.useProgram(program);const buffer=mainBuffer=gl.createBuffer();gl.bindBuffer(gl.ARRAY_BUFFER,buffer);gl.bufferData(gl.ARRAY_BUFFER,new Float32Array([-1,-1,1,-1,-1,1,-1,1,1,-1,1,1]),gl.STATIC_DRAW);const pos=mainPosition=gl.getAttribLocation(program,'aPosition');gl.enableVertexAttribArray(pos);gl.vertexAttribPointer(pos,2,gl.FLOAT,false,0,0);locations={};for(const key of ['uPanoramaPhoto','uPanorama','uOrbit','uViewportAspect','uPhoto','uZoom','uPhotoStep','uFieldSize','uDepth','uField','uFluidity','uMotion','uNearMotion','uLife','uFlow','uPointer','uTime','uAspect','uStrength','uScale'])locations[key]=gl.getUniformLocation(program,key);panoramaTexture=gl.createTexture();gl.uniform1i(locations.uPanoramaPhoto,3);photoTexture=gl.createTexture();depthTexture=gl.createTexture();gl.uniform1i(locations.uPhoto,0);gl.uniform1i(locations.uDepth,1);fieldTexture=gl.createTexture();gl.activeTexture(gl.TEXTURE2);gl.bindTexture(gl.TEXTURE_2D,fieldTexture);gl.texParameteri(gl.TEXTURE_2D,gl.TEXTURE_MIN_FILTER,gl.LINEAR);gl.texParameteri(gl.TEXTURE_2D,gl.TEXTURE_MAG_FILTER,gl.LINEAR);gl.texParameteri(gl.TEXTURE_2D,gl.TEXTURE_WRAP_S,gl.CLAMP_TO_EDGE);gl.texParameteri(gl.TEXTURE_2D,gl.TEXTURE_WRAP_T,gl.CLAMP_TO_EDGE);gl.texImage2D(gl.TEXTURE_2D,0,gl.RGBA,elasticField.w,elasticField.h,0,gl.RGBA,gl.UNSIGNED_BYTE,elasticField.encode());gl.uniform1i(locations.uField,2);try{flowEngine=new GPUFlow(gl);}catch(e){flowEngine=elasticField;}prepareMainDraw();}
function texture(unit,tex,source){gl.activeTexture(gl.TEXTURE0+unit);gl.bindTexture(gl.TEXTURE_2D,tex);gl.pixelStorei(gl.UNPACK_FLIP_Y_WEBGL,true);gl.texParameteri(gl.TEXTURE_2D,gl.TEXTURE_MIN_FILTER,gl.LINEAR);gl.texParameteri(gl.TEXTURE_2D,gl.TEXTURE_MAG_FILTER,gl.LINEAR);gl.texParameteri(gl.TEXTURE_2D,gl.TEXTURE_WRAP_S,gl.CLAMP_TO_EDGE);gl.texParameteri(gl.TEXTURE_2D,gl.TEXTURE_WRAP_T,gl.CLAMP_TO_EDGE);gl.texImage2D(gl.TEXTURE_2D,0,gl.RGB,gl.RGB,gl.UNSIGNED_BYTE,source);}
// A hand-authored depth field follows the subjects in these two illustrations.
// White is near; black is far. Soft boundaries prevent hard tearing.
function depthMap(kind){
if(kind==='hero'||country!=='tr'||ASSETS[country]?.depthMode==='approximate'){const map=document.createElement('canvas');map.width=256;map.height=256;const c=map.getContext('2d'),g=c.createLinearGradient(0,0,0,256);g.addColorStop(0,'#202020');g.addColorStop(1,'#dedede');c.fillStyle=g;c.fillRect(0,0,256,256);return map;}const map=document.createElement('canvas');map.width=kind==='cover'?1500:768;map.height=kind==='cover'?500:768;const c=map.getContext('2d'),w=map.width,h=map.height;c.scale(w,h);const g=c.createLinearGradient(0,0,0,1);g.addColorStop(0,'#141414');g.addColorStop(.24,'#292929');g.addColorStop(.48,'#777777');g.addColorStop(1,'#dedede');c.fillStyle=g;c.fillRect(0,0,1,1);
function shape(points,value,blur=.009){c.save();c.filter=`blur(${Math.max(1,blur*w)}px)`;c.fillStyle=`rgb(${value},${value},${value})`;c.beginPath();points.forEach((p,i)=>i?c.lineTo(p[0],p[1]):c.moveTo(p[0],p[1]));c.closePath();c.fill();c.restore();}
function ellipse(x,y,rx,ry,value){c.save();c.filter=`blur(${Math.max(1,w*.008)}px)`;c.fillStyle=`rgb(${value},${value},${value})`;c.beginPath();c.ellipse(x,y,rx,ry,0,0,Math.PI*2);c.fill();c.restore();}
if(kind==='cover'){
shape([[.007,.75],[.027,.42],[.105,.28],[.14,.19],[.17,.02],[.218,.035],[.248,.23],[.246,.31],[.222,.32],[.20,.24],[.188,.4],[.177,.64],[.19,.91],[.025,.93]],238,.004);
shape([[.063,.28],[.075,.065],[.093,.065],[.104,.28]],108,.004);
shape([[.29,.61],[.295,.29],[.319,.245],[.405,.245],[.427,.33],[.431,.63]],139,.004);
shape([[.425,.62],[.45,.21],[.48,.11],[.52,.3],[.55,.12],[.59,.29],[.62,.17],[.66,.4],[.68,.64]],120,.008);
ellipse(.843,.79,.052,.174,245);ellipse(.823,.615,.036,.083,242);
shape([[.793,.64],[.79,.535],[.811,.56],[.846,.53],[.851,.653]],245,.004);
ellipse(.891,.914,.046,.051,248);ellipse(.588,.805,.04,.093,225);ellipse(.598,.731,.015,.043,228);
ellipse(.343,.847,.058,.052,235);shape([[0,.83],[.12,.89],[.21,1],[0,1]],254,.01);shape([[.93,.80],[1,.68],[1,1],[.95,1]],253,.01);
}else{
shape([[.03,.60],[.075,.29],[.25,.21],[.29,.10],[.35,.015],[.427,.015],[.456,.20],[.445,.25],[.395,.20],[.382,.39],[.42,.62],[.445,.72],[.03,.73]],206,.004);
ellipse(.565,.668,.141,.204,242);ellipse(.585,.466,.098,.09,245);
shape([[.492,.48],[.505,.36],[.551,.412],[.615,.413],[.647,.359],[.659,.489]],244,.004);ellipse(.68,.858,.129,.055,248);
shape([[0,.80],[.24,.84],[.39,1],[0,1]],254,.009);shape([[.87,.92],[1,.84],[1,1],[.87,1]],254,.009);
}
return map;}
function resize(){if(!gl)return;const r=scene.getBoundingClientRect(),dpr=Math.min(window.devicePixelRatio||1,2,Math.sqrt(3500000/Math.max(1,r.width*r.height)))*qualityManager.scale;const w=Math.max(1,Math.round(r.width*dpr)),h=Math.max(1,Math.round(r.height*dpr));if(canvas.width!==w||canvas.height!==h){canvas.width=w;canvas.height=h;}let vw=w,vh=h;const imageAspect=textureSize[0]/textureSize[1];if(document.fullscreenElement===scene){if(w/h>imageAspect)vw=Math.round(h*imageAspect);else vh=Math.round(w/imageAspect);gl.clearColor(.137,.118,.169,1);gl.clear(gl.COLOR_BUFFER_BIT);}gl.viewport(Math.round((w-vw)/2),Math.round((h-vh)/2),vw,vh);const screenAspect=vw/vh;let scale=[1,1];if(screenAspect>imageAspect)scale[1]=imageAspect/screenAspect;else scale[0]=screenAspect/imageAspect;gl.uniform2f(locations.uScale,...scale);}

function prepareMainDraw(){
 gl.bindFramebuffer(gl.FRAMEBUFFER,null);gl.useProgram(program);gl.bindBuffer(gl.ARRAY_BUFFER,mainBuffer);gl.enableVertexAttribArray(mainPosition);gl.vertexAttribPointer(mainPosition,2,gl.FLOAT,false,0,0);gl.enable(gl.DITHER);
 gl.activeTexture(gl.TEXTURE3);gl.bindTexture(gl.TEXTURE_2D,panoramaTexture);gl.activeTexture(gl.TEXTURE0);gl.bindTexture(gl.TEXTURE_2D,photoTexture);gl.activeTexture(gl.TEXTURE1);gl.bindTexture(gl.TEXTURE_2D,depthTexture);gl.activeTexture(gl.TEXTURE2);gl.bindTexture(gl.TEXTURE_2D,flowEngine.gpu?flowEngine.texture:fieldTexture);
}

// Join opposite edge strips with a content-aware seam and a Laplacian pyramid.
// The source texture is untouched. Only the panorama texture uses an overlap.
function stitchPanorama(pixels,w,h){
const overlap=Math.max(8,Math.floor(w*.24)),ow=w-overlap;
const a=new Float32Array(overlap*h*3),b=new Float32Array(a.length);
for(let y=0;y<h;y++)for(let x=0;x<overlap;x++)for(let c=0;c<3;c++){
const i=(y*overlap+x)*3+c;a[i]=pixels[(y*w+w-overlap+x)*4+c]/255;b[i]=pixels[(y*w+x)*4+c]/255;
}
// Minimum-error path follows similar colors; the central bias avoids sliver joins.
const sw=Math.min(96,overlap),sh=Math.min(192,h),parents=new Int16Array(sw*sh);
let costs=new Float64Array(sw);
for(let y=0;y<sh;y++){
const next=new Float64Array(sw);
for(let x=0;x<sw;x++){
const sx=Math.round(x*(overlap-1)/(sw-1)),sy=Math.round(y*(h-1)/Math.max(1,sh-1)),i=(sy*overlap+sx)*3;
let error=0;for(let c=0;c<3;c++)error+=(a[i+c]-b[i+c])**2;
error+=.15*((x/(sw-1)-.5)*2)**4;
let best=x;if(y){for(let k=Math.max(0,x-1);k<=Math.min(sw-1,x+1);k++)if(costs[k]<costs[best])best=k;}
parents[y*sw+x]=best;next[x]=error+(y?costs[best]:0);
}costs=next;
}
const path=new Float32Array(sh);let last=costs.indexOf(Math.min(...costs));
for(let y=sh-1;y>=0;y--){path[y]=last/(sw-1)*(overlap-1);last=parents[y*sw+last];}
const mask=new Float32Array(overlap*h);
for(let y=0;y<h;y++){
const py=y*(sh-1)/Math.max(1,h-1),lo=Math.floor(py),cut=path[lo]+(path[Math.min(sh-1,lo+1)]-path[lo])*(py-lo);
for(let x=0;x<overlap;x++){const t=Math.max(0,Math.min(1,(x-cut)/6+.5));mask[y*overlap+x]=1-t*t*(3-2*t);}
}
function blur(src,width,height,channels){
const tmp=new Float32Array(src.length),out=new Float32Array(src.length),weights=[1,4,6,4,1];
for(let y=0;y<height;y++)for(let x=0;x<width;x++)for(let c=0;c<channels;c++){
let v=0;for(let k=-2;k<=2;k++)v+=src[(y*width+Math.max(0,Math.min(width-1,x+k)))*channels+c]*weights[k+2];tmp[(y*width+x)*channels+c]=v/16;
}
for(let y=0;y<height;y++)for(let x=0;x<width;x++)for(let c=0;c<channels;c++){
let v=0;for(let k=-2;k<=2;k++)v+=tmp[(Math.max(0,Math.min(height-1,y+k))*width+x)*channels+c]*weights[k+2];out[(y*width+x)*channels+c]=v/16;
}return out;
}
function down(src,width,height,channels){
const smooth=blur(src,width,height,channels),nw=Math.ceil(width/2),nh=Math.ceil(height/2),out=new Float32Array(nw*nh*channels);
for(let y=0;y<nh;y++)for(let x=0;x<nw;x++)for(let c=0;c<channels;c++)out[(y*nw+x)*channels+c]=smooth[((y*2)*width+x*2)*channels+c];
return {data:out,w:nw,h:nh};
}
function up(src,width,height,nw,nh){
const out=new Float32Array(nw*nh*3);
for(let y=0;y<nh;y++)for(let x=0;x<nw;x++){
const px=x/2,py=y/2,x0=Math.min(width-1,Math.floor(px)),y0=Math.min(height-1,Math.floor(py)),x1=Math.min(width-1,x0+1),y1=Math.min(height-1,y0+1),fx=px-x0,fy=py-y0;
for(let c=0;c<3;c++){const top=src[(y0*width+x0)*3+c]*(1-fx)+src[(y0*width+x1)*3+c]*fx,bot=src[(y1*width+x0)*3+c]*(1-fx)+src[(y1*width+x1)*3+c]*fx;out[(y*nw+x)*3+c]=top*(1-fy)+bot*fy;}
}return out;
}
const levels=[{a,b,m:mask,w:overlap,h}];
while(levels.length<7&&Math.min(levels.at(-1).w,levels.at(-1).h)>2){const prev=levels.at(-1),da=down(prev.a,prev.w,prev.h,3),db=down(prev.b,prev.w,prev.h,3),dm=down(prev.m,prev.w,prev.h,1);levels.push({a:da.data,b:db.data,m:dm.data,w:da.w,h:da.h});}
let result;
for(let l=levels.length-1;l>=0;l--){
const cur=levels[l],coarse=levels[l+1],ua=coarse?up(coarse.a,coarse.w,coarse.h,cur.w,cur.h):null,ub=coarse?up(coarse.b,coarse.w,coarse.h,cur.w,cur.h):null,base=coarse?up(result,coarse.w,coarse.h,cur.w,cur.h):new Float32Array(cur.a.length);
result=new Float32Array(cur.a.length);
for(let i=0;i<result.length;i++){const weight=cur.m[Math.floor(i/3)];result[i]=base[i]+(cur.a[i]-(ua?ua[i]:0))*weight+(cur.b[i]-(ub?ub[i]:0))*(1-weight);}
}
const out=new Uint8ClampedArray(ow*h*4),middle=w-2*overlap;
for(let y=0;y<h;y++)for(let x=0;x<ow;x++){
const dest=(y*ow+x)*4;
for(let c=0;c<3;c++){
if(x<middle)out[dest+c]=pixels[(y*w+overlap+x)*4+c];
else{
const sx=x-middle,i=(y*overlap+sx)*3+c;
// Exact input at overlap boundaries prevents a new seam after pyramid reconstruction.
const t=sx/(overlap-1),edge=Math.min(1,Math.min(t,1-t)*12),fade=edge*edge*(3-2*edge);
const endpoint=t<.5?a[i]:b[i];out[dest+c]=255*(endpoint*(1-fade)+result[i]*fade);
}
}out[dest+3]=255;
}
return {pixels:out,width:ow,height:h};
}
function uploadPanorama(artwork){
const width=Math.min(1536,artwork.naturalWidth),height=Math.max(8,Math.round(artwork.naturalHeight*width/artwork.naturalWidth));
const work=document.createElement('canvas');work.width=width;work.height=height;
const ctx=work.getContext('2d',{willReadFrequently:true});ctx.drawImage(artwork,0,0,width,height);
const joined=stitchPanorama(ctx.getImageData(0,0,width,height).data,width,height);
work.width=joined.width;work.height=joined.height;
const image=ctx.createImageData(joined.width,joined.height);image.data.set(joined.pixels);ctx.putImageData(image,0,0);
texture(3,panoramaTexture,work);
}
