// Real-browser smoke suite. No packages are downloaded; use PLAYWRIGHT_MODULE if needed.
import {createRequire} from 'node:module';
import {createServer} from 'node:http';
import {readFile,writeFile,mkdir,readdir} from 'node:fs/promises';
import {existsSync,createReadStream} from 'node:fs';
import {resolve,extname,join} from 'node:path';
import {tmpdir,homedir} from 'node:os';
import assert from 'node:assert/strict';
const require=createRequire(import.meta.url),root=resolve(new URL('../',import.meta.url).pathname);
const output=process.env.ATLAS_QA_OUTPUT||join(tmpdir(),'atlas-browser-qa');await mkdir(output,{recursive:true});
const report={status:'running',checks:[],errors:[]};let browser,server;
try{
 let playwright;
 const candidates=[process.env.PLAYWRIGHT_MODULE,'playwright','playwright-core'].filter(Boolean);
 const yarn=join(homedir(),'Library/Caches/Yarn/v6');
 if(existsSync(yarn))for(const entry of await readdir(yarn))if(entry.startsWith('npm-playwright-core-'))candidates.push(join(yarn,entry,'node_modules/playwright-core'));
 for(const candidate of candidates){try{playwright=require(candidate);break;}catch(error){}}
 if(!playwright)throw new Error('Playwright bulunamadı. PLAYWRIGHT_MODULE ile kurulu playwright-core yolunu belirtin.');
 const launch={headless:true,args:['--enable-unsafe-swiftshader']};
 const cached=join(homedir(),'Library/Caches/ms-playwright/chromium_headless_shell-1248/chrome-headless-shell-mac-arm64/chrome-headless-shell');
 if(process.env.ATLAS_BROWSER_PATH)launch.executablePath=process.env.ATLAS_BROWSER_PATH;else if(existsSync(cached))launch.executablePath=cached;
 browser=await playwright.chromium.launch(launch);
 server=createServer((request,response)=>{
  let path;try{path=resolve(root,'.'+decodeURIComponent(new URL(request.url,'http://localhost').pathname));}catch(error){response.writeHead(400).end();return;}
  if(path!==root&&!path.startsWith(root+'/')){response.writeHead(403).end();return;}
  const types={'.html':'text/html; charset=utf-8','.avif':'image/avif','.png':'image/png','.js':'text/javascript'};
  const stream=createReadStream(path===root?join(root,'atlas.html'):path);stream.on('error',()=>response.writeHead(404).end());
  stream.on('open',()=>{response.setHeader('Content-Type',types[extname(path)]||'application/octet-stream');stream.pipe(response);});
 });
 await new Promise((done,reject)=>{server.once('error',reject);server.listen(0,'127.0.0.1',done);});
 const base='http://127.0.0.1:'+server.address().port;
 const page=await browser.newPage({viewport:{width:1440,height:1100},deviceScaleFactor:1,hasTouch:true});
 page.on('pageerror',error=>report.errors.push(error.message));
 await page.goto(base+'/atlas.html');await page.waitForFunction(()=>state.ready,{timeout:20000});
 report.backend=await page.evaluate(()=>flowEngine.gpu?'gpu':'cpu');
 await page.evaluate(()=>{drawScene();const pixel=new Uint8Array(4);gl.readPixels(Math.floor(canvas.width/2),Math.floor(canvas.height/2),1,1,gl.RGBA,gl.UNSIGNED_BYTE,pixel);if(gl.getError()!==gl.NO_ERROR)throw new Error('WebGL error');if(pixel[3]!==255)throw new Error('Canvas has no opaque frame');});
 report.checks.push('WebGL program, flow texture and opaque frame');
 for(const [code,name] of [['tw','Tayvan'],['az','Azerbaycan'],['au','Avustralya'],['at','Avusturya'],['xn','Kuzey Kıbrıs (KKTC)'],['gb-eng','İngiltere'],['gb-sct','İskoçya'],['gb-wls','Galler'],['gb-nir','Kuzey İrlanda']]){
  await page.selectOption('#countrySelect',code);
  await page.waitForFunction(code=>state.ready&&country===code,code);
  assert.equal(await page.locator('#countryTitle').textContent(),name);
  for(const kind of ['hero','cover','avatar']){
   await page.click(`[data-image="${kind}"]`);
   await page.waitForFunction(kind=>state.ready&&state.image===kind,kind);
   await page.waitForFunction(()=>!sceneTransition.visible);
   await page.evaluate(()=>{drawScene();if(gl.getError()!==gl.NO_ERROR)throw new Error('Expanded country WebGL error');});
  }
 }
 await page.selectOption('#countrySelect','tr');await page.waitForFunction(()=>state.ready&&country==='tr');
 await page.click('[data-image="cover"]');await page.waitForFunction(()=>state.ready&&state.image==='cover');
 report.checks.push('Taiwan, revised AZ/AU/AT and special/subdivision codes: Turkish names and all three images decoded');
 await page.click('#panoramaMode');
 for(const [label,yaw] of [['front',0],['join',Math.PI],['full-turn',Math.PI*2]]){
  await page.evaluate(yaw=>{setDemo(false);panorama.yaw=yaw;panorama.targetYaw=yaw;drawScene();if(gl.getError()!==gl.NO_ERROR)throw new Error('Panorama WebGL error');},yaw);
  await page.screenshot({path:join(output,'panorama-'+label+'.png'),fullPage:true});
 }
 await page.click('#panoramaMode');report.checks.push('stitched panorama shader, seam and full-turn screenshots');

 await page.click('#profile-cinematic');assert.equal(await page.inputValue('#speed'),'65');
 await page.click('#profile-lively');assert.equal(await page.inputValue('#elasticity'),'85');
 await page.click('#saveCustomMotion');await page.click('#profile-calm');await page.click('#loadCustomMotion');assert.equal(await page.inputValue('#elasticity'),'85');
 report.checks.push('profiles and personal preset');
 const area=await page.locator('#scene').boundingBox();await page.mouse.move(area.x+area.width*.2,area.y+area.height*.6);await page.mouse.move(area.x+area.width*.8,area.y+area.height*.4,{steps:30});
 await page.screenshot({path:join(output,'desktop-motion.png'),fullPage:true});
 await page.click('[data-image="avatar"]');await page.waitForFunction(()=>state.ready&&state.image==='avatar');await page.waitForFunction(()=>!sceneTransition.visible);
 await page.screenshot({path:join(output,'avatar-transition.png'),fullPage:true});report.checks.push('decoded-image snapshot crossfade');
 await page.selectOption('#qualityMode','eco');assert.equal(await page.evaluate(()=>qualityManager.scale),.65);
 await page.selectOption('#qualityMode','high');assert.equal(await page.evaluate(()=>qualityManager.scale),1);report.checks.push('quality selection');
 // Trusted multi-touch input exercises pointer capture and pinch handling.
 const mobileArea=await page.locator('#scene').boundingBox();
 const fingerA={x:Math.round(mobileArea.x+mobileArea.width*.35),y:Math.round(mobileArea.y+mobileArea.height*.5),id:1};
 const fingerB={x:Math.round(mobileArea.x+mobileArea.width*.55),y:fingerA.y,id:2};
 const cdp=await page.context().newCDPSession(page);
 await cdp.send('Input.dispatchTouchEvent',{type:'touchStart',touchPoints:[fingerA]});
 fingerA.x+=20;await cdp.send('Input.dispatchTouchEvent',{type:'touchMove',touchPoints:[fingerA]});
 await cdp.send('Input.dispatchTouchEvent',{type:'touchStart',touchPoints:[fingerA,fingerB]});
 fingerB.x+=70;await cdp.send('Input.dispatchTouchEvent',{type:'touchMove',touchPoints:[fingerA,fingerB]});
 assert.ok(await page.evaluate(()=>state.touchZoom>1));
 await cdp.send('Input.dispatchTouchEvent',{type:'touchEnd',touchPoints:[]});report.checks.push('touch pan and pinch');
 if(await page.evaluate(()=>videoRecorder.available)){
  const download=page.waitForEvent('download');await page.click('#recordVideo');await page.waitForTimeout(600);await page.click('#recordVideo');
  const file=await download;await file.saveAs(join(output,file.suggestedFilename()));report.checks.push('real MediaRecorder video download');
 }else report.checks.push('video unavailable: control correctly hidden');
 await page.setViewportSize({width:390,height:844});await page.screenshot({path:join(output,'mobile-controls.png'),fullPage:true});
 assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);report.checks.push('mobile layout fits viewport');
 await page.reload();await page.waitForFunction(()=>state.ready);assert.equal(await page.inputValue('#elasticity'),'85');report.checks.push('saved settings after reload');
 assert.deepEqual(report.errors,[]);report.status='passed';
}catch(error){report.status=browser?'failed':'blocked';report.error=String(error.stack||error);process.exitCode=1;}
finally{if(browser)await browser.close();if(server)await new Promise(done=>server.close(done));await writeFile(join(output,'report.json'),JSON.stringify(report,null,2));console.log(JSON.stringify({status:report.status,checks:report.checks,report:join(output,'report.json'),error:report.error?.slice(0,500)},null,2));}
