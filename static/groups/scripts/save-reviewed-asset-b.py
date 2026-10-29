"""Persist reviewed source provenance and hand-authored approximate depth polygons."""
import json,sys,subprocess
from pathlib import Path
code,kind,source,observations,regions=sys.argv[1:]
prompt=Path(f'production/{code}_{kind}_prompt.txt').read_text()
Path(f'production/{code}_{kind}_source.json').write_text(json.dumps({'path':source,'prompt':prompt,'tool':'built-in image_gen'},indent=2))
spec={'output':f'depth/{code}_{kind}.png','width':768 if kind!='avatar' else 512,'height':256 if kind=='cover' else 512,'approximate':True,'stops':[[0,10],[.25,32],[.5,88],[.75,155],[1,228]],'regions':json.loads(regions)}
Path(f'data/depth-regions/{code}_{kind}.json').write_text(json.dumps(spec,indent=2))
subprocess.run(['python3','scripts/depth-map.py',f'data/depth-regions/{code}_{kind}.json'],check=True)
rpath=Path(f'production/{code}_review.json');r=json.loads(rpath.read_text()) if rpath.exists() else {}
r[kind]={'status':'reviewed','observations':observations,'depth':'approximate hand-authored subject masks; not measured geometry'}
rpath.write_text(json.dumps(r,indent=2))
