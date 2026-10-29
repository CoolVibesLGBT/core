"""Save reviewed group-3 generated artwork and numeric approximate depth metadata."""
from pathlib import Path
import json, subprocess, sys
code,kind,source,region_json,prompt=sys.argv[1:]
size={'hero':(1920,1280),'cover':(1500,500),'avatar':(512,512)}[kind]
subprocess.run([sys.executable,'scripts/encode-avif.py',source,f'images/{code}_{kind}.avif',*map(str,size)],check=True)
Path(f'production/{code}_{kind}_source.json').write_text(json.dumps({'path':source,'prompt':prompt},ensure_ascii=False,indent=2))
spec={'output':f'depth/{code}_{kind}.png','width':768 if kind!='avatar' else 512,'height':{'hero':512,'cover':256,'avatar':512}[kind],'stops':[[0,15],[.25,40],[.5,100],[1,230]],'regions':json.loads(region_json)}
p=Path(f'data/depth-regions/{code}_{kind}.json');p.parent.mkdir(parents=True,exist_ok=True);p.write_text(json.dumps(spec,indent=2))
subprocess.run([sys.executable,'scripts/depth-map.py',str(p)],check=True)
