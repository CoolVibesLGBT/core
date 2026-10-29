class ElasticField{
constructor(w=48,h=32){this.w=w;this.h=h;this.position=new Float32Array(w*h*2);this.velocity=new Float32Array(w*h*2);this.next=new Float32Array(w*h*2);this.pixels=new Uint8Array(w*h*4);this.accumulator=0;this.energy=0;this.dirty=true;this.encode();}
reset(){this.position.fill(0);this.velocity.fill(0);this.accumulator=0;this.energy=0;this.dirty=true;this.encode();}
splat(x,y,dx,dy,aspect=1){
const size=Math.hypot(dx,dy);if(size<.00001)return;
const limit=Math.min(1,.12/size);dx*=limit;dy*=limit;
for(let j=1;j<this.h-1;j++)for(let i=1;i<this.w-1;i++){
const rx=(i/(this.w-1)-x)*aspect,ry=j/(this.h-1)-y;
const weight=Math.exp(-(rx*rx+ry*ry)/.045);if(weight<.001)continue;
const k=(j*this.w+i)*2;
this.velocity[k]=Math.max(-12,Math.min(12,this.velocity[k]+(dx-ry*(dx-dy)*1.8)*30*weight));
this.velocity[k+1]=Math.max(-12,Math.min(12,this.velocity[k+1]+(dy+rx*(dx-dy)*1.8)*30*weight));
}this.energy=1;this.dirty=true;
}
step(dt,elasticity){
if(!this.energy)return;this.accumulator+=Math.min(dt,.05);const h=1/120,damping=8-2.5*elasticity;
while(this.accumulator>=h){
for(let j=1;j<this.h-1;j++)for(let i=1;i<this.w-1;i++)for(let axis=0;axis<2;axis++){
const k=(j*this.w+i)*2+axis,p=this.position;
const lap=p[k-2]+p[k+2]+p[k-this.w*2]+p[k+this.w*2]-4*p[k];
this.next[k]=(this.velocity[k]+(125*lap-18*p[k])*h)*Math.exp(-damping*h);
}
let energy=0;
for(let k=0;k<this.position.length;k++){
this.velocity[k]=this.next[k];this.position[k]=Math.max(-1.1,Math.min(1.1,this.position[k]+this.velocity[k]*h));
energy=Math.max(energy,Math.abs(this.position[k]),Math.abs(this.velocity[k])*.1);
}this.energy=energy;this.accumulator-=h;this.dirty=true;
}
if(this.energy<.0001&&this.energy!==0)this.reset();
}
encode(){for(let i=0;i<this.w*this.h;i++){this.pixels[i*4]=Math.round(128+this.position[i*2]*100);this.pixels[i*4+1]=Math.round(128+this.position[i*2+1]*100);this.pixels[i*4+2]=0;this.pixels[i*4+3]=255;}return this.pixels;}
}
const elasticField=new ElasticField();let fieldPointer=null;

// Float ping-pong textures keep the displacement and its velocity on the GPU.
// Actual framebuffer validation decides support; CPU remains the fallback.
class GPUFlow {
  constructor(context, w=96, h=64) {
    this.gl=context;this.w=w;this.h=h;this.gpu=true;this.energy=0;this.accumulator=0;this.remaining=0;this.queue=[];this.index=0;this.textures=[];this.frames=[];
    if(!context.getExtension('OES_texture_float')||!context.getExtension('OES_texture_float_linear'))throw new Error('Float flow unavailable');
    context.getExtension('WEBGL_color_buffer_float');
    const fragmentSource=`precision highp float;
    varying vec2 vUv;uniform sampler2D uPrevious;uniform vec2 uSize;uniform float uDamping;uniform float uAspect;uniform int uCount;uniform vec4 uSplats[8];
    vec2 displacement(vec2 uv){return (texture2D(uPrevious,uv).rg*255.0-128.0)/100.0;}
    void main(){
      vec2 texel=1.0/uSize;
      if(vUv.x<texel.x||vUv.x>1.0-texel.x||vUv.y<texel.y||vUv.y>1.0-texel.y){gl_FragColor=vec4(vec2(128.0/255.0),vec2(.5));return;}
      vec4 previous=texture2D(uPrevious,vUv);vec2 p=(previous.rg*255.0-128.0)/100.0;vec2 velocity=(previous.ba-.5)*24.0;
      for(int i=0;i<8;i++){if(i<uCount){vec4 s=uSplats[i];vec2 delta=(vUv-s.xy)*vec2(uAspect,1.0);float weight=exp(-dot(delta,delta)/.045);vec2 tangent=vec2(-delta.y,delta.x)*(s.z-s.w)*1.8;velocity=clamp(velocity+(s.zw+tangent)*30.0*weight,vec2(-12.0),vec2(12.0));}}
      vec2 lapX=displacement(vUv+vec2(texel.x,0.0))+displacement(vUv-vec2(texel.x,0.0))-2.0*p;
      vec2 lapY=displacement(vUv+vec2(0.0,texel.y))+displacement(vUv-vec2(0.0,texel.y))-2.0*p;
      vec2 ratio=uSize/vec2(48.0,32.0);
      velocity=(velocity+(125.0*(lapX*ratio.x*ratio.x+lapY*ratio.y*ratio.y)-18.0*p)/120.0)*exp(-uDamping/120.0);
      p=clamp(p+velocity/120.0,vec2(-1.1),vec2(1.1));
      gl_FragColor=vec4((128.0+p*100.0)/255.0,velocity/24.0+.5);
    }`;
    const shaders=[];
    try {
      this.program=context.createProgram();
      for(const [type,source] of [[context.VERTEX_SHADER,vertex],[context.FRAGMENT_SHADER,fragmentSource]]){
        const item=context.createShader(type);shaders.push(item);context.shaderSource(item,source);context.compileShader(item);
        if(!context.getShaderParameter(item,context.COMPILE_STATUS))throw new Error(context.getShaderInfoLog(item));context.attachShader(this.program,item);
      }
      context.linkProgram(this.program);if(!context.getProgramParameter(this.program,context.LINK_STATUS))throw new Error(context.getProgramInfoLog(this.program));
      this.position=context.getAttribLocation(this.program,'aPosition');this.uniforms={};
      for(const key of ['uPrevious','uSize','uDamping','uAspect','uCount','uSplats[0]'])this.uniforms[key]=context.getUniformLocation(this.program,key);
      this.buffer=context.createBuffer();context.bindBuffer(context.ARRAY_BUFFER,this.buffer);context.bufferData(context.ARRAY_BUFFER,new Float32Array([-1,-1,1,-1,-1,1,-1,1,1,-1,1,1]),context.STATIC_DRAW);
      this.blank=new Float32Array(w*h*4);for(let i=0;i<w*h;i++){this.blank[i*4]=128/255;this.blank[i*4+1]=128/255;this.blank[i*4+2]=.5;this.blank[i*4+3]=.5;}
      for(let i=0;i<2;i++){
        const texture=context.createTexture();this.textures.push(texture);context.bindTexture(context.TEXTURE_2D,texture);
        context.texParameteri(context.TEXTURE_2D,context.TEXTURE_MIN_FILTER,context.LINEAR);context.texParameteri(context.TEXTURE_2D,context.TEXTURE_MAG_FILTER,context.LINEAR);context.texParameteri(context.TEXTURE_2D,context.TEXTURE_WRAP_S,context.CLAMP_TO_EDGE);context.texParameteri(context.TEXTURE_2D,context.TEXTURE_WRAP_T,context.CLAMP_TO_EDGE);
        context.texImage2D(context.TEXTURE_2D,0,context.RGBA,w,h,0,context.RGBA,context.FLOAT,this.blank);
        const frame=context.createFramebuffer();this.frames.push(frame);context.bindFramebuffer(context.FRAMEBUFFER,frame);context.framebufferTexture2D(context.FRAMEBUFFER,context.COLOR_ATTACHMENT0,context.TEXTURE_2D,texture,0);
        if(context.checkFramebufferStatus(context.FRAMEBUFFER)!==context.FRAMEBUFFER_COMPLETE)throw new Error('Float framebuffer unavailable');
      }
    } catch(error) {this.destroy();throw error;}
    finally {for(const item of shaders)context.deleteShader(item);context.bindFramebuffer(context.FRAMEBUFFER,null);}
  }
  get texture(){return this.textures[this.index];}
  splat(x,y,dx,dy,aspect=1){
    const magnitude=Math.hypot(dx,dy);if(magnitude<.00001)return;const cap=Math.min(1,.12/magnitude);dx*=cap;dy*=cap;this.aspect=aspect;
    if(this.queue.length<8)this.queue.push([x,y,dx,dy]);
    else{const previous=this.queue[7];this.queue[7]=[x,y,Math.max(-.12,Math.min(.12,previous[2]+dx)),Math.max(-.12,Math.min(.12,previous[3]+dy))];}
    this.energy=1;this.remaining=4.5;
  }
  step(dt,elasticity){
    if(!this.energy)return;const g=this.gl;this.accumulator+=Math.min(dt,.05);
    g.useProgram(this.program);g.bindBuffer(g.ARRAY_BUFFER,this.buffer);g.enableVertexAttribArray(this.position);g.vertexAttribPointer(this.position,2,g.FLOAT,false,0,0);g.disable(g.DITHER);g.viewport(0,0,this.w,this.h);
    g.uniform1i(this.uniforms.uPrevious,0);g.uniform2f(this.uniforms.uSize,this.w,this.h);g.uniform1f(this.uniforms.uDamping,8-2.5*elasticity);g.uniform1f(this.uniforms.uAspect,this.aspect||1);
    while(this.accumulator+1e-10>=1/120){
      const input=new Float32Array(32);this.queue.forEach((s,i)=>input.set(s,i*4));g.uniform4fv(this.uniforms['uSplats[0]'],input);g.uniform1i(this.uniforms.uCount,this.queue.length);this.queue=[];
      g.bindFramebuffer(g.FRAMEBUFFER,this.frames[1-this.index]);g.activeTexture(g.TEXTURE0);g.bindTexture(g.TEXTURE_2D,this.texture);g.drawArrays(g.TRIANGLES,0,6);this.index=1-this.index;this.accumulator-=1/120;this.remaining-=1/120;
    }
    g.bindFramebuffer(g.FRAMEBUFFER,null);this.energy=Math.max(0,this.remaining/4.5);if(this.remaining<=0)this.reset();
  }
  reset(){
    this.queue=[];this.accumulator=0;this.remaining=0;
    if(this.energy){const g=this.gl;g.activeTexture(g.TEXTURE0);for(const texture of this.textures){g.bindTexture(g.TEXTURE_2D,texture);g.texSubImage2D(g.TEXTURE_2D,0,0,0,this.w,this.h,g.RGBA,g.FLOAT,this.blank);}}
    this.energy=0;
  }
  destroy(){const g=this.gl;for(const frame of this.frames)g.deleteFramebuffer(frame);for(const texture of this.textures)g.deleteTexture(texture);if(this.buffer)g.deleteBuffer(this.buffer);if(this.program)g.deleteProgram(this.program);}
}
let flowEngine=elasticField;
